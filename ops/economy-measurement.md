# Economy measurement

Run `psql -X -v ON_ERROR_STOP=1 -f ops/economy-report.sql` against the intended database. The report uses one read-only transaction and a 60-second statement timeout. Its queries explicitly use the public schema, so reset backups are excluded.

The report separates economy epoch, producing revision, and unclassified historical data. Wallet credits and debits include player transfers; they must not be described as new gold issuance or destruction. Gold-changing UTC days are activity observations, not online hours. The player field is a deterministic pseudonym; service accounts require separately verified identification.

New run provenance and live replay JSON carry `cohort` with measurement schema, economy epoch, and build revision. New bank/death audits copy the run cohort. Older records remain intact and are labeled with an unknown economy when attribution is unavailable. History identifies current/prior economies when recorded epochs exist.

`timing.resolution_ns` measures monotonic wall time inside live resolution phases, including database waits and scheduling during those phases. Planning windows, pauses, initial encounter preparation, and time between floors are excluded. `timing.estimated_action_window_ns` measures countdown elapsed time, capped at two minutes per window; it is an estimate of engagement because a player may be idle during that countdown. These values are not player-active-time denominators. `measured_floors` and `untimed_floors` expose mixed coverage; replay recovery retains the last persisted observation and cannot reconstruct time lost in a crash.

The report deliberately leaves gold per player-active hour and meaningful inventory upgrades unset: the former needs an engagement denominator, and the latter requires the canonical current build, slot, set, and diminishing-passive comparison. Raw item rarity or combat rating cannot substitute for that comparison. Known-death percentages use terminal runs as the denominator, including concessions which require a downed run in the current economy; unclassified endings are reported separately. Replay counts describe retained archive coverage, not every attempted fight.

Collect at least seven full days after the reset before changing general enemy difficulty. Compare credits/debits and death shares within the same epoch, revision, tier, and depth band; report sample sizes and missing source/request/timing data alongside every conclusion.
