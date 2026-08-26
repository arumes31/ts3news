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
- [x] **AAA-0005 / 5** — secure 25% or 50% of a live cache for a visible 10% fee and continue the run.
- [x] **AAA-0006 / 6** — one persisted coin flip can double or remove only the latest cleared floor's bonus.
- [x] **AAA-0007 / 7** — consumable-free momentum grants 2% strength per floor, capped at 20%.
- [x] **AAA-0008 / 8** — floors 1–3 return the full cache and loot without death or bank-streak penalties.
- [x] **AAA-0009 / 9** — hardcore runs disable protection and revival, double cache rewards, and use a dedicated board and badge.
- [x] **AAA-0010 / 10** — opt-in weekly expeditions retain one UTC ISO-week seed for the full run.
- [x] **AAA-0011 / 11** — lifetime banking loyalty lowers the charged and previewed insurance premium rate by up to 15 points.
- [x] **AAA-0012 / 12** — five percent of ordinary descents become a server-authoritative two-floor cursed-elevator drop; both forced combat floors resolve rewards and danger independently, stop on defeat, and play back in order (`web_abyss.go`, `web_abyss_live_layout_test.go`, `webassets/abyss.html`).
- [x] **AAA-0013 / 13** — an authoritative run cadence guarantees a sanctuary after seven floors without rest.
- [x] **AAA-0014 / 14** — depth-scaled escrow thresholds visibly reduce only new marginal growth to 25 percent.
- [x] **AAA-0015 / 15** — one token-priced Last Stand restores at least 25% HP and seals banking for two victories.
- [x] **AAA-0016 / 16** — each 25 best-depth milestone grants 1% permanent loot find, capped at 4%.
- [x] **AAA-0017 / 17** — a validated 5/10/20-token ante is debited atomically at entry, boosts every floor's cache by the same percentage, and returns only after a successful full bank (`web_abyss_stakes.go`, `web_abyss_stakes_test.go`).
- [x] **AAA-0018 / 18** — the live pace marker compares the current run with the player's server-authoritative all-history best floors-per-minute run while retaining the latest-run reference (`web_abyss_longterm.go`, `web_abyss_longterm_test.go`).
- [x] **AAA-0019 / 19** — five prestige tiers expose escalating names and cosmetic auras.
- [x] **AAA-0020 / 20** — the ten-week floor-progress journey presents free and premium cosmetic lanes; a one-time seasonal token unlock and every premium claim are atomic and idempotent (`web_abyss_season.go`, `web_abyss_season_premium.go`, `web_abyss_season_test.go`, `tests/e2e/abyss.spec.js`).
- [x] **AAA-0021 / 21** — Descend shows the server-estimated next-floor death risk.
- [x] **AAA-0022 / 22** — every third consecutive full bank atomically grants one non-stacking insurance voucher; the next valid coverage purchase consumes it in the same transaction (`web_abyss_bank_incentives.go`, `web_abyss_bank_incentives_test.go`).
- [x] **AAA-0023 / 23** — full banks convert a rounded 10% share of cache above the depth soft cap into tokens at the normal exchange rate, with the exact gold and tokens itemized before confirmation (`web_abyss_bank_incentives.go`, `web_abyss_bank_incentives_test.go`, `tests/e2e/abyss.spec.js`).
- [x] **AAA-0024 / 24** — three same-day deaths fund one clearly labeled +10% comeback run.
- [x] **AAA-0025 / 25** — a server-authoritative −20%…+50% dial moves encounter danger and floor cache together and can be changed between, but not during, live floors (`web_abyss_stakes.go`, `web_abyss_stakes_test.go`).

## Delivered tranche: floor ecology audit

- [x] **AAA-0026 / 26** — Three Chests persists one hidden answer and resolves a single hinted choice without leaking it to the client.
- [x] **AAA-0027 / 27** — Trap Chambers publish bounded DGE-based pass odds and atomically apply either cache gain or nonlethal damage.
- [x] **AAA-0028 / 28** — Locked Vaults consume run-scoped keys for one explicit cache, token, or material reward.
- [x] **AAA-0029 / 29** — rescuing a canonical Lost Explorer creates one server-owned ally strike per combat round for the next three cleared fights, exposes the delver in the live formation and initiative order, and retains the existing defensive map and vault-key rewards (`abyss_rescue_support.go`, `abyss_rescue_support_test.go`, `web_abyss_rooms.go`, `web_abyss_live.go`, `webassets/abyss.html`).
- [x] **AAA-0030 / 30** — Cursed Libraries trade either maximum health or a timed speed penalty for lore and a skill elixir.
- [x] **AAA-0031 / 31** — Mirror combat clones the player's current stats, skills, and equipped gear without aliasing mutable state.
- [x] **AAA-0032 / 32** — Gambling Dens expose posted stake, prize, and odds for five depth-gated games.
- [x] **AAA-0033 / 33** — the Silent Anvil offers one free eligible temper attempt, socket punch, or full equipped-gear repair; the existing forge mutation transaction locks and consumes the room only when the chosen operation commits, while quotes and the Forge tab expose the exact zero cost (`abyss_forge_floor.go`, `abyss_forge_floor_test.go`, `web_abyss_features.go`, `web_abyss_forge2.go`, `web_abyss_forge_quotes.go`, `webassets/abyss.html`).
- [x] **AAA-0034 / 34** — Abyssal Market prices scale with discovery depth and include one opaque fixed-price mystery slot.
- [x] **AAA-0035 / 35** — Scrying Rifts offer up to three paid visions and persist the revealed floor queue server-side.
- [x] **AAA-0036 / 36** — Storm combat telegraphs its next target side before applying bounded environmental damage.
- [x] **AAA-0037 / 37** — Darkness combat redacts enemy health from snapshots and timelines without mutating authoritative combat state.
- [x] **AAA-0038 / 38** — Sanctuary improvements persist token-funded upgrades to healing, repair, rerolls, maps, and crafting.
- [x] **AAA-0039 / 39** — the Triune Sigil Hunt binds a server-authoritative ten-floor event chain, marks three scheduled combat victories, expires without rollover, and atomically opens a depth-scaled cache chest on the third sigil (`abyss_event_chain.go`, `abyss_event_chain_test.go`, `web_abyss_victory_state.go`, `web_abyss_rooms.go`, `webassets/abyss.html`, `tests/e2e/abyss.spec.js`).
- [x] **AAA-0040 / 40** — the Lost Cartographer atomically sells one depth-priced, server-authoritative chart of the next five floor types; the persisted route accounts for bosses, sanctuaries, pacts, and event cadence, advances across single, live, and planned descents, and remains readable on mobile (`abyss_cartographer.go`, `abyss_cartographer_test.go`, `web_abyss_cartographer.go`, `web_abyss_cartographer_test.go`, `webassets/abyss.html`, `webassets/abyss_events.css`, `tests/e2e/abyss.spec.js`).
- [x] **AAA-0041 / 41** — Blood Altars consume an owned item for a timed buff, with explicit corrupted-item amplification.
- [x] **AAA-0042 / 42** — Echo Floors recover a bounded share of the last credited reward without rerolling it.
- [x] **AAA-0043 / 43** — Alchemy Labs atomically combine two owned consumables into a stronger result.
- [x] **AAA-0044 / 44** — Unstable Portals jump exactly three floors, disclose skipped room labels, and grant no skipped rewards.
- [x] **AAA-0045 / 45** — Delver Graveyards use a persisted real death when available and support honor, disturb, or echo-duel outcomes.
- [x] **AAA-0046 / 46** — Bounty Boards persist one five-floor side objective with explicit success and failure conditions.
- [x] **AAA-0047 / 47** — Collapsed Passages offer a safe dust detour or a visible health-for-shards squeeze.
- [x] **AAA-0048 / 48** — Wishing Well contributions feed a shared jackpot and retain bounded lifetime contribution context.
- [x] **AAA-0049 / 49** — Abyssal Gardens grant depth-selected materials and a persisted three-harvest mastery bonus.
- [x] **AAA-0050 / 50** — Hall of Mirrors presents three exact temporary buffs and empowers repeated choices across distinct runs.

## Delivered tranche: combat mechanics audit

- [x] **AAA-0051 / 51** — every fight ends with party/enemy damage totals plus non-zero Abyss breakdown lines for thorns and counters.
- [x] **AAA-0052 / 52** — known boss encounters preview the authoritative elemental signature, the equipped main-hand matchup, and explicit strong 2× / resisted ½× rules; the compact card advances to the next natural boss during planned descent and remains overflow-safe on mobile (`abyss_elemental_preview.go`, `abyss_elemental_preview_test.go`, `abyss_boss_affinity.go`, `webassets/abyss_boss_affinity.html`, `webassets/abyss_boss_affinity.css`, `tests/e2e/abyss.spec.js`).
- [x] **AAA-0053 / 53** — live floors expose server-persisted aggressive, balanced, defensive, and conserve-items tactics.
- [x] **AAA-0054 / 54** — a server-persisted, reconciled casting order makes automatic combat choose the first ready affordable skill while manual live actions retain priority; the compact pre-run editor supports drag, keyboard arrows, reset-to-slot-order, database-failure feedback, and mobile overflow protection (`abyss_skill_priority.go`, `abyss_skill_priority_test.go`, `web_abyss_skill_priority.go`, `web_abyss_skill_priority_test.go`, `webassets/abyss_skill_priority.html`, `webassets/abyss_skill_priority.css`, `tests/e2e/abyss.spec.js`).
- [x] **AAA-0055 / 55** — the pre-run hold-mana preference suppresses automatic ultimates on normal waves while explicit live ultimates bypass it.
- [x] **AAA-0056 / 56** — exact excess damage on the final enemy converts at 10:1 into escrow gold, capped at 25% of the fully modified floor reward and governed by the existing soft cap; ordinary, pet, rescued-delver, parry, and thorns finishers flow through authoritative single/live/planned settlement, while death summons and environmental kills cannot mint the reward (`xp_abyss_combat.go`, `web_abyss_risk_rewards.go`, `web_abyss.go`, `webassets/abyss.html`, `webassets/abyss_ui200.css`, `tests/e2e/abyss.spec.js`).
- [x] **AAA-0063 / 63** — poison and related damage-over-time effects tick per round and receive visible log and health-bar treatment.
- [x] **AAA-0064 / 64** — boss encounters announce enrage two rounds ahead and expose an imminent-enrage HUD state.
- [x] **AAA-0068 / 68** — downed responses and controls show the authoritative revive offer percentage and pity streak.
- [x] **AAA-0069 / 69** — persisted 1×, 2×, and 4× combat playback rates support instant, fast, and dramatic viewing preferences.
- [x] **AAA-0070 / 70** — Skip to Result drains only presentation while retaining authoritative reward, loot, and summary lines.
- [x] **AAA-0071 / 71** — round-thirty overtime warns before unavoidable bounded fatigue begins damaging both sides.

## Delivered tranche: loot and itemization audit

