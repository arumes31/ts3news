# Removing Abyss monsters without rewriting saved Brawl fights

Brawl reads Abyss's live roster for new encounters, but saves adapted actors in
an encounter plan for the current mission. Removing a monster from the catalog
does not remove that actor from an already-saved fight. This is intentional:
players should not lose a boss attempt or have its stats replaced on reload.

## What changes, and when

| Situation | Behavior after a monster is removed |
| --- | --- |
| Reload an existing run | JSON decoding restores saved enemies and the encounter plan. Loading does not look up each monster in the current roster. |
| Continue into another room of the same mission | Uses the saved room's actors when a full encounter plan exists, including removed names and their saved stats. |
| Retry a defeated boss | Uses the frozen boss encounter plan. A catalog deletion does not disqualify that boss by name. Normal retry eligibility still applies. |
| Seamlessly advance to a new mission | Builds a new encounter plan from the current server roster. Removed monsters are no longer eligible if absent from every source in that roster. |
| Start a new campaign run | Builds encounters from the current roster. Existing history can carry forward; the old actor plan does not. |
| Load an old run without a full encounter plan | Retains its saved current enemies until a room spawn is needed. At that point the engine builds a plan from `content.AbyssMobCatalog()`. This legacy fallback is narrower than the server's extended boss roster. |
| Reload selected-enemy skills practice | Restores the saved actor; it does not require a fresh catalog lookup. A new spawn request for a removed name fails. Clearing still removes the actor; resetting restores the normal drill target. |
| Reset boss practice without a new selection | Uses the saved `BossStart` actor. An explicit new boss selection is validated against the current server roster and rejects a removed name. |
| Browse the bestiary or choose a new practice enemy | Uses the current roster, so removed entries disappear from new selections. |

The production roster is a union: canonical Abyss mobs plus named bosses, weekly
bosses, depth/lore bosses and secret encounters, deduplicated by exact name.
Removing a name from one source is insufficient if another source still adds it.
The engine's `AdaptMonster` conversion derives action-combat stats from these
sources; no separate hand-maintained Brawl roster is required.

## Identity, records and presentation

Actor art/record identity is `monster:` followed by the exact monster name.
Renaming a monster therefore creates a different identity. There is no automatic
alias migration from the old name to the new name. Do not rename a monster and
assume old records/bookmarks will merge with its replacement.

Saved actors retain their name, art key, kind, element, stats and encounter data.
Monster records and encounter monster keys may retain the old identity. The
bestiary only renders current roster entries; encounter-result links are shown
only when their keys resolve to a current entry. A missing link does not mean
the saved encounter has been deleted.

Rendering uses the shared combat-art profile for the saved actor rather than
requiring membership in the current bestiary. Removing a catalog entry is not
permission to delete sprite sheets or profile support used by older saves.
Missing legacy artwork is a separate compatibility concern: verify the fallback
before removing an asset.

Frozen actors do not freeze the entire game implementation. Future changes to
shared AI, collision, damage formulas or art-profile selection can still affect
saved fights. The guarantee here is preservation of the serialized encounter
identity and actor values, not bit-for-bit simulation across software versions.

## Content-edit review

When retiring or renaming an entry, check all production roster sources and
confirm the intended new-run roster. Preserve a saved run containing the old
entry and verify reload, next-room entry and eligible boss retry. Also verify
that advancing to a new mission uses the updated roster. For practice, check
saved boss reset separately from a new explicit selection.

Do not erase saved enemies, substitute a different monster by name, or rewrite
old receipts as part of a roster-only edit. If identity migration is actually
needed, design an explicit migration for actors, encounter plans, practice boss
snapshots, records, bookmarks and art keys together.

## Implementation references

- [Live server roster and bestiary](../internal/bot/web_rift_monsters.go)
- [Actor adaptation and encounter planning](../internal/rift/monsters.go)
- [Saved-run decoding and loading](../internal/bot/web_rift.go)
- [Room spawning and legacy plan fallback](../internal/rift/combat.go)
- [New-mission planning](../internal/rift/levels.go)
- [Frozen boss retry](../internal/rift/boss_retry.go)
- [Practice reset](../internal/rift/practice.go)
- [Explicit boss selection](../internal/bot/web_rift_boss_practice.go)
- [Selected-enemy practice](../internal/rift/practice_enemy.go)
- [Bestiary identity lookup](../internal/bot/webassets/rift_bestiary.js)
- [Encounter-result links](../internal/bot/webassets/rift_hud.js)
