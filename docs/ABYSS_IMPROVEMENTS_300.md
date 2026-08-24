# The Abyss — 300 UI & Gameplay Improvement Suggestions (AB-1 … AB-300)

Fresh suggestions, mixed gameplay + UI. Deliberately excludes: `ABYSS_IDEAS.md`
(1–400), `ABYSS_UX_IDEAS.md` (UX-1…100), `ABYSS_UI_200.md` (UI-1…200, 177
implemented 2026-08-23), the ~145 quick wins, forge rounds 3–4 (20 mechanics),
Celestial/Eternal rarities, the Harvester set, the Insanity tier and the
daily-cap tax. Numbers are stable — reference them in issues.
Tags: **[G]** gameplay · **[U]** UI/UX · **[T]** tech/meta · *(heavy)* = multi-day effort.

## A. Core loop & risk/reward (AB-1 – AB-25)

1. **[G]** Greedy grip: every 10 floors without banking adds +2% escrow interest but −2% DEF (visible debuff chip).
2. **[G]** Heavy pockets: cache above 100k slows you −1% SPD per 50k, shown on the HUD with its exact penalty.
3. **[G]** Frantic fee: banking below 15% HP costs 5% of the cache — listed in the bank preview.
4. **[U]** Safe-word confirm for banks above 1M (type "BANK"), optional toggle.
5. **[G]** Echo banking: after a bank, the next run starts with 5% of the banked amount as a head-start cache.
6. **[G]** Anchor rune consumable: the next death forfeits only half the cache (one charge).
7. **[U]** Insurance slider (10–90% in 5% steps) replacing fixed steps.
8. **[G]** The abyss notices: idling >60s on a floor adds +1% danger per minute (HUD warning).
9. **[G]** Death wish: per-floor toggle raising next-floor danger ×3 for ×2 loot, with a skull chip.
10. **[G]** Rest-floor cache shrink: convert 50% of cache to tokens at a 70% rate.
11. **[G]** Second Last Stand charge purchasable at 3× price after the first is used.
12. **[G]** Experience vs killer: each death grants a permanent +0.1% damage vs that mob family (cap +5%).
13. **[G]** Cold muscles: express/checkpoint starts carry a 2-floor −10% stats debuff.
14. **[G]** Combo banking: banking exactly on a checkpoint floor refunds the checkpoint token cost.
15. **[G]** Record push: each floor past your best depth adds +3% cache.
16. **[G]** Bankers' raffle: 1% of every bank feeds a daily draw among everyone who banked that day.
17. **[G]** Downed timeout: 5 min without choosing auto-concedes and returns a 10% pity cache.
18. **[G]** Defensive momentum: clearing a floor untouched grants +2% DEF momentum variant (separate stack).
19. **[G]** Cheapskate flag: insuring <5% of the cache once grants a joke title.
20. **[G]** Perfect run: banking without taking damage on any floor pays +25% and a badge.
21. **[U]** Escrow interest tooltip shows the diminishing-returns curve with your position marked.
22. **[G]** Revive odds improve +5% per consecutive daily death (reset on success), shown on the gamble chip.
23. **[G]** Double bank: after banking, optionally seed a fresh run at floor 0 with 10% of the payout.
24. **[G]** Anti-stall: fights past 30 rounds add stacking fatigue damage to both sides.
25. **[G]** Hybrid runs: opt-in mode where every 5th floor rolls the next tier up's danger at half its reward bonus.

## B. Floors & events (AB-26 – AB-50)

