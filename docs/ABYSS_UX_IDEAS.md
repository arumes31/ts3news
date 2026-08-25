# The Abyss — 100 UX Improvement Ideas

UI/UX-focused suggestions (presentation, feedback, clarity, navigation, accessibility).
Referenced as **UX-1 … UX-100** to avoid clashing with the gameplay list in ABYSS_IDEAS.md.

## ✅ Implemented (2026-07-04)

**UX-16** entry cost confirmation modal · **UX-20** entry gate sweep · **UX-28** boss intro
name plate · **UX-33** escrow delta floats · **UX-34** floor-trail mini-map with next-boss
marker · **UX-36** momentum chip shows live STR bonus · **UX-45** rarity toasts for epic+
drops · **UX-46** loot rarity/upgrade filter chips (persisted) · **UX-47** hover compare
cards on loot rows · **UX-48** rarity dot counters in the loot header · **UX-49** bank
preview modal (server-computed breakdown incl. daily-cap warning) · **UX-52** screen-edge
glow on legendary+ drops · **UX-53** 10-second undo on auto-equips (`/api/abyss/unequip`) ·
**UX-54** vault subtotal count-up (base → multiplier → jackpot) · **UX-55** forge grouped
into accordions (Improve / Gems & Runes / Recovery / Maintenance / Fusion / History) ·
**UX-60** happy-hour countdown chip · **UX-89** per-button spinners · **UX-91** stacked
toast queue (errors never evicted by successes) · **UX-93** connectivity banner with
auto-probe. All reduced-motion safe.

## A. First-run onboarding & clarity (UX-1 – UX-10)

- **UX-1** First-visit guided tour: 4–5 dismissible tooltip steps (Enter → Descend → Escrow → Bank → Death rules) anchored to the real buttons.
- **UX-2** "How the Abyss works" collapsible primer card for accounts with 0 runs, replacing the empty history panel.
- **UX-3** Glossary tooltips on jargon terms (escrow, pity, tier, pact, momentum, temper) — dotted underline + hover/tap popover.
- **UX-4** Show a "recommended tier" hint on the tier picker based on player CR vs tier difficulty.
- **UX-5** Explain the risk % tag on the Descend button with a tooltip breaking down the estimate (depth, tier, CR).
- **UX-6** Inline "what happens if I die?" summary next to the insurance row: concrete numbers using the current escrow.
- **UX-7** Label the daily free entry on the Enter button ("Enter — FREE today") instead of only after the fact.
- **UX-8** First-death explainer modal: one-time card explaining revive vs Last Stand vs concede with their exact costs.
- **UX-9** Empty-state art + one-line hint for each empty panel (no lore, no bestiary, no history) instead of blank tables.
- **UX-10** "New!" badges on panels added by recent updates, cleared once opened (localStorage).

## B. Entry screen & run setup (UX-11 – UX-20)

- **UX-11** Combine tier / pact / start-mode pickers into one "Prepare your descent" stepper with a live summary line ("Nightmare · Checkpoint 20 · 2 pacts · cost 500g + 🜲10").
- **UX-12** Show projected first-floor risk for the selected tier before entering.
- **UX-13** Disable + explain locked tiers ("Reach depth 15 to unlock") directly on the button rather than erroring after click.
- **UX-14** Pact picker: show net effect preview ("+40% rewards / +25% danger") when combining multiple pacts.
- **UX-15** Remember last run's setup and preselect it ("Same as last run" one-click chip).
- **UX-16** Confirmation summary modal before paid entries over a threshold (e.g. hell tier + express) listing all charges.
- **UX-17** Show current daily affix on the entry screen, not only inside the run.
- **UX-18** Consumable loadout picker at entry: mark which buffs will carry in, with a capacity meter.
- **UX-19** Grey out the checkpoint select options beyond best depth with lock icons instead of hiding them.
- **UX-20** Animated door/gate transition on entry instead of an instant page state swap (respect reduced-motion).

## C. Combat log & fight feedback (UX-21 – UX-32)

