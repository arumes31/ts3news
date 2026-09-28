# Circular arenas (0416 in progress)

RoundArena saves a circular floor as an ellipse in the side-on depth projection.
Ground movement uses actor clearance and rejects endpoints outside the floor;
convexity keeps each accepted straight movement segment inside it. Enemy spawn
placement shares these checks. Geometry survives JSON and is independently copied.
Focused movement, spawn, save and bridge regression tests pass.

Server/browser validation now rejects invalid dimensions, off-map circles and
explicit entry/exit anchors outside the usable floor. Recovery and protocol tests
pass for valid geometry and legacy rooms.

Tier 1 of missions 2, 62 and 82 now uses this geometry. The raised platform and
central cover remain; two routes above and below the central hazard band connect
the original regional entrance to an exit inside the circular floor. Both routes
pass actual movement and hazard-clearance checks. Enemy placement stays inside
the floor, and curved-edge steering keeps fleeing treasure monsters moving.

The permanent-route audit now uses saved entrances/exits and checks a final short
segment for anchors that do not lie on its search grid. The lane-slam audit only
starts actors on walkable ground, as it already excluded positions inside walls.
The full Rift suite passes, including authored-spawn pursuit, every combat role,
regional arrival consistency, raised-platform access and hazard-safe routes.

Still required: renderer/map previews, real browser combat/recovery and screenshot
inspection. Do not mark 0416 complete until these paths are verified.
