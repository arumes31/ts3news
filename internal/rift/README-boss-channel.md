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


## Hazard reservation groundwork

The simulation reserves a bounded escape band during a live Time Pulse warning:
outside the125/62 blast ellipse and inside a200/90 ellipse at its locked center.
The blast boundary remains dangerous; the outer boundary is reserved. Contact
with arena hazards is suppressed in this band, including contact slow and credit.
Hazard clocks, enemy attacks and projectiles continue normally. Simultaneous lane,
ring or pulse danger overrides any reservation.

A released pulse retains the band for0.45 combat seconds using a saved
hazard-channel timer. Cancellation or defeat removes an unreleased warning;
defeat after release preserves its brief impact reservation. Pause freezes it,
and room changes clear it. Damage and reservation share the blast predicate.
Tests cover boundaries, contact effects, save/reload, pause, expiry, room reset,
cancellation, overlapping patterns and unrelated enemy damage.

This completes the Time Pulse portion of0326; see README-boss-safe-areas.md. A mint outer-band overlay and hazard-specific label
are implemented, including the saved impact interval and clean-screenshot hiding.
The label says HAZARD SHELTER because hazard clocks continue; it explicitly
retains enemy and overlapping-warning danger. Lane-slam labels use the same term.
Four browser tests passed for normal/reduced motion, interruption, saved impact
after boss death, expiry, clean screenshots and mobile label bounds. Normal
warning and reduced-motion mobile impact screenshots were visually reviewed.
The two shelter labels scale to the displayed canvas width, retain at least
9 CSS pixels at default text size, and reserve separate line spacing at the bottom.
Mobile text bounds and spacing assertions passed in the browser run.
Local artifacts: test-results/boss-shelter-final-20260928. These injected display
states verify presentation; the Go tests above verify authoritative lifecycle.

The finite terrain audit covers370 states: all300 campaign rooms, substituting
all eight floor/gate combinations for each of the ten wave rooms. It samples
X35..1565 in30-unit steps and ten Y values. After excluding11834 illegal starts,
all180566 legal starts per speed reach the protected band with one held direction.
Speeds are235 horizontal/141 depth normally and141/84.6 while slowed. Routes use
real movement in20ms steps after300ms reaction, within the canonical two-second
warning. This does not establish arbitrary-coordinate or concurrent-attack safety.

    go test -tags=brawl_audit ./internal/rift -run TestCampaignChannelShelterGridAudit -count=1 -v

A normal-suite regression also runs full combat ticks from all300 actual room
entrances, keeping slow active and allowing authored hazards/objectives to tick.
Each stationary control takes damage on the pulse's release; at least one held
walking direction reaches shelter after300ms reaction and avoids damage on that
release tick. Simulation HP is increased to keep earlier hazard contact from
ending the probe; this is not a normal-health campaign completion test. Jumping,
dodging, attacking and channel interruption are not used to manufacture escape.

    go test ./internal/rift -run TestCampaignChannelShelterWithSlowedCombatTicks -count=1 -v
