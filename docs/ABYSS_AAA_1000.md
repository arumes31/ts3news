# Abyss AAA 1000 delivery program

This is the canonical index for the 1,000-item Abyss overhaul. It does not
duplicate or renumber the existing suggestions; it gives their four established
ledgers one collision-free global ID space.

| Global IDs | Source ledger | Source IDs | Count |
|---|---|---:|---:|
| AAA-0001–AAA-0400 | `ABYSS_IDEAS.md` | 1–400 | 400 |
| AAA-0401–AAA-0500 | `ABYSS_UX_IDEAS.md` | UX-1–UX-100 | 100 |
| AAA-0501–AAA-0700 | `ABYSS_UI_200.md` | UI-1–UI-200 | 200 |
| AAA-0701–AAA-1000 | `ABYSS_IMPROVEMENTS_300.md` | AB-1–AB-300 | 300 |

The source ledgers are the suggestion text. A global item is implemented only
when an evidence entry names production code and a direct test or operational
check. A comment, broad commit, or existence of a similarly named feature is
not sufficient proof.

## Delivery standards

Every shipped tranche must preserve server authority, work without animation,
remain keyboard accessible, avoid unbounded browser/server state, expose useful
failure diagnostics without leaking player data, and pass the complete Go test
and vet gates. Performance claims require an enforceable bound or measurement.

## Delivered tranche: foundational run rules

- [x] **AAA-0001 / 1** — one transactionally claimed free paid-tier entry per database day.
- [x] **AAA-0002 / 2** — reached ten-floor checkpoints with token pricing and 75% floor rewards.
- [x] **AAA-0003 / 3** — best-depth-minus-five express starts with paid, reward-free catch-up floors.
- [x] **AAA-0004 / 4** — 0.5% base cache interest compounded before each new floor reward.
- [x] **AAA-0007 / 7** — consumable-free momentum grants 2% strength per floor, capped at 20%.
- [x] **AAA-0010 / 10** — opt-in weekly expeditions retain one UTC ISO-week seed for the full run.

## Active tranche: operational quality and client performance

- [x] **AAA-0986 / AB-286** — compare tier catalog values with the latest database constraints.
- [x] **AAA-0987 / AB-287** — golden-render threshold and active-run page fixtures.
- [x] **AAA-0990 / AB-290** — aggregate enter → floor 5 → bank funnel telemetry.
- [ ] **AAA-0991 / AB-291** — inspect a player's live run for support (requires explicit approval for operator-visible player data).
- [x] **AAA-0993 / AB-293** — bounded client error reporting.
- [x] **AAA-0994 / AB-294** — combat-log DOM virtualization above 500 lines.
- [x] **AAA-0995 / AB-295** — coalesce HUD chip recomputes to one animation frame.
- [x] **AAA-0996 / AB-296** — compute rarity metadata once and reuse it.
- [ ] **AAA-0997 / AB-297** — pause authoritative run timers during connectivity loss.
- [x] **AAA-0998 / AB-298** — expose per-locale Abyss i18n coverage to operators.
- [x] **AAA-0999 / AB-299** — validate unique gear IDs, set size, and effect descriptions.
- [x] **AAA-1000 / AB-300** — render an in-portal changelog from release-note data.

AAA-0997 intentionally remains pending while reconnect/session-cursor work is
already modified in the worktree. It requires a server-authoritative deadline
lease, not a cosmetic client countdown pause.

## Delivered tranche: first-run clarity and descent preparation

- [x] **AAA-0401 / UX-1** — anchored, dismissible five-step field-manual tour.
- [x] **AAA-0402 / UX-2** — zero-run primer that retires after experience or dismissal.
- [x] **AAA-0403 / UX-3** — keyboard-focusable glossary terms using the shared safe tooltip path.
- [x] **AAA-0404 / UX-4** — CR-aware recommended tier without changing the player's selection.
- [x] **AAA-0405 / UX-5** — floor-one risk breakdown with tier and equipped-CR inputs.
- [x] **AAA-0406 / UX-6** — live protected-versus-exposed cache values at defeat.
- [x] **AAA-0407 / UX-7** — server-authoritative daily-free-entry label before descent.
- [x] **AAA-0408 / UX-8** — one-time first-defeat decision explainer with current costs.
- [x] **AAA-0409 / UX-9** — illustrated, instructive history and bestiary empty states.
- [x] **AAA-0410 / UX-10** — locally dismissible new-content markers on updated panels.
- [x] **AAA-0411 / UX-11** — progressive tier, route, build, and pact preparation workflow.
- [x] **AAA-0412 / UX-12** — selected-tier floor-one risk projection.
- [x] **AAA-0413 / UX-13** — disabled tiers show their exact depth requirement.
- [x] **AAA-0414 / UX-14** — combined pact reward and danger preview.
- [x] **AAA-0415 / UX-15** — restore the last valid descent setup from local browser state.
- [x] **AAA-0416 / UX-16** — itemized confirmation for materially expensive entry routes.
- [x] **AAA-0417 / UX-17** — current rotating affix presented before entry.
- [x] **AAA-0418 / UX-18** — ownership-validated consumable carry loadout and capacity meter.
- [x] **AAA-0419 / UX-19** — show the next locked checkpoint and its unlock requirement.
- [x] **AAA-0420 / UX-20** — reduced-motion-safe gate transition into a descent.

