# Narrow bridges (verification in progress)

Tier 1 of missions 4, 14, ..., 94 now contains a saved wooden deck before its
combat obstacles. The deck spans X=230..410 with regional Y variation. Actor
footprints stay between the rails, including during jumps, dashes and knockback.
Projectiles can cross the open gaps. Wood footsteps use the existing audio path.

Movement tests cover large steps, both directions, actor clearance, jumping,
knockback, legacy rooms and saved geometry. Pursuers align with an entry before
crossing. Actual combat-loop tests also verify goblin, wolf and shield-carrier
pursuit in both directions with normal cooldowns. Enemy placement and ledge landing checks reject gaps. Campaign checks
verify all ten entrances/spawns, clear approaches, footstep surfaces and detached
copies. Rail-defeat tests also confirm that wolf, ordinary enemy and boss drops
can be reached and collected at both rails while another enemy remains alive;
the test does not rely on end-of-room automatic collection. The mission 74 hazard switch sits on the entrance bank at X=180 so it
remains reachable; the full Rift suite passed after fixing that conflict.

Server and browser validators bound bridge dimensions, count, IDs and separation.
The renderer uses the saved deck rectangle for planks, rail boundaries and dark
gaps. Minimap and mission previews identify the walkable deck.

Pending real-browser verification: tests/e2e/rift-bridges.spec.js exercises WASD,
Space, rails, crossing, recovery, reduced motion, mobile screenshots and campaign
preview. These tests are prepared and syntax-checked, not yet executed. Inspect
screenshots and verify actual campaign combat/loot and completion before marking
ledger 0407 complete. Engine tests alone do not prove the entire player journey.
