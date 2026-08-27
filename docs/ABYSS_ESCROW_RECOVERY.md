# Abyss escrow recovery drill

`cmd/abyss-escrow` protects active runs, escrowed loot, and persisted live
combat ownership without exposing recovery through the web portal. Snapshot
files contain player identifiers and item payloads; store them like database
backups and restrict access to operators.

## Routine integrity check

Run against the same `DATABASE_URL` used by the bot:

```sh
go run ./cmd/abyss-escrow -mode check
```

The command exits non-zero if loot has no active run, a combat session has no
active owner or owner member, a loot payload is not a JSON object, or a member
references a missing session. Schedule this check before backups and after
deployments that change Abyss persistence.

## Snapshot and offline verification

Create a new file with owner-only permissions. Existing files are never
overwritten:

```sh
go run ./cmd/abyss-escrow -mode backup -file /secure/abyss-escrow-2026-08-27.json
go run ./cmd/abyss-escrow -mode verify -file /secure/abyss-escrow-2026-08-27.json
```

The snapshot is captured in a repeatable-read transaction and includes a
SHA-256 checksum plus row counts. Verification is offline, bounded to 256 MiB
by default, and rejects unsupported formats, tampering, duplicate identities,
or broken cross-table ownership. A backup is still preserved when existing
relational corruption is detected so operators retain forensic evidence.

## Rollback-only restore drill

Exercise decoding and the current PostgreSQL table shapes without changing
live rows:

```sh
go run ./cmd/abyss-escrow -mode drill -file /secure/abyss-escrow-2026-08-27.json
```

The drill validates the snapshot before connecting, creates transaction-local
tables from the current production schema, imports every row, round-trips the
four tables, compares the checksum, and explicitly rolls the transaction back.
It does not provide a live restore switch. An actual disaster restore remains
an operator-reviewed database operation after a successful drill and a normal
full-database backup.
