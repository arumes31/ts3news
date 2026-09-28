# Brawl healing potions (integration in progress)

The engine action heals up to max HP, records actual healing and one potion use,
and sets an eight-combat-second cooldown in saved skill timers. It rejects full
health, death, paused/ended encounters, practice, cooldown and invalid amounts
without changing state. It reuses normal heal effects and sound presentation.

The campaign `potion` request accepts a `consumable_id` and runs after existing
owner, epoch, revision and replay checks. Standard healing items use the canonical
Abyss definitions (fixed amounts or fractions of Brawl max HP, rounded down).
Inventory decrement, zero-count cleanup and the updated run save share one SQL
transaction. Inventory count must be positive; duplicate requests return the
saved result before inventory mutation. Practice does not consume real items.

Corrupted items are not currently supported: their backlash must be implemented
before offering them. Buff, revive and repair consumables are also excluded.
The adapter currently uses base consumable values, not Abyss tree item-power
bonuses. That scope must be visible in the eventual player-facing presentation.

Focused engine tests pass for capped healing, use accounting, cooldown and all
rejection states. SQL transaction tests pass for success, missing inventory,
save-failure rollback and replay without a second charge. These focused checks
use sqlmock; real PostgreSQL recovery verification is recorded below.

Owned-item availability is exposed through the separate no-cache GET
`/api/abyss/rift?inventory=potions`. It returns canonical fixed/fractional healing
metadata and positive owned counts, excluding unsupported items. Practice returns
an empty array without reading real inventory. SQL and response tests pass.
Frequent combat snapshots do not query potion inventory.

The Healing potions panel now loads inventory on demand, lets the player choose
an owned item and queues one explicit use between combat updates. Selection and
repeat use are disabled while queued or in flight. Pause/input resets cancel an
unsent use; an uncertain response requires normal expedition recovery. Inventory
refreshes after confirmed use and recovery. Control-state tests pass for duplicate
clicks, pause cancellation, unavailable use and uncertainty. These tests use a
small DOM fixture; actual browser/server integration is still pending.

The optional Potions in reserve objective is now offered for new missions. It
uses a mission-start baseline of confirmed potion uses; any use fails it, while
healing skills and rejected uses do not. Saved progress, final completion/banking
and fresh-mission baselines pass focused tests. Existing objective tests also
pass. The normal five-gold optional-objective reward applies through existing
banking. Older saved objective lists are not retroactively expanded.

A saved eight-second potion cooldown remains frozen through pause and resumes
with combat time; the focused save/pause test passes.

Request/mode validation tests now cover missing and oversized potion IDs,
unrelated actions carrying an item, and practice isolation. Inventory error
messages remain visible through combat updates until refresh succeeds; the
control regression test passes.

The isolated E2E fixture and browser tests cover healing, inventory decrement,
objective failure, paused cooldown recovery, mobile layout and practice isolation.
The fixture compiles, but these browser tests have not run yet while the separate
long-session memory capture occupies the browser fixture.

The PostgreSQL integration test `TestRiftActualPotionCommitUncertainty` exercises
lost COMMIT and lost ROLLBACK confirmations with the existing disposable-database
helper. It checks inventory/run atomicity and repeated request recovery. Set
`RIFT_TEST_DATABASE_URL` to a disposable PostgreSQL server with CREATE DATABASE
permission to execute it; compilation or a skipped test is not live verification.

On 2026-09-28 the test passed against an isolated PostgreSQL 16 Alpine container
with real migrations and a disposable database (11.37 seconds). Both committed
and rolled-back lost confirmations passed, including two identical retries and
a shared transaction-ID check for inventory and the saved expedition. The
container was stopped after verification. This proves this potion transaction
path; it is not evidence for unrelated economy flows or browser presentation.

Still required: real-browser execution and inspection and broader integration
checks. Ledger 0363 remains open.
