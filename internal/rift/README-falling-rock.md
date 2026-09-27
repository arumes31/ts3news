# Falling rocks

Each Obsidian Citadel first-tier arena replaces its first fire patch with an
explicit overhead falling-rock hazard. It retains that patch's position, cycle
and phase offset, warns for 1.2 seconds, then has a 0.18-second impact window.
A dark outlined shadow sits within the exact rectangular danger boundary;
normal motion shows descending rocks, while reduced motion keeps the shadow
static until impact. The impact burst and MOVE cue match the active interval.

Move outside the rectangle or dodge to evade. Jumping cannot evade overhead
rocks; guard retains the existing hazard damage reduction. Per-hazard/shared
cooldowns prevent repeated hits within the brief window. Recovery has no damage.
Existing armor, grace, defeat attribution and saved hazard timing remain intact.

Warning and impact use separate sounds, including impacts that miss the player.
Saved active timers prevent impact sound from replaying every simulation tick.
Old frozen arenas retain their fire patches. This is a brief ordinary-damage
impact, not a lethal one-shot mechanic (ledger0465 remains separate).

Verification includes warning boundaries, one contact per impact, post-impact
safety, jump/dodge/movement/guard behavior, save continuation, all campaign
hazard escapes and maximum-difficulty warning timing. Browser tests check the
shadow, countdown, impact and removal in both motion settings; shared audio
checks verify sources, mute and retirement. These do not prove performance gates.