26. **[G]** Event floors expire if you walk 3 floors past them ("the moment passes").
27. **[G]** Event memory: the same event type revisited in one run improves its offer 10% per visit.
28. **[U]** Cursed elevator shows both destination floor previews before you commit.
29. **[U]** Trap floors show the pass chance computed from your DGE vs floor difficulty.
30. **[G]** Rescue NPCs get persistent names; rescuing the same one twice grants a keepsake trinket.
31. **[G]** Gambling den high-roller table (10× stakes) unlocks past depth 40.
32. **[G]** Cursed library: alternative price in SPD instead of max HP.
33. **[G]** Mirror-floor clone uses your *current* gear, with a pre-fight warning panel.
34. **[G]** Abyssal market: one mystery-box slot per visit (random rarity, flat price, posted odds).
35. **[G]** Rift floor: first peek per run is free; second and third peeks cost gold.
36. **[U]** Storm floors telegraph which side the lightning hits next round.
37. **[G]** Darkness floors also hide HP bars, not just the log.
38. **[G]** Sanctuary map-table upgrade: reveals event *types* (not just combat/rest) 2 floors ahead.
39. **[U]** Event-chain progress ribbon ("sigils 2/3") under the depth dial.
40. **[G]** Cartographer's map upgrade: also marks expected loot quality per floor.
41. **[G]** Blood altar: sacrificing a corrupted consumable doubles the buff duration.
42. **[G]** Echo floor taken twice in a row stacks to 75% of the original value.
43. **[G]** Alchemy lab risky brew: 50% stronger buff, 20% backfire — posted odds.
44. **[U]** Unstable portal shows the 3 skipped floors' types afterwards ("what you missed").
45. **[U]** Graveyard floors show the ghost's name and death depth before you commit.
46. **[G]** Bounty board: 3 completed bounties in a run double the 4th's reward.
47. **[G]** Collapsed passage: even the detour grants a small material find.
48. **[U]** Wishing well panel shows your lifetime contributions.
49. **[G]** Abyssal garden: harvesting the same node type 3× grants a permanent +1 material "green thumb".
50. **[G]** Hall of mirrors: picking the same reflection 3 runs running empowers it +25%.

## C. Combat & skills (AB-51 – AB-75)

51. **[G]** Elemental combo: two same-element skills in a row boost the second +10%.
52. **[G]** Mana overflow: casting at full mana overcharges the spell +15%.
53. **[G]** Weapon swap mid-boss (MainHand ↔ backpack backup), once per fight, 1-round penalty.
54. **[G]** Cursed mercy rule: the HP drain pauses below 20% HP, called out in the log.
55. **[G]** Executioner + Execute-affix days stack with a special log flourish and an extra +5%.
56. **[G]** Parry mastery: 3 parries in one fight grant a round of Stealth.
57. **[U]** Thorns/counter totals in the end-of-fight summary.
58. **[G]** Pet focus-fire: click a mob in multi-mob waves to set the pet's target.
59. **[G]** Stunbreak: one free stun cleanse per boss fight at 50% effectiveness.
60. **[U]** DoTs render as a striped segment on the mob HP bar.
61. **[U]** Enrage warning: boss card border flashes 2 rounds before enrage.
62. **[G]** Focus synergy: the auto-selected focus adds a matching micro-bonus (gold focus → +2% crit).
63. **[U]** Skill cooldown pips on the combat HUD.
64. **[G]** "Hold mana" toggle: save casts for boss floors while grinding normals.
65. **[U]** Ultimate charge ring around the stage (estimated rounds-to-ready).
66. **[U]** Counter-attack damage line in the fight summary for parry/dodge builds.
67. **[G]** Elemental resist: 3+ same-element runes grant 10% resist vs that element's mobs.
68. **[G]** Backstab: backline weapons deal +8% to mobs that didn't target you last round.
69. **[G]** Kill-chain: a ≤2-round kill grants +5% speed next floor (stacks ×3).
70. **[G]** Boss interrupt: an ultimate fired during a summon telegraph cancels the summon.
71. **[U]** Post-hoc odds: hovering your death shows the survival chance you actually had.
72. **[G]** Fumble recovery: the hit after a fumble gets +10% crit ("embarrassed rage").
73. **[G]** Pet betrayal foreshadowing: sub-10% loyalty makes its log barks turn nervous.
74. **[U]** Mind-controlled mobs visibly flip side with a color swap and 1 HP.
75. **[G]** Desperation: from round 25 both sides gain stacking +5% damage per round.

## D. Loot & items (AB-76 – AB-100)