- **UX-21** Log filter chips: All / Damage / Loot / Events / Summary — persisted per user.
- **UX-22** Collapse multi-hit exchanges into one expandable line ("⚔️ 4 exchanges — click to expand").
- **UX-23** Color-code damage numbers by source (you = blue, mob = red, DoT = purple) consistently and document it in a small legend.
- **UX-24** Floating damage numbers over the stage HP bars during log playback.
- **UX-25** Log playback speed control (1×/2×/4×/skip) next to the log; remember choice. (Swiftness node keeps its baseline effect.)
- **UX-26** "Skip to result" button while a long fight log is animating.
- **UX-27** Sticky one-line fight status above the log while animating ("Floor 23 · wave 2/3 · HP 64%").
- **UX-28** Boss intro cards: add a 1-second portrait zoom + name plate; keep text version for reduced motion.
- **UX-29** Critical events (crit kill, pet capture, phoenix proc, legendary drop) get inline icon flashes in the log line.
- **UX-30** After-fight diff panel: HP lost, gold gained, loot count, durability lost — as compact chips instead of buried log lines.
- **UX-31** Copy-fight-log button (plain text export) for sharing/bug reports.
- **UX-32** Auto-scroll toggle for the log with a "⬇ jump to latest" pill when scrolled up.

## D. HUD, status & run awareness (UX-33 – UX-44)

- **UX-33** Show escrow delta animations (+1,250) floating off the escrow counter when it changes.
- **UX-34** Depth progress rail: vertical mini-map of the last 10 floors with icons (⚔️🕊️❔👑) and the next boss marked.
- **UX-35** Persistent "distance to record" chip ("best 42 — 3 floors to go") that turns gold when passed.
- **UX-36** Momentum chip: show the actual bonus ("🔥 ×6 = +12% STR") not just the counter.
- **UX-37** Bank-lock state: countdown chip near the Bank button ("🔒 2 floors") in addition to the disabled button text.
- **UX-38** Insurance status badge on the escrow display (shield icon + %) instead of only in the sub-row.
- **UX-39** Consumable belt: icon row with counts above the Descend button; click to use; active buffs get a glowing ring + fights-left number.
- **UX-40** HP bar: add a ghost segment showing projected HP after estimated next-floor damage.
- **UX-41** Pity meter: show "N kills until guaranteed Legendary" as text on hover, not only a fill bar.
- **UX-42** Mini-HUD: add tiny Bank/Descend buttons so deep-scrolled users can act without scrolling back.
- **UX-43** Session timer + floors-this-session in the mini-HUD for self-pacing.
- **UX-44** Warning tint on the whole stage frame when HP < 25% (subtle red vignette, reduced-motion safe).

## E. Loot & rewards presentation (UX-45 – UX-54)

- **UX-45** Loot toast stack: rare+ drops get their own toast with rarity-colored border and item icon, not just a log line.
- **UX-46** Run Loot sidebar: rarity filter chips + "only upgrades" toggle using the existing ▲ CR delta data.
- **UX-47** Item hover cards: full stat block + comparison vs equipped on hover/tap of any loot label.
- **UX-48** Loot summary bar: counts by rarity as colored dots (● 3 ● 1 ● 1) in the sidebar header.
- **UX-49** Bank preview modal: itemized list of everything about to be claimed (gold, interest, multipliers, loot) before confirming the bank.
- **UX-50** Escrowed-vs-safe visual metaphor: locked cache items slightly desaturated with a small padlock; unlock animation on bank.
- **UX-51** "Best drop this run" pinned at the top of the loot sidebar.
- **UX-52** Legendary+ drops trigger a short screen-edge glow in the item's rarity color (skip on reduced motion).
- **UX-53** Auto-equip notices ("⬆️ Equipped") get an undo link for 10 seconds.
- **UX-54** Vault animation: show counted subtotals (base / interest / streak / jackpot) ticking in sequence rather than one number.

## F. Forge & workshop usability (UX-55 – UX-66)

- **UX-55** Replace the flat forge button list with grouped accordions: Improve / Gems / Runes / Recovery / Fusion.
- **UX-56** Item-first flow: pick the item once (large card with stats), then all applicable actions light up with per-action costs.
- **UX-57** Success-chance bar on Temper (visual % with pity bonus highlighted) instead of only title-attribute text.
- **UX-58** Before/after stat preview on every forge action (temper, gem upgrade, recalibrate ranges, cleanse) in a two-column diff.
- **UX-59** Cost preview refresh: show discounted price (rep + happy hour) with the base price struck through.
- **UX-60** Happy-hour countdown chip on the forge header ("⚒️ 20% off — 38 min left").
- **UX-61** Batch dismantle: confirmation modal listing exact items to be destroyed with rarity colors, not just counts.
- **UX-62** Materials wallet: persistent 4-icon strip (dust/shard/core/prism) in the workshop header with per-hour tooltips.
- **UX-63** Recipe cards: show craftable-count ("can make ×3") and grey out unaffordable ones with the missing material highlighted.
- **UX-64** Secret recipes: show silhouette cards ("??? — found via lore") so players know how many remain.
- **UX-65** Forge history: relative timestamps ("2 h ago") and an icon per action type; undo button inline on the newest entry.
- **UX-66** Fusion picker: dedicated modal with slot-filtered item grid and live "result preview" card, replacing raw inv-ID selection.

