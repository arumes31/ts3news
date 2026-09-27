# Ancient Dragon arena-wide Ground Surge

Ancient Dragon's eighth attack, and each eighth thereafter, is Ground Surge.
Its named warning lasts2s in every phase, or4s in slow practice, and starts from
anywhere in the arena. It supersedes a charge on that turn; earlier slams, volleys
and charges remain. Existing saved windup and attack count preserve the warning.

One pulse hits the whole arena floor, including positions behind cover and at
both boundaries. Jump>=0.1 or dodge invulnerability avoids it and exposes the
usual0.8s weak point. Guard and perfect guard do not reduce the surge or earn
blocking rewards. Armor, barriers, connection/entry grace and ordinary defeat
attribution still apply. Shared damage functions accept an explicit guardability
parameter internally; existing callers remain guardable. Recovery is2.3s and the
existing boss guard-break/phase cancellation rules stop pending surges.

A full-floor outline and restrained fill announce the area. Foreground text shows
ARENA-WIDE SURGE with countdown and JUMP NEAR IMPACT / GUARD WON'T STOP IT.
Optional intent identifies the surge. Ordinary projectile lines and localized
slam circles are suppressed for this attack. Reduced motion retains static cues;
clean screenshots hide them. Existing roar begins the warning and a distinct
synthesized impact uses the shared audio mixer. There is no full-screen flash.

Simulation tests cover full warning, save/resume, floor extremes, jump/dodge,
non-blockability, barriers, slow practice, charge precedence and guard-break
cancellation. A normal game-loop Space-jump at1.6s avoids the2s impact in every
one of the100 final arenas, retaining actual arena hazards/geometry. This checks
that entry-position timing; it does not prove every simultaneous crowded state.
Full combat suite passed27.758s. Implements0325 and0360. Reserved safe lanes are
a separate requirement: this surge is avoided by timing, not a clear floor lane.
