# Recovering wave-floor panels (0479 verified)

The engine supports authored FragileFloor rectangles and saved FloorSegments
inside a survive_waves objective. Each panel warns for three combat seconds,
with later panels staggered by 0.6 seconds. A panel waits for every living player
or enemy footprint to leave before it collapses. A jump above it still counts
as occupancy. Waiting causes no damage or forced relocation.

A collapsed panel removes walkable ground: walking, jumps, dodges and knockback
cannot cross its ground footprint. Projectiles still cross it. Pursuers use the
existing obstacle detour logic to take an upper or lower route. Ground checks
also apply to placement and path checks. Missing ground is derived from saved
objective state; it never modifies the frozen level definition or becomes
projectile-blocking cover.

All panels rebuild as soon as a wave is defeated, including the last wave. They
remain solid throughout intermission and restart their warning for the next
wave. Pause freezes the timer. Warning, collapse and rebuild each emit one event
per panel per wave for the forthcoming renderer/audio integration.

Focused tests cover warning and collapsed-state recovery, pause, occupied
panels, swept movement, jumps, knockback, both pursuit directions, projectile
paths, independent geometry, all three repair cycles and one-shot events.
The full Rift suite passed after this foundation change
(go test ./internal/rift -count=1).

Server and browser validation now bound panels to four separated rectangles,
with upper/lower floor margins and finite geometry. Active state must match the
frozen room panels exactly. Invalid countdowns, missing panels and collapsed
intermission/final-clear panels are rejected. Legacy rooms remain valid; content
updates do not substitute the current campaign definition into saved runs.
Focused decode tests and nine terrain protocol tests pass.

Tier 2 of missions 5, 15, ..., 95 now includes two panels. Authored movement
tests exercise both side routes with the player, boss, wolf and ordinary enemy
footprints, including a closed wave gate. Both routes remain walkable. They are
not immunity lanes: ordinary enemy attacks and floor hazards still apply.
Intermission repair also makes drops on a formerly missing panel reachable by
the normal pickup mechanism.

Original procedural cracked-tile and gap graphics show warning, occupied,
missing and rebuilt states. The minimap, mission preview and objective directions
explain the same rules. Distinct synthesized warning, collapse and rebuild cues
use the shared audio controls. Reduced motion preserves essential state changes.

Four focused browser cases passed: normal/reduced-motion collapse, swept jump
blocking, keyboard bypass, save/recovery, malformed geometry rejection, repair,
mobile layout, sound playback, preview and waiting for an occupying player.
Desktop/mobile captures were inspected under test-results/brawl-wave-floor-verified
and test-results/brawl-wave-floor-occupied. The final label check passed, and its screenshot was inspected under
test-results/brawl-wave-floor-label; waiting text is readable above the fighter.

The separate normal-combat mission 5 journey passed in
tests/e2e/rift-terrain-campaign-completion.spec.js. It observed collapsed panels,
completed all three tiers, defeated 18 enemies including one boss and banked
625 gold. Results are local under test-results/brawl-wave-floor-campaign. The
controlled one-hit terrain fixture is not used as evidence of full combat;
this campaign journey uses normal engine health and combat. Banking uses the
E2E fixture, with production persistence covered separately.

The full Rift suite passed with the authored panels. This completes ledger 0479.
