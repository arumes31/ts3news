# Sweeping flame jets

Ten Ember Forge second-tier arenas replace their first fixed fire patch with a
240-unit horizontal sweep. The authored rectangle is the warning envelope and
spawn exclusion. A full1.2-second warning precedes a36-unit damaging strip moving
left to right over1.8 seconds. Its position is derived from the saved hazard
phase; no unsaved animation timer drives damage. Previously saved arenas retain
their existing hazards.

Only the current strip deals damage. Players may jump/dodge or pass behind it;
existing hazard guard, grace, cooldown and damage rules remain. ContactBounds
provides the server footprint; renderer, player area-status and minimap use the
same36-unit strip and phase formula. Warning/escape audits conservatively retain
the entire authored envelope; damage-boundary tests place the player in the
actual strip. The authored sweep remains within arena bounds.

Arrows preview direction; an outlined moving strip and flames show contact.
Reduced motion removes flame flicker while preserving essential hazard movement.
Labels sit below the lane with dark outlines. A distinct activation whoosh plays
once per cycle; contact uses the fire sound. Saved cooldowns prevent cue replay.

Verification covers moving/empty positions, inactive phases, jump/dodge, saved
trajectory and activation cue persistence, all campaign escapes and warning
budgets. Browser tests measure strip displacement, narrow minimap width, HUD
contact state and normal/reduced rendering; shared audio checks mute/cleanup.
No performance gate is established here.
