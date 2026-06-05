# TeamSpeak 3 "free games" poke bot.
#
# The TeamSpeak SDK client cannot connect to a retail TS3 server, so this image
# runs the official TeamSpeak 3 client headless (Xvfb) connected to the server on
# 9987, and a small Go bot drives it through the ClientQuery plugin to poke users.

# ---- Stage 1: build the Go bot (pure Go, no cgo) ----
FROM golang:1.26-bookworm AS gobuilder
WORKDIR /app
COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -o /bot ./cmd/bot

# ---- Stage 2: download + extract the official TeamSpeak 3 client ----
FROM debian:bookworm-slim AS tsclient
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*
ARG TS3_VERSION=3.6.2
# The .run is a makeself archive; `yes |` auto-accepts the license prompt, --noexec
# extracts without running the installer script.
RUN curl -fsSL -o /tmp/ts3.run \
      "https://files.teamspeak-services.com/releases/client/${TS3_VERSION}/TeamSpeak3-Client-linux_amd64-${TS3_VERSION}.run" \
 && (yes | sh /tmp/ts3.run --noexec --keep --target /opt/ts3 >/dev/null 2>&1 || true) \
 && test -f /opt/ts3/ts3client_linux_amd64

# ---- Stage 3: runtime ----
FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    xvfb dbus dbus-x11 xdotool python3 sqlite3 ca-certificates fonts-dejavu-core procps \
    libxcb1 libxcb-render0 libxcb-render-util0 libxcb-shape0 libxcb-shm0 \
    libxcb-icccm4 libxcb-image0 libxcb-keysyms1 libxcb-randr0 libxcb-xfixes0 \
    libxcb-sync1 libxcb-xinerama0 libxcb-util1 libxcb-glx0 libxcb-xkb1 libxcb-cursor0 \
    libxkbcommon0 libxkbcommon-x11-0 libfontconfig1 libfreetype6 libdbus-1-3 \
    libnss3 libglib2.0-0 libgl1 libegl1 libpulse0 libasound2 libxi6 libxtst6 \
    libxrender1 libxrandr2 libxcomposite1 libxdamage1 libxcursor1 \
    libevent-2.1-7 libsm6 libice6 libxext6 libharfbuzz0b libpng16-16 \
    libpci3 libxslt1.1 liblcms2-2 libatomic1 \
    && rm -rf /var/lib/apt/lists/*

COPY --from=tsclient /opt/ts3 /opt/ts3
COPY --from=gobuilder /bot /usr/local/bin/bot

# Baked "golden" client profile: license accepted + ClientQuery plugin installed.
COPY runtime/ts3profile.tgz /tmp/ts3profile.tgz
RUN tar xzf /tmp/ts3profile.tgz -C /root && rm /tmp/ts3profile.tgz

COPY runtime/inject_identity.py /opt/inject_identity.py
COPY runtime/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /opt/ts3/ts3client_linux_amd64

WORKDIR /app
ENTRYPOINT ["/entrypoint.sh"]
