# Minimap tier progress

The minimap retains the current arena's terrain, hazards, enemies and player. A
three-tier strip below it shows cleared, current, ahead and not-cleared sections of
the current mission. Text/checkmarks carry state alongside color; aria-current
identifies the current tier and each list item names its authored arena.

A prior tier is cleared because server progression advances rooms sequentially.
This also works for legacy saves without split times. The current tier is cleared
when status is cleared, complete or banked, or a finite nonnegative room split
records its clear. Defeated/expired current tiers without that evidence remain
not cleared. Boss retry clears the current split on the server. Future tiers are
never cleared from history, and earlier mission completions do not mark progress
in a fresh attempt. The map floor gains a cleared treatment only for a cleared
current arena. This is display-only; it grants no rewards and changes no collision.

Practice uses its own arena and hides campaign tier progress. Minimap and clean
screenshot display preferences hide the complete map including the progress strip.
The strip is updated from confirmed snapshots and rebuilt only when tier state or
arena names change. It adds no local storage or live-region announcements.

Browser coverage verifies checkpoint advance and reload, legacy missing split
values, completed/banked/defeated/expired states, prior mission history exclusion,
practice exclusion and mobile rendering. Existing map checks cover live movement,
destroyed cover, disabled hazards, objective exits, preferences and landmark names.
