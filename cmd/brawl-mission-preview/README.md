# Deterministic Brawl mission preview

From the repository root, with Go on PATH:

```sh
go run ./cmd/brawl-mission-preview -mission 15 -seed forge-review
```

Emits one JSON document to stdout with schema version 1, the exact seed,
`catalog: canonical_abyss`, the mission definition and three ordered tier
previews. Each tier includes its prepared arena, player spawn, full planned
enemy roster and entry actors. Entry actors can include objective props; planned
enemies can include later-wave reserves. Do not add the two lists together.
Prepared arenas include objective setup such as linked hazards.

The command uses the combat engine, a fixed timestamp and a synthetic build
named Preview with 300 HP and 20 damage. It advances a temporary in-memory run
between tiers without playing combat or banking rewards. It requires no database,
performs no network calls and neither loads nor modifies player saves. Identical
source content, mission and seed yield identical bytes. Changing the seed can
change enemy selection; it does not change authored mission geometry.

The catalog is `content.AbyssMobCatalog()`, matching the standalone author report
commands. It excludes the server-only regular/weekly/depth-100/lore/secret boss
union. This is explicit in the output. Use the
[browser fixtures](../../tests/e2e/README-brawl-fixtures.md) to inspect the server
union or class-specific visual behavior. A preview is not proof of difficulty,
reachability or successful objective completion.

Options:

- `-mission`: 1–100, default 1; out-of-range IDs are rejected rather than clamped.
- `-seed`: nonblank, at most 128 UTF-8 bytes, default `author-preview`.
- `-h`: show help. Positional arguments and unknown flags are rejected.

Exit codes are 0 for success/help, 1 for preview/output failure and 2 for invalid
arguments. `go run` wraps nonzero program exit codes; build a binary when exact
codes are needed by automation. Diagnostics go to stderr, JSON only to stdout.

To save a UTF-8 artifact in Windows PowerShell:

```powershell
New-Item -ItemType Directory -Force .tmp | Out-Null
python -c "import pathlib,subprocess; pathlib.Path('.tmp/brawl-mission-15.json').write_bytes(subprocess.check_output(['go','run','./cmd/brawl-mission-preview','-mission','15','-seed','forge-review']))"
```

This replaces only that file after the command succeeds. Retain the checkout
commit with a preview when comparing content revisions.

```sh
go test ./cmd/brawl-mission-preview -count=1
```

The tests compare repeated output for all 100 missions, require three complete
tiers, check wave/entry separation, verify seed-sensitive encounters and exercise
invalid arguments and output failure. For all authored definitions at once, use
[the campaign export](../brawl-campaign-export/README.md). For actual validation,
run [the campaign author gates](../brawl-validate/README.md).
