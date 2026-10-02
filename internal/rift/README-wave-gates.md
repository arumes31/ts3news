# Wave-room gates

The ten authored Survive the waves tiers now have a central temporary gate. Its
24 by 50 world-unit footprint lies between the existing obstacles; upper and
lower routes remain open throughout the encounter. It closes during each living
wave and opens during the reinforcement interval and after the final wave.

Each wave starts with a 0.8-second warning measured by the combat clock. A gate
never closes around the player or a living enemy, including airborne actors and
large bosses. When the warning expires with an occupied threshold, it remains
open until every living actor clears its footprint plus collision radius and a
one-unit margin. It never moves, damages or teleports an occupant. Corpses do not
block closure. Warning, closing and opening each have distinct synthesized sounds;
the world label and objective directions provide the same information visually.

The frozen arena stores the authored threshold. The room objective stores a copy
of that geometry, closed state and remaining warning time. These survive save and
resume. Older saved arenas without the field retain their original terrain.
Campaign copies detach the gate pointer. Closed gates are exposed as tall cover
through Run.Arena, without mutating the frozen high-cover array: projectiles,
melee visibility, AI navigation and normal/jumping movement use that collision.
An additional swept-footprint test prevents a fast dodge from skipping the gate
between movement endpoints. Moving away from a touching boundary stays allowed.

The client validates finite bounded geometry, matching frozen coordinates, boolean
closure, the warning range, and open state during intermission/completion. The gate
is included in depth sorting, fades when it obscures the player, and appears as
tall cover in the nearby-terrain hint and minimap when closed. The mission preview
explains both bypasses. No new image download is required.

Reachability verification floods every legal 5-pixel grid cell in all ten gated
arenas at all supported actor radii (6, 10 and 18), including the arena boundary. Every such cell must connect to the
entrance lane when the gate is closed. The test does not substitute for arbitrary
future geometry: any new gated room must pass it and be reviewed. Closure also
checks actual actor footprints, avoiding trapping an occupant between grid cells.


## Verification

The full Rift engine suite passed in 30.225s. Focused gate checks also cover saved
warning time, pause/death/end inactivity, actor occupancy, non-aliasing geometry,
fast movement from all four directions, boundary release, projectile blocking,
intermission opening and repeated warnings. The expanded connectivity and actual
pursuit tests passed in 0.925s: small, normal and boss-sized actors navigate around
every authored gate in both directions. Embedded Brawl bot tests passed in 6.169s.

Six browser checks passed in 53.2s, including the complete three-wave reward flow,
all three synthesized gate cues, save/reload of a closed gate, jumping against it,
walking the upper bypass, intermission opening, and rejection of inconsistent
protocol data. Nine final browser checks passed after the visual corrections,
including mission previews and lean-response recovery. Final closed/open captures
in `test-results/wave-gates-final` were visually reviewed. Initial fixture failures
are preserved under `test-results/wave-gates`: the first fixture placed enemies
inside an existing pillar; the corrected fixture uses clear ground beyond it.

This completes authored wave gates and their reachability checks. It does not
establish the remaining frame, cold-start, long-session memory or release gates.
