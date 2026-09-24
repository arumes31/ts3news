# Brawl local fixture persistence and resets

The [Brawl fixture](../../internal/bot/web_rift_e2e_test.go) is compiled only with
`-tags=e2e`. It uses the production simulation with in-memory character and reward
storage. It does not prove production database persistence, transaction behavior,
or real inventory rewards; those require the server/storage tests.

## What resets state

| Operation | Result |
| --- | --- |
| Load a recognized `scenario` URL | Recreates that scenario on every page request, including reload. Most scenarios replace the campaign slot. |
| Load `scenario=practice-tools&practice=<valid-mode>` | Replaces only that practice mode's slot. |
| Include `subclass` on the page URL, even empty or unchanged | Rebuilds the selected character and deletes the campaign slot before scenario initialization. Invalid values fall back to vanguard. Existing practice slots are not deleted by this branch. |
| Reload `/abyss/rift` without reset parameters | Keeps the current campaign in the same fixture session. |
| Read `/api/abyss/rift` | Reads state; does not apply page scenario initialization. |
| Restart the fixture server | Loses all in-memory runs and builds. Browser preferences and cookies may remain. |
| Use a fresh browser context | Gets a separate cookie session and fresh browser storage. |
| Open another tab in the same context | Shares the fixture cookie, server slots and origin-local preferences. It is not an isolated player. |

The HTTP-only `rift_fixture` cookie identifies a session. The campaign uses its
cookie value as the storage key; each valid practice mode uses a separate
`cookie:mode` key. Visit the page first to establish the cookie: an API request
without it returns 401. Losing the cookie makes the old in-memory session
inaccessible to that browser; it does not erase that session from the server.

Browser settings such as automatic tier advancement, reduced motion and transition
delay live in local storage. Scenario resets and server restarts do not reset
these settings. For a fully clean test, use a fresh context and deliberately set
the viewport, motion preference and relevant game settings.

## Reload a saved run without reseeding

Initialize the scenario once. Before testing reload persistence, remove both
`scenario` and `subclass` without navigating. Keep `practice` when testing a
practice slot:

```javascript
await page.goto('/abyss/rift?scenario=checkpoint');
await expect(page.locator('#rift-start')).toBeEnabled();
// Perform the interaction whose saved result is under test.
await page.evaluate(() => {
  const url = new URL(location.href);
  url.searchParams.delete('scenario');
  url.searchParams.delete('subclass');
  history.replaceState(null, '', url.pathname + url.search + url.hash);
});
await page.reload();
await expect(page.locator('#rift-start')).toBeEnabled();
```

Pausing before the reload makes combat-state comparisons easier. Assert specific
persisted fields instead of byte-equal snapshots containing timestamps. To
intentionally reset, navigate to the scenario URL again. Unknown scenario names
have no general reset behavior; use an implemented scenario or a fresh context.

A fixed scenario name is not a guarantee of byte-identical fixtures: constructors
use current time, the live boss roster can reflect daily affinity, and checkpoint
gear is rolled. Visual comparisons must control those sources separately.

## Running and verifying

[Playwright configuration](../../playwright.config.js) defaults to loopback port
18082 and starts `TestAbyssE2EServer`. `ABYSS_E2E_PORT` selects another port;
`ABYSS_E2E_EXTERNAL_SERVER=1` tells Playwright to use an already-running fixture.
An existing binary serves its compiled code/assets until rebuilt and restarted.
Choose an unused port rather than stopping another developer's server.

The [touch checkpoint test](rift-touch-checkpoint.spec.js) banks a receipt,
removes the scenario URL and verifies that the receipt survives reload:

```powershell
node node_modules/@playwright/test/cli.js test tests/e2e/rift-touch-checkpoint.spec.js --reporter=line
```

This verifies persistence within a live fixture process. It does not establish
survival across a server restart or production reward durability.

## Terminal result fixtures

Each URL below initializes a campaign result with 50 gold previously secured.
They use isolated fixture storage and grant no real inventory rewards.

