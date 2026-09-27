# Kraken destructible shell

New canonical Kraken actors start with a90-point shell, shown separately from
health and stagger. Durability is stored on the actor and never regenerates;
new practice encounters and explicit resets start from their saved initial actor.
Older frozen actors without shell fields keep their original defenses.

Ordinary incoming damage consumes shell after armor and piercing calculation.
Only excess damage from a breaking hit reaches health. Shell absorption does not
count as health damage, kills, loot or stagger buildup. Zero damage cannot alter
the shell. Breaking it cancels the pending attack, grants1.2s weak-point exposure
and stagger, and imposes at least2s attack cooldown. Overflow can kill normally,
with exactly the ordinary defeat reward. Shatter feedback occurs once.

Existing earned weak points bypass the shell and retain their damage bonus.
This preserves the health counterattack earned by avoiding the opening slam;
when that exposure expires, remaining shell protects health again. The shield
label explicitly announces the bypass while exposure is active.

A durability bar, numeric remaining/max label and static shell outline show
progress. Broken state and shatter feedback remain distinct from guard break.
Impact/shatter sounds use the existing mixer. Reduced motion keeps static cues;
clean screenshots suppress them. Canonical art and existing combat animation
remain shared with Abyss.

Tests cover canonical initialization, partial save/resume, armor/piercing,
zero damage, overflow, one break event, reward accounting and weak-point bypass
then expiry. The original all-catalog opening counterplay test is unchanged.
Full combat suite passed28.306s after fixing the weak-point interaction; bot
TestRift passed5.796s. Initial browser cues/audio passed3 tests39.7s. Reviewed
reduced-motion shell warning and shatter screenshots for readable labels.
Implements0332.
