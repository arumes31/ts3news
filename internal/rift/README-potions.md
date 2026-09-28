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
save-failure rollback and replay without a second charge. These are sqlmock
checks; no live database economy test has run yet.

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

Still required: request/mode validation coverage, fixture inventory and real-
browser controls/recovery/objective tests, broader integration checks, and the
live-database economy verification. Ledger 0363 remains open. No potion UI has
been verified in a real browser yet.
