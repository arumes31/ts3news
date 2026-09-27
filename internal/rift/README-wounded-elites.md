# Wounded elite attack priorities

Canonical EliteMinion, Elite and Miniboss templates receive an elite marker when
adapted for new encounters. At or below 35% maximum health, living melee/ranged
elites enter a saved, one-way Last stand phase. Boss and treasure behavior remains
separate. Older frozen actors without elite metadata retain their previous rules.

Melee elites prioritize the existing aimed charge at 85-360 units instead of
walking into melee range. This also widens the 110-320 range of an elite that
already charges. The charge still locks its target, warns for 0.7s, stops on cover,
permits jump/dodge/interrupt counterplay and has 0.85s recovery. No damage, armor,
speed, reward or warning-time multiplier is added by the elite phase.

Ranged elites prioritize retreat below 220 units (normally 150), then stop to aim.
A blocked retreat falls through to the normal attack logic. Committed shots,
cooldowns and hit/control reactions are preserved. Healing and save/reload do not
reset the phase or replay its cue. The marker is stored as the existing optional
Enraged field; it denotes a behavior phase, not a damage buff.

A static LAST STAND badge and one-time synthesized effects cue mark the change.
The optional intent overlay remains independent; ranged movement says Moving so
retreat is not mislabeled as approaching. Reduced motion retains the badge and
clean screenshot mode hides it. Defeated actors lose the badge.

Tests cover the threshold, native tier adaptation, existing charger range changes,
normal windup, saved/healed phase continuity, normal versus wounded archer choices,
committed shots, pinned fallback, ordinary/boss/treasure exclusions and wounded
native elites reaching attacks from authored spawn geometry. Browser snapshots
verify badge contrast, intent coexistence and removal; audio tests verify source
creation, mute and pause cleanup. This implements 0308.
