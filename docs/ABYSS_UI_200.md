# The Abyss — 200 UI Improvement Suggestions (UI-1 … UI-200)

Fresh UI/UX suggestions, scoped to presentation and interaction (no new game
mechanics). Deliberately excludes: everything in `ABYSS_IDEAS.md` (1–400) and
`ABYSS_UX_IDEAS.md` (UX-1…UX-100, incl. their ✅ implemented batches), and the
~145 quick wins + forge-round-3/4 UI shipped 2026-08-22/23 (aria states, focus
rings, hover/active/disabled states, reduced-motion sweeps, rarity beams for
Celestial/Eternal, dynamic forge price labels, second-item picker, etc.).
Numbers are stable — reference them in issues.

## A. Stage & combat presentation (UI-1 – UI-20)

1. Stage header chip showing the current biome name + icon, not just tinted backdrop.
2. Mob sprite area: idle bob animation (2px translate) so the stage feels alive between actions.
3. First-strike indicator: a ⚡ flash on the side that acts first each round.
4. Damage-type glyph on each hit (🗡️ physical, 🔥 fire…) matching the weapon's etched rune element.
5. Pet HP mini-bar under the stage when a pet is active, updating with the fight.
6. "Wave x/y" pip row above the stage for multi-wave floors.
7. Boss floors: stage frame border shifts to the boss's element color.
8. Downed state: stage desaturates to grayscale until revive/concede is chosen.
9. Rest floors: gentle ember particle drift instead of the combat frame.
10. Event floors: a large centered event icon (❔/📚/🎰) so the floor type reads at a glance.
11. Kill counter chip on the stage ("3 slain this floor") resetting per floor.
12. Overkill flash: when a hit exceeds remaining mob HP ×2, a "OVERKILL" stamp animates across the stage.
13. Last-hit slow zoom on the killing blow line (reduced-motion safe).
14. Durability warning: cracked-shield overlay on the weapon card when durability < 20%.
15. Threat meter needle animates to the new value on descend instead of jumping.
16. Depth dial: floor number counts up/down with a roll animation between floors.
17. Milestone depth (10/20/30…): the dial ring briefly glows gold.
18. Active consumable buffs as small icons along the stage's bottom edge with fights-left badges.
19. Momentum flame icon grows through 3 visual sizes at ×5/×10 stacks.
20. Boss enrage: stage pulses a slow red breathing glow once enraged.

## B. Combat log (UI-21 – UI-40)

21. Per-line hover: highlight the matching combatant's HP bar.
22. "New floor" divider rows with the floor number, so the log reads as chapters.
23. Crit lines get a heavier font weight, not just a color.
24. Dodge lines rendered in italic to distinguish avoidance from mitigation.
25. Lifesteal numbers in green with a + prefix, separate from damage red.
26. Round numbers on the left gutter, toggleable.
27. Long fights: auto-collapse rounds 2..n−1 behind "… 14 rounds …" when a fight exceeds 20 lines.
28. Loot lines inside the log get the rarity left-border (same classes as the manifest).
29. Deaths: the fatal line is pinned (sticky) until the revive choice is made.
30. Pet capture success lines get a 🎯 target animation.
31. Log line for durability loss gets a 🔧 prefix so repairs stop being a surprise.
32. Momentum gain/loss lines get a 🔥 prefix and the current multiplier.
33. Pity counter increment lines get a ✨ prefix ("pity 31/40").
34. Affix-of-the-day reminder line at the start of each floor's log section.
35. Pact effects echoed once at run start ("Bloodlust: +40%/+25% danger") as a log header.
36. Log search box filtering visible lines by substring.
37. "Copy last fight" button copying only the most recent floor's lines.
38. Font ligature-safe mode for the log (some monospace fonts render 🜲 oddly).
39. Screen-reader summary line after each fight: one sr-only sentence with outcome, HP, loot.
40. Log line grouping: consecutive identical DoT ticks collapse to "×3" suffix.

## C. HUD & run awareness (UI-41 – UI-60)