- [x] **AAA-0076 / 76** — run-loot gear exposes exact combat-rating delta against the equipped slot on focus, hover, and expansion.
- [x] **AAA-0077 / 77** — account loot rules auto-salvage only Common or Uncommon gear and preserve an authoritative recap.
- [x] **AAA-0081 / 81** — persistent owner-scoped item locks prevent sale, salvage, dismantle, listing, fusion, and sacrifice accidents.
- [x] **AAA-0082 / 82** — Legendary-or-better recalibration rerolls one chosen stat line for a fixed five-token cost.
- [x] **AAA-0083 / 83** — corrupted gear carries oversized trade-off stats and supports explicit cleanse or embrace forge paths.
- [x] **AAA-0084 / 84** — gems have bounded tiers I–III and deterministic three-into-one upgrades.
- [x] **AAA-0086 / 86** — the catalog includes validated build-changing unique gear with described special effects.
- [x] **AAA-0088 / 88** — loot presentation assigns distinct reduced-motion-safe rarity beams through Eternal quality.
- [x] **AAA-0092 / 92** — three compatible same-slot Legendary ingredients fuse through server-validated preview and commit paths.
- [x] **AAA-0099 / 99** — Identify All quotes one combined cost and mutates the reviewed unidentified inventory atomically.
- [x] **AAA-0100 / 100** — keyboard and pointer expansion reveals complete escrow gear stats before the bank decision.

## Delivered tranche: forge and crafting audit

- [x] **AAA-0101 / 101** — rarity-scaled salvage and dismantle yield bounded Dust, Shard, Core, and Prism materials.
- [x] **AAA-0102 / 102** — a dedicated material wallet reports exact counts and acquisition sources.
- [x] **AAA-0103 / 103** — depth-gated recipes craft consumables through authoritative material deductions.
- [x] **AAA-0104 / 104** — lore-fragment progression unlocks persisted recipes instead of client-only visibility.
- [x] **AAA-0105 / 105** — the weekly crafting objective persists progress and pays its guaranteed reward once.
- [x] **AAA-0106 / 106** — Temper raises gear through bounded tiers with posted success odds and optional protection.
- [x] **AAA-0107 / 107** — failed Temper attempts persist a capped pity contribution used by the next quote and roll.
- [x] **AAA-0108 / 108** — equipped gear accumulates kill milestones that unlock durable bonus effects.
- [x] **AAA-0109 / 109** — Reforge previews bounded stat outcome ranges before an item is mutated.
- [x] **AAA-0110 / 110** — batch dismantle commits only the exact reviewed inventory manifest and revision.
- [x] **AAA-0111 / 111** — enchant transfer validates both owned items and atomically moves the effect for tokens.
- [x] **AAA-0112 / 112** — Mythic fusion publishes its Divine chance before consuming two compatible items.
- [x] **AAA-0113 / 113** — a token-purchased Sanctuary Crafting Station enables mid-run recipe access.
- [x] **AAA-0114 / 114** — persisted artisan reputation reduces forge prices and is reflected in server quotes.
- [x] **AAA-0115 / 115** — recalibration and reforge surfaces show exact bounded stat ranges before commit.
- [x] **AAA-0116 / 116** — one server-owned daily forge undo retains a bounded, item-specific recovery snapshot.
- [x] **AAA-0117 / 117** — socket extraction removes one selected gem intact for the quoted fee.
- [x] **AAA-0118 / 118** — etched rune families persist in an account library and discount future matching work.
- [x] **AAA-0119 / 119** — floors thirty and deeper add gem-focused material seams to authoritative loot rolls.
- [x] **AAA-0120 / 120** — a fixed 500-material contract crafts the chosen eligible Legendary catalog target.
- [x] **AAA-0121 / 121** — the forge header announces the daily discount hour and its live UTC countdown.
- [x] **AAA-0122 / 122** — applicable forge actions expose their quoted combat-rating and gear-score deltas.
- [x] **AAA-0123 / 123** — the last twenty persisted forge actions include cost, outcome, and relative time.
- [x] **AAA-0124 / 124** — Repair All previews its exact combined cost before atomically restoring eligible gear.
- [x] **AAA-0125 / 125** — opt-in Auto Repair quotes and charges its cost before each descent.

## Delivered tranche: economy and market audit

- [x] **AAA-0126 / 126** — the token exchange posts distinct server-authoritative buy and sell rates before either transaction.
- [x] **AAA-0130 / 130** — the Auction House exposes material listings and escrow-backed material buy orders.
- [x] **AAA-0140 / 140** — the live deep-cache jackpot balance is visible before entry and refreshes after authoritative economy responses.
- [x] **AAA-0142 / 142** — a bid placed during an auction's final minute extends its expiry by exactly sixty seconds.
- [x] **AAA-0149 / 149** — material buy orders reserve the maximum spend, fill at or below the posted unit price, and refund unused escrow.

## Delivered tranche: progression and talents audit

- [x] **AAA-0151 / 151** — three named Deep Delver loadout slots save and atomically reapply validated talent-tree allocations.
- [x] **AAA-0152 / 152** — a selected talent node can be refunded with an explicit quote, cascading only when allocated dependents require it.
- [x] **AAA-0154 / 154** — Swiftness shortens combat presentation delay at each purchased rank.
- [x] **AAA-0155 / 155** — Scavenger increases authoritative crafting-material yields by rank.
- [x] **AAA-0156 / 156** — Mercy raises both revive-offer odds and Last Stand healing through server calculations.
- [x] **AAA-0157 / 157** — Cartographer extends paid route visions and lowers their posted cost.
- [x] **AAA-0158 / 158** — Quartermaster adds one server-enforced consumable carry slot per rank.
- [x] **AAA-0159 / 159** — the talent interface is a connected, keyboard-accessible branched tree rather than a flat upgrade list.
- [x] **AAA-0161 / 161** — Delver, Plunderer, and Warden are mutually exclusive specializations with distinct live bonuses.
- [x] **AAA-0163 / 163** — prestige grants a dedicated Paragon point budget spendable only on post-prestige nodes.
- [x] **AAA-0168 / 168** — accumulated boss-family kills buy bounded +1% damage mastery ranks against chosen bestiary families.
- [x] **AAA-0173 / 173** — returning after fourteen days seeds exactly ten authoritative +25% XP and +10% cache catch-up charges.
- [x] **AAA-0175 / 175** — the post-prestige Paragon board grants bounded +0.1% ranks across seven permanent effects.

## Delivered tranche: pact program

- [x] **AAA-0176 / 176** — three account-level named preset slots save canonical pact combinations and restore them safely in the entry planner.
- [x] **AAA-0177 / 177** — each pact tracks authoritative completed runs; at ten completions its own cache bonus improves by 5% on subsequent floors.

## Delivered tranche: challenge pacts

- [x] **AAA-0178 / 178** — Abstinence persists an empty run consumable allowance and pays +15% cache.
- [x] **AAA-0179 / 179** — Pauper rejects entry while any equipped item is above Rare and pays +30% cache.
- [x] **AAA-0180 / 180** — Anemic halves authoritative combat maximum HP and pays +25% cache.
- [x] **AAA-0181 / 181** — Cursed Horde grants every spawned enemy one additional distinct beneficial affix and pays +20% cache.
- [x] **AAA-0182 / 182** — Deep Drums makes every third floor a forced boss across single and queued descents and pays +35% cache.
- [x] **AAA-0183 / 183** — Uninsured rejects cache-insurance purchases for the full run and pays +15% cache.
- [x] **AAA-0184 / 184** — Blind resolves paths server-side while concealing route forecasts and threat UI, paying +10% cache.
- [x] **AAA-0185 / 185** — Brittle applies a second authoritative durability-loss pass after combat and pays +10% cache.
- [x] **AAA-0186 / 186** — Famine removes sanctuary floors from single and queued descents and pays +20% cache.
- [x] **AAA-0187 / 187** — dangerous pact combinations display pre-entry warnings as selections change.

## Delivered tranche: pact and affix campaign

- [x] **AAA-0188 / 188** — a Monday-to-Sunday calendar previews all seven affixes from the same authoritative daily selector used by combat.
- [x] **AAA-0189 / 189** — authenticated players cast one replaceable weekday vote; the deterministic winner locks and becomes Saturday/Sunday's authoritative affix.
- [x] **AAA-0190 / 190** — active pact/affix synergies are called out before entry and included in the authoritative floor-reward multiplier.
- [x] **AAA-0191 / 191** — Mystery Pact uses a cryptographic server-side draw, conceals the concrete pact throughout the active run, pays +40%, and reveals only when the run ends.
- [x] **AAA-0192 / 192** — immutable run pact risk converts into a bounded token grant whose preview and transactional bank commit share the exact quote.
- [x] **AAA-0193 / 193** — bank confirmation itemizes every pact's base, mastery, featured, and synergy contribution plus the exact total multiplier.
- [x] **AAA-0194 / 194** — one deterministic ISO-week featured pact doubles its full pact contribution and is highlighted in the entry planner.
- [x] **AAA-0195 / 195** — every concrete pact awards a distinct, selectable achievement badge on the first transactionally completed run using it.
- [x] **AAA-0196 / 196** — one cryptographic personal affix reroll per UTC day costs ten tokens and is snapshotted authoritatively for the next run.
- [x] **AAA-0197 / 197** — the token-shop Affix Suppressor is consumed transactionally at entry and disables the daily affix for the full run.
- [x] **AAA-0198 / 198** — normalized per-player affix outcomes track authoritative runs, wins, win rate, and average depth across banks and forfeits.
- [x] **AAA-0199 / 199** — Flawless and Checkpoint contract pacts pay +20% while valid, expose live clause state, and transactionally forfeit 25% to the jackpot when broken.
- [x] **AAA-0200 / 200** — authoritative ISO-week banked gold fills a server-wide goal that unlocks a visible +10% cache buff for Saturday and Sunday.

## Delivered tranche: bosses and monsters

