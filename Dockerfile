# TeamSpeak 3 "free games" poke bot.
#
# The TeamSpeak SDK client cannot connect to a retail TS3 server, so this image
# runs the official TeamSpeak 3 client headless (Xvfb) connected to the server on
# 9987, and a small Go bot drives it through the ClientQuery plugin to poke users.

# ---- Stage 1: build the Go bot (pure Go, no cgo) ----
FROM golang:1.26.6-bookworm@sha256:116d58cbd88c1297624acc6e967a060012422bacf9930927e23fb719189c6f36 AS gobuilder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
# -ldflags "-s -w" strips debug info; embedded migrations (internal/db/migrations)
# are baked into the binary.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags "-s -w" -o /bot ./cmd/bot

# ---- Stage 2: download + extract the official TeamSpeak 3 client ----
FROM debian:bookworm-slim@sha256:88200866dfff7ea7f5cbcb6ec7c8a701889efe6fe859fe64d6990e4b07ea4171 AS tsclient
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates \
    && rm -rf /var/lib/apt/lists/*
ARG TS3_VERSION=3.6.2
ARG TS3_SHA256=59f110438971a23f904a700e7dd0a811cf99d4e6b975ba3aa45962d43b006422
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
# The .run is a makeself archive; one explicit "y" accepts the license prompt,
# while --noexec extracts without running the installer script.
RUN curl -fsSL -o /tmp/ts3.run \
      "https://files.teamspeak-services.com/releases/client/${TS3_VERSION}/TeamSpeak3-Client-linux_amd64-${TS3_VERSION}.run" \
 && echo "${TS3_SHA256}  /tmp/ts3.run" | sha256sum --check --strict \
 && printf 'y\n' | sh /tmp/ts3.run --noexec --keep --target /opt/ts3 >/dev/null 2>&1 \
 && test -f /opt/ts3/ts3client_linux_amd64

# ---- Stage 3: runtime ----
FROM debian:bookworm-slim@sha256:88200866dfff7ea7f5cbcb6ec7c8a701889efe6fe859fe64d6990e4b07ea4171
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
# Keep the tarball at /opt so the entrypoint can re-seed it into a fresh named
# volume mounted over ~/.ts3client (e.g. the Idely container's idely_profile).
COPY runtime/ts3profile.tgz /opt/ts3profile.tgz
RUN groupadd --gid 10001 ts3bot \
 && useradd --uid 10001 --gid ts3bot --create-home --shell /usr/sbin/nologin ts3bot \
 && tar xzf /opt/ts3profile.tgz -C /home/ts3bot \
 && chown -R ts3bot:ts3bot /home/ts3bot

COPY runtime/inject_identity.py /opt/inject_identity.py
COPY runtime/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh /opt/ts3/ts3client_linux_amd64 \
 && install -d -o ts3bot -g ts3bot /app /app/data

WORKDIR /app
ENV HOME=/home/ts3bot
USER 10001:10001
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
  CMD pgrep -x bot >/dev/null || exit 1
ENTRYPOINT ["/entrypoint.sh"]
