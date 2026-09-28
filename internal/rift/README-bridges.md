# Narrow bridges (0407 verified)

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

Real-browser verification passed in tests/e2e/rift-bridges.spec.js: WASD,
Space, rails, crossing, recovery, reduced motion, mobile layout and campaign
preview. Desktop/mobile screenshots were inspected. Results remain local under
test-results/brawl-potions-arenas-verification.

The keyboard-driven tests/e2e/rift-terrain-campaign-completion.spec.js also
completed all three tiers of mission 4, including the hold-circle objective and
boss, using normal health, combat and transitions. It reads saved geometry to
navigate but never changes actor state or submits simulated combat outcomes.
The initial run passed with ten kills and no runtime errors. A second run
asserted three cleared rooms, a defeated boss and positive banked gold; it passed.
Results are in test-results/brawl-terrain-completion-rewards. Banking here uses
the E2E fixture; production database persistence has separate coverage. Combined
with the rail-defeat pickup tests, this closes ledger 0407.