- [x] **AAA-0201 / 201** — every named boss opens with a distinct name, title, depth stake, and deterministic mechanic hint before combat.
- [x] **AAA-0202 / 202** — boss timelines emit one round-specific taunt at the first authoritative 50% and 25% health crossings.
- [x] **AAA-0203 / 203** — the four weekly server bosses rotate distinct visible drop tables, with one server-authoritative deterministic reward per player and UTC day.
- [x] **AAA-0204 / 204** — every third depth promotes an elite aura carrier whose rotating Blood, Iron, or Gale hymn visibly empowers the full enemy pack.
- [x] **AAA-0205 / 205** — four named rare elites can invade from depth 6, advertise and escrow a fixed signature relic, and preserve that reward for retry when storage fails.
- [x] **AAA-0206 / 206** — live boss combat exposes the authoritative round-30 enrage threshold in a persistent countdown chip that becomes urgent for the final two rounds.
- [x] **AAA-0207 / 207** — each named boss has a tier-specific fastest-clear board ranking one indexed personal best per player, with authenticated-player highlighting.
- [x] **AAA-0208 / 208** — every defeated boss unlocks a deterministic 30-round, reward-free refight sandbox with balanced, aggressive, and defensive tactic comparisons.
- [x] **AAA-0209 / 209** — boss kills atomically mint dedicated trophies spent through a guarded, all-or-nothing Trophy Quartermaster for materials and Abyss Tokens.
- [x] **AAA-0210 / 210** — every named boss performs distinct summon choreography with themed reinforcements while preserving the live one-round Ultimate interrupt window.
- [x] **AAA-0211 / 211** — below half health, bosses expose server-validated Head and Arms weakpoints for +20% hit damage or one-use spell suppression.
- [x] **AAA-0212 / 212** — each named boss's first authoritative kill unlocks a distinct dated Boss Chronicle in the lore codex while unbeaten entries remain sealed.
- [x] **AAA-0213 / 213** — the shared weekly boss enters a visible Saturday–Sunday UTC raid surge that transactionally doubles both strike damage and immediate material quantity.
- [x] **AAA-0214 / 214** — treasure encounters roll visible Gem, Token, or Key Goblins with guaranteed Prism, token, or atomic in-run vault-key rewards alongside the existing fleeing haul.
- [x] **AAA-0215 / 215** — surviving three actual mimic bites awakens a non-deferable Mimic King showdown whose bounded final bite atomically seals a unique crown and resets the run chain.
- [x] **AAA-0216 / 216** — active delvers see a server-derived Watcher stalking meter count down to the authoritative 15-minute ambush threshold and clearly announce when the next descent is compromised.
- [x] **AAA-0217 / 217** — delvers can replace random combat echoes with a specifically selected co-op bond only while that friend explicitly opts in to sharing their echo.
- [x] **AAA-0218 / 218** — run-bound boss contracts atomically debit a bounded 1/3/5 Boss Token stake and return twice the wager only when the declared natural boss floor is authoritatively defeated.
- [x] **AAA-0219 / 219** — every boss floor beyond depth 60 fields two distinct named Twin Tyrants with a deliberately bounded combined stat budget and increased total experience.
- [x] **AAA-0220 / 220** — a four-element UTC daily boss affinity changes authoritative incoming and outgoing combat multipliers and forecasts its exact counter, trap, target floor, and Twin Tyrant status before descent.
- [x] **AAA-0221 / 221** — immediately before a resolved natural boss floor, a server-quoted gold toll can atomically bypass it for the expected value of its real loot bands while granting no combat rewards or credit.
- [x] **AAA-0222 / 222** — each delver's authoritative deepest-and-fastest boss kill renders as a locally generated, privacy-bounded PNG or text trophy card backed by a matching database index.
- [x] **AAA-0223 / 223** — collecting all ten lore fragments unlocks a restart-safe three-sovereign boss chain whose escalating natural-boss replacements advance only on victory and end in a permanent title.
- [x] **AAA-0224 / 224** — boss kills roll duplicate-safe cosmetic-only mounts and banners at explicit 2/4/7/12% tier rates inside the atomic trophy transaction, with ownership-verified persistent loadouts.
- [x] **AAA-0225 / 225** — after ten confirmed defeats, each named boss permanently learns one deterministic combat trick; the next-boss forecast exposes kill progress and the learned counter before descent.
- [x] **AAA-0226 / 226** — the persisted Pet1/Pet2 equipment slots now form an explicit collar-and-charm loadout whose combined stats apply to active companions only, never the owning delver.
- [x] **AAA-0227 / 227** — formation slots grant independent Pounce and Healing Spell abilities whose cast logs expose exact cooldowns and announce when each companion ability becomes ready again.
- [x] **AAA-0228 / 228** — a successful Mind Control at the three-pet cap creates one restart-safe stable decision, where the owner explicitly recruits, declines, or replaces a chosen companion in a single transaction.

## Delivered tranche: operational quality and client performance

- [x] **AAA-0976 / AB-276** — deterministic ten-week campaigns now apply a visible seasonal palette and triple-weight their matching biome affinity through the replay-safe encounter RNG.
- [x] **AAA-0977 / AB-277** — a dedicated Season tab aggregates weekly cleared floors server-side and grants ten idempotent, cosmetic-only account collectibles through guarded claims.
- [x] **AAA-0978 / AB-278** — a dedicated operator dashboard charts authoritative thirty-day death rates and drops per cleared floor without adding controls to the player page.
- [x] **AAA-0979 / AB-279** — bearer-protected runtime controls update independently locked Abyss subsystem flags and rollout percentages, with strict bounded requests and restart semantics disclosed.
- [x] **AAA-0980 / AB-280** — stable control, treatment, and holdout cohorts apply an integer-exact bounded reward multiplier and expose revision-isolated death and anomaly guardrails.
- [x] **AAA-0981 / AB-281** — terminal live combats transactionally archive their deterministic seed and participant-visible event log, with an owner-authorized bounded portal viewer.
- [x] **AAA-0982 / AB-282** — a versioned anonymous public endpoint exposes only cached global and per-tier aggregates, with CORS, ETag, HEAD, and generic failure handling.
- [x] **AAA-0984 / AB-284** — live snapshots expose the authoritative per-round action-change budget, with proactive warnings and client lockout that preserves the queued or automatic fallback action.
- [x] **AAA-0985 / AB-285** — persisted killer-family counts select the player's most lethal family and mark every matching live enemy as a revenge target.
- [x] **AAA-0986 / AB-286** — compare tier catalog values with the latest database constraints.
- [x] **AAA-0987 / AB-287** — golden-render threshold and active-run page fixtures.
- [x] **AAA-0988 / AB-288** — Playwright exercises enter, victorious descend, bank preview/commit, and the fatal revive/concede decision in CI.
- [x] **AAA-0989 / AB-289** — the production descend HTTP handler admits 100 simultaneous independent delvers inside a bounded CI load gate.
- [x] **AAA-0990 / AB-290** — aggregate enter → floor 5 → bank funnel telemetry.
- [x] **AAA-0991 / AB-291** — an explicit confirmation creates a bounded, manually shared live-run support bundle that excludes names, IDs, logs, and RNG state.
- [x] **AAA-0992 / AB-292** — difficulty values load from a strict embedded JSON catalog or an optional startup-time operator override without recompilation.
- [x] **AAA-0993 / AB-293** — bounded client error reporting.
- [x] **AAA-0994 / AB-294** — combat-log DOM virtualization above 500 lines.
- [x] **AAA-0995 / AB-295** — coalesce HUD chip recomputes to one animation frame.
- [x] **AAA-0996 / AB-296** — compute rarity metadata once and reuse it.
- [x] **AAA-0997 / AB-297** — losing the final live event stream grants one persisted, server-authoritative twenty-second planning lease per participant and round.
- [x] **AAA-0998 / AB-298** — expose per-locale Abyss i18n coverage to operators.
- [x] **AAA-0999 / AB-299** — validate unique gear IDs, set size, and effect descriptions.
- [x] **AAA-1000 / AB-300** — render an in-portal changelog from release-note data.

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

## Delivered tranche: exact stage feedback

- [x] **AAA-0511 / UI-11** — per-floor kill counter reset before every combat playback.
- [x] **AAA-0512 / UI-12** — server-verified OVERKILL signal only above twice the target's remaining HP.
- [x] **AAA-0513 / UI-13** — reduced-motion-safe killing-line emphasis.
- [x] **AAA-0514 / UI-14** — cracked-shield treatment from exact sub-20 durability values.
- [x] **AAA-0515 / UI-15** — threat fill and needle share the existing half-second risk transition.
- [x] **AAA-0516 / UI-16** — reduced-motion-aware numeric depth transition.
- [x] **AAA-0517 / UI-17** — gold ring treatment on each ten-floor milestone.
- [x] **AAA-0518 / UI-18** — active buff badges hydrate and refresh from persisted server-owned remaining-fight durations, including after restart.
- [x] **AAA-0519 / UI-19** — three momentum-flame sizes at zero, five, and ten stacks.
- [x] **AAA-0520 / UI-20** — reduced-motion-safe enrage state driven by combat logs.

## Delivered tranche: authoritative combat log

- [x] **AAA-0521 / UI-21** — log hover targets the matching player, enemy, or pet HP area.
- [x] **AAA-0522 / UI-22** — numbered floor dividers split the log into readable chapters.
- [x] **AAA-0523 / UI-23** — critical-hit lines use heavier type as well as color.
- [x] **AAA-0524 / UI-24** — dodge and miss lines use an italic avoidance treatment.
- [x] **AAA-0525 / UI-25** — exact restored lifesteal HP is logged in green with a plus prefix.
- [x] **AAA-0526 / UI-26** — persisted round-number gutter uses authoritative combat round metadata.
- [x] **AAA-0527 / UI-27** — fights over twenty lines collapse complete middle rounds behind a keyboard-accessible expander.
- [x] **AAA-0528 / UI-28** — inline loot lines reuse manifest rarity borders.
- [x] **AAA-0529 / UI-29** — fatal blows remain pinned until revive resolution or the next fight.
- [x] **AAA-0530 / UI-30** — pet captures receive a reduced-motion-safe target flourish.
- [x] **AAA-0531 / UI-31** — durability-loss lines carry an explicit wrench prefix.
- [x] **AAA-0532 / UI-32** — momentum gain and consumable-break lines show a flame and exact multiplier.
- [x] **AAA-0533 / UI-33** — pity lines render the persisted post-floor counter instead of inferring it from loot HTML.
- [x] **AAA-0534 / UI-34** — every floor section begins with the active daily-affix reminder.
- [x] **AAA-0535 / UI-35** — selected pacts are echoed once with exact reward and danger percentages.
- [x] **AAA-0536 / UI-36** — substring search filters currently visible combat lines.
- [x] **AAA-0537 / UI-37** — plain-text copy is scoped to the most recently played fight.
- [x] **AAA-0538 / UI-38** — persisted monospace mode disables font ligatures.
- [x] **AAA-0539 / UI-39** — post-fight outcome, HP, and loot summaries are announced to screen readers.
- [x] **AAA-0540 / UI-40** — server-marked consecutive DoT ticks collapse independent of the active locale.

## Delivered tranche: authoritative run-awareness HUD

