# Wind corridors

Ten Storm Spires second-tier arenas have a saved wind corridor. Even layouts
push right and odd layouts push left at 60 world units/second. Each eight-second
cycle warns for 1.2 seconds, blows for three, then rests for 3.8 seconds. Arrows
and a frozen-clock countdown show direction and timing; reduced motion retains
static arrows. Warning and activation each have an audio cue once per cycle.

Both teams receive the same additive projectile drift when a shot starts its
simulation step inside an active corridor. Launch velocity, ownership, damage
and lifetime remain unchanged. Leaving the corridor restores original motion;
cover collision checks the resulting swept segment. Multiple corridors sum to
at most 80 units/second in either direction. Wind does not move actors.

The optional arena list is deep-copied from campaign definitions and saved with
the fight. Legacy arenas without it have no wind. Backend and browser validation
bound corridor geometry, speed, timing and count. Pause freezes its clock and
saved cooldowns prevent audio from repeating on resume.

Tests cover both teams, warning/rest/outside/pause/clear states, save trajectories,
cover, cue repetition, content copies and the overlap cap. Browser tests check
both arrow directions, countdown phases and reduced motion; audio tests check
sources, mute and cleanup. These checks do not establish performance gates.
