# Brawl enemy population report

Run from the repository root:

```sh
go run ./cmd/brawl-population-report > brawl-enemies.csv
go run ./cmd/brawl-population-report -format=json -max-enemies=6 > brawl-enemies.json
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

Mission and tier numbers are one-based. Names and objectives identify where to
edit content. CSV and JSON include all rooms even when some exceed the budget.
The summary goes to stderr, preserving machine-readable stdout.

Exit codes: 0 means within budget; 1 means at least one room exceeds it; 2 means
invalid arguments or output failure. `go run` may wrap a nonzero program status;
compile the command when a pipeline needs the precise 1-versus-2 distinction.

This is an enemy-count report, not a boss-composition or performance benchmark.
The server's additional named boss sources do not change authored encounter
counts. Staged waves reduce entry population; objective actors are reported
separately instead of being hidden in a single enemy total.

Tests:

```sh
go test ./cmd/brawl-population-report -count=1
```