- [x] **AAA-0541 / UI-41** — current escrow pace uses persisted run depth and cache totals.
- [x] **AAA-0542 / UI-42** — positive token deltas accumulate in a session-earned mini-HUD counter.
- [x] **AAA-0543 / UI-43** — run-wide insurance explicitly shows unlimited remaining floor coverage.
- [x] **AAA-0544 / UI-44** — the HUD shows exact distance to the next ten-floor checkpoint.
- [x] **AAA-0545 / UI-45** — a Last Stand bank seal mirrors its remaining floor count into the mini-HUD.
- [x] **AAA-0546 / UI-46** — HP bars carry non-color segment ticks at 25, 50, and 75 percent.
- [x] **AAA-0547 / UI-47** — the threat track uses green, yellow, and red danger bands behind its needle.
- [x] **AAA-0548 / UI-48** — depth-dial rest ticks use the persisted last-sanctuary depth.
- [x] **AAA-0549 / UI-49** — a proximity chip appears within five floors of leaderboard rank ten.
- [x] **AAA-0550 / UI-50** — the active daily bounty is mirrored as a compact progress ring.
- [x] **AAA-0551 / UI-51** — reward focus supports a persisted, server-validated HUD quick switch and automatic mode.
- [x] **AAA-0552 / UI-52** — active pact chips expose exact reward and danger effects to pointer and keyboard users.
- [x] **AAA-0553 / UI-53** — unavailable live items show their cooldown as a radial sweep.
- [x] **AAA-0554 / UI-54** — aggregate equipped durability warns below forty percent.
- [x] **AAA-0555 / UI-55** — Last Stand readiness and exact token cost remain visible before defeat.
- [x] **AAA-0556 / UI-56** — the one-run comeback bonus is visible in the HUD.
- [x] **AAA-0557 / UI-57** — the current server-side deep-cache jackpot is shown in the HUD.
- [x] **AAA-0558 / UI-58** — bank streak displays its exact current multiplier.
- [x] **AAA-0559 / UI-59** — forge happy hour receives a UTC-aware HUD marker.
- [x] **AAA-0560 / UI-60** — interest shows the authoritative per-floor rate and compounded run total.

## Delivered tranche: authoritative run-loot inventory

- [x] **AAA-0561 / UI-61** — escrowed gear carries its canonical slot icon from structured grant data.
- [x] **AAA-0562 / UI-62** — unidentified gear receives a pulsing question badge without parsing localized labels.
- [x] **AAA-0563 / UI-63** — each loot row expands by pointer or keyboard to show source, slot, CR delta, and quality.
- [x] **AAA-0564 / UI-64** — the strongest positive run-loot upgrade per slot can be transactionally reserved to equip on bank.
- [x] **AAA-0565 / UI-65** — existing floor and rarity sorting operates on exact manifest metadata.
- [x] **AAA-0566 / UI-66** — exact gear-ID ownership marks duplicate drops without name matching.
- [x] **AAA-0567 / UI-67** — set-piece tags and equipped counts use canonical SetID fields.
- [x] **AAA-0568 / UI-68** — corrupted drops receive a reduced-motion-safe blood-drip treatment from the persisted corruption flag.
- [x] **AAA-0569 / UI-69** — backpack durability bars use exact current and maximum durability.
- [x] **AAA-0570 / UI-70** — attuned backpack items carry a dedicated binding ribbon.
- [x] **AAA-0571 / UI-71** — quality stars use the persisted zero-to-five masterwork tier.
- [x] **AAA-0572 / UI-72** — backpack filters separate weapons, armor, jewelry, and other items.
- [x] **AAA-0573 / UI-73** — the backpack header totals the server's exact current vendor values.
- [x] **AAA-0574 / UI-74** — equipped temper levels render as compact pip rows.
- [x] **AAA-0575 / UI-75** — equipped sockets render filled and empty gem positions.
- [x] **AAA-0576 / UI-76** — backpack clicks open a side-by-side comparison with the equipped slot.
- [x] **AAA-0577 / UI-77** — negative comparison stats receive an explicit warning treatment.
- [x] **AAA-0578 / UI-78** — equipped and backpack set ribbons use persisted SetID values.
- [x] **AAA-0579 / UI-79** — new run loot glows until its first pointer or keyboard interaction.
- [x] **AAA-0580 / UI-80** — run-loot detail cards retain the authoritative persisted drop floor.

## Delivered tranche: forge decision clarity

- [x] **AAA-0581 / UI-81** — the selected forge item receives a compact exact-metadata summary card.
- [x] **AAA-0582 / UI-82** — equipped and backpack picker groups show their live item counts.
- [x] **AAA-0583 / UI-83** — affordability hints remain advisory while every commit refreshes an exact signed server quote.
- [x] **AAA-0584 / UI-84** — temper pity restores from account-wide server state across browsers and sessions.
- [x] **AAA-0585 / UI-85** — temper surge uses a reduced-motion-safe coin-flip confirmation with both outcomes visible.
- [x] **AAA-0586 / UI-86** — forge confirmations show the selected item's current stat block.
- [x] **AAA-0587 / UI-87** — awakening previews the possible Special pool before confirmation.
- [x] **AAA-0588 / UI-88** — imbue selection includes a live mechanical effect description.
- [x] **AAA-0589 / UI-89** — brand selection previews current and resulting equipped set progress.
- [x] **AAA-0590 / UI-90** — Special swaps render both selected items as mini-cards.
- [x] **AAA-0591 / UI-91** — gear-XP infusion previews the exact CR-derived sacrifice value and milestones.
- [x] **AAA-0592 / UI-92** — prismatic rune preview identifies the exact highest stat and projected value.
- [x] **AAA-0593 / UI-93** — rebalance preview shows the exact source-stat transfer before committing.
- [x] **AAA-0594 / UI-94** — gem transmutation compares persisted before/after gem stats at the retained tier.
- [x] **AAA-0595 / UI-95** — forge history can be filtered by its authoritative action labels.
- [x] **AAA-0596 / UI-96** — history rarity names reuse the standard rarity color language.
- [x] **AAA-0597 / UI-97** — undo readiness and daily usage come from server state rather than local storage.
- [x] **AAA-0598 / UI-98** — artisan reputation shows exact progress to the next discount tier.
- [x] **AAA-0599 / UI-99** — happy-hour status and countdown derive from the server's UTC window.
- [x] **AAA-0600 / UI-100** — material chips explain their canonical acquisition sources by pointer and keyboard.

## Delivered tranche: authoritative entry and navigation

- [x] **AAA-0601 / UI-101** — tier cards show entry fees and unlock requirements together.
- [x] **AAA-0602 / UI-102** — per-tier best depths come from account run history across devices.
- [x] **AAA-0603 / UI-103** — Insanity carries a reduced-motion-safe warning border.
- [x] **AAA-0604 / UI-104** — pact reward totals animate from the previous exact value.
- [x] **AAA-0605 / UI-105** — dangerous pact combinations receive explicit warning icons.
- [x] **AAA-0606 / UI-106** — checkpoint choices state their 0.75 reward multiplier inline.
- [x] **AAA-0607 / UI-107** — Express entry names skipped floors and forfeited pre-record rewards.
- [x] **AAA-0608 / UI-108** — yesterday's runs, wins, deaths, banked gold, and best depth are aggregated from authoritative history.
- [x] **AAA-0609 / UI-109** — entry setup surfaces the live shared jackpot with cache-building context.
- [x] **AAA-0610 / UI-110** — Cursed Bank explains both its 20% upside and three-fight penalty.
- [x] **AAA-0611 / UI-111** — initial reward focus uses icon-led choices with exact mechanical effects.
- [x] **AAA-0612 / UI-112** — last-used tier, route, build, pacts, and focus persist as validated account state.
- [x] **AAA-0613 / UI-113** — daily free-entry presentation follows locked server eligibility rather than browser dates.
- [x] **AAA-0614 / UI-114** — floor-one forecasts use exact equipped CR computed by the server.
- [x] **AAA-0615 / UI-115** — locked tiers visualize progress toward their required best depth.
- [x] **AAA-0616 / UI-116** — section tabs show dismissible unread-content dots.
- [x] **AAA-0617 / UI-117** — the section bar becomes a horizontal scroll strip below 900px.
- [x] **AAA-0618 / UI-118** — panel headers expose stable hover and keyboard deep links.
- [x] **AAA-0619 / UI-119** — a floating back-to-top control appears after two viewports.
- [x] **AAA-0620 / UI-120** — sidebar Armoury/Run Loot ordering is user-configurable and persisted.

## Delivered tranche: codex, history, and feedback clarity

- [x] **AAA-0621 / UI-121** — bestiary species group under server-recorded encounter families with persistent collapse state.
- [x] **AAA-0622 / UI-122** — the lore ring derives its segment count from the authoritative server catalog.
- [x] **AAA-0623 / UI-123** — leaderboard ownership uses authenticated UID identity, including duplicate nicknames, and auto-scrolls to the owned row.
- [x] **AAA-0624 / UI-124** — leaderboard and history headers remain sticky while their results scroll.
- [x] **AAA-0625 / UI-125** — run rows expand by pointer or keyboard into a bounded, server-persisted loot summary with exact total count.
- [x] **AAA-0626 / UI-126** — the server supplies earned state and unlock conditions for the complete achievement catalog.
- [x] **AAA-0627 / UI-127** — badge choices preview inline beside the current nickname before selection.
- [x] **AAA-0628 / UI-128** — section-tab scroll positions persist for the browser session and restore on return.
- [x] **AAA-0629 / UI-129** — print media hides controls and preserves a clean run-record summary.
- [x] **AAA-0630 / UI-130** — leaderboard navigation immediately substitutes shape-matched reduced-motion-safe shimmer rows.
- [x] **AAA-0631 / UI-131** — success, warning, error, and loot feedback use distinct icon categories.
- [x] **AAA-0632 / UI-132** — every stacked toast records and exposes its local creation time on hover.
- [x] **AAA-0633 / UI-133** — error toasts offer a bounded copyable diagnostic containing page, time, and user-agent context.
- [x] **AAA-0634 / UI-134** — paid confirmations state their exact gold, token, or material cost on the primary action.
- [x] **AAA-0635 / UI-135** — concede and prestige confirmations disable their destructive action synchronously for three seconds.
- [x] **AAA-0636 / UI-136** — Escape dismisses only the top consumable/shared-modal layer.
- [x] **AAA-0637 / UI-137** — confirmation focus lands on the safe cancel action while destructive actions remain unavailable.
- [x] **AAA-0638 / UI-138** — the vault count-up offers an immediate reduced-motion-compatible skip to the committed total.
- [x] **AAA-0639 / UI-139** — an observed level increase flashes the HUD level chip without requiring a reload.
- [x] **AAA-0640 / UI-140** — achievement responses raise a named, icon-led, reduced-motion-safe top banner.

## Delivered tranche: authoritative feedback and accessible interaction

