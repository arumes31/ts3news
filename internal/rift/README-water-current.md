# Water currents

Ten Drowned Temple first-tier arenas contain a marked water channel. Its arrows
show a steady 35-unit/second horizontal flow, alternating left/right by layout.
The server pushes living grounded fighters once per simulation step before
hazard contact and enemy actions. Normal collision and arena bounds still apply.

Players can walk upstream, guard to brace, dodge, or jump out of the flow.
Flying/burrowed actors, raised-platform occupants, objective props and knocked-
down fighters are excluded. Cleared or paused fights do not apply current drift.
This does not rewrite actor input velocity or affect loot/projectiles. Overlapping
current vectors are capped at a combined 40 units/second.

Saved arena geometry/direction and the combat clock preserve resumes. Old saved
arenas have no new currents. Campaign slices are detached; backend/client reject
out-of-bounds, zero or excessive flows and lists longer than four. A saved entry
cue prevents repeated water audio while continuously in the flow. Arrows remain
static with reduced motion; labels explain guard/jump counterplay.

Tests cover grounded actors, input counterplay, collision, exclusions, saves,
entry cues, detached templates, overlap bounds and snapshot validation. The full
combat suite includes all campaign hazard escape checks. Browser tests inspect
normal/reduced cues and cleared states; audio tests verify mute/source cleanup.
No performance gate is established by these tests.
