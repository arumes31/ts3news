# Validate Brawl campaign content

From the repository root:

```sh
go run ./cmd/brawl-validate
```

Requires Go on PATH. Runs all four gates sequentially, without a database,
production server, browser session or deployment:

1. All combat-engine tests in `internal/rift`, including mission structure,
   encounter previews, objective behavior and collision/hazard regressions.
2. Population and reward report for all 300 rooms, enforcing enemy/boss budgets.
3. Walking reachability report, including later-wave and initial objective actors.
4. Hazard route-availability report over the requested time window.

An ordinary failing gate does not prevent later gates from producing useful
reports. Interrupting the command stops subsequent checks.

```sh
go run ./cmd/brawl-validate -max-enemies=8 -max-bosses=1 -hazard-seconds=120 -seed=author-review -out-dir ".tmp/brawl-review"
```

Default output is `.tmp/brawl-validation`. Existing files with these names are
replaced: `engine-tests.txt`, `population.json`, `reachability.json`,
`hazards.json` and `summary.json`. Other directory contents are left alone.
The summary records each gate's Go arguments, output filename, pass status and
error. It is reset before execution and updated after each gate, so an interrupted
rerun cannot leave an old success recorded as current. Unrun gates say `not run`.
Failed child processes can leave partial report output; trust the summary status
before consuming a report as successful evidence.

Exit codes: 0 means all gates pass; 1 means at least one gate fails; 2 means
configuration, cancellation or output failure. `go run` wraps nonzero child
statuses; compile this command when a pipeline needs exact exit codes.

Enemy/boss counts are not difficulty or frame-time measurements. Reward ceilings
are conservative, not promised payouts. Reachability is sampled walking access;
hazard intervals measure safe-route availability, not reaction time or timed
player traversal. See the dedicated report READMEs for their precise scopes:

- [Population and rewards](../brawl-population-report/README.md)
- [Walking reachability](../brawl-reachability-report/README.md)
- [Hazard intervals](../brawl-hazard-report/README.md)

This author command supplements browser and production transaction checks; it
is not a complete release gate.

```sh
go test ./cmd/brawl-validate -count=1
```