## Delivered tranche: combat recorder and fight readability

- [x] **AAA-0421 / UX-21** — persisted All, Damage, Loot, Events, and Summary log filters.
- [x] **AAA-0422 / UX-22** — keyboard-expandable exchange compaction for long fights.
- [x] **AAA-0423 / UX-23** — source-coded log borders with a non-color-only legend.
- [x] **AAA-0424 / UX-24** — timeline-driven overhead damage and healing values.
- [x] **AAA-0425 / UX-25** — persisted 1×, 2×, and 4× playback speeds.
- [x] **AAA-0426 / UX-26** — bounded skip-to-result drain for long resolved fights.
- [x] **AAA-0427 / UX-27** — sticky floor, exchange, wave, and HP recorder status.
- [x] **AAA-0428 / UX-28** — reduced-motion-safe boss introduction card and name plate.
- [x] **AAA-0429 / UX-29** — inline flashes for critical, capture, phoenix, and high-rarity events.
- [x] **AAA-0430 / UX-30** — post-fight HP, gold, loot, and gear-wear delta chips.
- [x] **AAA-0431 / UX-31** — plain-text clipboard export of the latest complete fight.
- [x] **AAA-0432 / UX-32** — explicit follow ownership and jump-to-latest recovery.

## Delivered tranche: run-awareness HUD

- [x] **AAA-0433 / UX-33** — reduced-motion-safe cache delta values at the escrow counter.
- [x] **AAA-0434 / UX-34** — persisted ten-floor trail with encounter icons and next-boss marker.
- [x] **AAA-0435 / UX-35** — sticky distance-to-record readout with gold new-record state.
- [x] **AAA-0436 / UX-36** — momentum displays its current strength bonus.
- [x] **AAA-0437 / UX-37** — bank-lock countdown mirrored beside the sticky actions.
- [x] **AAA-0438 / UX-38** — visible insured-share badge on the cache.
- [x] **AAA-0439 / UX-39** — compact, labelled quick-use consumable belt plus active-buff strip.
- [x] **AAA-0440 / UX-40** — threat-derived projected-health ghost segment with explicit forecast text.
- [x] **AAA-0441 / UX-41** — exact drops-left text beside the Legendary pity meter.
- [x] **AAA-0442 / UX-42** — state-aware Bank and Descend controls in the sticky mini-HUD.
- [x] **AAA-0443 / UX-43** — reload-stable run timer and floors-cleared counter.
- [x] **AAA-0444 / UX-44** — opaque low-health stage warning at under 25% HP.

## Delivered tranche: loot custody and reward presentation

- [x] **AAA-0445 / UX-45** — rarity-framed toast queue for Epic-or-better drops.
- [x] **AAA-0446 / UX-46** — persisted rarity and upgrade-only manifest filters.
- [x] **AAA-0447 / UX-47** — comparison cards on hover, keyboard focus, and touch focus with Escape dismissal.
- [x] **AAA-0448 / UX-48** — live rarity counts in the run-loot header.
- [x] **AAA-0449 / UX-49** — server-computed, read-only itemized bank preview before commit.
- [x] **AAA-0450 / UX-50** — visible escrow seal and bank-release transition for unsecured loot.
- [x] **AAA-0451 / UX-51** — read-only best-drop pin ranked by rarity, upgrade, CR, and gear score.
- [x] **AAA-0452 / UX-52** — reduced-motion-safe edge treatment for Legendary-or-better drops.
- [x] **AAA-0453 / UX-53** — item-bound ten-second undo for auto-equipped loot.
- [x] **AAA-0454 / UX-54** — sequential cache, multiplier, jackpot, and tax bank summary.

## Delivered tranche: forge and workshop usability

- [x] **AAA-0455 / UX-55** — grouped Improve, Gems & Runes, Recovery, Fusion, and History workstations.
- [x] **AAA-0456 / UX-56** — item-first workpiece card with applicable-action count and live costs.
- [x] **AAA-0457 / UX-57** — visual Temper success meter with a separately colored pity contribution.
- [x] **AAA-0458 / UX-58** — current, expected, and maximum two-column/stat-card forge outcomes.
- [x] **AAA-0459 / UX-59** — struck-through base gold price beside the current account-discounted price.
- [x] **AAA-0460 / UX-60** — live happy-hour status and countdown in the forge header.
- [x] **AAA-0461 / UX-61** — bounded exact-item dismantle manifest committed by reviewed inventory IDs.
- [x] **AAA-0462 / UX-62** — sticky four-material wallet with source and flow context.
- [x] **AAA-0463 / UX-63** — craftable counts, disabled unaffordable recipes, and explicit missing materials.
- [x] **AAA-0464 / UX-64** — locked recipe silhouettes with discovery guidance and completion count.
- [x] **AAA-0465 / UX-65** — relative, icon-coded forge history with newest-action inline undo.
- [x] **AAA-0466 / UX-66** — same-slot manual fusion picker with exact selection and live result odds.

