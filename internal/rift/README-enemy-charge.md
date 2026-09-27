# Charging enemy recovery

Raging Behemoth receives the charger role when adapted from the canonical Abyss
catalog, using the current localized canonical name. Existing explicit charger
flags also work on melee fighters. Archers, bosses and treasure goblins retain
their specialized behavior. Frozen older encounters are not retroactively changed.

At 110-320 units and within 100 units of the player's lane, an eligible fighter
warns for 0.7 seconds. It locks the player's position at warning time. The dashed
path and SIDESTEP label remain static with reduced motion. A rush moves at 360
units/second using steps no larger than six units, stopping on solid cover or the
arena boundary. The enemy cannot steer toward a player who leaves the marked path.
Contact can be avoided with a jump or dodge; a rush deals at most one contact hit.

Contact, a missed endpoint, collision or interruption starts 0.85 seconds of
stationary recovery. Existing hit and knockdown reactions can prolong the opening.
The recovery pose and countdown identify the counterattack window; no bonus damage,
armor bypass or additional reward is granted. Normal pursuit resumes afterward.
Warning, rushing and recovery have separate synthesized sound cues using the
existing effects mix, mute and pause cleanup. Attack concurrency and pack spacing
apply before a charge begins.

Charge state uses optional actor fields and existing target/windup fields, so
missing fields in older saves remain inert. JSON roundtrip tests cover warning, rush and recovery
midpoints. Tests cover fixed aim, counterattack damage, collision, single contact,
damage/control interruption, role exclusions, pack spacing and canonical usage.
Renderer snapshots verify warning/recovery labels; these are visual checks, not
proof of a live server fight. Simulation tests exercise actual enemy combat.

This implements improvement 0306. Boss charge patterns (0330) remain separate.
