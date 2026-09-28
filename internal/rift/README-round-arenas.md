# Circular arenas (0416 in progress)

RoundArena saves a circular floor as an ellipse in the side-on depth projection.
Ground movement uses actor clearance and rejects endpoints outside the floor;
convexity keeps each accepted straight movement segment inside it. Enemy spawn
placement shares these checks. Geometry survives JSON and is independently copied.
Focused movement, spawn, save and bridge regression tests pass.

No campaign room currently uses this geometry. Still required: server/browser
validation, authored upper and lower safe lanes around central defenses, hazard
compatibility and reachability audits, entry/exit placement, renderer/map previews,
and real browser combat/recovery checks. Do not mark 0416 complete from this
geometry foundation. Decorative ellipses alone do not meet the safe-lane request.
