# Brawl save schema and migration contract

The persisted `rift.Run` currently uses `schema: 1`. The storage envelope and the
run schema are separate: `rift:gzip:v1:` identifies gzip/base64 transport, not a
second version of game state. `internal/bot/web_rift_snapshot.go` accepts legacy
plain JSON up to 2 MiB decoded; new writes use at most 256,000 stored bytes and
compress larger JSON within those limits. Never relax these bounds to migrate data.

## Current read/write path

`decodeRift` in `internal/bot/web_rift.go` decodes the envelope, unmarshals JSON,
checks schema/identity/room/timer-map presence, validates vitals, cooldowns, hazard
timing, actor positions and level metadata, then bounds event presentation to the
latest 40 entries. It does not write the database. Invalid or unsupported saves
return an error; they are not silently reset or replaced with a fresh expedition.

Legacy schema-1 runs can lack level metadata, mission history, content-version
hashes, practice state and other optional fields. Missing fields retain Go zero/nil
values; optional maps must be initialized before mutation. A missing historical
value is not evidence of a zero-duration clear, completed objective or new reward.
Do not synthesize such evidence during recovery. `skill_timers` is required by the
existing schema and is not one of these optional fields.

Saved terrain, labels and mission definitions belong to the expedition. Validation
checks structural invariants without replacing those values from today's campaign.
Economic epoch expiration is handled separately by `loadRiftMode`; it is not a
schema migration. Inventory delivery remains in the banking transaction.

## Adding a compatible field

1. Add an optional JSON field with a safe zero/nil interpretation. Use a pointer
   when absence differs from a genuine zero. Document that distinction at the type.
2. Derive a missing value only from authoritative information already in the save.
   Keep normalization pure and idempotent, before validation that requires it.
   Preserve run ID, revision, economy epoch, request identity, rewards, banked flags,
   campaign history and content versions.
3. Test literal old JSON with the field absent, explicit zero, valid populated data,
   corrupt values, and repeated decode/encode. Constructors alone cannot prove old
   data compatibility. Exercise both plain and compressed envelopes when relevant.
4. Persist only through the existing successful transaction/save path. A GET must
   not perform reward delivery or a destructive backfill as a side effect of decode.

JSON reads tolerate unknown optional fields, but older writers discard fields they
do not understand. This is read compatibility, not safe mixed-version writing.
Do not deploy older writers against saves whose new fields carry authoritative
meaning. Keep the previous binary and save format compatible during rollout, or
coordinate writer replacement and rollback with a versioned migration.

## When a new schema is required

Changing the meaning of an existing field, losing information, or requiring new
state that cannot be safely inferred requires a schema bump. A future migration
must dispatch by stored schema after envelope decode, apply explicit sequential
pure transformations (for example 1 to 2), validate the result, and write the new
schema only within the normal transaction. Such a migration is not implemented
now. Unknown future schemas currently fail closed; do not change that to assume
schema 1 or bypass validation. Keep old-save fixtures and failure atomicity tests
for every supported conversion. A rollback must either understand the new schema
or restore from a tested backup; it must never replay banking to reconstruct it.

## Receipt compaction prerequisite

`BankedItems` currently supplies authoritative expedition item counts, while
`BankedLoot` supplies detailed receipt provenance. Simply truncating either is
not a complete compaction solution. Introduce a separate authoritative total,
deriving an absent total from the full legacy item list before trimming. Update
career totals, encounter summaries, HUD, receipt/export text and banking-change
announcements to use it. Bound presentation independently and tell the player
when only recent entries are shown. Prove every collected item is inserted once,
that totals survive reload/expiration/new expeditions, and that retries never
redeliver inventory. This work remains pending under improvement 0825.

## Verification

`TestRiftDecodeLegacyOptionalFields` exercises literal legacy JSON, unchanged
identity/rewards, optional absence, repeated save/load, harmless unknown fields,
and unsupported schema rejection. The `TestRiftDecode*` tests cover field validation,
all 300 campaign tiers and simulated movement roundtrips. Snapshot codec tests
cover bounded plain/compressed storage; saved-content tests cover historical
mission identity across tier transitions.

Run the focused backend checks from the repository root:

```text
go test ./internal/bot ./internal/rift -run 'Test(Rift|Campaign|Hazard|EventHistory|SavedContent)' -count=1
```
