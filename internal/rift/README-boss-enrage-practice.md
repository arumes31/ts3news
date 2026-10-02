# Optional timed boss practice

Boss practice has an unchecked timed-challenge option. When enabled, the saved
threshold is 30 seconds of combat time. The practice panel and boss health HUD
show the countdown, then ENRAGED / Boss damage +50%. It works in reduced motion;
there is no flashing or camera shake. A single synthesized cue marks the change,
and screen readers receive a transition announcement rather than every second.

At the threshold, boss-owned impacts deal 1.5 times their normal damage before
armor and guard. This includes existing projectiles when they land; projectile
power and the frozen boss template are not mutated. Hazards and other owners
are unaffected. Untimed practice and campaign damage are unchanged. This is an
optional pressure challenge, not an automatic defeat when the timer expires.

The countdown uses the authoritative run clock, so pausing freezes it. Reloads
retain the threshold and one-shot cue marker. Reset restarts at zero, clears the
marker, preserves the choice, and applies newly selected boss/phase/options.
The long-warning and freeze tools remain available. Local best times separate
timed settings and preserve the old record keys for untimed drills. Practice
still earns no campaign rewards or records.

Snapshot validation accepts only an omitted/zero threshold or 30 in boss practice,
and rejects a triggered marker before its threshold. The request toggle is only
accepted for start/reset, and other practice modes reject it. Existing old saves
without these fields remain untimed.

Tests cover exact impact boundaries, ownership, hazard/campaign isolation, paused
clock, cue deduplication, save/reset, invalid states, explicit opt-in, both motion
settings, desktop/mobile HUD visibility, unchanged campaign saves, separate local
records, and cue mute/pause retirement. A dedicated synthetic browser fixture
starts paused at 28.5 seconds to cross the real server threshold promptly; it does
not replace the engine's damage/timing tests or establish performance gates.
