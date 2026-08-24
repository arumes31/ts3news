# The Abyss — 400 Improvement Ideas

Brainstormed backlog for gameplay, UI, playability, economy, social and tech.
Grouped into 16 areas × 25 ideas. Numbers are stable — reference them in issues.

## ✅ Implemented (2026-07-04, migrations 0062–0063)

**Core loop:** 1 (daily free entry) · 2 (checkpoint starts, tokens, ×0.75 rewards) ·
3 (express elevator, no bonus until past record) · 4 (escrow interest — was already live) ·
7 (momentum, +2%/floor STR, breaks on consumable use) · 15 (Last Stand token revive at 25%+Mercy,
2-floor bank seal; double-or-nothing now offered on only ~45%+Mercy of downs) · 16 (loot-find
depth milestones) · 19 (prestige tier names/auras) · 21 (risk % on Descend button) · 24 (comeback buff).

**Events:** 26 (three chests) · 30 (cursed library) · 32 (gambling den, five posted-odds games) ·
35 (rift peek + floor queue) · 38/113 (sanctuary upgrades incl. crafting station) · 41 (blood altar) ·
43 (alchemy lab) · 50 (hall of mirrors).

**Combat/bosses/boards:** 51 (end-of-fight summary line) · 201 (boss intro cards) · 276 (per-tier board tabs).

**Loot:** 76 (▲/▼ CR delta on drops) · 83 (corrupted gear + forge cleanse) · 84 (gem tiers I–III + upgrades) ·
86 (four build-changing uniques) · 88/329 (rarity beams) · 92 (Ancient fusion) · 99 (identify-all) ·
112 (Mythic fusion) · 313 (run loot grouped by floor) · 315 (run loot count/cache summary).

**Crafting/forge (101–126):** 101–103 (materials, tab, recipes) · 104 (recipe discovery via lore) ·
105 (weekly craft quest) · 106–107 (temper + fail pity) · 108 (weapon gear-XP milestones) ·
109/115 (recalibrate ranges shown) · 110 (dismantle filters + preview) · 111 (enchant transfer) ·
114 (artisan rep discounts) · 116 (daily forge undo) · 117 (gem extraction) · 118 (rune library) ·
119 (deep material seams 30+) · 120 (deterministic legendary craft) · 121 (forge happy hour) ·
122 (delta hints on forge buttons) · 123 (forge history) · 124 (repair-all with preview) ·
125 (auto-repair toggle) · 126 (token exchange, 1 🜲 = 100K gold buy / 90K sell).

**Talents/progression:** 154–158 (Swiftness, Scavenger, Mercy, Cartographer, Quartermaster) ·
159 (branched tree layout) · 160 (depth-gated nodes) · 161 (Delver/Plunderer/Warden specs) ·
168 (bestiary mastery damage).

**UI bundle:** 301 (sticky mini-HUD) · 303 (section tabs) · 326 (elevator animation) · 333 (pity pulse) ·
335/336 (gold/token count-ups) · 338 (revive dice animation) · 339 (bank vault overlay) ·
341 (run summary card) · 347 (insurance glow) · 349 (dial shimmer) · 350 (escrow coin pile).

## A. Core loop & risk/reward (1–25)