41. Escrow counter: small "per floor" rate estimate (+X g/floor this run).
42. Tokens-today chip: tokens earned this session next to the balance.
43. Insurance countdown: floors of coverage left shown on the shield badge.
44. Checkpoint chip: next checkpoint depth and floors-to-go ("checkpoint in 3").
45. Bank-lock chip also appears on the mini-HUD, not only near the Bank button.
46. HP bar segment ticks at 25/50/75% for quick glance reading.
47. Threat meter: colored bands (green/yellow/red) behind the needle.
48. Depth dial: arc segments for floors since last rest floor.
49. "Distance to leaderboard" chip when within 5 floors of the #10 board spot.
50. Daily bounty progress ring on the HUD while a run is active.
51. Current focus (gold/XP/materials/tokens) shown as a HUD chip with quick-swap dropdown.
52. Active pacts get individual danger/reward tooltip breakdowns on hover.
53. Consumable cooldowns shown as radial sweep on the belt icons.
54. Durability total: "gear condition 82%" aggregate chip, warning color under 40%.
55. Last Stand availability: token cost preview on the HUD when downed is possible.
56. Comeback buff active indicator (after 3 daily deaths) with remaining run count.
57. Jackpot ticker: current deep-cache pot in the HUD, pulsing when it grows.
58. Streak flame: bank-streak counter with the multiplier it currently grants.
59. Time-of-day happy-hour icon on the HUD during forge happy hour.
60. Escrow interest rate chip (+0.5%/floor) with current accumulated %.

## D. Loot & inventory presentation (UI-61 – UI-80)

61. Loot manifest rows: slot icon before the item name.
62. Unidentified items get a pulsing "?" badge until identified.
63. Run-loot row click: expand inline detail card instead of hover-only.
64. "Equip best" one-click button per slot for run loot with a positive CR delta.
65. Loot manifest: sort toggle (by floor / by rarity / by CR delta).
66. Duplicate indicator: "already owned" tag on drops matching an owned item ID.
67. Set-piece drops get a 🧩 tag with current set count ("predator 2/4").
68. Corrupted drops get an animated blood drip on the 🩸 icon.
69. Inventory: durability bars on each backpack row.
70. Inventory: attuned items show the 🔗 ribbon on the row, not only in the name.
71. Inventory: quality tier shown as small stars (★ Fine … ★★★★★ Masterwork).
72. Inventory: filter chips (weapons / armor / jewelry / consumables).
73. Inventory: total vendor value estimate in the header.
74. Equipped cards: temper level as "+N" pip row instead of text.
75. Equipped cards: gem sockets rendered as filled/empty gem slots graphically.
76. Compare view: side-by-side card when clicking a backpack item vs its equipped slot.
77. Negative stat lines rendered in red with a warning tint on item cards.
78. Set membership shown as a colored ribbon per set (predator red, warden blue, harvester gold).
79. Newly acquired items get a "new" glow until first hover/click.
80. Item card footer: source line (dropped floor N / forged / shop) when known.

## E. Forge & workshop UI (UI-81 – UI-100)

81. Forge picker: item CR shown right-aligned in each option row.
82. Forge picker: optgroup headers show counts ("Backpack (14)").
83. Per-action affordability: buttons grey out when gold/tokens/materials are short, with the missing amount.
84. Temper: fail-stack pity shown as a growing luck meter next to the button.
85. Temper surge: big 50/50 visual (coin-flip card) instead of a text confirm.
86. Reforge: show the current stat block beside the confirm modal so the gamble is informed.
87. Awaken: show the possible Special pool as a row of icons in the confirm modal.
88. Imbue: selected effect's description updates live under the select.
89. Brand: show the set's current tier progress inside the brand row.
90. Special swap: both items rendered as mini-cards with a ⇄ arrow between them.
91. Gear-XP infusion: preview the XP amount and resulting milestone before confirming.
92. Prismatic rune: preview which stat gets the +5% before committing.
93. Rebalance: live preview of the from/to amounts before confirming.
94. Gem transmute: before/after stat diff shown inline.
95. Forge history: filter by action type (dropdown).
96. Forge history: item names colored by rarity.
97. Undo button: countdown text ("available today") vs ("used — resets midnight").
98. Artisan rep: progress bar to the next discount tier.
99. Happy hour: banner on the forge header while active, ticking down.
100. Materials strip: click a material to see its sources (tooltip).