| Page URL | Result |
| --- | --- |
| `/abyss/rift?scenario=terminal-complete` | Final tier of mission 1 completed, victory pose and replay action. |
| `/abyss/rift?scenario=terminal-banked` | Exited at the first checkpoint with secured gold. |
| `/abyss/rift?scenario=terminal-defeated` | First-tier player death with zero health and no unbanked drops. |
| `/abyss/rift?scenario=terminal-expired` | Obsolete-epoch view; 50 gold is historical and spendable/banked gold is zero. |

Completion, exit and defeat use the engine's transitions. Expiry supplies the
read-side projection directly; it does not simulate a database economy reset.
These cover all four terminal wire statuses; `fighting` and `cleared` are active
states. Remove the scenario parameter before testing persistence, as above.
`rift-terminal-fixtures.spec.js` checks the four result screens and reloads;
`TestRiftTerminalFixtureRoutes` checks their server snapshots under `-tags=e2e`.

## Authored hazard fixtures

Open `/abyss/rift?scenario=hazard-<kind>` for `fire`, `ice`, `poison`, `thorns`,
`rune`, `radiant`, `void` or `spikes`. Each scene selects the first enabled example
of that kind from the current campaign, enters its authored room, and keeps only
that hazard. Geometry, period, duration, offset and jumpability are unchanged.
The run starts paused halfway through the active pulse; resume to play the room
with its normal enemies. Other hazards are omitted to isolate the selected type.

The selector reads campaign definitions, so a future authored kind needs no new
fixture registration. Its first-example location can change when content is
reordered; these URLs are inspection fixtures, not version-independent visual
baselines. Strip the scenario parameter before reload to preserve state.
`rift-hazard-fixtures.spec.js` checks all eight browser scenes and frozen phase
persistence. `TestRiftHazardFixtureRoutes` checks authored hazard equality and
coverage under `-tags=e2e`; it prompts a coverage review when the kind count changes.

## Class-resource fixtures

Use `/abyss/rift?scenario=checkpoint&subclass=<style>&charges=<0-3>` to inspect
empty, one-charge, two-charge and full class resources. The `subclass` parameter
accepts either an Abyss foundation class ID or a subclass ID. For example:

- `/abyss/rift?scenario=checkpoint&subclass=warrior&charges=0` shows empty Momentum.
- `/abyss/rift?scenario=checkpoint&subclass=arcanist&charges=2` shows two Spark charges.
- `/abyss/rift?scenario=checkpoint&subclass=vanguard&charges=3` shows full Resolve.

The fixture uses canonical resource labels and signature pairs for all six
foundation classes and twelve subclasses. It is a cleared checkpoint, so it
isolates charge display from ongoing combat. Resume through the checkpoint to
exercise the next room; use resource practice to test earning and spending
charges. Charge count is distinct from mana, barrier, target marks or cooldowns.
Invalid/out-of-range charge parameters leave the constructor's zero value.

`TestRiftResourceFixtureAllStylesAndCharges` checks all 72 style/charge snapshots
and their canonical signatures under `-tags=e2e`. `rift-resource-fixtures.spec.js`
checks the displayed resource label and count for the same combinations.

## Long banked-loot receipt

Open `/abyss/rift?scenario=long-receipt` (optionally with `&subclass=arcanist`) for
a completed 100-mission campaign containing more than 1,000 banked gear records.
It uses the same generator as the large-snapshot codec regressions. Every authored
room contributes one synthetic gear reward per spawned enemy, with mission/tier
origins and gear constrained by that tier's rarity cap. This is a stress fixture,
not the expected drop rate or proof of combat completion.

The run ID and found-at time are fixed; the gear selection follows the current
catalog's longest eligible names. Content edits can therefore change the receipt.
The chosen class remains available, final enemies are defeated, and no real
inventory is touched. The UI groups repeated item names and bounds displayed
origins while keeping the complete saved receipt. Use search and expand a group
to inspect origins. Remove both `scenario` and `subclass` before testing reload.

`TestRiftLongReceiptFixture` verifies campaign coverage, selected build and a
save-codec round trip. `rift-long-receipt-fixture.spec.js` verifies desktop/narrow
layout, grouping, origin details, search and complete receipt persistence.