1. Daily free descent: first run of the day waives tier entry gold.
2. Checkpoint floors every 10 depths — pay tokens to start a later run there at reduced rewards.
3. Express elevator: skip to (best depth − 5) for gold, loot disabled until you pass your record.
4. Escrow interest: +0.5% cache per floor survived without banking (stacking greed incentive).
5. Partial banking: bank 25%/50% of the cache mid-run for a 10% fee, keep descending with the rest.
6. "Double or descend": optional coin-flip on the floor bonus you just earned before the next floor.
7. Momentum stacks: +2% damage per consecutive floor cleared without consumables, reset on use.
8. Grace floors: no death/abandon penalties above depth 3 (new-player friendliness).
9. Hardcore mode: no revive, no insurance, 2× rewards, separate leaderboard and badge.
10. Weekly gauntlet: fixed-seed run identical for everyone; compare apples to apples.
11. Insurance loyalty: premium rate drops slightly with lifetime banked gold.
12. Cursed elevator event: 5% chance a descend drops 2 floors — both rewards, both dangers.
13. Guaranteed rest floor if none rolled naturally within 7 floors.
14. Soft escrow cap with visible diminishing returns to discourage infinite hoard-running.
15. Last stand: on death, a one-time token purchase revives instantly at 25% HP (no double-or-nothing).
16. Depth milestones grant tiny permanent account bonuses (+1% loot find at 25, 50, 75…).
17. Token ante: wager tokens before a run for a reward multiplier, lost on death.
18. Pace ghost: show your best run's depth-per-minute as a marker during the current run.
19. Prestige tiers II–V with escalating cosmetic auras and +% per tier.
20. Season "descent pass": per-depth reward track (free + premium-with-tokens lanes).
21. "One more floor" prompt shows exact estimated death risk from CR vs next-floor difficulty.
22. Bank-streak insurance: after 3 banked runs in a row, next insurance is free.
23. Overcap conversion: cache above the soft cap partially converts to tokens on bank.
24. Comeback buff: after 3 deaths in a day, +10% stats for the next run (clearly labeled).
25. Elective difficulty dial per floor (−20%…+50% danger) fine-tuning rewards between tiers.

## B. Floor types & events (26–50)

26. Puzzle floor: pick 1 of 3 chests with readable hints; skill-check flavor, no combat.
27. Trap gauntlet floor resolved by SPD/DGE checks with a visible pass chance.
28. Locked treasure vault floor — needs a Key item that drops from elites.
29. Rescue floor: free a lost delver NPC who fights beside you for 3 floors.
30. Cursed library: trade max-HP% for lore fragments and a rare skill book.
31. Mirror floor: fight an exact stat-clone of yourself.
32. Gambling den floor: three mini-games (dice, cards, wheel) with posted odds.
33. Forge floor: one free temper/socket/repair action mid-run.
34. Abyssal market floor: merchant stock scales with depth, rotating rare stock.
35. Rift floor: pay gold to peek at the next 3 floor types.
36. Storm floor: ticking hazard damage all fight, +50% loot quality.
37. Darkness floor: combat log hidden until the end — you only see the outcome.
38. Sanctuary upgrades: spend tokens to permanently improve rest-floor options.
39. Event chains: multi-floor mini-quests (collect 3 sigils across 10 floors → chest).
40. Lost cartographer: buy his map to reveal the next 5 floor types.
41. Blood altar: sacrifice a consumable for a strong 3-floor buff.
42. Echo floor: replay your previous floor's reward roll at 50% value.
43. Alchemy lab: combine two consumables into one stronger one.
44. Unstable portal: jump +3 floors instantly, skipping their rewards.
45. Graveyard floor: fight the ghost of a real player who died at this depth (async).
46. Bounty board floor: pick a side objective for the next 5 floors (e.g. no healing).
47. Collapsed passage: dig through (STA check) or detour (+1 floor danger).
48. Community wishing well: pot fed by all players' coin tosses; jackpot announced server-wide.
49. Abyssal garden: harvest a random crafting material node.
50. Hall of mirrors: choose one of three temporary "reflection" buffs shown with exact numbers.

## C. Combat mechanics (51–75)

51. End-of-fight summary: total damage dealt/taken, crits, dodges, lifesteal, thorns.
52. Elemental matchup preview (strong/weak icons) before descending into a known mob type.
53. Stance selector per floor: aggressive (+dmg/−def), defensive, balanced.
54. Skill priority editor: drag to reorder which skills fire first.
55. Manual vs auto ultimate toggle set before the floor.
56. Overkill damage on the last mob converts to bonus gold.
57. Execute threshold marker visible on mob HP in the log.
58. Mob abilities telegraphed one floor in advance ("next: a summoner").
59. Variety bonus: using 3+ different skills in a fight grants +5% reward XP.
60. First-strike bonus scales with your SPD advantage and is called out in the log.
61. Per-gear damage-type contribution shown in an expandable tooltip.
62. Shield/absorb mechanic so DEF builds get pre-HP mitigation.
63. Bleed/poison DoTs with per-round icons in the combat log.
64. Boss enrage timer displayed as a countdown chip.
65. Weakness window: stunning a mob guarantees the next hit crits.
66. Pet commands: focus my target / guard me / free-for-all.
67. Expose the existing front/backline position mechanic as a pre-run choice.
68. Show revive-gamble odds explicitly (e.g. "48% to return, doubled cache").
69. Combat speed setting: instant / fast / dramatic (current).
70. Skip-log button that still prints the loot + summary lines.
71. Round-cap/fatigue indicator so stalemates are legible.
72. Token-priced "simulate" button: run 100 shadow fights vs the next floor, show win%.
73. Auto-descend mode with stop rules (stop at HP < 50%, at depth X, on legendary drop).
74. Mob affix chips (armored, swift, vampiric) with hover tooltips.
75. Critical-fail drama: 1% fumble events with funny log lines (no mechanical downside).

