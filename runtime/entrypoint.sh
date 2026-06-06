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

# Inject the configured (level-29) identity into the baked client profile.
if [ -n "${TS3_IDENTITY:-}" ]; then
  echo "[entrypoint] Injecting identity into settings.db..."
  python3 /opt/inject_identity.py "$TS3_IDENTITY" /root/.ts3client/settings.db || echo "[entrypoint] WARNING: identity injection failed"
else
  echo "[entrypoint] WARNING: TS3_IDENTITY not set; using the profile's default identity"
fi

# Virtual display + dbus session (shared across all cycles).
# Clear stale X locks/sockets left by a previous run (a `docker restart` reuses
# the container filesystem, so an old /tmp/.X99-lock would stop Xvfb from
# starting and the client would exit immediately with no display).
echo "[entrypoint] Clearing stale X locks..."
rm -f /tmp/.X99-lock 2>/dev/null || true
rm -rf /tmp/.X11-unix 2>/dev/null || true
mkdir -p /tmp/.X11-unix && chmod 1777 /tmp/.X11-unix

echo "[entrypoint] Starting Xvfb + dbus..."
Xvfb :99 -screen 0 1280x720x24 -ac >/tmp/xvfb.log 2>&1 &
sleep 3
export DISPLAY=:99
eval "$(dbus-launch --sh-syntax)"

echo "[entrypoint] Handing off to the bot supervisor..."
exec /usr/local/bin/bot
