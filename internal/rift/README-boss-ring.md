# Void Lord delayed ring

Canonical Void Lord actors replace every second slam with Void Ring, starting
with their third attack. Opening slam and alternating projectile attacks remain.
The ring locks its center when the 1.6-second warning starts, across all phases;
slow practice doubles this warning. Recovery lasts 2.3 seconds.

The pulse occupies an elliptical annulus with outer radii 200/90 and inner radii
100/45. Its inner edge, center and exterior are safe from this attack. A 90-degree
opening faces toward the arena interior: down when the boss is above Y402.5,
otherwise up. Gap boundaries are safe; the outer ring boundary is dangerous.
Saved target coordinates and gap preserve the committed attack across reloads.

One impact uses normal guard and armor damage rules. Jump height >=0.1, dodge
invulnerability or moving out of the pulse avoids damage and opens a 0.8-second
weak point. Guard break and existing phase transitions cancel pending attacks.
The pulse is an area attack, so cover does not block it. Other enemies can still threaten the marked gap. Arena hazard contact is
suppressed in the center and gap, within the outer ellipse, during the warning
and 0.45-second impact. Exterior hazards stay active. Every simultaneous ring
or lane danger zone takes precedence over a reserved area. Hazard clocks and
cues continue; only contact damage/status is suppressed. No new rewards or economy rules apply.

The renderer draws the exact annulus, a countdown, center/gap/outside guidance,
and a foreground escape arrow. Reduced motion retains static geometry and text;
clean screenshot mode suppresses cues. The impact has a distinct synthesized
sound through the existing audio mixer. No external image assets are needed.

Verification covers attack sequence, delayed release, saved center, exact damage
boundaries, counterplay, interruption and slow practice. A finite escape audit
checks all 100 final arenas across three boss phases, with real collision geometry,
150-unit walking speed and 300ms reaction time. Each stationary control is hit and
at least one of eight walking directions escapes. This does not establish safety
from every coordinate, simultaneous hazards or crowded encounters.

Browser checks cover both gap directions, ordinary and reduced motion, exact
ellipse arcs, non-overlapping gap/intent labels, cancellation and clean mode.
Shared combat audio coverage includes the ring impact. Implements item 0329;
arena-wide warnings and reserved safe lanes remain separate requirements.

Impact reservation time is saved per actor using a hazard-prefixed skill timer.
The existing recovery keeps target coordinates fixed longer than the impact;
defeated actors retain completed impact protection until it expires. Pause
freezes the timer and room start clears it. Cancelled warnings have no impact
reservation. No whole-arena immunity or protection against other enemies is added.
Tests cover geometric boundaries, hazard damage/slow/contact, saved impact,
pause/expiry, cancelled/dead warnings, overlapping rings/lanes, completed-impact
defeat, room cleanup and unrelated enemy damage. This is further progress on
0326, not a guarantee of reachable escape space from every possible coordinate.
