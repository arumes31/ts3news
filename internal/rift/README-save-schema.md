# Brawl save schema and migration contract

The persisted `rift.Run` currently uses `schema: 1`. The storage envelope and the
run schema are separate: `rift:gzip:v1:` identifies gzip/base64 transport, not a
second version of game state. `internal/bot/web_rift_snapshot.go` accepts legacy
plain JSON up to 2 MiB decoded; new writes use at most 256,000 stored bytes and
compress larger JSON within those limits. Never relax these bounds to migrate data.

## Current read/write path

`decodeRift` in `internal/bot/web_rift.go` decodes the envelope, unmarshals JSON,
checks schema/identity/room/timer-map presence, validates vitals, cooldowns, hazard
timing, actor positions and level metadata, then bounds receipt presentation to the
latest 200 items and event presentation to the latest 40 entries. It does not write the database. Invalid or unsupported saves
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

## Revision request identity

Optional `last_request_id` records the request that produced the current revision.
The normal transaction writes it with the snapshot, after successful simulation
and reward work. Equal-revision retries return the confirmed state only when the
request ID matches. A different ID produces a conflict without applying input or
banking. Older revisions still return newer state, which the browser treats as a
conflict and explicitly reloads before resuming. Start retries retain `start_key`.

Legacy saves remain readable without this field. Their equal-revision retries are
ambiguous and require reload; the next normal mutation records the new ID. Do not
invent a historical ID or accept an unknown winner as a confirmed retry. As with
other additive state, mixed old/new writers are unsupported: old writers drop this
field and do not enforce its arbitration rule. This prevents silent concurrent
control by rejecting stale actions, rather than assigning a permanent tab lease.
A player can explicitly reload and resume in a different tab; stale tabs must then
recover their state before controlling the expedition again.

## Uncertain client starts

The client retains one pending start ID and its request-content fingerprint per
practice mode in tab-scoped session storage. Only an unchanged explicit start
retry reuses that ID; mission, skills, previous run/revision or practice settings
changes create a new intent. Storage content supplies only a validated request ID,
never executable code or arbitrary request fields. Reads are size-bounded. Storage
failure falls back to page memory, so persistence across reload then depends on
session storage availability. Server start-key idempotency and active-run guards
remain authoritative regardless of browser storage.

A confirmed POST or GET whose `start_key` matches clears the pending metadata.
Recovery itself never automatically posts: a committed start is resumed from the
saved run, while an uncommitted unchanged start keeps its original identity when
the user tries again. Metadata contains request IDs and game selections, not
credentials, actor profiles, loot or inventory contents.

## Uncertain checkpoint banking

The client retains one pending banking ID per tab/practice mode. Its checkpoint
scope contains only run/epoch/mission/tier identity, room starting time, mission
attempt number and the last confirmed banking timestamp; the action must also match. A pause or resume
changes the optimistic revision without changing that checkpoint intent, so a
retry uses the current revision with the original banking ID. Changing action,
encounter, mission, run or economy produces a new intent. Session-storage reads
are bounded to 2 KiB and failures fall back to page memory.

Validated responses clear the entry when `last_request_id` confirms the bank, or
when the checkpoint is no longer the same cleared encounter. Thus recovery of a
committed advance/exit does not leave an ID available for a later checkpoint.
Recovery remains GET-only until explicit resume/action. Client metadata never
supplies loot, gold, inventory contents or arbitrary request fields; transactional
revision checks and stored banked flags remain authoritative for delivery.

## Bounded receipt history

`BankedItemsTotal` supplies the independent confirmed item count; absent or zero
legacy totals fall back to the complete `BankedItems` list through
`TotalBankedItems()`. Decode rejects negative, inconsistent nonzero, or JavaScript-
unsafe totals. `BoundReceiptHistory()` persists the derived legacy total in memory
before copying only the latest 200 names and provenance entries. The next normal
save persists that representation. Both lists release oversized backing storage.

Banking records the total and recent receipt only after each inventory insertion
succeeds; trimming does not skip delivery or change drop banking flags. Career,
encounter, HUD, terminal results, exports and banking-change announcements use the
independent count. The UI and copied receipt explain when only recent items are
shown. Old full receipts without the new total still render correctly.

Tests cover a 5,000-item legacy receipt, retained exact recent provenance and total,
idempotent saves, 250 inventory inserts despite the 200-entry presentation cap,
repeat settlement without redelivery, economy expiration and fresh expeditions.
Browser coverage checks desktop/mobile totals, filters, copied history notices,
record exports and feedback when the receipt length remains constant.

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


## PostgreSQL commit-uncertainty verification

`TestRiftActualBankCommitUncertainty` is an integration-tagged test using real
PostgreSQL and every embedded schema migration. Set `RIFT_TEST_DATABASE_URL` to a
disposable PostgreSQL administrator URL with CREATE DATABASE permission, then run:

```text
go test -tags=integration ./internal/bot -run '^TestRiftActualBankCommitUncertainty$' -count=1 -v -timeout=3m
```

The helper creates its own uniquely named database and drops that database on
cleanup; it never migrates or seeds the supplied administrator database. The test
injects confirmation loss at the SQL driver transaction boundary: real COMMIT or
ROLLBACK completes, then the connection closes and returns `driver.ErrBadConn`.
This covers both unknown outcomes without substituting mock reward storage.

For bank, exit, next and advance, recovery and late identical retries must produce
exactly one gear row, fight gold plus objective bonus once, one settlement
transaction in the economy ledger, and a matching persisted revision/request ID.
Row transaction IDs additionally prove balance, gear and run snapshot were written
together. Rolled-back attempts leave no inventory, ledger or snapshot changes.
The driver fault is deliberate test instrumentation, not a claim that every possible
network/server failure has been reproduced. Without the environment variable the
test skips; a skipped run is not integration evidence.