- [x] **AAA-0641 / UI-141** — personal and global record bursts use separate, one-per-depth server-confirmed signals.
- [x] **AAA-0642 / UI-142** — the pity flash is emitted only by the exact guaranteed-legendary award branch after escrow succeeds.
- [x] **AAA-0643 / UI-143** — Celestial and Eternal loot toasts receive distinct sound-free starburst treatments.
- [x] **AAA-0644 / UI-144** — downed revive choices receive a reduced-motion-safe heartbeat emphasis.
- [x] **AAA-0645 / UI-145** — the connectivity banner distinguishes blocked and uncertain actions, probes with HEAD only, and requires state review before manual retry; mutations are never replayed.
- [x] **AAA-0646 / UI-146** — enabled primary actions provide a short tactile scale response with keyboard and reduced-motion fallbacks.
- [x] **AAA-0647 / UI-147** — confirmed bank success animates coins from the depth dial toward the gold counter.
- [x] **AAA-0648 / UI-148** — defeat visually drains escrow toward the jackpot without changing authoritative balances.
- [x] **AAA-0649 / UI-149** — an active run waiting five minutes gently marks the document title until the player returns.
- [x] **AAA-0650 / UI-150** — versioned, dismiss-on-visit dots identify newly updated panels once per browser.
- [x] **AAA-0651 / UI-151** — decorative vault and boss overlays are inert and hidden from the accessibility tree.
- [x] **AAA-0652 / UI-152** — an Abyss-only first-tab-stop skip link moves focus directly to run controls.
- [x] **AAA-0653 / UI-153** — combat feeds stop announcing every mutation and post-fight summaries collapse into a throttled dedicated live region.
- [x] **AAA-0654 / UI-154** — rarity presentation adds explicit spoken labels instead of relying on color or dots.
- [x] **AAA-0655 / UI-155** — the depth dial announces current depth, next boss distance, threat, and cache state.
- [x] **AAA-0656 / UI-156** — optional colorblind mode adds a distinct shape for every rarity tier.
- [x] **AAA-0657 / UI-157** — reduced-motion preferences disable dial, counter, feedback, and gesture animations.
- [x] **AAA-0658 / UI-158** — high-contrast mode uses solid panels and brighter muted text.
- [x] **AAA-0659 / UI-159** — deliberate horizontal stage swipes offer confirmed descend or bank actions.
- [x] **AAA-0660 / UI-160** — long-pressing an item opens its detail surface for touch users without hover.

## Delivered tranche: mobile, zoom, and adaptive presentation

- [x] **AAA-0661 / UI-161** — Forge accordions start collapsed on phone-sized viewports to reduce initial overload.
- [x] **AAA-0662 / UI-162** — the mobile tier picker becomes a sticky, accessible select with live risk context.
- [x] **AAA-0663 / UI-163** — consumables form a horizontal, snap-aligned touch belt without trapping page scroll.
- [x] **AAA-0664 / UI-164** — loot and backpack filter chips meet a 36px coarse-pointer target minimum.
- [x] **AAA-0665 / UI-165** — short landscape phones move the combat log beside the stage and restore its DOM position on rotation.
- [x] **AAA-0666 / UI-166** — zoom-responsive controls wrap, shrink safely, and break long labels instead of clipping at 200%.
- [x] **AAA-0667 / UI-167** — backpack rows support arrow navigation, Enter-to-equip, and confirmed Delete-to-salvage.
- [x] **AAA-0668 / UI-168** — closing a run modal restores keyboard focus to the available Descend action.
- [x] **AAA-0669 / UI-169** — light color-scheme users receive readable solid panels, controls, logs, and muted text.
- [x] **AAA-0670 / UI-170** — a screen-reader-only page map explains the run stage and the major navigation sections.

## Delivered tranche: authoritative records and progression insight

- [x] **AAA-0671 / UI-171** — the record panel graphs the last 30 persisted runs rather than the eight visible table rows.
- [x] **AAA-0672 / UI-172** — completed runs persist explicit bank, defeat, failed-revival, concede, and timeout outcomes for the death-cause chart.
- [x] **AAA-0673 / UI-173** — current cache per hour derives from the persisted run start time and authoritative escrow.
- [x] **AAA-0674 / UI-174** — per-tier win-rate chips aggregate the account's complete run history across devices.
- [x] **AAA-0675 / UI-175** — bestiary family bars compare persisted unique encounters against server-defined family targets totaling 50 species.
- [x] **AAA-0676 / UI-176** — weekly Forge ROI uses successful audited before/after snapshots for materials spent and positive CR gained.
- [x] **AAA-0677 / UI-177** — the skill-web header exposes spent, available, and total points and updates all three after mutations.
- [x] **AAA-0678 / UI-178** — prestige preview values come from the server's current prestige state and permanent 5% progression rule.
- [x] **AAA-0679 / UI-179** — tier/result filters drive both the history table and bounded 30-run CSV/JSON exports.
- [x] **AAA-0680 / UI-180** — leaderboard arrows compare current ranks with a server-computed snapshot excluding today's runs, including new entries.

## Delivered tranche: persistent pace, polish, and focused settings

- [x] **AAA-0681 / UI-181** — the pace ghost compares the current persisted run start and cleared-floor count with the previous completed run across devices.
- [x] **AAA-0682 / UI-182** — the seven-day bounty strip is built from authoritative claim rows rather than browser marks.
- [x] **AAA-0683 / UI-183** — the drop-streak widget uses persisted pity, streak, bonus, and the server's guarantee cap.
- [x] **AAA-0684 / UI-184** — the seven-day material chart aggregates persisted Forge sources and sinks.
- [x] **AAA-0685 / UI-185** — milestone nodes are authored from server boss cadence, checkpoint cadence, jackpot depth, tier gates, and best depth.
- [x] **AAA-0686 / UI-186** — an optional depth-graded vignette follows authoritative run depth.
- [x] **AAA-0687 / UI-187** — optional biome ambience follows the server-selected stage biome.
- [x] **AAA-0688 / UI-188** — the Insanity visual treatment activates only for the authoritative Insanity run tier.
- [x] **AAA-0689 / UI-189** — server season labels drive bounded seasonal design variables.
- [x] **AAA-0690 / UI-190** — active runs receive a distinct torch favicon and restore the application icon when idle.
- [x] **AAA-0691 / UI-191** — empty run history includes a friendly illustrated first-run state.
- [x] **AAA-0692 / UI-192** — empty bestiary state explains how the first encounter is recorded.
- [x] **AAA-0693 / UI-193** — terse persistence errors are translated to actionable player-facing copy.
- [x] **AAA-0694 / UI-194** — gold, token, and material values retain consistent currency symbols at decision points.
- [x] **AAA-0695 / UI-195** — a shared pluralizer keeps dynamic floor, item, and drop copy grammatical.
- [x] **AAA-0696 / UI-196** — action labels use consistent sentence casing and verb-first wording.
- [x] **AAA-0697 / UI-197** — emoji-bearing feedback uses a cross-platform fallback font stack.
- [x] **AAA-0698 / UI-198** — completed runs can export a print-friendly local PNG recap from server-confirmed results.
- [x] **AAA-0699 / UI-199** — leaderboard transitions use reduced-motion-safe skeleton rows.
- [x] **AAA-0700 / UI-200** — every persisted presentation preference lives in one searchable settings modal.

## Delivered tranche: core risk and reward protocols

- [x] **AAA-0701 / AB-1** — every ten unbanked floors add 2% interest and remove 2% DEF, capped and exposed as one exact HUD chip.
- [x] **AAA-0702 / AB-2** — caches above 100k apply a capped, server-computed SPD penalty shown with its current percentage.
- [x] **AAA-0703 / AB-3** — banking below 15% HP deducts 5% of the banked cache in both authoritative preview and commit results.
- [x] **AAA-0704 / AB-4** — payouts above 1M require the normalized `BANK` safe word unless the account-level preference is disabled.
- [x] **AAA-0705 / AB-5** — a full bank atomically seeds the next run with 5% of the credited payout and consumes that seed only when entry commits.
- [x] **AAA-0706 / AB-6** — a one-charge Anchor Rune is bought transactionally and guarantees at least half-cache recovery on the next forfeit.
- [x] **AAA-0707 / AB-7** — fixed insurance buttons are replaced by a 10–90% slider whose five-point steps are validated by the server.
- [x] **AAA-0708 / AB-8** — floor idling adds 1% danger per complete minute after the first minute, capped at 50% and surfaced on the HUD.
- [x] **AAA-0709 / AB-9** — Death Wish is a one-fight wager with triple danger, double floor reward, and explicit armed/consumed state.
- [x] **AAA-0710 / AB-10** — each rest floor can atomically compress half the cache into tokens at 70% of the normal exchange value.
- [x] **AAA-0711 / AB-11** — a spent Last Stand unlocks one final charge at triple cost, with both charges enforced by persisted run state.
- [x] **AAA-0712 / AB-12** — each defeat adds 0.1% permanent damage against every unique killer family, capped at 5% per family.
- [x] **AAA-0713 / AB-13** — checkpoint and express starts apply a visible 10% stat penalty that expires after two cleared floors.
- [x] **AAA-0714 / AB-14** — banking exactly on a checkpoint refunds the authoritative token cost paid at entry in the bank transaction.
- [x] **AAA-0715 / AB-15** — combat and event rewards beyond personal best depth gain an additional 3% record-push bonus.
- [x] **AAA-0716 / AB-16** — every full bank contributes 1% to a transactional daily raffle with one entry per banker and a lazy atomic draw.
- [x] **AAA-0717 / AB-17** — the downed screen schedules a server-authoritative five-minute timeout that auto-concedes for a 10% pity cache.
- [x] **AAA-0718 / AB-18** — untouched combat clears build a capped 2%-per-stack defensive momentum variant and damage resets it.
- [x] **AAA-0719 / AB-19** — buying insurance for a premium below 5% of cache value can award the seven-day Cheapskate title.
- [x] **AAA-0720 / AB-20** — banking a run with no recorded incoming or self damage pays 25% extra and awards the Untouchable badge.
- [x] **AAA-0721 / AB-21** — the risk console plots the server soft-cap curve, marks current cache position, and explains 100% versus 25% marginal growth.
- [x] **AAA-0722 / AB-22** — same-day defeats add five points to the revive-offer chance up to +25, reset on success, and show the exact chance.
- [x] **AAA-0723 / AB-23** — the bank confirmation can double the next-run echo seed from 5% to 10% without allowing partial-bank ambiguity.
- [x] **AAA-0724 / AB-24** — Abyss rounds past 30 deal stacking max-HP fatigue to both sides and report each pulse in the combat timeline.
- [x] **AAA-0725 / AB-25** — optional Hybrid runs surge every fifth combat floor to next-tier danger with half of its additional reward multiplier.

## Delivered tranche: remembered encounters and deep-floor markets