## Delivered tranche: navigation and information architecture

- [x] **AAA-0467 / UX-67** — sticky section tabs below the global navigation.
- [x] **AAA-0468 / UX-68** — active section persisted across browser visits.
- [x] **AAA-0469 / UX-69** — stable section aliases and panel-level deep links.
- [x] **AAA-0470 / UX-70** — counted, persisted collapse controls for long library and history panels.
- [x] **AAA-0471 / UX-71** — one keyboard-friendly search across lore and bestiary entries.
- [x] **AAA-0472 / UX-72** — active runs automatically prioritize Run Loot ahead of Armoury.
- [x] **AAA-0473 / UX-73** — persisted compact veteran layout setting.
- [x] **AAA-0474 / UX-74** — guarded D/B/R/L shortcuts plus an in-page shortcut guide.
- [x] **AAA-0475 / UX-75** — live depth and escrow breadcrumbs in the browser title.
- [x] **AAA-0476 / UX-76** — prestige and badge controls moved into a rare-actions overflow.

## Delivered tranche: mobile and accessibility

- [x] **AAA-0477 / UX-77** — viewport-pinned Descend, Bank, and usable-item actions during mobile runs.
- [x] **AAA-0478 / UX-78** — guarded horizontal swipe navigation between section tabs.
- [x] **AAA-0479 / UX-79** — 44px minimum forge and workshop targets on coarse pointers.
- [x] **AAA-0480 / UX-80** — polite, atomic result banner and additive toast announcements.
- [x] **AAA-0481 / UX-81** — focus-trapped shared and consumable dialogs with Escape and trigger restoration.
- [x] **AAA-0482 / UX-82** — persisted high-contrast theme with solid panels and stronger focus indicators.
- [x] **AAA-0483 / UX-83** — visible tier-count glyphs supplement every supported rarity color.
- [x] **AAA-0484 / UX-84** — page-wide reduced-motion coverage including coins, vault, and numeric effects.
- [x] **AAA-0485 / UX-85** — persisted Small, Medium, and Large combat-log text sizes.
- [x] **AAA-0486 / UX-86** — opt-in, coarse-pointer haptics for downed and Legendary-or-better events.

## Delivered tranche: feedback, errors, and recovery

- [x] **AAA-0487 / UX-87** — human error messages plus safe connectivity rechecks without replaying ambiguous mutations.
- [x] **AAA-0488 / UX-88** — optimistic loot-rule and reservation updates with authoritative rollback and stale-response protection.
- [x] **AAA-0489 / UX-89** — clicked-button loading indicators delayed until an action exceeds 400 ms.
- [x] **AAA-0490 / UX-90** — concede confirmation names exact cache, insured refund, and forfeited amount.
- [x] **AAA-0491 / UX-91** — three-toast queue that preserves visible errors ahead of later successes.
- [x] **AAA-0492 / UX-92** — ten-minute background-tab reminder with live floor and downed context.
- [x] **AAA-0493 / UX-93** — persistent offline banner with browser-online and bounded HEAD-probe recovery.
- [x] **AAA-0494 / UX-94** — HTTP 401 detection, re-login guidance, and same-tab workspace restoration.

## Delivered tranche: delight, stats, and long-term polish

- [x] **AAA-0495 / UX-95** — locally rendered 1200×630 run recap with image clipboard and PNG fallback.
- [x] **AAA-0496 / UX-96** — exact recent depth, banked-gold, average, bank-rate, and bounded browser death-cause dashboard.
- [x] **AAA-0497 / UX-97** — four depth-band background treatments driven only by live run depth.
- [x] **AAA-0498 / UX-98** — bounded, reduced-motion-safe celebration particles for records, achievements, and first boss.
- [x] **AAA-0499 / UX-99** — unified opt-in sound setting for descend, rare-loot, downed, and live-combat cues.
- [x] **AAA-0500 / UX-100** — UTC-season cosmetic variables for stage accent and page wash with no gameplay effect.

## Delivered tranche: authoritative stage presentation

- [x] **AAA-0501 / UI-1** — biome name and inferred biome glyph in the stage header.
- [x] **AAA-0502 / UI-2** — two-pixel idle motion on rendered combat sprites with reduced-motion fallback.
- [x] **AAA-0503 / UI-3** — first-strike side flash driven by the first authoritative action line.
- [x] **AAA-0504 / UI-4** — elemental damage glyphs derived from combat effectiveness and equipped rune data.
- [x] **AAA-0505 / UI-5** — pet health mini-bar updated from authoritative combat timeline frames.
- [x] **AAA-0506 / UI-6** — current, completed, and remaining wave pips for multi-wave encounters.
- [x] **AAA-0507 / UI-7** — boss-family element borders retained independently of motion preferences.
- [x] **AAA-0508 / UI-8** — grayscale downed-stage treatment cleared by normal run-state rendering.
- [x] **AAA-0509 / UI-9** — bounded rest-floor ember layer disabled by reduced-motion preferences.
- [x] **AAA-0510 / UI-10** — centered event-type glyph for library, den, gambler, and unknown events.
