# Frost traction

New campaign ice hazards opt into `slippery`. While an enabled patch is active
and the player is grounded, velocity approaches movement input at 600 world
units/s² horizontally and 360 vertically. Releasing movement brakes to zero;
a full-speed horizontal release stops within 0.4 seconds and 47 world units.
Leaving the patch restores direct control. Guard brakes immediately; dodge and
an available jump retain direct control. Repeated unavailable jump inputs do
not suppress traction. Collision consumes blocked momentum.

Player Vx/Vy and the frozen arena flag persist in existing saves. Old arenas
without the flag retain their movement. Traction does not change ice contact
damage, its lingering slow, its sound effects, or the warning/active timing.
The hazard label explains SLIPPERY / GUARD BRAKES beside its jump countdown,
including reduced motion. Inactive and disabled patches remove that warning.

Verification: frost movement tests cover release, acceleration, reversal,
collision, dodge, guard, jump, saved continuation, campaign inclusion and legacy
behavior. The full combat suite includes the campaign hazard escape audit.
Browser coverage renders the cue with normal/reduced motion and checks removal
when inactive/disabled. It does not represent a physical-device performance test.