76. **[G]** Loot vacuum: rest floors recover missed common drops from the last 3 floors at 50% value.
77. **[U]** Threat-meter tooltip shows the expected drop-rarity distribution for the next floor.
78. **[G]** Foil items: 1% of drops get a cosmetic animated shine (no stat change, pure brag).
79. **[G]** Doomed items: cursed+eldritch co-occurrence (×2 stats, both drawbacks) with a red-black beam.
80. **[U]** Gear sets get lore tooltips ("worn by the first delvers…").
81. **[G]** Lucid variants: 10% of insanity drops lose the negative stat but −20% stats.
82. **[G]** Perfect punch: a 4th punched socket has a 10% chance to grant a 5th.
83. **[G]** Gem resonance: 3 same-type gems across your gear grant +5% to that gem's stat.
84. **[G]** Defensive runes (new): armor-etchable, +5% elemental resist instead of damage element.
85. **[G]** Corrupted consumables: stronger effect plus self-damage, red-flagged in the belt.
86. **[G]** "of the Deep" suffix: drops past depth 40 can roll it, adding STA.
87. **[G]** Duplicate legendaries auto-convert to 5 cores if you already own two (toggleable).
88. **[U]** Loot beam intensity scales with rarity (subtle rare → dramatic eternal).
89. **[U]** Charm-slot items get a tiny dangle animation in the inventory.
90. **[U]** Item compare sorts by your build's main stat first.
91. **[G]** Sentimental value: items kept 30+ days gain +1% stats ("broken in").
92. **[U]** Unidentified items show slot icon and rarity silhouette before identifying.
93. **[G]** Eternal drops push a TS3 channel announcement (extending the Mythic+ fanfare).
94. **[U]** Run-loot manifest floats upgrade rows (▲CR) to the top.
95. **[U]** Pity display splits per tier: legendary counter and celestial counter.
96. **[G]** Treasure goblins drop goblin-token collectibles; 5 unlock a cosmetic title.
97. **[U]** Boss relics get flavor inspect text in the inventory.
98. **[G]** Consumable stacking: identical potions merge into stacks of 5 with count badges.
99. **[G]** Set trading post: swap 2 same-set duplicates for a missing slot (rotating offer).
100. **[U]** Salvage preview hover on any item shows its expected material yield.

## E. Forge & crafting (AB-101 – AB-125)

101. **[G]** Batch temper: pick a target level; auto-stops after 2 fails, with total cost preview.
102. **[G]** Temper insurance stone (new recipe): prevents one downgrade/fail.
103. **[G]** Forge queue: queue up to 3 actions, executed with a single confirm.
104. **[G]** Bulk gem upgrade: all tier-I → II in one action with summed cost.
105. **[G]** Rune scraping: remove a rune, recovering 50% of its cost as dust.
106. **[G]** Un-attune: remove attunement for 50 tokens (item loses the +5%).
107. **[G]** Masterwork transfer: move quality levels to a replacement item at 80% efficiency.
108. **[G]** Perfect corruption: corrupting has a hidden 5% roll for no HP malus, revealed after.
109. **[G]** Reforge lock: pay 2× to exclude one stat line from the reroll.
110. **[G]** Bulk rebalance: shift 10% of all stats toward one chosen stat at once.
111. **[G]** Brand removal: strip a set brand for 2 cores (item returns to the legacy pool).
112. **[G]** Special reroll: reroll an item's Special outright for 6 cores.
113. **[G]** Guided awaken: pay double cores to pick 1 of 3 rolled Specials.
114. **[G]** Imbue removal: strip an imbued effect for 1 prism.
115. **[G]** Polish-all: one click polishes every equipped piece, summed cost shown.
116. **[G]** Repair Kit II recipe: +25 durability, costs 8 dust.
117. **[G]** Forge mastery: every 50 actions grants +1% permanent discount (cap +5%), separate from rep.
118. **[G]** Socket relocation: move a gem between sockets for a small fee (order matters for gem tools).
119. **[U]** Fusion preview shows the survivor's projected stats before consuming.
120. **[G]** Celestial fuse safety: pay 10 prisms to make the ascension roll 50% instead of 25%.
121. **[G]** Eternal privilege: Eternal items may reforge twice per day.
122. **[G]** Crafting crit: 5% chance a craft doubles its output, celebrated in the toast.
123. **[U]** Recipe favorites pinned atop the craft list.
124. **[G]** Material conversion at the exchange: 10 dust → 1 shard, 10 shard → 1 core, 5 core → 1 prism.
125. **[G]** Sanctuary upgrade: a second forge undo per day.

## F. Economy, shop & auction house (AB-126 – AB-150)

