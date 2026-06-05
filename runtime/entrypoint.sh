#!/bin/bash
# Entry point: run the official TeamSpeak 3 client headless, connected to the
# configured server on 9987, then run the poke bot which talks to it via ClientQuery.
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

# Load config.env (so TS3_HOST/PORT/IDENTITY are available). Strip CR so the file
# works even when edited on Windows (CRLF line endings).
if [ -f /app/config.env ]; then
  set -a
  . <(sed 's/\r$//' /app/config.env)
  set +a
fi

: "${TS3_HOST:?TS3_HOST is required}"
: "${TS3_PORT:=9987}"
NICK="${TS3_NICKNAME:-MrFree}"

# Inject the configured (level-29) identity into the baked client profile.
if [ -n "${TS3_IDENTITY:-}" ]; then
  echo "[entrypoint] Injecting identity into settings.db..."
  python3 /opt/inject_identity.py "$TS3_IDENTITY" /root/.ts3client/settings.db || echo "[entrypoint] WARNING: identity injection failed"
else
  echo "[entrypoint] WARNING: TS3_IDENTITY not set; using the profile's default identity"
fi

# Virtual display + dbus session
Xvfb :99 -screen 0 1280x720x24 -ac >/tmp/xvfb.log 2>&1 &
sleep 2
export DISPLAY=:99
eval "$(dbus-launch --sh-syntax)"

URL="ts3server://${TS3_HOST}?port=${TS3_PORT}&nickname=${NICK}"
echo "[entrypoint] Launching TeamSpeak client -> ${TS3_HOST}:${TS3_PORT} as ${NICK}"
cd /opt/ts3
./ts3client_linux_amd64 -nosingleinstance "$URL" >/tmp/ts3.log 2>&1 &
TS3PID=$!
trap 'kill $TS3PID 2>/dev/null || true' EXIT

# Dismiss any first-run/promo popups periodically so they never pile up (the
# connection and ClientQuery work regardless, but this keeps the UI clean).
( for _ in $(seq 1 8); do xdotool key --clearmodifiers Escape 2>/dev/null || true; sleep 2; done ) &

# Wait for the connection to be established (or fail loudly after a while)
echo "[entrypoint] Waiting for connection to be established..."
for i in $(seq 1 40); do
  if grep -q "Connection established" /root/.ts3client/logs/*.log 2>/dev/null; then
    echo "[entrypoint] TeamSpeak connection established."
    break
  fi
  if ! kill -0 $TS3PID 2>/dev/null; then
    echo "[entrypoint] TeamSpeak client exited unexpectedly:"; tail -20 /tmp/ts3.log; exit 1
  fi
  sleep 2
done

echo "[entrypoint] Starting bot..."
exec /usr/local/bin/bot
