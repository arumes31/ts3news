#!/bin/bash
# Entry point: prepare the headless environment (virtual display, dbus, identity)
# and then hand off to the Go bot, which now owns the full lifecycle — launching
# the TeamSpeak client each cycle, the watchdog, graceful shutdown and the random
# interval loop. We `exec` the bot so it becomes PID 1 and receives SIGTERM
# directly (graceful shutdown).
set -e

export HOME=/root
export XDG_RUNTIME_DIR=/tmp/runtime-root
mkdir -p "$XDG_RUNTIME_DIR"; chmod 700 "$XDG_RUNTIME_DIR"

# Headless / software-rendering settings for the Qt + WebEngine client.
export QT_QPA_PLATFORM=xcb
export QT_XCB_GL_INTEGRATION=none
export LIBGL_ALWAYS_SOFTWARE=1
export QTWEBENGINE_DISABLE_SANDBOX=1
export QTWEBENGINE_CHROMIUM_FLAGS="--no-sandbox --disable-gpu --disable-dev-shm-usage"
export LD_LIBRARY_PATH=/opt/ts3

# Load config.env (CR-stripped) without overwriting variables already in the
# environment (docker-compose env_file / environment take precedence).
if [ -f /app/config.env ]; then
  echo "[entrypoint] Loading config.env (preserving existing environment)..."
  while IFS='=' read -r key value || [ -n "$key" ]; do
    [[ "$key" =~ ^[[:space:]]*#.*$ || -z "$key" ]] && continue
    key=$(echo "$key" | xargs)
    value=$(echo "$value" | sed "s/^[[:space:]]*[\"']//;s/[\"'][[:space:]]*$//" | xargs)
    if [ -n "$key" ] && [ -z "${!key}" ]; then
      export "$key=$value"
    fi
  done < <(sed 's/\r$//' /app/config.env)
fi

: "${TS3_HOST:?TS3_HOST is required}"
: "${TS3_PORT:=9987}"

# Seed the golden client profile into HOME if it is missing. This happens when a
# fresh named volume is mounted over ~/.ts3client (e.g. the Idely container's
# idely_profile): without seeding, the client would start with no accepted
# license or ClientQuery plugin. Copying the baked profile in keeps a working,
# license-accepted profile and lets the volume persist the identity across
# restarts. When the baked profile is already present (no volume), this is a no-op.
if [ ! -f "$HOME/.ts3client/settings.db" ] && [ -f /opt/ts3profile.tgz ]; then
  echo "[entrypoint] Seeding golden TS3 client profile into $HOME/.ts3client..."
  tar xzf /opt/ts3profile.tgz -C "$HOME"
fi

# Inject the configured identity into the client profile. When unset, the profile
# keeps its own identity — fine for the Idely client (IDELY_IDENTITY is optional),
# but the poke bot should set TS3_IDENTITY to its own (ideally leveled) identity.
if [ -n "${TS3_IDENTITY:-}" ]; then
  echo "[entrypoint] Injecting identity into settings.db..."
  python3 /opt/inject_identity.py "$TS3_IDENTITY" /root/.ts3client/settings.db || echo "[entrypoint] WARNING: identity injection failed"
else
  echo "[entrypoint] TS3_IDENTITY not set; using the profile's own identity."
fi

# Virtual display + dbus session (shared across all cycles).
# Clear stale X locks/sockets left by a previous run (a `docker restart` reuses
# the container filesystem, so an old /tmp/.X99-lock would stop Xvfb from
# starting and the client would exit immediately with no display).
echo "[entrypoint] Clearing stale X locks and QtSingleApplication sockets..."
rm -f /tmp/.X99-lock 2>/dev/null || true
rm -rf /tmp/.X11-unix 2>/dev/null || true
mkdir -p /tmp/.X11-unix && chmod 1777 /tmp/.X11-unix
rm -f /tmp/qtsingleapp-* 2>/dev/null || true

echo "[entrypoint] Starting Xvfb + dbus..."
Xvfb :99 -screen 0 1280x720x24 -ac >/tmp/xvfb.log 2>&1 &
sleep 3
export DISPLAY=:99
eval "$(dbus-launch --sh-syntax)"

echo "[entrypoint] Handing off to the bot supervisor..."
exec /usr/local/bin/bot
