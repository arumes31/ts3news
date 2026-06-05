# ts3-free-game-bot

A Dockerized TeamSpeak 3 bot that pokes users about free Steam/Epic games.

## How it works

The TeamSpeak **SDK** client cannot connect to a retail TeamSpeak 3 server (they
use incompatible protocols/licensing), so this image instead runs the **official
TeamSpeak 3 client headless** (under Xvfb) and connects it to the server on the
voice port (UDP 9987). A small Go bot then drives that client through the
**ClientQuery** plugin (telnet on `127.0.0.1:25639`) to poke users.

```
┌─────────────────────── container ───────────────────────┐
│  Xvfb ── ts3client (official) ──UDP 9987──► TS3 server   │
│                  │ ClientQuery :25639                    │
│                  ▼                                        │
│   Go bot ── fetches free games ── clientpoke ──► users   │
└──────────────────────────────────────────────────────────┘
```

The server requires an identity **security level 29**; the client's profile is
baked into the image with the license pre-accepted and the ClientQuery plugin
installed, and the configured identity is injected into the client's
`settings.db` at startup.

## Setup

1. Edit `config.env`:
   ```
   TS3_HOST=217.154.216.239        # public IP of the server (UDP 9987)
   TS3_PORT=9987
   TS3_NICKNAME=MrFree
   TS3_IDENTITY=<exported identity string with the required security level>
   CHECK_INTERVAL_HOURS=12
   TS3_TARGET_NICK=Daniel          # testing: only poke this nickname (empty = everyone)
   ```
2. `docker compose up -d --build`

## Notes

- `TS3_IDENTITY` is the identity string from a TeamSpeak client identity export
  (the `identity="..."` value). It must already meet the server's security level.
- `TS3_TARGET_NICK` restricts pokes to a single nickname — handy for testing.
  Leave it empty in production to poke every online user.
- Logs show the connection progress (`Connection established`) and each poke sent.
