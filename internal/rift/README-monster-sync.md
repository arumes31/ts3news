# Adding Abyss monsters to Brawl

Brawl builds its encounter pool from Abyss content. Add a monster to its owning
Abyss source; do not create a second Brawl roster. A newly eligible monster can
be selected in future missions, but random selection does not guarantee it will
appear in every run.

## Source to encounter

[The canonical catalog](../content/mobs.go) returns detached copies of authored
base templates plus named treasure goblins. These are authored identities, not
every procedural stat or name variant produced during an Abyss encounter roll.

[The server union](../bot/web_rift_monsters.go) starts with that catalog, then adds:

- The regular Abyss boss roster and weekly boss names.
- The depth-100 boss and bosses named in the lore catalog.
- Every monster returned by each secret boss encounter definition.

Additional entries are deduplicated by exact name; an existing canonical entry
wins over an additional source with the same name. The union is sorted by name.
Named bosses use the Abyss boss constructor with the current daily affinity.
The same union supplies the bestiary and new practice selections.

[AdaptMonster](monsters.go) preserves the name, canonical tier and element, and
uses `monster:` plus the exact name as the art identity. It converts canonical
stats into action-combat health, damage, armor and speed. Tier selects the initial
combat role; spells, relative intelligence and name hints can select ranged
behavior. Element and name hints select projectile effects. This conversion does
not import every Abyss spell's turn-based behavior as a bespoke Brawl attack.

Encounter planning separates bosses from regular enemies, shuffles the regular
pool using the run seed, and chooses a boss for the final tier. Mission rules
then apply encounter scaling and composition. No generated per-monster manifest
or Brawl registration is needed for a new catalog entry to become eligible.

## When content updates take effect

| Action | Roster used |
| --- | --- |
| Start a new production campaign | Current server union. |
| Advance seamlessly to the next mission | Current server union, with a newly frozen mission plan. |
| Enter the next tier of the current mission | Saved encounter plan. |
| Reload or retry an eligible saved boss fight | Serialized actors and frozen encounter data. |
| Request the bestiary or a new practice selection | Current server union. |
| Spawn a room from a legacy save without a complete plan | Canonical `content.AbyssMobCatalog()` fallback, which excludes the server-only extended union. |

Sync means the running server reads its current content sources when it creates
new plans; editing source files still requires the normal build/restart process.
Existing mission plans deliberately retain their identities and adapted values.
Shared AI and combat code can still change their behavior across software updates.
See [removal and rename compatibility](../../docs/brawl-removed-monsters.md) before
changing an existing name: names are identities and have no automatic aliases.

## Art and author checks

The shared combat-art profile can derive a rig from an actor's saved name, role
and element without an exact catalog match. The renderer falls back to the base
mob atlas if the shared frame or image is unavailable. A bespoke sprite is optional
for encounter eligibility; preserve fallback assets and old saved identities when
adding art. See [shared profiles](../bot/webassets/abyss_combat_art.js) and
[the renderer](../bot/webassets/rift_renderer.js).

After adding content:

1. Check the owning Abyss source and exact name for accidental identity collisions.
2. Verify canonical tier, stats, element and spell hints produce the intended
   Brawl role. Boss/legendary entries should enter the boss pool.
3. Check bestiary and practice selection, then inspect a newly planned mission.
   Use deterministic tests or practice for a specific identity instead of assuming
   one random campaign must roll it.
4. Verify old saved fights still load and render. New source eligibility should
   not replace their saved actors.
5. Run the focused roster regressions from the repository root:

```powershell
go test ./internal/rift -run 'TestCatalogEncountersCoverEveryTemplate|TestNewCatalogEntryNeedsNoBrawlRegistration|TestFutureCatalogTiersEnterCampaignWithoutRegistration' -count=1
go test ./internal/bot -run 'TestRiftCatalogIdentityMatchesEverySharedSource|TestRiftRosterTracksAllAbyssCatalogs|TestRiftBestiary|TestRiftBossPracticeAcceptsWholeLiveRoster' -count=1
```

For mission composition changes, also run the
[campaign author validator](../../cmd/brawl-validate/README.md). Its population,
reachability and hazard reports check campaign structure; they do not prove every
monster's attack balance or visual quality.

The complete identity gate compares the bestiary against every source above,
including the depth-100 boss and every actor returned by secret encounters.
It rejects unknown/duplicate Brawl identities, verifies canonical tier, element
and art keys, and checks boss-pool classification across seven dates. Each
source is also planned alone to verify identity survives encounter creation
without relying on random full-roster coverage. Existing seeded roster tests
continue to cover mixed-pool selection.
