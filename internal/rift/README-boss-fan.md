# Kraken rotating projectile fan

Canonical Kraken actors in newly planned encounters replace their alternating
single-shot volleys with Rotating Fan. Slam turns remain intact. Other bosses and
frozen actors without VolleyFan keep their previous patterns.

The fan engages at up to 430 horizontal units in the normal boss attack lane.
Each warning lasts 1.25 seconds in every phase, doubled by slow practice. The
player position and fan rotation are saved when warning begins. Five rays use
0.22-radian spacing, with the complete fan rotated -0.18, 0, +0.18 radians on
successive volleys, then repeating. Moving during warning does not retarget it.

Each shot travels at 260 units/second, deals 65% of the ordinary volley power,
lasts up to four seconds, and retains canonical shot effects and boss ownership.
There are exactly five new shots per release and a two-second attack cooldown.
Existing projectile collision, cover damage, dodge/jump behavior and dead-boss
projectile cleanup apply. A blocked origin does not release shots. Guard breaks
and phase interruption cancel pending fans through the normal windup lifecycle.

Five dashed paths show exactly the saved shot directions. FAN / MOVE BETWEEN
SHOTS replaces ordinary volley and slam guidance; reduced motion keeps it static.
Clean screenshots omit the warning. Existing projectile artwork is reused, with
a new synthesized fan-release cue and existing boss warning roar.

Tests compare every projectile direction across a full rotation cycle, including
saved warnings and player movement. They verify IDs, ownership, reduced power,
recovery, other pattern preservation, slow practice and guard-break cancellation.
The all-campaign boss audit retains stationary-threat and spawn/approach checks;
fan cases allow three seconds for warning plus flight and require a safe tested
movement direction after a 300ms reaction delay. This is finite directional
counterplay coverage, not proof of every possible player position or crowd state.
Browser tests compare all five drawn rays for each rotation, check misleading
slam guidance is absent, verify clean suppression, and exercise audio lifecycle.
This implements 0327.
