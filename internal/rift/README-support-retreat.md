# Defender-aware support retreat

Healer actors, including canonical Frost Lich in new encounters, receive the
support role. When pressured within 240 units, a ready support actor can seek
cover behind a living knight or shielded ally within 220 units. It chooses the
nearest directly reachable cover point that moves away from the player. Defenders
that are dead, unalerted, airborne, staggered, knocked down or actively charging
are excluded, as are bosses, treasure goblins and objective props.

Cover is computed from the current player-to-defender direction: aim 90 units
behind the defender, staying within arena bounds. Once at least 60 units behind
and within 55 units across that line, normal behavior resumes. Candidate paths use
actual collision clearance. Blocked movement falls back to ordinary behavior;
there is no teleport or collision bypass. A chosen retreat can finish beyond the
initial pressure radius, and is recomputed as actors move or defenders disappear.

Committed shots and mends remain intact. Eligible healing takes priority over
starting a retreat. Attack cooldowns are preserved. The saved cover ID identifies
an ongoing retreat and suppresses duplicate cues after reload; it is cleared on
hits and recomputed by the next enemy tick. No economic reward or damage change.

Existing running/backpedal sprites and a static SEEKING COVER label show movement.
Optional training intent says Seeking defender, and a short effects cue plays
when the chosen defender changes or a new retreat begins. Reduced motion retains
the label; clean screenshot mode suppresses it.

Tests cover reaching cover and resuming fire, save/reload cue continuity, lane
changes and flanking, committed attacks/mends, dead or unsuitable defenders, blocked
routes, damage cancellation, canonical role assignment and completing a retreat
beyond its initial trigger radius. Browser snapshots verify readable guidance and
its removal; sound tests cover source creation, mute and pause cleanup.
This implements 0302.
