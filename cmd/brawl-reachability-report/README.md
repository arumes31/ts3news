# Brawl walking-reachability report

Run from the repository root:

```sh
go run ./cmd/brawl-reachability-report > brawl-reachability.json
go run ./cmd/brawl-reachability-report -seed=another-layout-sample > brawl-reachability-other.json
```

This read-only, database-free command reports every mission/tier, its number of
checked targets, and failures containing `actor_id`, `x`, `y` and `reason`.
It checks the frozen encounter plan (including future waves) plus objective
actors present at room entry. The fixed seed makes runs reproducible; specify
another seed to sample different roster selections and spawn placement.

The engine helper uses the same 10-unit walking-grid approach as the existing
campaign regression test, a 10-unit player footprint and a 16-unit target
approach tolerance. Low/high cover and intact terrain cover block the flood.
`blocked_start` means the starting grid cell is obstructed; `invalid_start`
means the starting coordinate is outside the supported walking bounds;
`unreachable` identifies a target without a reachable approach cell.

This is sampled walking reachability, not a continuous-geometry proof. It does
not model jumping, broken cover, abilities, hazard timing, combat survival or
all optional objective waypoints. Use the full combat tests and hazard checks
alongside it. No player saves or campaign definitions are changed.

Exit codes: 0 means no reported failure, 1 means at least one failure, and 2
means invalid arguments or output failure. JSON remains on stdout; summary goes
to stderr. `go run` wraps nonzero program statuses; compile the command if exact
exit-code distinctions are needed in automation.

```sh
go test ./cmd/brawl-reachability-report ./internal/rift -count=1
```
