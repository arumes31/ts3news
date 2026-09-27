# Shield-carrier turn window

Knights (including legacy knights without the shield flag) and shield-flagged
actors brace between attacks when their player target is within 120 horizontal
and 50 vertical world units. Their attack cooldown must still be active, with no
windup or timed hit/attack pose. Guard uses the existing guard flag and sprite pose.
Bracing holds position until attack readiness or until the player leaves range.

Crossing behind a braced carrier starts a 0.35-second turn window. Facing stays fixed
until the timer expires. Returning in front cancels the turn; crossing behind again
starts a fresh window. Existing rear strikes still bypass most armor and turn the
carrier toward the hit immediately. This does not add another armor bonus.
Unshielded actors and shield carriers winding up, reacting to hits, pursuing distant
targets or ready to attack retain immediate facing behavior.

The existing serialized Actor.TurnDelay stores remaining turn time; no new save
field or economy state is introduced. Values are bounded before use. Paused runs
do not tick enemy AI. Simulation tests cover the actual enemy tick path, save/load
mid-turn, rear-strike damage, cancellation and re-entry, legacy/flagged roles,
stagger/knockdown cancellation, bounded stale timer values and ordinary turning outside guard. The full simulation
suite covers interaction with existing movement and enemy attack behavior.
