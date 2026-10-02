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
impact, not an inherently fatal mechanic. The conditional lethal-contact warning
is described in README-lethal-hazards.md.

Verification includes warning boundaries, one contact per impact, post-impact
safety, jump/dodge/movement/guard behavior, save continuation, all campaign
hazard escapes and maximum-difficulty warning timing. Browser tests check the
shadow, countdown, impact and removal in both motion settings; shared audio
checks verify sources, mute and retirement. These do not prove performance gates.

## Enemy contact

Falling rocks also damage monsters, including bosses, within the same rectangular
contact bounds during the impact window. Damage starts at12+region, then uses the
enemy's normal armor, guardian bond and boss shell. Airborne enemies remain
exposed to this overhead strike; burrowed enemies and objective props are excluded.
Other hazard kinds keep their existing player-only behavior.

Saved per-hazard/per-enemy cooldowns allow one contact per cycle. A shared enemy
cooldown of0.35s prevents overlapping rocks from stacking damage. These timers
freeze with the run and clear at room start. Rock warnings and impact cues keep
running even when no actor is hit. Player safe-area reservations do not shield
monsters. Player and enemy contact cooldowns are independent.

Damage reuses the enemy damage/defeat pipeline, preserving alerting, hurt sounds,
interruptions, boss phase changes, shell breaks, defeat records and ordinary loot.
Environmental hits do not borrow Berserker fury, rear-strike, summon-arrival or
weak-point attack bonuses. They add no player damage, largest-hit, attack-chain,
combo, practice-hit or boss-stagger credit, and cause no player-facing recoil.
Boss shells still absorb a rock during a player weak-point opportunity. Normal
practice reward suppression remains in force. Kill/loot credit rewards using the
terrain, and the HP guard prevents duplicate death drops.

The visible warning/impact label now says HITS ENEMIES while retaining movement
and overhead-jump guidance. Existing rock and monster hurt/death effects and
sounds are reused. Tests cover exact mitigation, saved contact, next cycle,
warning/recovery/disabled/outside/burrow/prop/airborne rules, boss phase/shield,
kill/drop deduplication, player-bonus isolation, overlap, pause, room cleanup and
practice rewards. Implements0456.

The isolated enemy-rock browser fixture starts a real paused run just before an
impact with one low-health monster under the rock. Resuming verifies server-side
defeat, exactly one normal drop, zero player damage/combo credit, and persistence
after reload. This complements engine mitigation and lifecycle tests; it is not
a performance measurement.