- [x] **AAA-0727 / AB-27** — event familiarity resets atomically on entry, adds a capped 10% offer improvement per repeat encounter, and posts the exact bonus.
- [x] **AAA-0728 / AB-28** — Cursed Elevator plates preview both server-sealed destination depths and room types before a route is committed.
- [x] **AAA-0729 / AB-29** — Trap Chamber posts the authoritative DGE-versus-depth pass chance and uses that exact value for settlement.
- [x] **AAA-0731 / AB-31** — depth 41 unlocks five server-validated high-roller tables with tenfold stakes and payouts at unchanged posted odds.
- [x] **AAA-0732 / AB-32** — the Cursed Library accepts a three-fight 15% SPD curse as an alternative to permanent-run HP damage.
- [x] **AAA-0733 / AB-33** — Mirror Floor constructs its hostile clone from the player's current gear, stats, and active skills and posts a pre-fight warning.
- [x] **AAA-0734 / AB-34** — every merchant visit carries one opaque 750g mystery box with enforced 55/28/12/4/1 rarity odds and escrowed loot.
- [x] **AAA-0735 / AB-35** — the first rift vision per run is free, the next two cost progressively more, and a fourth server request is rejected.
- [x] **AAA-0736 / AB-36** — Storm Floor selects a side with the authoritative combat RNG, telegraphs it one full round ahead, and applies a five-percent strike.
- [x] **AAA-0737 / AB-37** — Darkness Floor redacts enemy health from live cards, pixel bars, and serialized replay timelines without mutating combat state.
- [x] **AAA-0741 / AB-41** — a corrupted blood-altar sacrifice doubles the familiar-event-adjusted buff duration, with the committed duration returned to the UI.
- [x] **AAA-0743 / AB-43** — risky alchemy posts its 50% longer duration and 20% backfire odds, then settles ingredients, HP damage, and the floor atomically.
- [x] **AAA-0744 / AB-44** — Unstable Portal advances exactly three depths and reports the three server-derived floor types forfeited by the jump only after commitment.
- [x] **AAA-0745 / AB-45** — Delver Graveyard reveals a persistent ghost identity and death depth before the player chooses honor or plunder.
- [x] **AAA-0748 / AB-48** — wishing wells display and transactionally update the account's lifetime gold contribution.
- [x] **AAA-0750 / AB-50** — mirror choices count once per distinct run; a three-run same-reflection streak empowers the timed buff and is persisted transactionally.

## Delivered tranche: room identities and route intelligence

- [x] **AAA-0730 / AB-30** — lost explorers receive stable encounter identities, with per-player rescue memory and a Cursed Compass keepsake escrowed on the second rescue.
- [x] **AAA-0738 / AB-38** — the Sanctuary Map Table rolls one server-owned event preview and reveals its type only when the scheduled room is within two floors.
- [x] **AAA-0739 / AB-39** — an event-chain ribbon below the depth dial reports authoritative per-run sigils and completed three-sigil chains.
- [x] **AAA-0740 / AB-40** — Cartographer-revealed route choices include a depth-aware expected loot-quality forecast without leaking hidden routes.
- [x] **AAA-0747 / AB-47** — Collapsed Passage offers a risky shortcut while its safe detour still grants a transactionally settled material find.
- [x] **AAA-0749 / AB-49** — Abyssal Garden tracks each material node permanently and grants Green Thumb's +1 yield from the third matching harvest onward.

## Delivered tranche: deferred contracts and combat expression

- [x] **AAA-0726 / AB-26** — event floors can be deferred once, returned to within three cleared floors, and visibly expire when their server-owned deadline passes.
- [x] **AAA-0742 / AB-42** — consecutive Echo Floors preserve the original completed-floor reward and increase the repeat payout from 50% to 75%.
- [x] **AAA-0746 / AB-46** — signed bounties pay on the next combat victory, while the fourth completed contract in one run doubles exactly once.
- [x] **AAA-0751 / AB-51** — consecutive same-element skill casts gain a logged 10% elemental combo bonus, while valid mixed elements retain their stronger reactions.
- [x] **AAA-0752 / AB-52** — skills cast at full mana consume their normal cost and land with a logged 15% overcharge bonus.
- [x] **AAA-0753 / AB-53** — a weak MainHand can swap once to a compatible backpack weapon during a boss fight at the cost of the next action.
- [x] **AAA-0754 / AB-54** — cursed-gear health drain pauses below 20% health and reports Cursed Mercy once per fight.
- [x] **AAA-0755 / AB-55** — Executioner and the daily Execute affix stack with one extra 5% flourish and an explicit combat callout.
- [x] **AAA-0756 / AB-56** — a third parry in one fight grants a full next round of Stealth without leaking into ordinary channel combat.
- [x] **AAA-0757 / AB-57** — the authoritative fight summary reports total reflected Thorns damage when any was dealt.
- [x] **AAA-0758 / AB-58** — live companion commands bind pets to the selected living enemy, with random targeting retained only as a fallback.
- [x] **AAA-0759 / AB-59** — one boss-fight stunbreak restores the turn at 50% effectiveness and remains unavailable on ordinary waves.
- [x] **AAA-0760 / AB-60** — poisoned, bleeding, burning, and other damage-over-time targets receive animated striped health segments on both tactical and pixel bars.
- [x] **AAA-0761 / AB-61** — the live combat cards flash a high-contrast warning when the authoritative log announces boss enrage in two rounds.
- [x] **AAA-0762 / AB-62** — each reward focus supplies its matching small combat bonus, including the documented two-point Gold crit bonus.
- [x] **AAA-0763 / AB-63** — unavailable skills display one exact cooldown pip per remaining round, bounded for compact action slots.
- [x] **AAA-0764 / AB-64** — a persisted Hold Mana control suppresses only automatic normal-wave casts; bosses and explicit live selections still execute.
- [x] **AAA-0765 / AB-65** — the pixel stage carries an ultimate readiness ring with the nearest authoritative cooldown and a ready-state pulse.
- [x] **AAA-0766 / AB-66** — the authoritative fight summary separately reports total parry counter-attack damage.
- [x] **AAA-0767 / AB-67** — three equipped runes matching an attack element reduce its resolved incoming damage by 10% and report the ward once.
- [x] **AAA-0768 / AB-68** — backline attacks gain 8% Backstab damage only against mobs that did not target that delver in the preceding enemy phase.
- [x] **AAA-0769 / AB-69** — combat floors cleared within two rounds grant a persisted next-floor speed chain capped at three stacks.
- [x] **AAA-0770 / AB-70** — bosses telegraph a one-round summon channel that any ultimate fired in that window interrupts before reinforcements spawn.
- [x] **AAA-0771 / AB-71** — defeat responses expose a bounded post-fight survival estimate derived from the completed floor's authoritative risk model.
- [x] **AAA-0772 / AB-72** — an Abyss fumble halves the current hit but grants ten additional crit points to the following non-fumbled strike.
- [x] **AAA-0773 / AB-73** — mind-controlled enemies are captured at one HP with proportional loyalty, warn below ten percent, and gain loyalty after victories.
- [x] **AAA-0774 / AB-74** — captured companions appear as converted allies with health, element, and identity on both tactical and pixel combat stages.
- [x] **AAA-0775 / AB-75** — rounds after 25 add a logged, stacking 5% desperation multiplier to both sides.

## Delivered tranche: loot identity and collection agency

- [x] **AAA-0776 / AB-76** — Sanctuary floors sweep a depth-scaled missed-loot consolation into escrow, preserving the run's bank-or-lose tension.
- [x] **AAA-0777 / AB-77** — the threat tooltip previews next-floor drop bands using the same ordered, bounded probability cascade as the authoritative roller.
- [x] **AAA-0778 / AB-78** — one percent of gear drops become stat-neutral Foil collectibles with an animated shine and explicit label.
- [x] **AAA-0779 / AB-79** — simultaneous cursed and eldritch rolls create a named Doomed variant with a distinct red-black beam treatment.
- [x] **AAA-0780 / AB-80** — named Predator, Warden, and Harvester pieces carry inspectable set-specific lore.
- [x] **AAA-0781 / AB-81** — Insanity loot can roll a Lucid variant that removes negative stats and scales the remaining positive package by 80%.
- [x] **AAA-0782 / AB-82** — punching the fourth socket has a posted ten-percent Perfect result that grants a fifth socket at once.
- [x] **AAA-0783 / AB-83** — equipping three or more gems from one family activates a matching five-percent resonance bonus with a stat-summary note.
- [x] **AAA-0784 / AB-84** — armor-etched elemental wards stack five-percent matching resistance per piece, cap below immunity, and report their resolved reduction.
- [x] **AAA-0785 / AB-85** — corrupted potions and elixirs roll as stronger drops and inflict their advertised five- or ten-percent backlash in lobby and live combat.
- [x] **AAA-0786 / AB-86** — deep-floor gear can gain an “of the Deep” identity and an additional Stamina roll after floor 40.
- [x] **AAA-0787 / AB-87** — an account-persisted opt-in converts third-and-later duplicate Legendary drops into five Umbral Cores.
- [x] **AAA-0788 / AB-88** — loot rows receive rarity-scaled beams from Common through Eternal with reduced-motion support.
- [x] **AAA-0789 / AB-89** — Charm icons use a restrained hanging animation with an explicit reduced-motion fallback.
- [x] **AAA-0790 / AB-90** — run loot can sort by the selected build's authoritative main stat, including Survival's combined HP and DEF value.
- [x] **AAA-0791 / AB-91** — equipped gear held for 30 days gains a derived one-percent positive-stat bonus without mutating its stored roll.
- [x] **AAA-0792 / AB-92** — unidentified drops expose only slot and rarity silhouette information until identification.
- [x] **AAA-0793 / AB-93** — committed Eternal drops and successful Celestial ascensions trigger the sanitized TS3-wide fanfare only after persistence succeeds.
- [x] **AAA-0794 / AB-94** — the run-loot manifest defaults to upgrades-first ordering with deterministic CR and acquisition tie-breaks.
- [x] **AAA-0795 / AB-95** — the HUD separates Legendary guarantee progress from the independent Celestial drop counter.
- [x] **AAA-0796 / AB-96** — Treasure Goblins escrow collectible tokens, and five banked tokens unlock the timed Goblin King title.
- [x] **AAA-0797 / AB-97** — named boss-relic drops display unique lore tied to the defeated boss instead of a generic reward line.
- [x] **AAA-0798 / AB-98** — identical escrowed consumables merge into bounded five-charge stacks instead of overwriting or multiplying rows.
- [x] **AAA-0799 / AB-99** — a weekly Set Trading Post atomically exchanges two true spare duplicates for a rotating piece the account is missing.
- [x] **AAA-0800 / AB-100** — every gear drop label previews its expected rarity-based salvage material yield before banking.

## Delivered tranche: Forge control and crafting depth

