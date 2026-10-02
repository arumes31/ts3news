# Rotating blades

Ten Bloodrust Barracks second-tier arenas replace their first floor patch with
a 140x70 rotating-blade mechanism. A 1.2-second warning shows the clockwise path.
The active blade then completes one circuit over 2.4 seconds (150 degrees/second)
before the existing cycle's rest interval. Its 20x20 contact square follows an
ellipse inside the warning envelope. Only that square damages; the hub stays
safe. Jump/dodge and existing guard/grace rules remain available.

The blade centre derives from the saved hazard phase. ContactBounds, rendered
outline, player area-status and minimap all use the same ellipse. Reduced motion
removes tooth spinning while retaining the hazard's essential position changes.
A direction arrow previews clockwise motion, and labels explain the safe centre
and jump counterplay. Separate activation/contact sounds identify the mechanism.
Old frozen arenas retain their original hazards.

Tests check all quarter-turn contact positions, safe hub, warning/rest phases,
jump/dodge, saved continuation and ten authored mechanisms. The full combat
suite audits campaign escape routes. Browser checks measure quarter-turn motion,
HUD contact/safe centre, minimap width and normal/reduced rendering; shared flame
checks cover the adjacent geometry code and audio checks cover mute/cleanup.
These tests do not establish the outstanding performance gates.