## F. Entry & run setup (UI-101 – UI-115)

101. Tier cards: show each tier's entry fee and unlock requirement in one line under the name.
102. Tier cards: personal best depth per tier on the card.
103. Insanity tier card gets a flickering warning border to set expectations.
104. Pact picker: total reward multiplier recompute animates when toggling pacts.
105. Pact picker: conflict icons (⚠️) on pacts that contradict (e.g. Famine + rest-reliant).
106. Checkpoint dropdown: show the ×0.75 reward penalty inline per option.
107. Express start: show exactly which floors are skipped and what loot is forfeited.
108. Entry screen: yesterday's summary line ("yesterday you banked 42k across 3 runs").
109. Entry screen: current jackpot amount with a "feed it" flavor line.
110. Cursed bank checkbox: explain the +20% and the 3-fight hex in a tooltip.
111. Focus picker: icons + one-line effect per focus, not just names.
112. "Last used setup" chip also remembers focus, not only tier/pacts.
113. Entry button shows the exact gold cost struck through when the daily free entry applies.
114. Entry screen: threat forecast for floor 1 of the selected tier ("first floor: ~8% death risk").
115. Locked tiers show a progress bar toward the unlock depth.

## G. Navigation & layout (UI-116 – UI-130)

116. Section tabs: unread-dot on tabs with new content since last visit (e.g. new lore).
117. Section tab bar collapses to a horizontal scroll strip under 900px.
118. Panel headers get anchor links (hover #) for direct linking.
119. "Back to top" floating button after 2 viewport heights of scroll.
120. Sidebar order toggle: let users pin Run Loot or Armoury first (persisted).
121. Bestiary: family grouping headers with collapse toggles.
122. Lore fragments: a 10-segment progress ring in the panel header.
123. Leaderboard: your row highlighted and auto-scrolled into view.
124. Leaderboard: sticky header row on scroll.
125. History table: expand a run row to see its loot summary.
126. Achievements: locked ones show their unlock condition on hover.
127. Badge picker: preview the badge inline next to your nickname.
128. Forge/shop/record panels: remember scroll position per section tab.
129. Print stylesheet: clean run-summary and record printing.
130. Panel loading: shimmer placeholders shaped like the final content.

## H. Feedback, toasts & modals (UI-131 – UI-150)

131. Toast icons per category (✅ success, ⚠️ warning, ⛔ error, 💎 loot).
132. Toast timestamps on hover for "when did that happen".
133. Error toasts offer "Copy details" for bug reports.
134. Confirm modals: the primary action button shows the cost in its label ("Bank 41.5k").
135. Destructive modals: 3-second countdown before the confirm button enables (concede/prestige).
136. Modal stack: ESC closes only the top modal when two overlap.
137. Modal focus: initial focus lands on the safest button (cancel), not the destructive one.
138. Vault summary: skip button for the count-up animation.
139. Level-up flash on the HUD level chip when XP crosses a level mid-run.
140. Achievement unlock: banner slides from the top with the badge icon.
141. New record: the depth dial emits a one-time ring burst.
142. Pity proc: screen-wide subtle golden flash on the guaranteed legendary.
143. Celestial/Eternal drops: distinct toast sound-free visual (starburst rays behind the toast).
144. Downed: heartbeat pulse on the revive options.
145. Network error banner: counts blocked or uncertain actions and requires state review before manual retry; mutations are never queued or replayed.
146. Button press: short haptic-style scale bounce (0.96) on primary actions.
147. Bank success: gold coins arc from the depth dial to the gold counter.
148. Death: the escrow number visually crumbles/drains into the jackpot ticker.
149. Idle run reminder: browser tab title flashes gently when a run waits >5 min.
150. First visit after an update: one-time "what's new" dot on changed panels.

## I. Accessibility & mobile (UI-151 – UI-170)

151. Focus trap audit for the vault overlay and boss card (they're not real modals yet).
152. Skip-link ("skip to run controls") as the first tab stop.
153. aria-live region announcements throttled to one per fight (floods screen readers now).
154. Rarity chips get text labels for screen readers, not only colors/dots.
155. Depth dial: full aria-description ("depth 12, boss in 3 floors, threat 41%").
156. Colorblind mode: rarity shapes (◆●▲) next to names everywhere, toggled in settings.
157. Reduced-motion: also disable the depth-dial roll and counter animations.
158. High-contrast mode: bump muted text from #8a93a8 to #b8c0d0.
159. Touch: swipe left/right on the stage to descend/bank (with confirm).
160. Touch: long-press an item row for the detail card (no hover on mobile).
161. Mobile: forge accordions default collapsed under 700px.
162. Mobile: the tier picker becomes a bottom-sheet select.
163. Mobile: consumable belt becomes horizontally scrollable with snap.
164. Tap targets: loot filter chips to 36px minimum on touch.
165. Orientation: landscape phone layout puts log beside the stage.
166. Text scaling: verify 200% browser zoom doesn't clip the stage controls.
167. Keyboard: arrow keys move between loot rows, Enter equips, Delete salvages.
168. Keyboard: focus returns to Descend after any modal closes mid-run.
169. prefers-color-scheme: respect light-mode users with a readable light theme.
170. Screen-reader cheat sheet: a sr-only description of the page layout at the top.

## J. Stats, records & progression panels (UI-171 – UI-185)

171. Personal record panel: sparkline of depth over the last 30 runs.
172. Death-causes breakdown (boss/normal/trap/downed-timeout) as a mini bar chart.
173. Gold-per-hour meter for the current session.
174. Tier win-rate chips on the record panel (Normal 82% · Nightmare 61% …).
175. Bestiary completion % with a progress bar per family.
176. Forge ROI line: "materials spent vs CR gained this week".
177. Talent tree page: total points spent/available in the header.
178. Prestige panel: preview next prestige's permanent bonus before confirming.
179. History: CSV/JSON export button per filter.
180. Leaderboard: delta arrows since yesterday (▲2) per row.
181. Compare-self: overlay your last run's pace ghost on the current depth rail.
182. Bounty history: last 7 days claimed/missed icons.
183. Drop-streak widget: floors since legendary with the 40-cap marked.
184. Material income chart: dust/shard/core/prism per day for a week.
185. "Milestone map": a static depth chart showing where bosses/checkpoints/jackpot depth sit.

## K. Theming, polish & micro-copy (UI-186 – UI-200)

186. Depth-graded vignette: corners darken every 10 floors (toggleable).
187. Biome ambiences: faint animated fog/embers/void stars per biome (CSS only).
188. Insanity tier: UI chrome gets a subtle glitch jitter on its accents.
189. Seasonal palette variables documented in one CSS block for easy themes.
190. Favicon: swap to a torch icon while a run is active, grey when idle.
191. Empty history: illustrated delver with "your legend starts at floor 1".
192. Empty bestiary: "the abyss keeps its secrets — descend" placeholder art.
193. Error copy pass: replace every remaining "db" toast with human text.
194. Consistent currency icons: audit every place gold appears without 🪙 or tokens without 🜲.
195. Unify pluralization helper for all toasts ("1 items" class of bugs, systematically).
196. Button label casing audit (Title Case vs sentence case — pick one).
197. Emoji fallback font stack for platforms missing newer glyphs (🜲, 🌫️).
198. Print/PDF-friendly run recap card layout.
199. Loading skeleton for the leaderboards panel (currently pops in).
200. A single `abyss-settings` modal collecting every toggle added above, searchable.
