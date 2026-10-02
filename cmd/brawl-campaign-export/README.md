# Export the Brawl campaign

From the repository root, with Go on PATH:

```sh
go run ./cmd/brawl-campaign-export
```

Writes one indented JSON document to stdout. It contains schema version 1,
mission/room/region totals, counts of rooms by objective, and the complete
serialized `rift.Campaign()` definitions in mission order. Empty objective IDs
are summarized as `combat`; individual mission definitions retain their original
empty value. Definitions include names, tactics, encounter previews, rarity
ceilings, geometry, hazards, platforms, cover, drop edges and regional presentation.

There is no timestamp or random sampling, so the same authored content produces
identical output. This is an export of the current checkout's compiled content,
not a server query or a player's saved expedition. It contains no player data,
requires no database and performs no game actions. Mission encounters and rolled
gear are not generated; use the population/reachability reports for planned
actors and the browser fixtures for visual previews.

To retain a UTF-8 artifact in Windows PowerShell without its legacy redirect
encoding behavior, capture stdout as bytes:

```powershell
New-Item -ItemType Directory -Force .tmp | Out-Null
python -c "import pathlib,subprocess; pathlib.Path('.tmp/brawl-campaign.json').write_bytes(subprocess.check_output(['go','run','./cmd/brawl-campaign-export']))"
```

This replaces only the named output file after a successful command. Keep the
checkout commit alongside the export when sharing or comparing revisions.
The schema version describes the export envelope; it is not a saved-game version.
Unknown options or positional arguments exit 2; output failures exit 1; success
and `-h` exit 0. `go run` wraps nonzero program exits, so compile the command if
an automated consumer needs exact exit codes.

```sh
go test ./cmd/brawl-campaign-export -count=1
```

Tests require complete equality with authored campaign definitions, stable bytes,
summary counts and reported failures. An export is descriptive, not a validation
result. Run the [author gates](../brawl-validate/README.md) before accepting content
changes, and consult the [mission schema](../../internal/rift/README-mission-variants.md)
for field semantics and save compatibility.
