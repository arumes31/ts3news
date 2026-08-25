# The Abyss — What's New

## 2026-08-25 — Combat clarity and reliability

- Live-combat reconnect now resets its event cursor for every new session, preventing stale reconnect loops after descending.
- Combat feedback adds opt-in synthesized hit, cast, heal, ultimate, and defeat cues with saved mute and volume settings.
- Visual impact flashes can be disabled independently and respect the operating system's reduced-motion preference.
- Pixel combatants now use overhead health bars, animated actions, stable monster art, boss variants, and readable hit/heal numbers.
- Forge and shop tools live in dedicated tabs, keeping the main descent flow focused.
- Community tools can read versioned, anonymous run and tier aggregates from `/api/abyss/public/stats`.
- Finished live combats retain their deterministic seed and bounded event history in an owner-authorized replay viewer.