126. **[U]** AH "Insanity" filter tab for tier-exclusive gear.
127. **[G]** AH watchlist with a toast when a watched item type is listed.
128. **[U]** AH bulk relist: "relist all mine at −1%".
129. **[G]** AH seller reputation (completed sales) on listings.
130. **[U]** AH listing tooltip explains why attuned items can't be listed.
131. **[G]** Shop "happy accident": one random item discounted 40% daily.
132. **[G]** Token shop: rotating insanity-tier cosmetic tab.
133. **[G]** Token bundles: gold→tokens at a sliding daily rate.
134. **[G]** Consumable subscription: 2 potions auto-delivered daily for a flat fee (opt-in).
135. **[G]** Vendor loyalty: frequently sold item types earn +2% resale.
136. **[G]** Cache-backed loan: borrow up to 50% of escrow mid-run, 10% fee due on bank.
137. **[G]** Jackpot split: winning the pot pays 10% to your last co-op helper.
138. **[G]** Tax rebate: a bank-free day refunds 10% of the week's paid abyss tax.
139. **[G]** Bounty insurance: one missed daily bounty per week doesn't break the streak.
140. **[U]** AH header ticker: cheapest legendary listing price.
141. **[U]** AH sale proceeds arrive with a mail-style summary toast.
142. **[G]** "Most taxed this season" hall-of-fame board with an ironic badge.
143. **[G]** Material buy-orders: post dust/shard/core requests others can fill.
144. **[G]** Shop gift codes: one-time codes to give a shop item to a friend.
145. **[G]** Price alerts: toast when your listed item's type drops 20% in average price.
146. **[G]** Repair subscription: flat daily fee covering all repairs.
147. **[G]** Token scratch card: daily 50-token gamble with posted odds.
148. **[G]** AH anti-snipe: bids in the last 60s extend the auction by 60s.
149. **[U]** AH history tabs: sold / listed / expired with totals.
150. **[U]** Fee transparency: every economy action shows its exact fee breakdown on hover.

## G. Progression & talents (AB-151 – AB-175)

151. **[G]** Three saved talent loadouts with one-click respec into them.
152. **[G]** Keystone loadouts: swap the active keystone without refunding its path.
153. **[U]** Tree search filters nodes by stat keyword ("lifesteal").
154. **[U]** Pathfinder: pick a target node, the tree highlights the cheapest path.
155. **[G]** Jewel crafting: 3 identical jewels fuse into a +1 tier jewel.
156. **[U]** Timeless jewels show their affected node radius visually.
157. **[G]** First respec each week is free.
158. **[U]** Depth-gated nodes glow when you first qualify.
159. **[G]** Mastery shard (boss drop): refunds a single branch.
160. **[G]** Paragon board post-prestige: a small hex grid of +0.1% nodes (visual successor of the flat paragon list).
161. **[G]** Bestiary talents: spend boss-kill counts on family-specific nodes.
162. **[U]** Node tooltips show branch totals ("+15% loot from this branch").
163. **[U]** Tree minimap with a viewport rectangle for the 1100-node web.
164. **[G]** Node queue: shift-click queues allocations, applied as points arrive.
165. **[G]** Achievement nodes: some nodes unlock via achievements instead of points.
166. **[G]** Set-mastery node: wearing a full set unlocks a matching mini-node.
167. **[G]** Node of the day: one node costs half points, rotating daily.
168. **[G]** Prestige memory: each prestige keeps one chosen node allocated for free.
169. **[U]** Shift-hover a node to see the stat delta vs your current build.
170. **[G]** Build codes: export/import a talent build as a shareable string.
171. **[T]** Tree canvas-rendering toggle for low-end devices.
172. **[G]** Free undo-last-allocation within 60 seconds.
173. **[U]** Keystone cooldown timers rendered on the node itself.
174. **[U]** Major nodes get a subtle glow tier by power.
175. **[U]** Beginner path overlay: suggested first 10 nodes for new delvers.

## H. Bosses, pets, co-op & social (AB-176 – AB-200)

