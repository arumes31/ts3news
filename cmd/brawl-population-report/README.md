# Brawl population and reward ceiling report

Run from the repository root:

```sh
go run ./cmd/brawl-population-report > brawl-enemies.csv
go run ./cmd/brawl-population-report -format=json -max-enemies=6 > brawl-enemies.json
go run ./cmd/brawl-population-report -format=json -max-bosses=0 > brawl-bosses.json
```

The command reads campaign definitions and the canonical Abyss monster catalog,
constructs each mission with a fixed seed, and emits all 300 room rows without a
database or running server. It does not change campaign content or saved games.

- `planned_enemies`: the frozen encounter-plan count, including later waves.
- `entry_enemies`: planned enemies actually present when entering the room.
- `objective_actors`: extra target actors created at entry, such as cages,
  generators and totems. These do not count as monster loot-bearing enemies.
- `max_attackers`: effective simultaneous attack-start cap, not population.
- `enemy_budget` and `over_budget`: compare planned enemies with the supplied
  limit. The default is eight, the current largest authored room population.

- `planned_bosses` and `entry_bosses`: boss-kind actors in the frozen plan and
  at room entry. Bosses are included in enemy counts, not added to them.
- `boss_budget` and `boss_over_budget`: independently compare planned bosses
  against `-max-bosses` (default one). Zero can flag every boss encounter;
  negative limits are rejected.

Reward columns are conservative upper bounds for one successful room clear:

- `drop_gold_ceiling`: planned enemies times the runtime gold-per-defeat rule.
- `gear_count_ceiling`: at most one gear drop per planned enemy. Actual eligibility
  depends on monster kind and encounter index; this is not a guaranteed count.
- `gear_rarity_ceiling`: the runtime rarity cap as a stable numeric value: 3 is
  Epic and 4 is Legendary. It is not a sale-value estimate.
- `objective_gold_ceiling`: zero before the final tier; at the final tier, the
  frozen per-challenge reward times all current challenge options with equipped
  builder/finisher and ultimate. This includes mutually incompatible challenges
  and encounter-dependent goals, so the bound need not be achievable.
- `total_gold_ceiling`: drop gold plus the optional objective bound. It excludes
  selling, recycling or upgrading gear and applies before any such inventory use.

Props generate no loot. Escapes, defeat, missing optional equipment and failed
challenges can all lower actual rewards. The command never grants or banks them.

Mission and tier numbers are one-based. Names and objectives identify where to
edit content. CSV and JSON include all rooms even when some exceed the budget.
The summary goes to stderr, preserving machine-readable stdout.

Exit codes: 0 means within budget; 1 means at least one room exceeds either budget; 2 means
invalid arguments or output failure. `go run` may wrap a nonzero program status;
compile the command when a pipeline needs the precise 1-versus-2 distinction.

This reports population counts, not boss identities, boss ability difficulty or
measured combat performance.
The server's additional named bosses extend the boss pool; they do not change
the current planner's per-room counts while the regular monster pool is nonempty. Staged waves reduce entry population; objective actors are reported
separately instead of being hidden in a single enemy total.

Tests:

```sh
go test ./cmd/brawl-population-report -count=1
```