## G. Navigation, layout & information architecture (UX-67 – UX-76)

- **UX-67** Sticky section tab bar (Progression / Lore / Leaderboards) that pins below the topbar on scroll.
- **UX-68** Remember the active section tab per user (localStorage) across visits.
- **UX-69** Deep links for every section (#forge, #workshop, #talents) so chat/tooltips can link directly.
- **UX-70** Collapse long panels by default with counts in the header ("Bestiary (34)"); expand state persisted.
- **UX-71** Search box for the bestiary and lore panels.
- **UX-72** Reorder: put Run Loot sidebar above Armoury while a run is active; swap back when idle.
- **UX-73** Compact mode toggle: tighter paddings and smaller cards for veterans on laptops.
- **UX-74** Keyboard shortcuts: D = descend, B = bank, R = revive, L = focus log; "?" opens a cheat-sheet overlay.
- **UX-75** Breadcrumb state in the page title ("Abyss · F23 · 12.4k escrow") so browser tabs are informative.
- **UX-76** Move rarely-used actions (prestige reset, badge picker) into a "⋯" overflow menu to reduce button noise.

## H. Mobile & accessibility (UX-77 – UX-86)

- **UX-77** Bottom action bar on mobile: Descend / Bank / Items pinned to the viewport bottom during a run.
- **UX-78** Swipe between section tabs on touch devices.
- **UX-79** Larger tap targets (min 44px) for the xs ghost buttons in forge/workshop on touch.
- **UX-80** aria-live="polite" region for banners and toasts so screen readers announce floor results.
- **UX-81** Full keyboard focus order audit: modals trap focus, Escape closes, focus returns to the trigger.
- **UX-82** High-contrast theme toggle (thicker borders, no translucent panels) persisted per user.
- **UX-83** Don't rely on color alone for rarity: add a small tier glyph (◆ count) next to rarity-colored names.
- **UX-84** Respect prefers-reduced-motion for the coin pile, dice spin, and count-up numbers (already partial — complete the sweep).
- **UX-85** Font-size preference (S/M/L) applied to the combat log specifically.
- **UX-86** Haptic feedback (navigator.vibrate) on mobile for downed/legendary events, behind a setting.

## I. Feedback, errors & confirmations (UX-87 – UX-94)

- **UX-87** Replace generic "db" error toasts with human messages + retry button where safe ("Couldn't save — nothing was charged. Retry?").
- **UX-88** Optimistic UI with rollback for toggles and cheap actions; spinner overlay only for >400 ms calls.
- **UX-89** Button-level loading states (spinner inside the clicked button) instead of the global busy flag disabling everything silently.
- **UX-90** Danger confirmations use typed context: concede modal shows exact escrow + insurance refund you're giving up.
- **UX-91** Toast queue: stack up to 3 with slide, never overwrite an unread error with a success message.
- **UX-92** Idle-run reminder: if a run is active and the tab regains focus after >10 min, show a "resume your run at floor N" banner.
- **UX-93** Network-loss detection: banner with auto-retry when abPost hits a network error, instead of a one-off toast.
- **UX-94** Session-expiry handling: catch 401s and show a re-login modal preserving page state, instead of failing actions silently.

## J. Delight, stats & long-term polish (UX-95 – UX-100)

- **UX-95** Run recap share card: rendered PNG/canvas summary (depth, gold, best drop, deaths) with a copy button.
- **UX-96** Personal stats dashboard panel: sparklines for gold/day, average depth, death causes breakdown.
- **UX-97** Ambient depth theming: page background hue shifts subtly with depth band (shallow blue → deep purple → abyssal red).
- **UX-98** Milestone confetti (first boss, new record, prestige) — one short burst, reduced-motion safe.
- **UX-99** Sound design behind a mute-by-default toggle: soft ticks for descend, chime for legendary, low drone when downed.
- **UX-100** Seasonal cosmetic themes for the stage frame (event-driven CSS variables only, no gameplay impact).