176. **[U]** Boss execution cinematic: the killing blow briefly zooms the stage.
177. **[U]** Post-boss DPS estimate in the summary.
178. **[G]** Sanctuary practice dummy: test your build log, no rewards or risk.
179. **[G]** Per-boss player tips: one-line notes on the intro card (local-only curation).
180. **[G]** Pet moods (content/hungry/scared) with ±2% performance and icons.
181. **[U]** Pet collar/charm stats shown on the pet panel.
182. **[G]** Pet training: 1000g raises a pet's lowest stat 1% (daily cap).
183. **[G]** Second active pet slot unlocks at prestige 2.
184. **[U]** Co-op helper ping: helpers get a toast when their assist secures the kill.
185. **[G]** Helper reliability badge for frequent co-op partners.
186. **[G]** Graveyard ghosts prefer players near your level.
187. **[U]** Death wall: your last 10 deaths with killer names (honor log panel).
188. **[G]** Weekly rival: the system picks a player one rank above you; passing them pays tokens.
189. **[U]** Bank feed: banks by players near your depth appear in a side feed (opt-in).
190. **[U]** Revenge marker: the mob family that killed you most gets a target icon.
191. **[U]** Boss kill-cam: replay the final 5 rounds from history.
192. **[G]** Duo bonus: helping the same player 5× unlocks +2% for both.
193. **[U]** Pet graveyard: a memorial list of lost pets 🕯️.
194. **[G]** *(heavy)* Weekly server boss: shared HP pool, per-contribution loot rolls.
195. **[G]** Boss taunts: quips in the log at 50%/25% HP.
196. **[G]** Per-ability pet autoskill toggles.
197. **[G]** Ghost courtesy: beating a player's ghost pays them a 5% consolation fee.
198. **[G]** Insanity whispers: insanity runs occasionally print unsettling flavor lines in the log.
199. **[U]** First-kill trophy cards per boss, dated.
200. **[G]** *(heavy)* Spectator link: read-only live floor feed of a friend's run.

## I. UI — stage, log & HUD (AB-201 – AB-225)

201. **[U]** Damage vignette: screen edge flashes red proportional to big hits taken.
202. **[U]** Mob intent icon above the stage (heavy/normal/special next attack).
203. **[U]** Round-timer chip: average round duration for the current fight.
204. **[U]** Pin any log line to the top of the log.
205. **[U]** Log export as a styled HTML snippet (colors preserved).
206. **[U]** HP forecast ghost bar: projected HP if you descend now.
207. **[U]** Consumable belt drag-to-reorder, persisted.
208. **[U]** HUD edit mode: drag chips to reorder, persisted.
209. **[U]** Threat sparkline over the run in the HUD.
210. **[U]** Floor timer with your session average.
211. **[U]** Boss HP numeric overlay option (exact HP vs %).
212. **[U]** Floor clear-rating flair (S/A/B/C, purely cosmetic).
213. **[U]** Log bookmarks: jump links for boss floors in long runs.
214. **[U]** Multi-wave progress as segment colors on the depth rail.
215. **[U]** Pet status line in the HUD (name + loyalty heart).
216. **[U]** Buffs with 1 fight left blink gently.
217. **[U]** HP-bar tooltip: damage-taken breakdown (physical/elemental/DoT).
218. **[U]** Long-press Descend for ×5 auto-descend with the stop-rule summary.
219. **[U]** Hovering loot greys out non-matching equipped slots.
220. **[U]** Right-click skips the boss intro card.
221. **[U]** Run clock in the HUD (pauses when idle).
222. **[U]** Mana bar segmented by spell cost (casts remaining at a glance).
223. **[U]** Depth-dial tooltip with full run stats.
224. **[U]** Escrow tooltip splits the cache into gold/items/materials.
225. **[U]** Sub-20% HP bar slow pulse (visual heartbeat).

## J. UI — panels, forge & inventory (AB-226 – AB-250)

