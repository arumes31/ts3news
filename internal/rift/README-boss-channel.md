# Chronos interruptible Time Pulse

Chronos replaces every fourth attack with a two-second Time Pulse channel.
Alternating lane slams and the earlier projectile turn remain unchanged. Slow
practice doubles the channel time. The impact center locks to the player position
when the channel starts; its ellipse has radii125/62, matching the warning.

Any positive damage, knockdown or stagger interrupts the channel, advances the
attack sequence once, grants one second of weak-point vulnerability and imposes
at least2.3s cooldown. Zero damage cannot interrupt. Phase-changing and lethal
hits also cancel; no interruption grants extra rewards. Existing windup, target,
attack count, cooldown and weak-point state survive saves without a new schema.

Uninterrupted channels release one area pulse with normal guard/armor damage,
then2.3s recovery. Moving outside the ellipse, jumping>=0.1 or dodge invulnerability
avoids damage. Cover does not block this area pulse. The boss remains stationary
during the committed warning and occupies the existing simultaneous-attack budget.

The display shows channel time and HIT BOSS TO INTERRUPT above the boss, with the
impact area and JUMP OR MOVE fallback guidance. Optional intent identifies the
channel. Cancellation announces the weak point, and pulse/interruption sounds
use the shared audio mixer. Reduced motion retains static cues; clean screenshots
hide them. Existing boss roar starts the warning.

Tests cover the attack sequence, delayed/saved release, positive/zero damage,
control/phase/death interruption, locked elliptical geometry, movement/jump/dodge,
and slow practice. Full combat suite passed31.832s. Browser cues in both motion
modes plus shared audio passed3 tests34.2s; reduced-motion screenshot reviewed.
Implements0333. This is a localized pulse, not an arena-wide or jump-required attack.
