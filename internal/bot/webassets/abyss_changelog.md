# The Abyss — What's New

## 2026-08-26 — A distinct Skill Web

- Every live Skill Web node now has a unique pixel-art cell drawn from one of six discipline atlases, with recognizable silhouettes, high-contrast support, and crisp canvas rendering at every zoom level.
- Atlas generation is deterministic and validated against the complete node catalog, so new nodes cannot silently ship with missing or duplicate artwork.
- The seven Abyss workspaces remain on one keyboard-accessible navigation rail that scrolls cleanly on narrow screens and restores the selected workspace.
- Identified rings can reroll a one-to-three-socket layout for five Void Shards without losing or downgrading fitted gems.

## 2026-08-25 — Combat clarity and reliability

- Live-combat reconnect now resets its event cursor for every new session, preventing stale reconnect loops after descending.
- Combat feedback adds opt-in synthesized hit, cast, heal, ultimate, and defeat cues with saved mute and volume settings.
- Visual impact flashes can be disabled independently and respect the operating system's reduced-motion preference.
- Pixel combatants now use overhead health bars, animated actions, stable monster art, boss variants, and readable hit/heal numbers.
- Forge and shop tools live in dedicated tabs, keeping the main descent flow focused.
- Community tools can read versioned, anonymous run and tier aggregates from `/api/abyss/public/stats`.
- Finished live combats retain their deterministic seed and bounded event history in an owner-authorized replay viewer.
- Live combat now shows the exact action changes remaining each round and preserves the queued or timeout action when that safety budget is exhausted.
