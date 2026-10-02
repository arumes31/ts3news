# Brawl client control contract

The authenticated account is supplied by the server, never by the JSON body.
Use `GET /api/abyss/rift` to load a campaign snapshot and POST to the same URL to
submit controls. Add `?practice=<mode>` for the isolated practice snapshot.
An omitted/empty practice query selects campaign; unknown modes are rejected.
See [practice and rewards](brawl-practice-and-rewards.md) for the storage boundary.

POST requires `Content-Type: application/json`. The production handler limits
the body to 4096 bytes, rejects unknown fields and trailing JSON, validates
same-origin headers when present, and rejects cross-site fetches. JSON field
types below describe client output; omitted fields use Go zero values unless
noted. Length limits on strings are UTF-8 byte limits (`len` in Go).

## Top-level fields

| Field | Type | Meaning and validation |
| --- | --- | --- |
| `kind` | string | Action name from the list below; also checked against the URL's mode. |
| `run_id` | string | ID from the last saved snapshot, at most 80 bytes. Empty before the first start. Mutating an existing run requires matching its ID. A new start replacing an old terminal run also identifies that old run. |
| `request_id` | string | 16–80 bytes; browser sends a UUID. Deduplicates a repeated start through saved `StartKey`; also supplies economic request context. It does not replace revision checks for other actions. |
| `revision` | integer | Nonnegative. Browser sends last saved revision plus one (1 when no run exists). For existing-run actions, a revision at/below the saved value returns saved state; skipping ahead conflicts. Run ID and economy epoch checks still apply. |
| `input` | object | Movement/action state for `step`; browser sends `{}` for other actions. Fields below. |
| `skills` | string array | On start, choose at most three distinct owned optional skill IDs, each at most 100 bytes. Omitted/null retains the server's default first three available skills; an empty array selects none. Class signatures and the ultimate use their separate build slots. |
| `level_id` | integer | Campaign mission selection on start: 1–100. Omitted/0 starts mission 1. Negative or greater than 100 is rejected at request validation. Practice uses its own arena. |
| `boss_name` | string | Exact live-roster boss name, at most 512 bytes, for boss-practice start/reset selection. Supply together with a valid phase. Other actions do not select bosses. |
| `boss_phase` | integer | Boss-practice starting phase 1–3; request validation allows 0 as the omitted value. Name and phase both omitted use default boss setup on start or reset the saved setup. Partial/nonexistent selections fail constructor validation. |
| `slow_telegraphs` | boolean | Optional boss-practice start/reset setting for doubled warning time. Omitted preserves the saved setting on reset and defaults false for a new drill. Does not slow projectile travel or recovery. |
| `hazard_intensity` | string | Hazard-practice start/reset preset: `gentle`, `standard`, `intense`, or omitted/empty. New drills default to standard; omission on reset preserves the saved preset. Nonempty intensity outside hazard practice is rejected by practice setup. |
| `enemy_name` | string | Exact live-roster monster name for `practice_spawn`, at most 512 bytes and nonblank for that action. Selection is accepted only in active free skill practice. |

## Step input fields

| Field | Type | Meaning and validation |
| --- | --- | --- |
| `x` | number | Horizontal axis in [-1, 1], including analog fractions. Negative moves left; positive moves right. |
| `y` | number | Depth axis in [-1, 1]. Negative moves up; positive moves down. |
| `attack` | boolean | Request a basic attack. Cooldowns, guard/recovery and melee geometry determine its result. |
| `guard` | boolean | Hold directional guard. The simulation determines facing, timing and blocked damage. |
| `jump` | boolean | Request jump. Server cooldown/state determines acceptance. |
| `skill` | string | Equipped ability ID to cast, at most 100 bytes. Empty requests no cast. Ownership/build membership, mana and cooldowns are checked by the server. |

Axes must be finite; NaN/infinity are not JSON numbers and out-of-range values
fail movement validation. Practice may filter input to match its exercise.
No client timestamp, damage amount, HP, loot amount, completion flag or account
identifier is accepted as an input field. The server advances and saves combat.

## Action scope

- Shared campaign/practice actions: `start`, `step`, `pause`, `resume`.
- Campaign only: `bank`, `next`, `advance`, `exit`, `retry_boss`.
- Practice only: `practice_reset`, `practice_health`, `practice_mana`,
  `practice_cooldowns`, `practice_freeze`.
- Free skill practice only: `practice_spawn`, `practice_clear`.
- Banking demonstration only: `practice_bank` (never real economic banking).

Scope alone does not authorize an action: the current status, checkpoint/drill
conditions and revision must also permit it. Practice setup rejects missing
required equipped abilities. Reload authoritative state after an uncertain reply;
do not increment revisions and repeat a banking action to guess whether it ran.

## Implementation references

- [Request struct, HTTP validation, action scopes and transactions](../internal/bot/web_rift.go)
- [Input struct and combat acceptance](../internal/rift/combat.go)
- [Practice start/reset options](../internal/bot/web_rift_boss_practice.go)
- [Hazard presets](../internal/rift/hazard_practice.go)
- [Browser request construction](../internal/bot/webassets/rift.js)
- [Response validation](../internal/bot/webassets/rift_protocol.js)

Keep the tables synchronized with every JSON field in `riftRequest` and
`rift.Input`. New actions need explicit scope, state checks and replay tests.