- [x] **AAA-0801 / AB-101** — batch temper shows its authoritative minimum-to-maximum cost, then stops at the target, two failures, or insufficient gold.
- [x] **AAA-0802 / AB-102** — a two-core one-shot temper guard converts the next failed single or batch temper on its bound item into a success.
- [x] **AAA-0803 / AB-103** — the Forge executes a validated queue of up to three item actions atomically behind one confirmation.
- [x] **AAA-0804 / AB-104** — bulk gem upgrade advances every eligible socket to a chosen tier using one exact accumulated price.
- [x] **AAA-0805 / AB-105** — rune scraping removes an etched rune and transactionally recovers half of its dust value.
- [x] **AAA-0806 / AB-106** — un-attuning costs 50 tokens, reverses the baked five-percent stat bonus, and restores item mobility.
- [x] **AAA-0807 / AB-107** — same-slot masterwork transfer moves bounded quality at 80% efficiency and snapshots both items.
- [x] **AAA-0808 / AB-108** — corruption has a server-rolled five-percent Perfect outcome that retains the boon without the HP penalty.
- [x] **AAA-0809 / AB-109** — locked reforge preserves one selected stat line at double cost with a persisted daily limit.
- [x] **AAA-0810 / AB-110** — bulk rebalance shifts ten percent of every other positive stat toward one validated target stat.
- [x] **AAA-0811 / AB-111** — two cores remove an item's set brand and return it to the unbranded item pool.
- [x] **AAA-0812 / AB-112** — six cores reroll an item's Special while honoring validated effect exclusions.
- [x] **AAA-0813 / AB-113** — guided awaken persists three distinct server-rolled Specials and commits exactly the player's selected option.
- [x] **AAA-0814 / AB-114** — one prism strips an imbued effect and its corresponding stat bonus from an item.
- [x] **AAA-0815 / AB-115** — polish-all quotes the summed price and upgrades every eligible equipped piece in one transaction.
- [x] **AAA-0816 / AB-116** — the eight-dust Repair Kit II recipe creates an upgraded consumable that restores 50 durability across equipped gear.
- [x] **AAA-0817 / AB-117** — every 50 committed Forge actions grants a global one-percent price discount, capped at five percent and stacked after reputation.
- [x] **AAA-0818 / AB-118** — paid socket relocation preserves every gem while moving the chosen gem to an exact socket index.
- [x] **AAA-0819 / AB-119** — fusion preview exposes the server-selected survivor, projected stats, odds, and costs before consumption.
- [x] **AAA-0820 / AB-120** — spending ten prisms raises Celestial-to-Eternal fusion odds from 25% to 50% in preview and settlement.
- [x] **AAA-0821 / AB-121** — Eternal items receive a second persisted locked-reforge use each UTC day.
- [x] **AAA-0822 / AB-122** — Repair Kit II crafting has a server-rolled five-percent critical result that doubles output and celebrates it.
- [x] **AAA-0823 / AB-123** — up to eight valid discovered recipe favorites persist per account and pin above other recipes.
- [x] **AAA-0824 / AB-124** — the exchange converts bounded quantities upward at 10 dust to shard, 10 shards to core, and 5 cores to prism.
- [x] **AAA-0825 / AB-125** — a permanent Sanctuary purchase unlocks and independently tracks a second daily Forge undo.

## Delivered tranche: Economy, shop, and Auction House

- [x] **AAA-0826 / AB-126** — the Auction House exposes a persisted Insanity-only filter for tier-exclusive gear and preserves it through pagination.
- [x] **AAA-0827 / AB-127** — per-player item watchlists create durable, one-shot market notices surfaced as Auction House toasts.
- [x] **AAA-0828 / AB-128** — one guarded action relists every expired, unbid player listing for seven days at an integer-exact one-percent lower price.
- [x] **AAA-0829 / AB-129** — every listing displays its seller's authoritative completed-sale count as reputation.
- [x] **AAA-0830 / AB-130** — attuned inventory items disable listing and explain their soulbound state and un-attune remedy on hover.
- [x] **AAA-0831 / AB-131** — one deterministic UTC-daily shop item receives an exact forty-percent server-side discount.
- [x] **AAA-0832 / AB-132** — an accessible Insanity cosmetic shop tab rotates one permanent, power-neutral collectible each UTC day.
- [x] **AAA-0833 / AB-133** — bounded gold-to-token bundles start at 100,000 gold and increase by exactly 20,000 per token bought that UTC day.
- [x] **AAA-0834 / AB-134** — an opt-in subscription atomically charges 25,000 gold and delivers two Great Health Potions at most once per UTC day.
- [x] **AAA-0835 / AB-135** — five completed vendor sales of the exact same catalog item unlock a persisted two-percent resale bonus.
- [x] **AAA-0836 / AB-136** — live runs may borrow at most half their original escrow, with a ceiling-rounded ten-percent fee previewed and collected at banking.
- [x] **AAA-0837 / AB-137** — a banked jackpot transactionally transfers exactly ten percent to the run's last distinct co-op helper.
- [x] **AAA-0838 / AB-138** — one bank-free day per ISO week can refund ten percent of that week's recorded Abyss cap tax.
- [x] **AAA-0839 / AB-139** — one insured missed bounty day per ISO week can bridge an otherwise consecutive claim streak.
- [x] **AAA-0840 / AB-140** — the Auction House header shows the authoritative cheapest active Legendary buy-now price.
- [x] **AAA-0841 / AB-141** — completed sales create durable mail-style proceeds notices with gross, fee, and exact net amounts.
- [x] **AAA-0842 / AB-142** — a monthly most-taxed board ranks recorded tax totals and gives its leader the ironic Tax Titan badge.
- [x] **AAA-0843 / AB-143** — material buy orders escrow their full value, settle partial fills atomically, and allow owners to cancel for the exact remainder.
- [x] **AAA-0844 / AB-144** — cryptographically random, recipient-bound, single-use gift codes transfer supported shop consumables without exposing player data.
- [x] **AAA-0845 / AB-145** — new listings at least twenty percent below the active same-item average create durable price-alert toasts for current sellers.
- [x] **AAA-0846 / AB-146** — a seven-day repair subscription covers both combat durability loss and manual full repairs for exactly zero additional gold.
- [x] **AAA-0847 / AB-147** — the once-daily 50-token scratch card enforces its posted 70/20/9/1 reward distribution on the server.
- [x] **AAA-0848 / AB-148** — valid reserved-gold bids placed in the final sixty seconds extend their auction by exactly sixty seconds.
- [x] **AAA-0849 / AB-149** — keyboard-accessible Sold, Listed, and Expired history tabs display authoritative totals and recent item details.
- [x] **AAA-0850 / AB-150** — every new shop and Auction House action discloses its exact price, escrow, fee, or refund formula on hover.

## Delivered tranche: Progression and talent mastery

- [x] **AAA-0851 / AB-151** — three named, server-persisted loadout slots save and atomically apply complete connected skill-web builds.
- [x] **AAA-0852 / AB-152** — a validated keystone swap replaces one allocated keystone while preserving the paid path around it.
- [x] **AAA-0853 / AB-153** — tree search discovers nodes by name, type, description, and normalized stat/effect vocabulary.
- [x] **AAA-0854 / AB-154** — Alt-click Pathfinder computes and highlights the cheapest point-cost route from the allocated web.
- [x] **AAA-0855 / AB-155** — three identical loose or socketed jewels fuse transactionally into one next-tier jewel.
- [x] **AAA-0856 / AB-156** — timeless jewels render their authoritative small, medium, or large effect radius and every affected node.
- [x] **AAA-0857 / AB-157** — the first full respec in each ISO week is free and consumed in the same transaction as the reset.
- [x] **AAA-0858 / AB-158** — newly qualified depth-gated nodes receive a distinct accessible glow without changing their server gate.
- [x] **AAA-0859 / AB-159** — bosses can drop persisted Mastery Shards that transactionally refund one dependent branch for free.
- [x] **AAA-0860 / AB-160** — the first Abyss prestige unlocks a seven-node hexagonal Paragon board with ten points per prestige and exact +0.1% ranks.
- [x] **AAA-0861 / AB-161** — recorded boss-kill counts are cumulatively spent across six family-specific Bestiary talents that add one percent damage per rank without erasing codex records.
- [x] **AAA-0862 / AB-162** — every node tooltip summarizes allocated node, stat, and effect totals for its discipline branch.
- [x] **AAA-0863 / AB-163** — a low-cost minimap mirrors the full web and keeps its clickable viewport rectangle synchronized while panning and zooming.
- [x] **AAA-0864 / AB-164** — Shift-click queues ordered allocations locally and automatically applies each affordable connected prefix atomically as points arrive.
- [x] **AAA-0865 / AB-165** — Victor's Trophy is an achievement-condition node whose allocation stays locked until the depth-25 feat is satisfied.
- [x] **AAA-0866 / AB-166** — Set Resonance activates a matching mini-mastery only for each fully equipped six-piece set and scales four base attributes.
- [x] **AAA-0867 / AB-167** — one deterministic non-aura node rotates daily at half point cost with quote, commit, refund, and plan parity.
- [x] **AAA-0868 / AB-168** — prestige memory preserves one chosen valid node allocation for free across an Abyss prestige reset.
- [x] **AAA-0869 / AB-169** — Shift-hover shows the direct stat and percentage delta for adding a cheapest route or removing a dependent branch.
- [x] **AAA-0870 / AB-170** — versioned, schema-bound Base64 build codes export and import complete builds with strict size and field validation.
- [x] **AAA-0871 / AB-171** — a persisted canvas renderer replaces the SVG's filters, gradients, and animation work while retaining its authoritative hit targets.
- [x] **AAA-0872 / AB-172** — the most recent allocation can be undone for free within sixty seconds, including dependent connectivity cleanup.
- [x] **AAA-0873 / AB-173** — active and cooldown seconds render directly beneath the timed keystone and update without server polling.
- [x] **AAA-0874 / AB-174** — notable, bridge, keystone, and aura nodes use restrained power-tier glow treatments with reduced-motion support.
- [x] **AAA-0875 / AB-175** — a beginner overlay highlights a connected suggested first-ten-node route and can be toggled without mutating the build.

## Delivered tranche: Boss spectacle, companions, and fellowship

- [x] **AAA-0876 / AB-176** — a server-confirmed boss killing blow triggers a short stage execution zoom with a reduced-motion fallback.
- [x] **AAA-0877 / AB-177** — each boss result reports a duration-derived estimated DPS value beside the victory summary.
- [x] **AAA-0878 / AB-178** — Sanctuary practice produces a five-round build log and DPS estimate without changing health, consumables, cooldowns, run state, or rewards.
- [x] **AAA-0879 / AB-179** — each boss intro card loads and saves one device-local strategy note without sending it to the server.
- [x] **AAA-0880 / AB-180** — companion mood derives from health and loyalty, displays an icon, and applies a temporary two-percent combat-stat modifier.
- [x] **AAA-0881 / AB-181** — companion cards show the equipped collar or charm and its exact positive combat stats.
- [x] **AAA-0882 / AB-182** — a locked 1,000-gold training transaction raises the companion's lowest combat stat by at least one percent, capped at three sessions per UTC day.
- [x] **AAA-0883 / AB-183** — prestige two unlocks a server-enforced second active companion formation slot.
- [x] **AAA-0884 / AB-184** — a successful helper floor clear creates a durable notification for the assisting player.
- [x] **AAA-0885 / AB-185** — helper selection exposes persisted assist counts and promotes five-assist pairs to Trusted Ally.
- [x] **AAA-0886 / AB-186** — Graveyard rooms prefer a persistent fallen-player ghost nearest the current player's level.
- [x] **AAA-0887 / AB-187** — the Fellowship death wall lists the player's ten most recent depths, killers, families, and times.
- [x] **AAA-0888 / AB-188** — each ISO week assigns the nearest higher-depth rival and atomically pays tokens only after the target is passed.
- [x] **AAA-0889 / AB-189** — the nearby-depth bank feed requires opt-in from both viewer and publisher before showing a bank event.
- [x] **AAA-0890 / AB-190** — the most lethal recorded enemy family receives a visible Revenge Target mark in tactical and pixel combat views.
- [x] **AAA-0891 / AB-191** — a boss victory retains one authoritative snapshot for each of the final five rounds and exposes a compact kill-cam replay.
- [x] **AAA-0892 / AB-192** — five shared clears unlock a logged two-percent combat-stat bonus for the established duo.
- [x] **AAA-0893 / AB-193** — a defeated companion moves atomically from the living roster into a dated memorial list.
- [x] **AAA-0894 / AB-194** — the weekly server boss has shared locked HP, one contribution per player per UTC day, immediate material loot, and a contributor-wide defeat payout.
- [x] **AAA-0895 / AB-195** — boss combat logs deliver threshold-specific taunts at the first authoritative 50% and 25% health crossings.
- [x] **AAA-0896 / AB-196** — each active healing companion has a persisted, server-enforced auto-heal toggle.
- [x] **AAA-0897 / AB-197** — defeating a named Graveyard echo transfers five percent of its cache reward to the fallen player and posts a durable courtesy notice.
- [x] **AAA-0898 / AB-198** — Insanity combat occasionally appends one deterministic-RNG atmospheric whisper to its authoritative log.
- [x] **AAA-0899 / AB-199** — the Fellowship trophy gallery records and dates the first persisted kill of every distinct boss.
- [x] **AAA-0900 / AB-200** — authenticated bearer links open a read-only live spectator deck whose snapshot strips controls, results, RNG state, and raw combatant IDs.

