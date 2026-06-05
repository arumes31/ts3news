#!/bin/bash
# Entry point: each cycle, connect the official TeamSpeak 3 client to the server
# on 9987, run one poke cycle (the Go bot), then disconnect. Repeat on an interval.
set -e

export HOME=/root
export XDG_RUNTIME_DIR=/tmp/runtime-root
mkdir -p "$XDG_RUNTIME_DIR"; chmod 700 "$XDG_RUNTIME_DIR"

# Headless / software-rendering settings for the Qt + WebEngine client
export QT_QPA_PLATFORM=xcb
export QT_XCB_GL_INTEGRATION=none
export LIBGL_ALWAYS_SOFTWARE=1
export QTWEBENGINE_DISABLE_SANDBOX=1
export QTWEBENGINE_CHROMIUM_FLAGS="--no-sandbox --disable-gpu --disable-dev-shm-usage"
export LD_LIBRARY_PATH=/opt/ts3

# Load config.env (CR-stripped) but do NOT overwrite existing environment variables.
# This ensures that variables set in docker-compose.yml 'environment' take precedence.
if [ -f /app/config.env ]; then
  echo "[entrypoint] Loading config.env (preserving existing environment)..."
  while IFS='=' read -r key value || [ -n "$key" ]; do
    # Skip comments and empty lines
    [[ "$key" =~ ^[[:space:]]*#.*$ || -z "$key" ]] && continue
    # Trim whitespace from key and value
    key=$(echo "$key" | xargs)
    value=$(echo "$value" | sed "s/^[[:space:]]*[\"']//;s/[\"'][[:space:]]*$//" | xargs)
    if [ -n "$key" ] && [ -z "${!key}" ]; then
      export "$key=$value"
    fi
  done < <(sed 's/\r$//' /app/config.env)
fi

: "${TS3_HOST:?TS3_HOST is required}"
: "${TS3_PORT:=9987}"
: "${CHECK_INTERVAL_HOURS:=12}"
NICK="${TS3_NICKNAME:-MrFree}"
INTERVAL_SECONDS=$(( CHECK_INTERVAL_HOURS * 3600 ))

# Inject the configured (level-29) identity into the baked client profile (once).
if [ -n "${TS3_IDENTITY:-}" ]; then
  echo "[entrypoint] Injecting identity into settings.db..."
  python3 /opt/inject_identity.py "$TS3_IDENTITY" /root/.ts3client/settings.db || echo "[entrypoint] WARNING: identity injection failed"
else
  echo "[entrypoint] WARNING: TS3_IDENTITY not set; using the profile's default identity"
fi

# Virtual display + dbus session (shared across all cycles)
Xvfb :99 -screen 0 1280x720x24 -ac >/tmp/xvfb.log 2>&1 &
sleep 2
export DISPLAY=:99
eval "$(dbus-launch --sh-syntax)"

URL="ts3server://${TS3_HOST}?port=${TS3_PORT}&nickname=${NICK}"
TS3PID=""
stop_client() {
  if [ -n "$TS3PID" ]; then
    echo "[entrypoint] Stopping TS3 client (PID $TS3PID)..."
    kill "$TS3PID" 2>/dev/null || true
    # Give it 5 seconds to exit gracefully
    for _ in 1 2 3 4 5; do
      if ! kill -0 "$TS3PID" 2>/dev/null; then break; fi
      sleep 1
    done
    # Force kill if still running
    if kill -0 "$TS3PID" 2>/dev/null; then
       echo "[entrypoint] TS3 client still running, sending SIGKILL..."
       kill -9 "$TS3PID" 2>/dev/null || true
    fi
    wait "$TS3PID" 2>/dev/null || true
  fi
  TS3PID=""
}
trap 'stop_client; exit 0' INT TERM

run_cycle() {
  echo "[entrypoint] Connecting to ${TS3_HOST}:${TS3_PORT} as ${NICK}..."
  : > /tmp/ts3.log
  # Run directly to capture correct PID
  /opt/ts3/ts3client_linux_amd64 -nosingleinstance "$URL" >/tmp/ts3.log 2>&1 &
  TS3PID=$!

  # Clear any first-run/promo popups that could appear.
  ( for _ in 1 2 3 4; do xdotool key --clearmodifiers Escape 2>/dev/null || true; sleep 2; done ) &

  # The bot waits (via ClientQuery) until the client is connected, then pokes.
  /usr/local/bin/bot || echo "[entrypoint] bot exited with an error"

  echo "[entrypoint] Poke cycle done; disconnecting client."
  stop_client
}

while true; do
  run_cycle
  echo "[entrypoint] Sleeping ${INTERVAL_SECONDS}s until next cycle."
  sleep "$INTERVAL_SECONDS"
done
