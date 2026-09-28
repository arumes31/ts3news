# Tracking lightning

The first hazard in each of Storm Spires' ten first tiers is a tracking strike.
Its 84-by-48 warning box follows the player for the first 0.5 seconds of the
existing hazard cycle. It then locks until the strike at 1.2 seconds. The impact
lasts 0.18 seconds. The marker stays inside playable bounds and uses the saved
simulation clock, never wall time. Its coordinates persist in the arena hazard,
so pause/resume cannot retarget an already locked strike. Disabled hazards stop
tracking; existing hazards retain their authored behavior.

Move out after lock or dodge the strike. Jump does not evade overhead lightning;
guard retains the ordinary damage reduction. Existing grace and contact limits
apply. Strike audio fires once even on a miss; the existing warning cue announces
the cycle. The renderer labels tracking, locked and impact phases over the exact
damage rectangle; minimap and HUD use those same saved bounds. Reduced motion
retains a thin static bolt, without screen flashes or decorative motion.

Tests cover tracking/locking, saved impact, next-cycle reacquisition, arena bounds,
pause/disable, authored placement, dodge/movement/jump behavior and cue cadence.
The full combat suite retains hazard warning and walking-escape checks. Browser
checks inspect phase labels, map alignment, both motion modes and sound lifecycle.
These checks do not establish the remaining performance gates.
