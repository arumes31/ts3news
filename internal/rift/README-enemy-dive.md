# Flying enemy dive cue

Common bats from the shared Abyss noun pool receive the Flying role in Brawl.
Identity, generated art, strengths and rewards remain canonical. Frozen actors
without this role retain their saved behavior. This is a low swoop with ordinary
hittable ground coordinates; no new immunity, aerial collision bypass or loot
bonus is introduced. Bosses, ranged actors, treasures and objective props retain
their existing behaviors.

Ready bats110–300 units away, with vertical separation at most100, lock the
player's position and warn for1s. Both actors must have ground elevation0 and the
initial route must clear cover with the actor's footprint. The swoop travels at
300 units/s in substeps no larger than6 units. It uses the normal attacker budget
and pack lockout; its attack slot remains occupied for the full saved flight.
The target never follows subsequent movement.

Contact within30 horizontal/20 vertical units deals one normal enemy hit, minimum
19 before ordinary guard/armor handling. Jump >=0.2, dodge invulnerability or
leaving the route avoids it. Collision or reaching the fixed target also ends the
swoop. Landing creates1s stationary recovery and at least2.2s cooldown. Positive
damage, stagger, knockdown and death interrupt warning or flight. Interruption
preserves the hit/control pose and emits only the interruption cue, not a false
successful landing cue. Warning, flight, target and recovery survive saves.

A cyan dashed route and target marker distinguish the dive from charges and
burrowing. The bat has a low hover/raised preparation pose, drops during its
attack pose and returns to ground during recovery. Hover decoration stops with
reduced motion. Warning, fixed-path flight, recovery and optional intent text
remain readable without animation. Clean screenshots hide warning overlays.
Warning, swoop, landing and interruption use distinct sounds through the normal
mix, mute and pause cleanup.

Simulation tests cover canonical bats, warning duration, single impact, contact
and missed-flight recovery, saved warning/active flight/recovery, movement/jump/
dodge, wall collision, damage/control/death interruption, role exclusions,
committed attacks and full-duration attack-slot/pack restrictions. The complete
campaign pursuit and combat suite also runs. Browser tests exercise normal and
reduced-motion warnings, flight, grounded recovery, interruption and clean mode;
the shared audio test verifies source creation, mute and pause retirement.
