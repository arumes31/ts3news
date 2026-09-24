# Brawl mission variant schema

The authoring source is [levels.go](levels.go), specifically `Campaign`, `Level`,
`Arena`, `Obstacle` and `Hazard`. Missions are generated Go content, not imported
JSON files. The JSON keys below describe the serialized form used by previews
and saved expeditions; they do not introduce a content-upload endpoint.

## Mission identity and variants

The campaign combines ten regions with ten layout blueprints. Zero-based
`region` and `layout` produce `id = region * 10 + layout + 1`. IDs are contiguous
1–100 and their slice position is `id - 1`; selection depends on this ordering.
Keep existing IDs stable when editing content. Display names must be unique.
[Startup validation](campaign_identity.go) rejects missing missions, duplicate or
out-of-range IDs, incorrect ordering, blank names, and names that collide after
trimming surrounding whitespace and ignoring case. Invalid compiled content
stops startup before game requests; historical saved expeditions are not rewritten.

| Level key | Type | Authoring contract |
| --- | --- | --- |
| `id` | integer | Stable mission identity, 1–100, matching slice position. |
| `region` | integer | Zero-based background panel and regional prop index, 0–9. |
| `name` | string | Unique display name, currently region plus layout name. |
| `region_name` | string | Human-readable area name. |
| `tactic` | string | Player-facing guidance matching the actual objectives. |
| `difficulty` | string | Wayfarer, Veteran, Champion or Mythic display tier. |
| `color` | string | Regional CSS color, currently a six-digit hex color. |
| `rooms` | Arena array | Exactly three entries, ordered tiers 1–3. |

A variant changes actual geometry, timing or encounters. Renaming a mission
alone does not establish a distinct level. Regional offsets, room offsets,
cover widths, floor materials and hazard periods are applied by `Campaign`.
Do not duplicate those rules in the browser. `Campaign()` returns fresh content;
callers may mutate their own expedition copy without modifying future runs.

## Arena contract

| Arena key | Type | Meaning |
| --- | --- | --- |
| `name` | string | Layout name and regional tier landmark. |
| `obstacles` | Obstacle array | Low cover; supports jumping over it. |
| `high_cover` | Obstacle array | Tall collision geometry. |
| `hazards` | Hazard array | Timed floor dangers, described below. |
| `floor` | string | grass, metal, ice, stone, mud, water or wood. |
| `objective` | string | Empty for ordinary combat, otherwise an implemented objective ID. |
| `max_attackers` | integer | Authored concurrent attack allowance; campaign uses 2, 3, 4 by tier. |
| `encounter` | object | Preview: `enemies`, `health_multiplier`, `damage_multiplier`. |
| `loot_rarity_ceiling` | string | Derived from `LootRarityCap`: Epic in tiers 1–2, Legendary in tier 3. |
| `camera_lead` | number | Preferred player screen X; zero uses the default 350. |
| `cover` | TerrainCover array | Breakable wood or permanent stone cover. |
| `platforms` | RaisedPlatform array | Walkable elevated surfaces with sloped edges. |
| `drop_edges` | DropEdge array | One-way descents with authored safe landing Y. |
| `steam_vents` | Obstacle array | Regional steam decoration geometry. |

Encounter previews must match the frozen monster plan after room and mission
scaling. The rarity ceiling is a cap, not a guaranteed drop. Use the shared
server helpers instead of separately authored preview numbers or gear rolls.
See [monster synchronization](README-monster-sync.md) for catalog ownership.

An `Obstacle` is `{x, y, w, h}` in world units, with positive dimensions.
Actor travel uses the ground lane; spawn/route validation must account for the
actor footprint, not merely a point. Do not infer passability from artwork.

Additional terrain records extend this rectangle as follows:

- `TerrainCover`: `id`, `material`, `hp`, `max_hp`. Stone stays solid; wood is
  solid while HP is positive. Broken cover is saved with the arena.
- `RaisedPlatform`: `id`, `rise`, `ramp`, `floor`. Require positive rise/ramp,
  `rise <= ramp`, and `2 * ramp <= min(w, h)`. Elevation does not provide jump
  immunity. See [platforms.go](platforms.go).
- `DropEdge`: `id`, `x`, `y`, `w`, `landing_y` (no rectangle height). Landings must
  fit the actor and avoid solid cover and enabled hazards. See
  [drop_edges.go](drop_edges.go).

## Hazard contract

Hazards flatten their rectangle into `x`, `y`, `w`, `h` and add:

| Key | Type | Contract |
| --- | --- | --- |
| `kind` | string | fire, ice, poison, thorns, rune, radiant, void or spikes. |
| `period` | seconds | Greater than `1.2 + duration`, leaving a safe phase. |
| `offset` | seconds | Nonnegative cycle offset; phase is `(clock + offset) % period`. |
| `duration` | seconds | Positive active interval after the 1.2-second warning. |
| `jumpable` | boolean | Explicitly authored; current campaign floor hazards use true. |
| `disabled` | boolean | Runtime state for disabled hazards; normally false initially. |
| `tile` | boolean | Optional checkerboard tile presentation. |
| `generator_id` | string | Optional link to the objective generator controlling this hazard. |

Warning occupies phase `[0, 1.2)`; damage occupies
`[1.2, 1.2 + duration)`. Preserve a navigable safe route and enough escape time.
Overlapping rectangles and linked timings need review across the full cycle.
Saved jumpability migration is handled by [hazard_json.go](hazard_json.go);
new authoring should always set the flag explicitly.

## Objective identifiers

These are arena objectives, separate from optional reward challenges in
[objectives.go](objectives.go). Their current layout assignments are:

| Layout | Tier 1 | Tier 2 |
| --- | --- | --- |
| Pilgrim's Gate | Combat | `interrupt_ritual` except mission 1 |
| Broken Well | Combat | `moving_beacons` |
| Pillar Watch | Combat | `sigils` |
| Crossroads | Combat | `hold_circle` |
| Twin Bastions | `escape_collapse` | `survive_waves` |
| Serpent Walk | `linked_guardians` | `destroy_totems` |
| Hidden Alcoves | `rescue_companions` | `carry_relic` |
| Shattered Bridge | `protect_lantern` | `disable_generators` |
| Sentinel Rows | `rune_gate` | `marked_hunt` |
| Crown Arena | `split_defense` | `escort_spirit` |

Tier 3 uses ordinary combat, with alternating hazards in Crossroads and
checkerboard hazards in Sentinel Rows. A new objective string needs engine,
renderer, controls, completion and persistence support; adding a label does
not implement its behavior.

## Saved content-version labels

A saved expedition's `mission_definition` is its frozen mission-content label.
[record_definition.go](record_definition.go) computes `level-v1:` followed by
64 lowercase hexadecimal SHA-256 characters over the serialized `Level` fields,
excluding the derived `definition` field itself. The algorithm prefix versions
the fingerprint format; it is not a deployment version.

The label is captured by `beginMissionHistory` before mission play. It persists
through saves and tier transitions and is copied into completion/attempt records.
Selecting the next mission captures that mission's label. Runtime changes such
as broken cover must not replace the captured label with a recomputed value.
Personal bests are grouped by the captured definition, so changed terrain is not
silently treated as the same challenge.

Serialized `Level.definition` is a separate, derived hash of the level as it is
serialized now. It may differ from `mission_definition` after mutable arena state
changes. Use the saved run's frozen field for attempt provenance, not that derived
field. Old saves with no captured label remain unknown; do not infer their initial
content from today's catalog. Nil or unencodable definitions also remain unknown.

The hash covers the complete mission definition, including names, presentation,
terrain, objectives and encounter preview values. It does not fingerprint combat
engine code, the selected monster catalog, player equipment, assets or a deployed
binary. Record the build revision separately when those distinctions matter.
`Run.schema` and the storage compression prefix describe serialization, not content.

## Saves, editing and verification

`setLevel` copies the authored level into the run and freezes its encounter
plan. Resuming that run uses saved geometry, including broken cover and disabled
hazards. Content edits apply when a new mission is selected; they must not
silently replace an existing saved arena. Maintain serialization compatibility
when changing these types.

After an edit, run the [campaign author gate](../../cmd/brawl-validate/README.md):

```sh
go run ./cmd/brawl-validate
go test ./internal/bot -run 'TestRiftRegionBackgroundPanelsCoverCampaign|TestRiftCoverPropIndicesFitAtlas' -count=1
```

The first command checks engine behavior, population/reward budgets, walking
reachability and hazard route intervals. The second checks actual embedded
background/prop atlas references. Inspect the reports and play the changed
missions with [reproducible fixtures](../../tests/e2e/README-brawl-fixtures.md).
These checks do not by themselves prove difficulty, timed escape feasibility,
visual quality or release readiness; use the [release checklist](README-release.md).

For an offline copy of every definition and objective totals, use the
[static campaign export](../../cmd/brawl-campaign-export/README.md).

For seeded encounter and entry-state inspection of one mission, use the
[mission preview command](../../cmd/brawl-mission-preview/README.md).

For source crop layouts, foot placement and the separation between painted art
and physics, see the [art contract](README-art-contract.md).
