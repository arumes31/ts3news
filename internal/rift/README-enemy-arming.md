# Cancellable explosive-enemy arming

Common canonical Abyss monsters whose adapted shot theme is fire receive the
explosive role in new encounters. This derives from the existing shared-catalog
adaptation rather than creating separate Brawl monster templates. Frozen actors
without the flag retain their previous attacks.

When ready and within the inner part of a blast ellipse with radii of 115 and 55 units, an enemy
stops and arms for 1.1 seconds. Windup occupies its normal attacker slot; pack
spacing and arena attacker limits apply. Already committed attacks stay intact.
Bosses, treasure enemies and objective props do not use this move.

Any damaging hit, stagger or knockdown cancels arming and imposes at least a
1.4-second retry cooldown. Death through damage also cancels it. The unfinished
blast cannot resolve later. Saved arming resumes with its remaining warning and
without replaying the start cue.

Completion emits one blast with normal enemy damage (minimum 20), a 0.35-second
attack pose and two-second cooldown. The enemy survives: no self-kill, fabricated
player damage credit or loot. Moving outside the ellipse, jumping, dodging, or
intervening cover avoids damage. The enemy remains stationary during arming.

Existing cast/attack sprites and fire effect artwork show the attack. A static
ellipse marks the actual blast extent; ARMING / HIT TO CANCEL and optional intent
explain the opening. Cancellation removes the warning and shows BLAST CANCELLED.
Start, cancellation and detonation have distinct synthesized sounds. Reduced
motion retains area/intent text and a brief impact outline; clean screenshots
hide warning and cancellation labels.

Simulation tests cover catalog integration, timing, interruption/retry, death,
save continuity, damage once, no self-kill rewards, escape/cover, role exclusions,
committed attacks and attack budgets. Browser tests check ellipse dimensions,
readable warning/intent, cancellation feedback, clean suppression and audio.
This implements improvement 0303.
