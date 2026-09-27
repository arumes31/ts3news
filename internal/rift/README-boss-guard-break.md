# Boss stagger threshold and recoverable guard break

Living bosses gain 20 stagger per successful damaging hit, independently of hit
size or their health percentage. At 100 the meter resets and a 1.5-second guard
break cancels the pending attack, charge or mend. The boss stays stationary in
its existing stagger animation and cannot start an attack. Its effective armor
is halved during the opening, then restored without mutating its base armor.

The break sets a minimum two-second attack cooldown. Buildup is disabled during
the break and for two seconds afterwards; incoming hits cannot extend the break
or refill the meter during this grace period. Existing brief heavy/third-strike
staggers and health phase transitions remain, but cannot shorten a guard break.
Guard break supersedes charge recovery rather than stacking a second immobile
window. The next attack still has its ordinary warning.

Zero-damage hits and other actor roles do not build stagger. Death clears the
meter and timers and does not trigger a break event. Partial buildup, breaks and
grace timers survive saving. No additional reward or player guard-break statistic
is awarded. The mechanic applies to living boss actors, including saved bosses
whose new fields begin at zero.

A separate STAGGER n/100 label shows buildup alongside existing health. It changes
to GUARD BROKEN with remaining opening time, then STAGGER RECOVERING during grace.
Ordinary meter/grace labels follow the health-bars preference; an active opening
remains visible even with health bars hidden. Clean screenshots suppress all
labels. Reduced motion retains static text; optional intent says Guard broken.
A distinct boss_guard_break event uses the existing boss stagger impact sound.

Simulation tests cover damage-independent buildup, threshold/cancellation, armor
restoration, no extension, grace, saves, death/role exclusions, charge and phase
interaction, and resuming with a telegraph. Browser coverage checks visible
states, intent, clean suppression and audio; screenshots reviewed for legibility.
Implements 0323 and 0324.