226. **[U]** Inventory grid (icons) vs list toggle.
227. **[U]** Forge action search box ("gem" highlights matching actions).
228. **[U]** Recently-used forge actions row atop the workshop.
229. **[U]** Item-card keyboard navigation (arrows + Enter).
230. **[U]** Set panel: per-set progress bars with next-tier preview.
231. **[U]** Materials strip warns under 5 of a kind.
232. **[U]** "Next run" consumable loadout tray (drag-and-drop).
233. **[U]** Forge price change notes ("was 100k, now 80k — happy hour").
234. **[U]** Backpack multi-select with a batch action bar (salvage/dismantle/attune).
235. **[U]** Item lock toggle on cards *(needs a backend flag — see ABYSS_IDEAS #81)*.
236. **[U]** Pinnable tooltips for side-by-side comparison.
237. **[U]** Armoury footer with summed equipped stats.
238. **[U]** Shop stock countdown ("new stock in 3h").
239. **[U]** Recipe discovery progress ("7/12 known").
240. **[U]** History filters: victories / deaths / records.
241. **[U]** Achievements sorted nearest-completion first.
242. **[U]** Lore reading-progress checkmarks.
243. **[U]** Bounty card shows claims left this week.
244. **[U]** Talents tab badge with unspent point count.
245. **[U]** Insanity items get a fractured-glass card border.
246. **[U]** Compare cards show per-stat percentages ("+13% STR").
247. **[U]** Green affordability dot on every forge action.
248. **[U]** "Recently looted" inventory section (last 10 min).
249. **[U]** Consumable fade warnings between floors.
250. **[U]** Sidebar panel transparency/density slider.

## K. UI — feedback, onboarding & flow (AB-251 – AB-275)

251. **[U]** First-death coaching targeted at the real cause (out-CR'd, uninsured, no consumables).
252. **[U]** Progressive tips: run 2 shows checkpoints, run 3 shows insurance.
253. **[U]** Confirm-dialog heuristic: only pester when the action risks >10% of net worth.
254. **[U]** "While you were away" summary on return (regen, jackpot growth).
255. **[U]** Monday weekly-recap modal (depths, banks, best drop).
256. **[U]** Toast severity uses shapes + colors (colorblind-safe audit).
257. **[U]** Busy indicator names the running action ("tempering…").
258. **[U]** Undo-toast pattern extended to auto-salvage.
259. **[U]** Share a run summary as a markdown card for chat.
260. **[U]** Death screen "near miss" line within 5% of a checkpoint.
261. **[U]** XP tooltip previews next level's rewards.
262. **[U]** Every empty panel gets a one-action empty state.
263. **[U]** Rotating "did you know" tip on the entry screen.
264. **[U]** Milestone celebrations queue instead of stacking.
265. **[U]** Latency dot in the HUD (lightweight ping).
266. **[U]** Session stats drawer (time, floors, gold/hour).
267. **[U]** Failed descend re-offers one-click retry with preserved settings.
268. **[U]** Visual-only confetti toggle for celebrations.
269. **[U]** Simultaneous warnings merge into one counted chip.
270. **[U]** 10-minute AFK overlay pauses run timers.
271. **[U]** Bookmarkable URL restoring exact panel state.
272. **[U]** First-legendary-of-the-day special toast variant.
273. **[U]** One-time guided warning before your first insanity entry.
274. **[U]** Post-prestige checklist (what reset, what stayed).
275. **[U]** Hint dismissal memory — never show the same tip twice.

## L. Meta, seasons & tech (AB-276 – AB-300)

276. **[G]** Season theme example: "Ember Descent" — fire-biome weighting + matching palette.
277. **[G]** Season journey page: 10-week track of weekly cosmetic unlocks.
278. **[T]** Admin balance dashboard: drop-rate and death-rate charts.
279. **[T]** Per-subsystem feature flags with an admin toggle panel.
280. **[T]** A/B harness for reward multipliers with guardrail metrics.
281. **[T]** *(heavy)* Server-side run replays (seed + log) with a UI viewer.
282. **[T]** Read-only `/api/abyss/public/stats` JSON for community tools.
283. **[T]** Discord webhook for Eternal drops and world-first depth records.
284. **[U]** Rate-limit feedback: show remaining actions instead of failing.
285. **[T]** Aggregate "killed by" analytics powering the AB-190 revenge markers.
286. **[T]** Pre-deploy schema check comparing DB constraints to the tier catalog (prevents the insanity CHECK-constraint bug class).
287. **[T]** Golden-template snapshot tests for the abyss page per fixture state.
288. **[T]** Playwright E2E for enter → descend → bank → death in CI.
289. **[T]** Load test for the descend endpoint at 100 concurrent delvers.
290. **[T]** Anonymous funnel telemetry (enter → floor 5 → bank) for balance calls.
291. **[T]** Admin god-view: inspect any player's live run state for support.
292. **[T]** Config-driven tiers (JSON file) instead of recompiles.
293. **[T]** Client error reporting: `window.onerror` → `/api/abyss/client-error`.
294. **[T]** Combat-log virtualization for 500+ line fights.
295. **[T]** HUD chip recomputes debounced to one frame.
296. **[T]** Rarity color map computed once instead of per-row scans.
297. **[T]** Connectivity loss pauses run timers (not just tab focus).
298. **[T]** i18n coverage meter: missing abyss keys per locale, admin view.
299. **[T]** Content validation CI test: unique item IDs, ≥2 pieces per set, every effect described.
300. **[U]** In-portal changelog panel fed from a CHANGELOG section.
