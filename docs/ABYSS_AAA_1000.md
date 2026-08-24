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