## Delivered tranche: Stage, combat log, and run HUD

- [x] **AAA-0901 / AB-201** — authoritative HP-frame losses trigger a proportional red edge vignette, with reduced-motion suppression.
- [x] **AAA-0902 / AB-202** — the stage HUD mirrors the lead enemy intent as a normal, special, or heavy icon and accessible description.
- [x] **AAA-0903 / AB-203** — live encounters report the rolling average duration of rounds in the current session only.
- [x] **AAA-0904 / AB-204** — any rendered combat-log line can be pinned as the sticky top line and toggled off again.
- [x] **AAA-0905 / AB-205** — the visible, filtered combat log exports as a self-contained styled HTML file with its color classes preserved.
- [x] **AAA-0906 / AB-206** — the HP bar overlays a clearly labelled next-floor threat forecast without presenting it as guaranteed damage.
- [x] **AAA-0907 / AB-207** — quick-belt consumables support persisted drag ordering by stable consumable ID.
- [x] **AAA-0908 / AB-208** — an explicit HUD edit mode persists chip ordering with stable semantic keys.
- [x] **AAA-0909 / AB-209** — the HUD draws the last twenty server risk readings as a compact accessible threat sparkline.
- [x] **AAA-0910 / AB-210** — cleared-floor client durations feed a session average chip beside the stage.
- [x] **AAA-0911 / AB-211** — the boss HP overlay toggles between exact health and percentage by pointer or keyboard and remembers the preference.
- [x] **AAA-0912 / AB-212** — completed floors receive a cosmetic S/A/B/C tempo grade that has no reward or combat effect.
- [x] **AAA-0913 / AB-213** — boss floors create session-persisted jump bookmarks containing escaped plain-text fight logs.
- [x] **AAA-0914 / AB-214** — multi-wave pips use repeating segment colors to distinguish current and completed waves.
- [x] **AAA-0915 / AB-215** — the HUD shows the primary active companion's name, loyalty heart, and exact loyalty value.
- [x] **AAA-0916 / AB-216** — buffs with one fight remaining receive a gentle blink that is disabled for reduced motion.
- [x] **AAA-0917 / AB-217** — the HP bar tooltip groups observed fight damage into physical, elemental, and damage-over-time estimates.
- [x] **AAA-0918 / AB-218** — long-pressing either Descend control opens a five-floor plan with its authoritative stop conditions stated before confirmation.
- [x] **AAA-0919 / AB-219** — hovering escrow loot desaturates equipped cards outside the matching gear slot.
- [x] **AAA-0920 / AB-220** — right-clicking the boss intro dismisses and makes the card inert immediately.
- [x] **AAA-0921 / AB-221** — the run clock counts visible, recently active play time and pauses after idle or tab hiding.
- [x] **AAA-0922 / AB-222** — live mana bars are segmented by the cheapest currently available spell and report the casts remaining.
- [x] **AAA-0923 / AB-223** — the depth dial tooltip summarizes depth, cleared floors, risk, HP, active time, and average floor time.
- [x] **AAA-0924 / AB-224** — the escrow tooltip separates current gold from item and material-stack counts.
- [x] **AAA-0925 / AB-225** — health below twenty percent receives a slow heartbeat pulse with a reduced-motion fallback.

## Delivered tranche: Bestiary Codex Explorer

- [x] **AAA-0926 / AB-226** — the Bestiary searches discovered monsters by normalized name or encounter family.
- [x] **AAA-0927 / AB-227** — a generated family selector filters only families present in the authoritative codex.
- [x] **AAA-0928 / AB-228** — the codex sorts deterministically by descending kill count with a name tie-breaker.
- [x] **AAA-0929 / AB-229** — an alphabetical sort provides a stable species index.
- [x] **AAA-0930 / AB-230** — a recent-encounter sort uses the server-persisted latest kill timestamp.
- [x] **AAA-0931 / AB-231** — a discovery-order sort uses the immutable first-kill timestamp.
- [x] **AAA-0932 / AB-232** — minimum-kill filters jump directly to tracked, hunted, or mastered entries.
- [x] **AAA-0933 / AB-233** — the explorer reports discovered species against the fifty-species account goal.
- [x] **AAA-0934 / AB-234** — a lifetime total sums authoritative kill counts across every discovered species.
- [x] **AAA-0935 / AB-235** — the summary reports how many species have reached one hundred-kill mastery.
- [x] **AAA-0936 / AB-236** — each encountered family receives a bounded discovery-progress meter and exact count.
- [x] **AAA-0937 / AB-237** — activating a family progress chip applies that family as the current filter.
- [x] **AAA-0938 / AB-238** — every species retains and displays its first authoritative discovery time.
- [x] **AAA-0939 / AB-239** — every kill refreshes a server-owned latest-encounter time without rewriting first discovery.
- [x] **AAA-0940 / AB-240** — eight monotonic kill milestones progress from Discovered through Mythic.
- [x] **AAA-0941 / AB-241** — the inspector shows exact remaining kills and a bounded meter to the next milestone.
- [x] **AAA-0942 / AB-242** — mastered species receive a distinct, power-neutral visual treatment.
- [x] **AAA-0943 / AB-243** — device-local favorites use stable monster names and persist across visits.
- [x] **AAA-0944 / AB-244** — a favorites-only view composes safely with search, family, and kill filters.
- [x] **AAA-0945 / AB-245** — persistent table and responsive card layouts serve dense and visual browsing styles.
- [x] **AAA-0946 / AB-246** — a focused monster inspector exposes family, kills, chronology, mastery, and practice access.
- [x] **AAA-0947 / AB-247** — monster summaries copy as plain text without interpreting player-visible names as markup.
- [x] **AAA-0948 / AB-248** — device-shareable deep links reopen the exact discovered monster inspector.
- [x] **AAA-0949 / AB-249** — the discovered codex exports to bounded structured JSON and a print-focused layout.
- [x] **AAA-0950 / AB-250** — a keyboard-operable two-slot comparison bench reports the exact kill-count delta.

## Delivered tranche: Inventory and panel command tools

- [x] **AAA-0951 / AB-251** — the backpack searches owned gear by normalized item name and slot.
- [x] **AAA-0952 / AB-252** — backpack rows sort by recency, combat rating, name, or equipment slot with deterministic tie-breakers.
- [x] **AAA-0953 / AB-253** — device-local list and responsive grid layouts persist as an explicit inventory preference.
- [x] **AAA-0954 / AB-254** — gear acquired during the last ten minutes receives a server-derived recent-loot marker protected against future timestamps.
- [x] **AAA-0955 / AB-255** — owned gear can be persistently locked through an owner-scoped server endpoint.
- [x] **AAA-0956 / AB-256** — sales, salvage, dismantling, forge consumption, auctions, and set sacrifices all reject persistently locked gear.
- [x] **AAA-0957 / AB-257** — keyboard-focusable backpack rows support explicit multiselection without triggering equip behavior.
- [x] **AAA-0958 / AB-258** — selected unlocked items can be salvaged in one reviewed, server-authoritative batch.
- [x] **AAA-0959 / AB-259** — up to three inventory items remain pinned in a device-local comparison rack.
- [x] **AAA-0960 / AB-260** — comparison cards show exact item stats and percentage deltas against the currently equipped slot.
- [x] **AAA-0961 / AB-261** — the armoury sidebar summarizes non-zero stats across the equipped loadout.
- [x] **AAA-0962 / AB-262** — equipped and progression set cards expose bounded progress toward their next tier.
- [x] **AAA-0963 / AB-263** — scarce crafting materials receive a reduced-motion-safe low-stock warning.
- [x] **AAA-0964 / AB-264** — the Workshop reports discovered recipes against all currently visible recipes.
- [x] **AAA-0965 / AB-265** — the same Workshop summary reports how many known recipes are presently affordable.
- [x] **AAA-0966 / AB-266** — a sticky forge search highlights matching actions and reports their count.
- [x] **AAA-0967 / AB-267** — the five most recently used forge actions persist as device-local jump shortcuts.
- [x] **AAA-0968 / AB-268** — forge controls display immediate affordable or unavailable status dots without overriding server quotes.
- [x] **AAA-0969 / AB-269** — locked achievements expose exact progress and highlight the nearest supported target.
- [x] **AAA-0970 / AB-270** — the Skill Web tab displays the authoritative count of unspent talent points.
- [x] **AAA-0971 / AB-271** — the shop shows the UTC stock-refresh countdown and states that server pricing is authoritative.
- [x] **AAA-0972 / AB-272** — unlocked lore entries support keyboard-operable, device-local read tracking.
- [x] **AAA-0973 / AB-273** — the command page clarifies the UTC daily bounty cadence and warns when buffs expire after the next floor.
- [x] **AAA-0974 / AB-274** — a persisted, reorderable three-consumable next-run shelf feeds the authoritative entry request.
- [x] **AAA-0975 / AB-275** — responsive sidebar opacity and density controls persist independently and respect reduced-motion preferences.

## Delivered tranche: player display controls audit

- [x] **AAA-0353 / 353** — persisted semantic color profiles combine colorblind-safe hues with the existing non-color rarity glyphs.
- [x] **AAA-0354 / 354** — a persisted motion policy can suppress every descendant animation and transition while honoring the system preference.
- [x] **AAA-0372 / 372** — the dynamic page title marks downed and decision-floor states while retaining the existing live depth title.
