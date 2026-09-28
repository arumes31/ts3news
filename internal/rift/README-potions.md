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

Still required: player controls and
recovery feedback, the no-potion optional objective, cooldown save/pause coverage,
request/mode validation coverage, and real-browser integration. The objective is
not offered yet, so it cannot award an automatic success without a potion action.
Ledger 0363 remains open. No potion UI has been verified.
