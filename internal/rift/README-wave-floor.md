# Recovering wave-floor panels (0479 in progress)

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

Still required: authored rooms with verified bypasses for
all supported enemy sizes, rendering and sound cues, minimap and preview,
accessible warning/occupancy guidance, and real browser combat/loot/completion
journeys. No campaign room uses the new panels yet. Keep ledger 0479 open.