## D. Loot & itemization (76–100)

76. Run-loot rows show a compare-vs-equipped delta (▲/▼ CR) on hover.
77. Loot filter: auto-salvage commons/uncommons toggle with recap line.
78. Smart loot bias toward your empty or weakest slots.
79. Set-piece pity: owning 3 pieces biases drops toward the 4th.
80. Transmog: unlock appearances, apply skins for gold.
81. Item lock flag preventing salvage/dismantle/auto-list accidents.
82. Partial recalibrate: reroll one stat line for fewer tokens.
83. Corrupted items: oversized stats + a drawback, cleansable via a quest chain.
84. Gem tiers (I–III) and 3-into-1 gem merging.
85. Rune+element synergy: matching rune to your element gives +5%.
86. Build-changing unique legendaries ("your Thorns also heals you").
87. Weekly featured drop pool with 2× chance, shown on the abyss page.
88. Distinct drop animation/beam per rarity in the loot lines.
89. Server-wide TS3 announcement for Mythic+ drops.
90. Wishlist: mark 3 catalog items, gain slow pity toward them.
91. Duplicate protection: no identical item within 20 floors.
92. Ancient upgrade: fuse 3 same-slot legendaries into one boosted piece.
93. Compact number formatting toggle for all item stats.
94. Item provenance in tooltip: depth found, boss, date.
95. Visible set-piece bad-luck-protection progress bar.
96. Craftable insurance charms consumed on death in place of gold insurance.
97. Socket rerolling for rings (count capped, positions rerollable).
98. First identify of the day is free.
99. Identify-all button with a single combined cost confirm.
100. Escrow inspection: expand a run-loot gear row to full stats before deciding to bank.

## E. Forge & crafting (101–125)

101. Salvage yields tiered crafting materials (dust/shards/cores) by rarity.
102. Materials tab with counts and sources.
103. Craft consumables from materials (recipes gated by depth).
104. Recipe discovery through lore fragments (lore becomes useful).
105. Weekly crafting quest with a guaranteed epic+ result.
106. Temper system: +1…+15 with rising fail chance and protection stones.
107. Fail-stack pity: each failed temper adds +5% to the next attempt.
108. Gear XP: items gain kills, unlocking a bonus affix at max level.
109. Reforge preview: show the 3 possible outcome ranges before committing.
110. Batch dismantle with rarity/slot filters and a total-yield preview.
111. Enchant transfer between items for tokens.
112. Mythic fusion: combine two Mythics for a Divine chance (posted odds).
113. Crafting stations as purchasable rest-floor upgrades.
114. Artisan reputation levels granting forge discounts.
115. Exact stat ranges displayed before recalibration.
116. One free "undo last forge action" per day (60s window).
117. Socket extraction: remove a gem intact for a fee.
118. Rune library: once etched, a rune type is unlocked for cheaper re-etching.
119. Gem-only drop table on depths 30+.
120. Deterministic legendary crafting: 500 materials → chosen catalog item.
121. Daily forge discount hour announced in the header.
122. Every forge button shows the CR/GS delta it would produce.
123. Forge history log (last 20 actions with costs).
124. Bulk repair-all with cost preview.
125. Auto-repair toggle before each descent (opt-in, shows cost each time).

## F. Economy & sinks (126–150)

