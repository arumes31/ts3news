# Brawl hazard-safety report

```sh
go run ./cmd/brawl-hazard-report > brawl-hazards.json
go run ./cmd/brawl-hazard-report -seconds=120 > brawl-hazards-120s.json
```

Run from the repository root. The command reads all 100 missions/300 arenas
without a database. JSON records mission, tier, analysis horizon, unsafe
intervals and invalid-hazard errors. Summary goes to stderr. Exit codes are 0
for no findings, 1 for unsafe intervals/invalid hazards, and 2 for command/output
errors. `go run` can wrap program exit codes; compile for exact status handling.

Each unsafe interval is half-open (`from_seconds` inclusive, `until_seconds`
exclusive) and lists zero-based `active_hazards` indices into the arena's hazard
array. It means no sampled hazard-free entrance-to-exit walking route exists
under that phase's active footprints. Hazard durations/periods/offsets determine
all activation changes inside the window, rather than fixed time sampling.
Offsets already active at time zero and intervals clipped by the horizon are
included. Disabled hazards are ignored. Enabled hazards require finite values,
positive geometry/duration and recovery time after the 1.2-second warning.

The route check is the existing campaign safe-ground regression: a 10-unit grid,
12-unit safety margin, entrance (160,410), exit (1450,320), intact cover and actual
movement/ledge collision checked in two-unit steps. Arenas with a permanent
route outside every hazard footprint pass immediately for all phase combinations.
Other arenas are checked once per distinct active-hazard set. The horizon defaults
to 30 seconds and must be greater than zero and at most 3600 seconds.

This is phase-by-phase route availability, not a time-dependent player simulation,
reaction-time test, continuous geometry proof, or promise of combat survival.
Warnings are not damaging; jumpable hazards are conservatively treated as ground
danger while active. Player jumps, abilities, enemy attacks and moving practice
hazards are outside this report. A player may need to wait for a route to reopen.

```sh
go test ./cmd/brawl-hazard-report ./internal/rift -count=1
```
