# Persistent floor-hazard warnings

Current floor hazards remain harmful throughout their active duration. Before
activation, the renderer now adds LINGERS followed by that full duration above
its existing kind/activation countdown. During activation, DANGER REMAINS sits
above the existing jump/move advice and remaining danger time. Safe, disabled and
cleared states omit the persistence cue.

Both lines use the saved combat clock, static text, explicit hazard colors and
dark backplates rendered after actors. Their spacing follows the HUD text scale.
Reduced motion retains both lines; hiding hazard labels or enabling clean
screenshots suppresses both. This changes neither damage nor hazard timing.

Browser coverage checks warning/activation/safe boundaries, cycle restart,
pause timing, disabled/cleared states and label suppression in both motion modes.
The visibility fixture checks both backplates and text above a large boss on
bright ice. Warning and active screenshots were inspected for legibility.

Implements improvement 0464. Current floor hazards are not one-shot strikes;
0465 needs a real one-shot hazard lifecycle before assigning it a distinct cue.