126. Token ⇄ gold exchange with a visible spread (sink both ways).
127. Weekly rotating cosmetic stock in the token shop.
128. Escrow "armored transport": bank remotely mid-run at 85% value.
129. Auto-insure subscription: always insure 25% at run start (opt-in).
130. Materials section on the Auction House.
131. Repair-cost curve shown on a tooltip so durability costs are legible.
132. Vanity gold sinks: statues/trophies on a public plaza page.
133. Vendor buyback list (last 10 sold items).
134. Consumable pouch upgrades raising stack/carry caps.
135. AH price history sparkline per item.
136. Dynamic shop pricing nudged by server-wide demand.
137. Loyalty punch card: 10th token-shop purchase free.
138. Bundle deals (potion packs, repair kits + insurance combo).
139. Consumable gifting with a small gold fee.
140. Deep-cache jackpot ticker visible on the abyss page.
141. Sell junk directly from the run-loot list at 50% while descending.
142. AH anti-snipe: bids in the last minute extend the timer.
143. AH sales tax feeds the community jackpot.
144. Unified currency bar: gold, tokens, scrap, materials.
145. Daily market flash sale with a header notification.
146. Convert duplicate lore fragments into tokens.
147. Bounty-streak token interest (streak already exists — pay interest on it).
148. Escrow tithe pact: donate 10% of banks to the jackpot for +luck.
149. Buy-order system on the AH (post what you want, price you'd pay).
150. Season-end token conversion into cosmetics instead of hard reset.

## G. Progression & talents (151–175)

151. Talent loadouts: save/swap two Deep Delver configurations.
152. Refund a single talent node instead of all-or-nothing.
153. Soft caps: nodes go to Lv10 with decreasing increments.
154. New node — Swiftness: +combat speed / shorter logs.
155. New node — Scavenger: +crafting material yield.
156. New node — Mercy: +revive gamble odds.
157. New node — Cartographer: floor-type preview range +1.
158. New node — Quartermaster: +1 consumable carry cap.
159. Visual talent tree with branches instead of a flat list.
160. Milestone talents unlocked by depth records, not tokens.
161. Specializations: Delver / Plunderer / Warden with exclusive perks.
162. Weekly challenge that grants talent XP as an alternative to tokens.
163. Prestige-only talent currency and nodes.
164. Achievements grant one-off talent points.
165. Per-tier mastery bars (Normal → Nightmare) with completion rewards.
166. Biome mastery: clear 50 floors of a biome for a permanent biome bonus.
167. Set collection book with rewards at 25/50/100% completion.
168. Bestiary milestones: 100 kills of a family → +1% damage vs it.
169. Lore completion grants an exclusive title and badge.
170. Two badge slots: prefix + suffix combinations.
171. Seasonal temporary talent that resets each season.
172. New-player boost: first 10 floors at +100% XP.
173. Returning-player catch-up buff after 14 days away.
174. Skill rank-up progression bars visible in the armoury.
175. Paragon system post-prestige: infinite +0.1% nodes.

## H. Pacts, affixes & modifiers (176–200)

176. Pact presets: save favorite combinations.
177. Pact mastery: 10 completed runs with a pact → +5% to its bonus.
178. New pact — Abstinence: no consumables (+15%).
179. New pact — Pauper: no gear above Rare equipped (+30%).
180. New pact — Anemic: half max HP (+25%).
181. New pact — Cursed Horde: mobs gain +1 affix (+20%).
182. New pact — Deep Drums: boss every 3rd floor (+35%).
183. New pact — Uninsured: insurance disabled (+15%).
184. New pact — Blind: no floor previews or threat meter (+10%).
185. New pact — Brittle: double durability loss (+10%).
186. New pact — Famine: no rest floors (+20%).
187. Conflicting-pact warnings before entering.
188. Affix calendar: see the week's daily affixes ahead of time.
189. Player-voted weekend affix (poll on the page).
190. Pact+affix synergy bonuses called out ("Bloodlust + Anemic: +5% extra").
191. Mystery pact: random hidden pact for +40%.
192. Pact rewards paid partly in tokens, not just gold.
193. Show the full pact bonus math on the bank confirmation.
194. Featured pact of the week at double bonus.
195. Pact-specific achievements and badges.
196. Personal affix reroll once per day for tokens.
197. Affix suppressor consumable: ignore today's affix for one run.
198. Per-affix personal stats: your win rate and average depth under each.
199. Contract pacts: mid-run failure conditions with forfeiture clauses.
200. Community affix weekends: server-wide banked-gold goal unlocks a global buff.

## I. Bosses & monsters (201–225)

201. Boss intro cards: name, title, mechanic hint before the fight.
202. Boss phases at 50%/25% HP with distinct log flavor.
203. Weekly boss rotation with unique per-boss drop tables.
204. Elite mobs on every 3rd floor with aura modifiers.
205. Named rare spawns carrying fixed unique drops.
206. Boss enrage round displayed as a chip during the fight.
207. Per-boss kill-time leaderboards (not just global).
208. Practice mode: refight beaten bosses without rewards or risk.
209. Boss tokens: currency for a boss-vendor with mechanic-countering gear.
210. Telegraphed summon phases ("the Firelord raises his horn…").
211. Weakpoint choice mid-boss: target head (dmg) or arms (disable ability).
212. First-kill lore entries per boss in the codex.
213. World boss weekends: server-shared HP pool with contribution rewards.
214. Treasure goblin variants: gem goblin, token goblin, key goblin.
215. Mimic King: rare chain event started by surviving 3 mimics.
216. The Watcher: visible "stalking meter" that fills while idling.
217. Ghost echoes of specific friends (opt-in) instead of random players.
218. Boss contracts: declare a kill attempt and wager tokens on it.
219. Double-boss floors past depth 60.
220. Daily rotating boss elemental affinities shown in advance.
221. Boss toll: pay expected-loot value to skip a boss floor entirely.
222. Shareable best-kill summary cards.
223. Secret boss chain unlocked by completing all lore fragments.
224. Cosmetic-only mount/banner drops from tier-scaled boss kills.
225. Adaptive bosses: a boss you've beaten 10× gains one new trick.

## J. Pets & companions (226–250)

226. Pet gear slots: collar + charm with pet-only stats.
227. Pet abilities with cooldowns shown in the combat log.
228. Capture management UI: at the 3-pet cap, choose who to release.
229. Pet stable page: rename, release, favorite.
230. Pet feeding: spend consumables for pet XP.
231. Loyalty stat: fights survived together grants bonuses.
232. Loyalty reduces the 3% betrayal chance down to 0%.
233. Pet classes: tank / damage / support with distinct AI.
234. Support pets heal their owner between rounds.
235. Pet fusion: merge similar pets into a stronger one.
236. Shiny variants: 1% recolor captures with a badge.
237. Pet cosmetics from the token shop.
238. "Strongest captured pet" leaderboard.
239. Daycare: benched pets gain slow passive XP.
240. Pet expeditions: send benched pets on timed missions for materials.
241. Mind Control rework: choose which weakened mob to capture.
242. Talent node raising the pet cap to 4–5.
243. Pet revival scroll for pets lost to betrayal or HP loss.
244. Pet gifting between players.
245. Bestiary integration: capture rates per mob family.
246. Companion-slot gear that synergizes with your active pet's class.
247. Pet emotes/barks in the combat log (mutable).
248. Pet naming with a profanity filter.
249. Show active pets in the abyss Armoury sidebar with HP.
250. Ultra-rare boss-variant pet captures (mini versions of bosses).

## K. Social & co-op (251–275)

251. Full co-op descents: party of 2–3 shares the run and splits escrow on bank.
252. Spectator mode: watch a friend's live run log read-only.
253. Daily cheer: send one buff to a friend mid-run.
254. Rescue missions: clear a floor where a friend died to recover 10% of their lost cache for both.
255. Guilds with shared weekly depth goals.
256. Guild leaderboards and guild banner cosmetics.
257. Abyss-page shoutbox bridged to a TS3 channel.
258. Consumable trading window.
259. Mentor pairing: veteran + newbie both earn bonus tokens.
260. Server goal bar: “10,000 floors cleared today” unlocks a happy hour.
261. Duels: wagered arena fights using abyss builds.
262. Floor messages: leave one hint/taunt per run for others to find (Dark Souls style).
263. Kudos: post-run appreciation points with a weekly board.
264. Referral rewards for recruiting delvers.
265. Scheduled team tournaments with brackets.
266. Raid lobbies: 5 players vs a mega-boss with shared loot rolls.
267. Web friend list with online/in-run status.
268. Activity feed: "X banked 50k", "Y hit depth 60", "Z captured a shiny".
269. Exportable run-summary image cards for sharing.
270. Need/greed loot rolls on co-op boss drops.
271. Helper rewards scale with the depth of the boss they helped on.
272. Invite links that deep-link straight into a co-op lobby.
273. Social achievements (help 10 allies, win 5 duels).
274. Rival system: auto-matched rival near your depth; beating their record pays extra.
275. Emote wheel whose picks appear as log flavor lines.

## L. Leaderboards & competition (276–300)

276. Boards per tier (Nightmare, Hell…) as tabs, not just Normal.
277. Build/class filters on boards once specializations exist.
278. Weekly and seasonal splits with automatic reward payouts.
279. Percentile line: "you are top 12% this season".
280. Pagination + "jump to me" button.
281. Speedrun boards: fastest real-time to depth 20.
282. Economy board: most banked this week.
283. Pact board: highest total pact multiplier survived.
284. Bestiary kill boards per mob family.
285. Opt-in "hall of shame": funniest deaths with log excerpt.
286. Bank-streak leaderboards.
287. Pet power leaderboards.
288. Archived season snapshots browsable by season.
289. Season rewards: cosmetics top 10, tokens top 100, badge for all participants.
290. Server-side audit trail for record runs (anti-cheat evidence).
291. Personal bests panel with deltas vs last week.
292. Rank-change notifications ("you were passed by X").
293. Per-TS3-channel boards for local bragging rights.
294. Season-themed trophies rendered on profiles.
295. Wager ladders: entry-fee brackets with prize pools.
296. Tie-break rules displayed on every board.
297. Board embeds pushed into TS3 channel descriptions.
298. "Close to record" nudge when within 2 floors of a board spot.
299. Depth-over-time graph on your profile.
300. Hall of fame page for past season champions.

## M. UI/UX — layout & readability (301–325)

301. Sticky mini-HUD (HP, escrow, depth, threat) once the stage scrolls out of view.
302. Collapsible panels with remembered open/closed state per user.
303. Tab the lower page (Delver / Shop / Forge / Record / History) instead of one long scroll.
304. Real mobile layout for stage controls (buttons reachable one-handed).
305. Font-size setting (S/M/L) persisted per account.
306. Compact mode for run-loot rows (single line, tooltip detail).
307. Rarity color legend available from a "?" tooltip.
308. Tier picker cards get icons and win-rate hints.
309. Threat meter shows the numeric % and a tooltip explaining the formula.
310. Depth ring shows "boss in N floors" around its edge.
311. Toggle exact vs abbreviated numbers (41,468 vs 41.5k) everywhere.
312. Search/filter inside the forge item dropdown.
313. Group run loot by floor using the stored Depth (headers: "Floor 6").
314. Run-loot filters: gear only / consumables / gold, plus a gold-total row.
315. Total estimated value of the current run loot in the sidebar header.
316. Combat log timestamps toggle.
317. Auto-scroll lock button on the combat log.
318. Keyboard shortcuts: D descend, B bank, I insure, with a cheat-sheet modal.
319. Active pacts shown as chips near the depth dial during a run.
320. Insurance nudge: highlight the insure row when escrow is large and 0% insured.
321. Consumable prep shows remaining counts and per-item cooldowns.
322. Uniform confirm modals for every destructive/costly action.
323. Illustrated empty states with a tip ("No loot yet — descend!").
324. Skeleton loaders instead of blank panels during fetches.
325. A single settings page collecting all toggles introduced above.

## N. UI/UX — feedback & juice (326–350)

326. Elevator descent transition animation between floors.
327. Subtle screen shake on boss spawns (disabled with reduced-motion).
328. Floating damage numbers in the log area for big hits.
329. Rarity-colored light beams on loot drop lines.
330. Optional sound effects with a mute toggle (drops, bank, death).
331. Tactile button press states across all abyss buttons.
332. Confetti burst on a new best depth.
333. Pity bar pulses when ≥90% toward guaranteed legendary.
334. Drop-streak flame animation grows with the streak.
335. Token gains tick up with a counting animation.
336. Gold counter count-up on bank.
337. Boss HP bar with phase markers at 50/25%.
338. Slot-machine style animation for the revive gamble.
339. Vault-door animation on successful bank.
340. Death screen with a full run recap (floors, damage, lost cache, lesson tip).
341. End-of-run summary card: floors, loot count, records, biggest hit.
342. Milestone toasts at depths 10/20/30… with distinct styling.
343. Parallax background that darkens as depth increases.
344. Biome-tinted stage backdrop matching the current biome.
345. Lantern-glow vignette effect deepening with depth (toggleable).
346. Achievement banners queue instead of overlapping.
347. Insurance shield icon glows while coverage is active.
348. Escrow value pulses red while you are downed.
349. Idle micro-animations on the depth dial between actions.
350. Animated "Cache at stake" coin pile that grows with escrow size.

## O. Accessibility & QoL (351–375)

351. Full keyboard navigation with visible focus rings on all abyss controls.
352. ARIA labels/live regions audited across every dynamic element.
353. Colorblind-safe rarity palette option (shape + color coding).
354. Reduced-motion audit for every animation added above.
355. High-contrast theme toggle.
356. Text alternatives for all emoji-as-icon uses.
357. Log verbosity setting: full / summary-only.
358. Descend debounce + disabled-state to prevent double-click mishaps.
359. Harden state resume: mid-fight refresh restores the exact log position.
360. Session-expiry warning before the login token dies mid-run.
361. Complete abyss i18n coverage in all shipped locales.
362. Locale-aware number formatting everywhere.
363. Tooltips for every stat abbreviation (CR, GS, STA, DGE…).
364. First-visit guided tour of the abyss page.
365. Help modal with a mechanics glossary (pity, escrow, threat, pacts).
366. "What's new" changelog panel fed from release notes.
367. Error toasts with a retry button instead of dead ends.
368. API latency indicator (subtle dot: green/yellow/red).
369. Persist tier/pact/focus selections across sessions.
370. Monospace-log font option.
371. 44px minimum touch targets on mobile.
372. Dynamic page title: "Depth 12 · The Abyss" during runs.
373. Favicon badge showing current depth during an active run.
374. Opt-in browser notifications: bounty complete, revive ready, AH sold.
375. Export run history and stats as CSV.

## P. Meta, seasons, retention & tech (376–400)

376. Seasonal themes with a season-wide twist mechanic.
377. Season journal: 50 objectives with a final cosmetic.
378. Abyss-flavored daily login calendar.
379. Weekly personal digest (poke or web): depth trend, gold/day, near-records.
380. Endless mode past depth 100 with cosmetic-only scaling rewards.
381. Biome selection at rest floors (pick your poison for the next 5 floors).
382. NG+ tier unlock quests instead of pure depth gates.
383. Authored story campaign: 10 handcrafted floors with fixed setpieces.
384. Run-only relics: roguelike passives that drop and expire when the run ends.
385. Boon drafts: pick 1 of 3 run-buffs every 5 floors.
386. Player analytics dashboard: charts of your own performance.
387. Read-only public stats API with personal tokens.
388. Discord/webhook integration for milestones and legendary drops.
389. Replay system: store seed + choices, replay any past run's log.
390. Deterministic seeds recorded per run for dispute resolution.
391. Rate limiting + macro detection on descend/bank endpoints.
392. Load tests for the descend endpoint at peak concurrency.
393. Browser E2E tests (Playwright) covering enter→descend→bank→death paths.
394. Feature flags per mechanic for gradual rollouts.
395. Funnel telemetry: where players stop descending, tune difficulty there.
396. Balance dashboard built on the existing cmd/simulation harness.
397. A/B framework for reward tuning with guardrails.
398. Escrow table backup/restore drills and integrity checks.
399. Idempotency keys on bank/descend/revive APIs to kill double-submit bugs.
400. Auto-generated player wiki from the content tables (gear, mobs, pacts, affixes).
