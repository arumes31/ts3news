# Volatile terrain reactions

Status: implemented in nine campaign rooms and verified in the browser.
Ledger item 0476 is complete. Engine tests establish bounded propagation and
damage; browser tests verify the player-facing path and recovery.

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
active fuses on intact or nonvolatile cover. Missions 11, 21, 31, 41, 51, 61, 71, 81 and 91 now have a three-crate
cluster in tier 1. Existing saved room geometry is unchanged.

## Browser verification

Four tests in `tests/e2e/rift-terrain-reaction.spec.js` pass against the dedicated
fixture on port 18098 (`test-results/terrain-reaction-readable`). They cover:

- Real keyboard melee ignition, the shared warning, pause beyond its fuse,
  save/reload mid-warning, silent recovery, three detonation cues, one damage
  dose per target, a normal enemy kill and loot, and no repeated damage.
- No crate-art download at idle or for ordinary cover; preparation includes
  future mission tiers and reuses the decoded image across tiers.
- Failed art loading offers recovery, preserves the saved expedition, and
  prevents combat until required decoding completes.
- Actual mission 11's preview and three authored marked crates.

Desktop and 390px mobile/reduced-motion screenshots were inspected. Additional
`-world.png` captures hide only the pause panel for render inspection; companion
captures retain the real UI. Warning instructions maintain 12 CSS pixels times
the player's text setting, countdowns 10, with measured instruction-width edge
clamping. Blast ellipses retain the engine's exact world dimensions. The spent
state leaves visible debris. Existing cover/collision/projectile browser tests
and event-sound synthesis checks also passed in the preceding combined run.

The first browser run exposed a fixture-only null loot list, which the protocol
correctly rejected. The fixture now supplies an empty array. Projectile fuse
timing, lethal-tick attack prevention and defenses are covered by engine tests;
the browser path above uses melee. Geometry tests verify all nine authored
clusters and escape lanes, not every possible combined hazard arrangement.

## Verification

`go test ./internal/rift -run '^TestTerrainReaction' -count=1`

Tests cover connected warnings, saved fuses, overlapping damage, walls, ordinary
wood, the eight-prop hard cap, invalid input, jump/dodge/grace escape, protected
enemies, pause/cleared states, single kill/drop, lethal attribution and the
absolute regional damage cap. Combat-loop tests also cover melee/projectile warning timing, a last-tick jump
and lethal-tick attack prevention. The public protocol tests run with
`node --test tests/performance/terrain-protocol.test.cjs`. Browser evidence is described above.

## Original artwork

`internal/bot/webassets/rift_volatile_cover.png` is an original transparent
three-state sheet generated with the built-in image tool. The exact prompt and
SHA-256 are in the adjacent `.prompt.json`. The generator returned 2172 × 724
RGBA (three 724-pixel cells), rather than the requested 1536 × 512. The original
alpha is preserved without image editing. The cells show intact, cracked/glowing
and spent cover. Each has a triangular spark danger mark where applicable.

Alpha >= 16 bounds relative to each cell: intact (126,214)-(590,627), cracked
(125,214)-(604,626), spent (74,344)-(657,629). Use a common crop and scale with a
shared baseline; do not stretch each body independently. Small-size readability was inspected in the browser captures above.

Presentation wiring uses a shared (64,200,600,440) crop per 724-pixel cell and a
uniform scale, anchoring all three states to the same baseline. It preloads the
sheet through the existing per-expedition art preparation only when a frozen
room has volatile cover. Intact/cracked/spent states derive from HP and fuse.
Ground warnings use the authoritative fuse and the same 115 × 55 radii as the
engine; no new animation timer is created. Existing fire atlas frames and a
brief ellipse provide the explosion. Audio uses short synthesized warning and
blast envelopes within the existing voice lifecycle and event deduplication.
The browser checks above exercise this integration, including synthesized cues.

Warning presentation uses one shared visible escape instruction plus a short
countdown on each armed crate, avoiding overlapping long labels. The instruction
stays inside the viewport. Cleared/ended rooms show consumed crates as spent,
even if their saved fuse was still positive when combat ended; they cannot blast
outside fighting status. These states were visually inspected in the browser captures.
