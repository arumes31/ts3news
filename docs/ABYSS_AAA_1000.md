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
