# Brawl practice and campaign rewards

Practice uses the player's Abyss build and Brawl combat rules, but it is not a
reward-bearing expedition. Practice exists to learn controls, class sequences,
enemy attacks and checkpoint behavior without changing the campaign save.

| Behavior | Campaign expedition | Practice drill |
| --- | --- | --- |
| URL scope | `/abyss/rift` | `/abyss/rift?practice=<mode>` |
| Saved snapshot key | `rift_brawl:<account>` | `rift_practice:<account>:<mode>` |
| Build | Abyss character, selected abilities and equipment snapshot | Same build system; drill setup may adjust training HP or targets |
| Enemy defeat | Can produce expedition drops and records | No campaign drops, defeat records or kill credit |
| Collected loot | Unbanked until the server confirms banking | Training tokens have no gold, gear or pending gear roll |
| Banking | Transaction may credit gold and inventory and save a receipt | Real banking is rejected; the banking demo only marks valueless tokens |
| Completion | Campaign progression and applicable records | Drill completion; eligible best times are local browser records |
| Reset | Separate campaign restart/continuation rules | Recreates the current drill from its saved build; no campaign reset |

## Save and reward boundaries

The practice query parameter selects a validated mode, not a client assertion
that a campaign run is safe to modify. `riftModeKey` isolates account and drill
storage. `matchingRiftMode` rejects a snapshot from the wrong scope.
`validRiftModeAction` rejects campaign economic actions in practice and practice
actions in campaign before the update transaction starts.

`bankRift` also rejects every run with a non-null `Practice` state before setting
an economy context or writing inventory/gold. Keep this guard even though the
request layer already checks scope: callers must not be able to bypass the
reward boundary by reusing a lower-level helper.

Practice uses the usual run ID, revision and economy epoch checks. Duplicate
revisions return saved state; they do not repeat a drill action. The account/drill
snapshot is saved transactionally. Browser recovery reads that saved state when
a reply is lost or incomplete, rather than guessing whether an action succeeded.

Sources: [request routing, storage and banking](../internal/bot/web_rift.go),
[practice engine and reset](../internal/rift/practice.go),
[combat pickup and practice defeat handling](../internal/rift/combat.go).

## Practice-specific controls

- Health/mana refill and ability cooldown reset affect the practice snapshot.
  The UI pauses before applying these controls. They do not advance the campaign.
- Freeze stops enemy movement; attacks, warnings and projectiles continue.
  Freeze-assisted local best times are kept separately.
- Free skill practice can replace its dummy with one monster from the current
  Abyss roster, including bosses. Selected monsters fight and can be defeated;
  the default dummy remains immortal. Selection and clearing remove old
  projectiles and target marks. Clearing does not refill player resources.
  Reset restores the default dummy. Other drills cannot use these enemy controls.
- The pickup demo uses three zero-value tokens and the real pickup radius.
- The banking demo requires collecting all three tokens and reaching the
  checkpoint before `practice_bank` is accepted. It persists a completed demo
  receipt by marking those tokens banked. It does not call `bankRift`, credit
  gold, create inventory items or complete a campaign mission.

Sources: [enemy controls](../internal/rift/practice_enemy.go),
[live roster action adapter](../internal/bot/web_rift_practice_enemy.go),
[browser controls](../internal/bot/webassets/rift.js),
[local practice records](../internal/bot/webassets/rift_records.js).

## Evidence and maintenance

Engine tests verify the actual pickup boundary, explicit checkpoint confirmation,
full-charge practice cycles and zero-reward monster defeat across the live roster.
Transaction tests verify isolated keys, rejected premature actions and replayed
receipts without new writes. Browser journeys verify real controls, persistence,
reset and an unchanged authoritative campaign snapshot.

Relevant tests:

- [Practice scope/storage](../internal/bot/web_rift_practice_test.go)
- [Enemy action transactions](../internal/bot/web_rift_practice_enemy_test.go)
- [Banking demo transactions](../internal/bot/web_rift_banking_practice_test.go)
- [Enemy catalog/no rewards](../internal/rift/practice_enemy_test.go)
- [Pickup boundary](../internal/rift/pickup_practice_test.go)
- [Banking demo engine](../internal/rift/banking_practice_test.go)
- [All-subclass resource journey](../tests/e2e/rift-resource-practice.spec.js)
- [Enemy selection journey](../tests/e2e/rift-practice-enemy.spec.js)
- [Pickup journey](../tests/e2e/rift-pickup-demo.spec.js)
- [Banking journey](../tests/e2e/rift-banking-demo.spec.js)

When adding a drill, keep its economy effects confined to demonstration state,
add its mode to protocol validation, enforce its allowed actions server-side,
and verify that playing/resetting it leaves a previously saved campaign intact.
Do not infer safety solely from practice labels or disabled buttons.
