# Volatile terrain reactions (engine groundwork)

Status: engine implemented and unit tested; **not enabled in campaign rooms**.
Ledger item 0476 remains open until authored rooms,
presentation, audio and real-browser verification are done.

Breaking volatile wooden cover arms a connected cluster of up to eight props.
Every selected prop becomes non-solid and receives the same 1.2-second fuse.
Ignition links use the existing blast ellipse (115 world units horizontally,
55 vertically). Static obstacles, high cover, stone and ordinary intact wooden
cover stop links. No recursive explosion calls or growing per-actor maps exist.

The tick helper consumes expired fuses before applying any damage. All blast
ellipses expiring in one tick form one contact set: each living grounded actor
can receive at most one dose, even in overlapping circles. Damage is 12–21 before
normal defenses, clamped by campaign region. Solid cover blocks blast contact.
Jumping, burrowed enemies and objective props are excluded. Player damage uses
normal guard, dodge, grace, barrier and named-hazard accounting. Enemy damage
uses the environmental path, preserving normal deaths and loot without player
combo, attack-chain or damage-stat credit. Independent later detonations may
hit the same actor again; this is not permanent immunity.

`volatile` and `blast_fuse` are saved on TerrainCover. HP zero and fuse zero means
spent. A saved warning resumes with its remaining time; spent props cannot fire
again. The helper does nothing while paused or outside fighting status. Invalid
durations cannot advance it. Legacy cover has neither field and is unchanged.

The combat loop advances existing warnings after movement and jump, before
attacks can arm new clusters. A triggering melee attack or projectile retains
the full warning. Lethal blasts prevent a subsequent attack or cast that tick.
The browser protocol rejects non-finite/out-of-range fuses, volatile stone and
active fuses on intact or nonvolatile cover. No authored cover is volatile yet,
keeping incomplete warning presentation out of gameplay.

## Remaining integration

- Author compact clusters in selected rooms with escape space, entrances and
  exits clear; keep room geometry frozen in saved expeditions.
- Clearly mark intact volatile cover and show the exact blast ellipses while
  arming, including reduced-motion and pause behavior, HUD and minimap cues.
- Provide original marked-cover art if existing terrain art cannot communicate
  the mechanic, preserving the generator prompt beside the output.
- Add warning and detonation sounds, replay deduplication and silent recovery.
- Verify melee/projectile triggering, save/reload mid-warning, terrain collision,
  enemy loot and simultaneous player/enemy death through the real game loop.
- Browser-check at desktop and mobile sizes once the active memory capture ends.

## Verification

`go test ./internal/rift -run '^TestTerrainReaction' -count=1`

Tests cover connected warnings, saved fuses, overlapping damage, walls, ordinary
wood, the eight-prop hard cap, invalid input, jump/dodge/grace escape, protected
enemies, pause/cleared states, single kill/drop, lethal attribution and the
absolute regional damage cap. Combat-loop tests also cover melee/projectile warning timing, a last-tick jump
and lethal-tick attack prevention. The public protocol tests run with
`node --test tests/performance/terrain-protocol.test.cjs`. These checks do not
prove the unfinished player-facing feature.
