package bot

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

// Abyss loot escrow
// -----------------------------------------------------------------------------
// Items found inside the Abyss are not granted to the character mid-run. Instead
// each drop is rolled into a per-run escrow (the abyss_escrow_loot table) and held
// there, locked, exactly like the gold cache. Banking the run replays the escrow
// through the live granters; dying (concede / failed revival) discards it.
//
// This roller is intentionally self-contained — it mirrors the category cascade of
// rollLootForUser but writes to escrow instead of applying — so the live
// channel-cycle loot path stays untouched.

// abyssLegendaryPityCap is the number of gear drops without a Legendary that
// forces the next drop to be one (see the pity escalation in rollAbyssLootToEscrow).
const abyssLegendaryPityCap = 40

// Drop-streak consolation: each consecutive floor with no gear drop adds
// abyssDropStreakBonusPerFloor to loot find, capped at abyssDropStreakBonusCap.
const (
	abyssDropStreakBonusPerFloor = 0.02
	abyssDropStreakBonusCap      = 0.30
)

// abyssGearLabel builds the display label for an Abyss gear drop; unidentified
// pieces render as a silhouette (slot icon + rarity hint only, AB-92) so the
// stats stay hidden until the item is identified.
func abyssGearLabel(g content.Gear) string {
	if g.Unidentified {
		return fmt.Sprintf("%s Unidentified %s (R:[color=%s]%s[/color])", content.SlotIcon(g.Slot), string(g.Slot), g.Rarity.Color(), g.Rarity.String())
	}
	label := fmt.Sprintf("%s [s:%s] (gs:%d CR:%.1f R:[color=%s]%s[/color])", g.Name, string(g.Slot), g.Stats.Score(), g.CombatRating(), g.Rarity.Color(), g.Rarity.String())
	// Foil flair (AB-78): cosmetic shine, no stat change.
	if g.Foil {
		label += " [color=#fff59d]✨FOIL[/color]"
	}
	// Doomed (AB-79): cursed+eldritch co-occurrence — red-black "beam-doomed".
	if g.Doomed {
		label += " [color=#b71c1c]☠DOOMED[/color]"
	}
	// Set/item lore tooltip line (AB-80).
	if g.Lore != "" {
		label += fmt.Sprintf(" [color=#8a93a8]❝%s❞[/color]", g.Lore)
	}
	return label
}

// awardCombatLoot routes a defeated mob's drops either to the normal inline loot
// path or, for Abyss combatants, into the run's loot escrow. It is the single
// loot entry point used by the combat engine.
func (b *Bot) awardCombatLoot(winner *UserInCombat, mob content.Mob, zone content.Zone, logs *[]string, loots *[]LootResult) {
	if winner.EscrowLoot {
		for _, label := range b.rollAbyssLootToEscrow(winner.UID, mob, zone.Difficulty, winner.LootFocus) {
			*logs = append(*logs, fmt.Sprintf("[color=#b9a36b]🔒 %s — sealed into the cache (lost if you fall): %s[/color]", winner.Nickname, label))
			*loots = append(*loots, LootResult{UID: winner.UID, Note: label})
		}
		return
	}
	note, poke := b.rollLootForUser(winner.UID, mob, zone.Difficulty, winner.LootFocus)
	if note != "" {
		*logs = append(*logs, i18n.T("bot.combat.looted", winner.Nickname, mob.DisplayName(), note))
		*loots = append(*loots, LootResult{UID: winner.UID, Note: note, Poke: poke})
	}
}

// abyssLootGrant is the serialized payload of a single escrowed drop. Only the
// fields relevant to its Type are populated; the whole struct is stored as JSONB
// and replayed through the live granters when the run is banked.
type abyssLootGrant struct {
	Type      string               `json:"type"` // gear|cons|skill|ultimate|artifact|title|unique|ench|gold
	Gear      *content.Gear        `json:"gear,omitempty"`
	ConsID    string               `json:"cons_id,omitempty"`
	ConsDur   int                  `json:"cons_dur,omitempty"`
	Skill     *content.Skill       `json:"skill,omitempty"`
	Ench      *content.Enchantment `json:"ench,omitempty"`
	ArtName   string               `json:"art_name,omitempty"`
	ArtMult   float64              `json:"art_mult,omitempty"`
	ArtDura   int                  `json:"art_dura,omitempty"`
	UltID     string               `json:"ult_id,omitempty"`
	TitleName string               `json:"title_name,omitempty"`
	TitleMult float64              `json:"title_mult,omitempty"`
	UniqName  string               `json:"uniq_name,omitempty"`
	UniqRar   content.Rarity       `json:"uniq_rar,omitempty"`
	UniqPow   float64              `json:"uniq_pow,omitempty"`
	Gold      int64                `json:"gold,omitempty"`
	MatID     string               `json:"mat_id,omitempty"` // crafting material (#101/#119)
	MatN      int                  `json:"mat_n,omitempty"`
	Tokens    int64                `json:"tokens,omitempty"`
	// GoblinTokens is a treasure-goblin collectible grant (AB-96); 5 banked
	// tokens unlock the "Goblin King" title.
	GoblinTokens int `json:"goblin_tokens,omitempty"`
}

// lootRarityScale dampens high-rarity drop chances for low-level / low-difficulty
// content so early-game (and the shallow Abyss) doesn't rain ultimates and
// artifacts. It ramps from 0.3 at level 1 to 1.0 by level 50. Applied to the
// top-tier loot rolls in both the Abyss roller and the normal loot path.
func lootRarityScale(level int) float64 {
	if level >= 50 {
		return 1.0
	}
	if level < 1 {
		level = 1
	}
	return 0.3 + 0.7*float64(level-1)/49.0
}

// rollAbyssLootToEscrow rolls the drops for one defeated mob and writes them to the
// run's loot escrow, returning the display labels for the combat log.
func (b *Bot) rollAbyssLootToEscrow(uid string, mob content.Mob, zoneDifficulty float64, focus string) []string {
	count := 1
	switch mob.Type {
	case content.MobBoss:
		count = 3
	case content.MobLegendary:
		count = 5
	case content.MobTreasureGoblin:
		count = 2
	}

	qualityMult := zoneDifficulty
	if qualityMult < 1.0 {
		qualityMult = 1.0
	}
	lootFindBonus := 0.0
	if focus == "loot" {
		qualityMult *= 1.2
		lootFindBonus += 0.50
	}
	st := b.loadAbyssStats(uid)
	if st.UpFortune > 0 {
		qualityMult *= 1.0 + float64(st.UpFortune)*0.06
		lootFindBonus += float64(st.UpFortune) * 0.04
	}
	// Depth milestones (#16): +1% permanent loot find per 25 best depth, cap +4%.
	if ms := st.BestDepth / 25; ms > 0 {
		if ms > 4 {
			ms = 4
		}
		lootFindBonus += float64(ms) * 0.01
	}
	// Skill web: Fortune-sector loot_find notables and the Midas keystone;
	// gold_find scales the gold drop rolls below.
	treePct := b.treeBonusFor(uid).Pct
	lootFindBonus += treePct["loot_find"]
	goldFindMult := 1 + treePct["gold_find"]
	rareScale := lootRarityScale(mob.Level)

	// Check if user has Lucky Coin equipped (the coin is Trinket1-only —
	// see its definition in internal/content/artifacts.go).
	equipped := b.getEquippedItems(uid)
	hasLuckyCoin := false
	if it, ok := equipped[content.SlotTrinket1]; ok && it.ID == "ABYSS_LUCKY_COIN" {
		hasLuckyCoin = true
	}

	// Dynamic Scaling: load active run depth
	run := b.loadAbyssRun(uid)
	scale := 1.0
	if run.Active && run.Depth > 0 {
		scale = 1.0 + float64(run.Depth)*0.02 // +2% stats per floor depth
	}

	// Load legendary pity and the drop streak (floors in a row with no gear item,
	// distinct from legendary pity — see abyss_drop_streak).
	var legendaryPity int
	var dropStreak int
	if err := b.DB.QueryRow("SELECT legendary_pity, abyss_drop_streak FROM users WHERE client_uid=$1", uid).Scan(&legendaryPity, &dropStreak); err != nil {
		log.Printf("abyss pity/streak read failed for %s: %v", uid, err)
	}
	if dropStreak > 0 {
		bonus := float64(dropStreak) * abyssDropStreakBonusPerFloor
		if bonus > abyssDropStreakBonusCap {
			bonus = abyssDropStreakBonusCap
		}
		lootFindBonus += bonus
	}
	// Celestial pity counter (AB-95): gear drops since the last Celestial+ piece,
	// tracked in app_meta (display-only; there is no celestial pity cap).
	celestialPity := b.abyssCelestialPity(uid)
	gotGearThisCall := false

	// Duplicate protection: gear rolls in this call retry (capped) to avoid an
	// exact catalog ID the player already owns, equipped or in the backpack.
	// ownedGearCount powers the duplicate-legendary auto-convert (AB-87).
	ownedGear := make(map[string]bool)
	ownedGearCount := make(map[string]int)
	if gearRows, err := b.DB.Query("SELECT gear_id, COUNT(*) FROM (SELECT gear_id FROM user_gear WHERE client_uid=$1 UNION ALL SELECT gear_id FROM user_inventory WHERE client_uid=$1) owned GROUP BY gear_id", uid); err == nil {
		for gearRows.Next() {
			var id string
			var n int
			if gearRows.Scan(&id, &n) == nil {
				ownedGear[id] = true
				ownedGearCount[id] = n
			}
		}
		if err := gearRows.Err(); err != nil {
			log.Printf("abyss owned-gear scan failed for %s: %v", uid, err)
		}
		_ = gearRows.Close()
	}

	var labels []string
	add := func(label string, g abyssLootGrant) bool {
		if b.escrowAbyssLoot(uid, label, g) {
			labels = append(labels, label)
			return true
		}
		return false
	}

	// processGear applies the shared post-roll treatment to a gear drop — dynamic
	// stat scaling (all stats, MNA included), unidentified chance, sockets and the
	// eldritch/cursed affix rolls — and returns its display label. Shared by the
	// forced-legendary pity path and the ordinary gear roll so they stay in sync.
	processGear := func(g content.Gear) (string, content.Gear) {
		g.Stats = g.Stats.Scaled(zoneDifficulty * scale)

		// 20% chance to drop Unidentified
		// #nosec G404 -- non-cryptographic loot roll
		if rand.Float64() < 0.20 {
			g.Unidentified = true
		}

		// Sockets & Gemstones: Epic+ items roll with 1-3 sockets
		if g.Rarity >= content.RarityEpic {
			// #nosec G404 -- non-cryptographic loot roll
			g.Sockets = 1 + rand.IntN(3)
		}

		// Eldritch Gear Tier: 5% chance Legendary gear drops as Eldritch (Mythic rarity, +50% stats)
		// #nosec G404 -- non-cryptographic loot roll
		if g.Rarity == content.RarityLegendary && rand.Float64() < 0.05 {
			g.Rarity = content.RarityMythic
			g.Eldritch = true
			g.Stats = g.Stats.Scaled(1.5)
		}

		// Celestial ascension: 4% of Mythic drops arrive Celestial (+50% stats).
		// Eternal gear never drops — it is forge ascension/fusion only.
		// #nosec G404 -- non-cryptographic loot roll
		if g.Rarity == content.RarityMythic && rand.Float64() < 0.04 {
			g.Rarity = content.RarityCelestial
			g.Stats = g.Stats.Scaled(1.5)
		}

		// Cursed Weapons: 10% chance Epic+ weapon (MainHand, OffHand, Ranged) drops as Cursed (+50% stats, but -2% HP/turn)
		isWeapon := g.Slot == content.SlotMainHand || g.Slot == content.SlotOffHand || g.Slot == content.SlotRanged
		// #nosec G404 -- non-cryptographic loot roll
		if isWeapon && g.Rarity >= content.RarityEpic && rand.Float64() < 0.10 {
			g.Cursed = true
			g.Stats = g.Stats.Scaled(1.5)
		}

		// Corrupted drops (#83): 8% of Epic+ gear lands oversized (+50% offensive
		// stats) but carries an HP malus equal to its score, cleansable at the forge.
		// #nosec G404 -- non-cryptographic loot roll
		if !g.Corrupted && g.Rarity >= content.RarityEpic && rand.Float64() < 0.08 {
			g.Corrupted = true
			g.Stats.STR = g.Stats.STR * 3 / 2
			g.Stats.DEF = g.Stats.DEF * 3 / 2
			g.Stats.SPD = g.Stats.SPD * 3 / 2
			g.CorruptHP = g.Stats.Score()
			g.Stats.HP -= g.CorruptHP
			g.Name = "🩸 Corrupted " + g.Name
		}

		// Doomed items (AB-79): the cursed+eldritch co-occurrence — both
		// drawbacks (−2% HP/round and the eldritch price) on one oversized
		// piece, rendered with the red-black "beam-doomed" beam.
		if g.Cursed && g.Eldritch {
			g.Doomed = true
		}

		// Life-regen affix (real-time Abyss web regen): Rare+ gear can roll a slow
		// out-of-combat heal of 1-5 HP every 5-60s (both rolled). Ticks live on the
		// Abyss dashboard; see (*Bot).applyAbyssRegen.
		// #nosec G404 -- non-cryptographic loot roll
		if g.Rarity >= content.RarityRare && rand.Float64() < 0.20 {
			g.RegenAmount = 1 + rand.IntN(5)       // 1-5 HP
			g.RegenIntervalSec = 5 + rand.IntN(56) // 5-60 seconds
		}

		// Foil variants (AB-78): 1% of drops get a cosmetic animated shine
		// (no stat change, pure brag).
		// #nosec G404 -- non-cryptographic loot roll
		if rand.Float64() < 0.01 {
			g.Foil = true
		}

		// "of the Deep" suffix (AB-86): drops past depth 40 can roll bonus STA.
		// #nosec G404 -- non-cryptographic loot roll
		if run.Active && run.Depth > 40 && rand.Float64() < 0.10 {
			g.Stats.STA += 10 + run.Depth/5
			g.Name += " of the Deep"
		}

		// Acquisition timestamp for the sentimental-value "broken in" bonus
		// (AB-91; the +1% stats are applied by the stat aggregation in xp.go).
		g.FoundAt = time.Now().UTC().Format(time.RFC3339)

		label := abyssGearLabel(g)
		if !g.Unidentified && g.RegenAmount > 0 {
			label += fmt.Sprintf(" [color=#63b3ff]♻ +%d HP/%ds[/color]", g.RegenAmount, g.RegenIntervalSec)
		}
		// Upgrade delta vs the equipped piece in this slot (#76).
		if !g.Unidentified {
			if cur, has := equipped[g.Slot]; has {
				if d := g.CombatRating() - cur.CombatRating(); d > 0 {
					label += fmt.Sprintf(" [color=#41c97a]▲+%.0f CR[/color]", d)
				} else if d < 0 {
					label += fmt.Sprintf(" [color=#8a93a8]▼%.0f CR[/color]", d)
				}
			} else {
				label += " [color=#41c97a]▲ empty slot[/color]"
			}
			// Salvage preview (AB-100): expected material yield for this rarity.
			if mat, n := materialYieldForRarity(g.Rarity); n > 0 {
				label += fmt.Sprintf(" [color=#8a93a8]⚒ ~%s ×%d[/color]", abyssMaterialName(mat), n)
			}
		}
		return label, g
	}

	// Bosses and legendaries always seal a guaranteed consumable.
	if mob.Type == content.MobBoss || mob.Type == content.MobLegendary {
		c := rollAbyssConsumable()
		cl := i18n.T("bot.loot.item", c.Name, c.ID)
		if content.IsCorruptedConsumable(c.ID) {
			cl = "🩸 " + cl // corrupted consumables (AB-85) are red-flagged
		}
		add(cl, abyssLootGrant{Type: "cons", ConsID: c.ID, ConsDur: c.Duration})
	}

	// Deep material seams (#119): from depth 30 the dark bleeds crafting
	// materials — 15% per kill, richer the deeper you are.
	// #nosec G404 -- non-cryptographic loot roll
	if run.Active && run.Depth >= 30 && rand.Float64() < 0.15 {
		mat, n := "shard", 2+rand.IntN(3) // #nosec G404
		if run.Depth >= 50 {
			mat, n = "core", 1+rand.IntN(2) // #nosec G404
		}
		add(fmt.Sprintf("⛏️ Material seam: %s ×%d", abyssMaterialName(mat), n), abyssLootGrant{Type: "mat", MatID: mat, MatN: n})
	}

	// Unique Boss Relics (5% chance)
	// #nosec G404 -- non-cryptographic loot roll
	if mob.Type == content.MobBoss && rand.Float64() < 0.05 {
		var relicName string
		switch mob.Name {
		case "Gorgoroth the Firelord":
			relicName = "Gorgoroth's Obsidian Heart"
		case "Malakor the Voidweaver":
			relicName = "Malakor's Void Conduit"
		case "Azazoth the Slumbering Eye":
			relicName = "Azazoth's Dream Catalyst"
		case "Abyssus, Heart of the Void":
			relicName = "Abyssus's Shattered Core"
		default:
			relicName = mob.Name + "'s Ancient Sigil"
		}
		add(fmt.Sprintf("✨ Unique Relic: %s [Legendary] [color=#8a93a8]❝%s❞[/color]", relicName, abyssBossRelicLore(mob.Name)), abyssLootGrant{
			Type: "unique", UniqName: relicName, UniqRar: content.RarityLegendary, UniqPow: 15.0,
		})
	}

	for i := 0; i < count; i++ {
		// #nosec G404 - loot rolls are not security-sensitive
		r := rand.Float64() - lootFindBonus

		// Legendary pity: once the counter reaches the cap the very next drop is a
		// guaranteed Legendary, resolved *before* every other branch — including the
		// gold-focus / treasure-goblin payout and the ordinary reward switch — so
		// nothing can skip the pity payout. Pity is only reset once the drop is
		// actually escrowed.
		if legendaryPity >= abyssLegendaryPityCap {
			pg := content.RandomAbyssGearDropExcluding(ownedGear)
			pg.Rarity = content.RarityLegendary
			label, g := processGear(pg)
			// Escrow it like any other run drop — even an empty slot — so it stays
			// forfeitable on death. Equipping straight to user_gear here would let the
			// player keep a guaranteed Legendary for free by dying (escrow bypass).
			if add(label, abyssLootGrant{Type: "gear", Gear: &g}) {
				legendaryPity = 0
				if g.Rarity >= content.RarityCelestial {
					celestialPity = 0
				} else {
					celestialPity++
				}
				gotGearThisCall = true
				ownedGear[g.ID] = true // don't re-award this exact item on a later roll
			}
			continue
		}

		// XP focus skips all loot drops in this loot iteration loop
		if focus == "xp" {
			continue
		}

		// Materials focus drops material drops instead of other loot
		if focus == "materials" {
			mat, n := "shard", 3+rand.IntN(4)
			if run.Active && run.Depth >= 50 {
				mat, n = "core", 2+rand.IntN(2)
			}
			add(fmt.Sprintf("⛏️ Material Drop: %s ×%d", abyssMaterialName(mat), n), abyssLootGrant{Type: "mat", MatID: mat, MatN: n})
			continue
		}

		// Tokens focus drops tokens instead of other loot
		if focus == "tokens" {
			tks := int64(1 + rand.IntN(2))
			if mob.Type == content.MobBoss || mob.Type == content.MobLegendary {
				tks = int64(3 + rand.IntN(4))
			}
			add(fmt.Sprintf("🜲 %d Abyss Tokens", tks), abyssLootGrant{Type: "tokens", Tokens: tks})
			continue
		}

		// Gold-focus rolls and treasure goblins pay gold, escrowed like everything else.
		if focus == "gold" || mob.Type == content.MobTreasureGoblin {
			var gold int64
			if mob.Type == content.MobTreasureGoblin {
				gold = int64(1000 + rand.IntN(2000)) // #nosec G404
			} else {
				gold = int64(10 + rand.IntN(mob.RewardXP/2+10)) // #nosec G404
			}
			if hasLuckyCoin {
				gold = int64(float64(gold) * 1.5) // Lucky Coin: +50% gold drop rate
			}
			gold = int64(float64(gold) * goldFindMult) // skill web gold_find
			add(fmt.Sprintf("💰 %d gold", gold), abyssLootGrant{Type: "gold", Gold: gold})
			// Treasure goblins also drop a goblin-token collectible (AB-96);
			// 5 banked tokens unlock the "Goblin King" title at bank time.
			if mob.Type == content.MobTreasureGoblin {
				add(fmt.Sprintf("🪙 Goblin Token (%d/%d banked)", b.abyssGoblinTokens(uid)+1, abyssGoblinTitleAt), abyssLootGrant{Type: "goblin_token", GoblinTokens: 1})
			}
			continue
		}

		// Cumulative loot bands (ascending), built from the shared per-type
		// probabilities (abyssDropForecastData) that also feed the drop-quality
		// forecast (AB-77). These MUST accumulate: the per-type chances
		// are individual probabilities, so used as bare thresholds in a first-match chain
		// the duplicated ones (title==ultimate=0.005, artifact==unique=0.01, gear==cons=0.10)
		// collapse to zero-width bands and gear/title/artifact never drop. Summing them
		// gives each type its own slice: ~0.5/0.5/1/1/2/5/10/10% before the common default.
		fc := abyssDropForecastData(qualityMult, rareScale)
		cUlt := fc.Ultimate
		cTitle := cUlt + fc.Title
		cUniq := cTitle + fc.Unique
		cArt := cUniq + fc.Artifact
		cEnch := cArt + fc.Enchant
		cSkill := cEnch + fc.Skill
		cCons := cSkill + fc.Consumable
		cGear := cCons + fc.Gear
		switch {
		case r < cUlt:
			us := content.RandomUltimateSkill()
			add(fmt.Sprintf("🌟 Ultimate: %s", us.Name), abyssLootGrant{Type: "ultimate", UltID: us.ID})
		case r < cTitle:
			t := content.RandomTitle()
			add(fmt.Sprintf("🏷️ Title: %s", t.Name), abyssLootGrant{Type: "title", TitleName: t.Name, TitleMult: t.XPMultiplier})
		case r < cUniq:
			ui := content.RandomUniqueItem()
			add(fmt.Sprintf("✨ %s [%s]", ui.Name, ui.Rarity.String()), abyssLootGrant{Type: "unique", UniqName: ui.Name, UniqRar: ui.Rarity, UniqPow: ui.Power})
		case r < cArt:
			a := content.RandomArtifact()
			add(fmt.Sprintf("🔮 Artifact: %s", a.Name), abyssLootGrant{Type: "artifact", ArtName: a.Name, ArtMult: a.Mult, ArtDura: a.MaxDurability})
		case r < cEnch:
			ench := content.RandomEnchantment()
			ench.Stats.STR = int(float64(ench.Stats.STR) * zoneDifficulty)
			ench.Stats.SPD = int(float64(ench.Stats.SPD) * zoneDifficulty)
			add(fmt.Sprintf("💠 Enchant: %s (gs:%d)", ench.Name, ench.Stats.Score()), abyssLootGrant{Type: "ench", Ench: &ench})
		case r < cSkill:
			s := content.RandomSkill()
			s.Power *= zoneDifficulty
			add(fmt.Sprintf("📘 Skill: %s (gs:%d)", s.Name, s.Score()), abyssLootGrant{Type: "skill", Skill: &s})
		case r < cCons:
			c := rollAbyssConsumable()
			cl := i18n.T("bot.loot.item", c.Name, c.ID)
			if content.IsCorruptedConsumable(c.ID) {
				cl = "🩸 " + cl // corrupted consumables (AB-85) are red-flagged
			}
			add(cl, abyssLootGrant{Type: "cons", ConsID: c.ID, ConsDur: c.Duration})
		case r < cGear:
			g := content.RandomGearDropExcluding(ownedGear)
			// #nosec G404 -- non-cryptographic loot roll
			if rand.Float64() < 0.20 {
				g = content.RandomAbyssGearDropExcluding(ownedGear)
			}
			// Insanity-tier exclusives: 25% of gear drops in an Insanity run come
			// from the Insanity-only catalog (huge stats, harsh trade-offs).
			insanityDrop := false
			// #nosec G404 -- non-cryptographic loot roll
			if run.Tier == "insanity" && rand.Float64() < 0.25 {
				g = content.RandomInsanityGearDropExcluding(ownedGear)
				insanityDrop = true
			}
			// Lucid variants (AB-81): 10% of Insanity drops lose the negative
			// trade-off stat but scale 20% down.
			// #nosec G404 -- non-cryptographic loot roll
			if insanityDrop && rand.Float64() < 0.10 {
				g = lucidInsanityVariant(g)
			}
			label, g := processGear(g)
			// Escrow every drop (empty slots included) so it stays forfeitable on death.
			// Only touch pity once the drop is actually escrowed, so a failed save
			// can't reset (or skip incrementing) the counter.
			switch {
			// Duplicate-legendary auto-convert (AB-87): with the toggle on, a
			// Legendary+ piece the player already owns twice becomes 5 Umbral
			// Cores instead. Still counts as a legendary drop for pity.
			case g.Rarity >= content.RarityLegendary && b.abyssDupLegendConvert(uid) && ownedGearCount[g.ID] >= 2:
				cvLabel := fmt.Sprintf("♻ Duplicate converted: %s → %s ×5", g.Name, abyssMaterialName("core"))
				if add(cvLabel, abyssLootGrant{Type: "mat", MatID: "core", MatN: 5}) {
					legendaryPity = 0
					celestialPity++
					gotGearThisCall = true
					ownedGear[g.ID] = true
				}
			default:
				if add(label, abyssLootGrant{Type: "gear", Gear: &g}) {
					if g.Rarity >= content.RarityLegendary {
						legendaryPity = 0
					} else {
						legendaryPity++
					}
					if g.Rarity >= content.RarityCelestial {
						celestialPity = 0
					} else {
						celestialPity++
					}
					gotGearThisCall = true
					ownedGear[g.ID] = true // don't re-award this exact item on a later roll
					// Eternal drops push a TS3 channel announcement (AB-93).
					// Eternal gear is forge-ascension-only today, so this is a
					// defensive hook; the ascension path calls the same fanfare.
					if g.Rarity >= content.RarityEternal {
						go b.broadcastAbyssEternalDrop(uid, g.Name)
					}
				}
			}
		default:
			// 100% drop guarantee → a common gear or a small potion.
			// #nosec G404 -- non-cryptographic loot roll
			if rand.Float64() < 0.7 {
				g := content.RandomStarterGear()
				// Sockets / unidentified checks on common starter gear too
				// #nosec G404 -- non-cryptographic loot roll
				if rand.Float64() < 0.20 {
					g.Unidentified = true
				}
				label := abyssGearLabel(g)
				// Escrow it (empty slots included) so it stays forfeitable on death.
				if add(label, abyssLootGrant{Type: "gear", Gear: &g}) {
					legendaryPity++
					gotGearThisCall = true
					ownedGear[g.ID] = true // don't re-award this exact item on a later roll
				}
			} else {
				add(i18n.T("bot.loot.small_health_potion"), abyssLootGrant{Type: "cons", ConsID: "small_health_potion", ConsDur: 0})
			}
		}
	}

	if gotGearThisCall {
		dropStreak = 0
	} else {
		dropStreak++
	}
	if _, err := b.DB.Exec("UPDATE users SET legendary_pity=$1, abyss_drop_streak=$2 WHERE client_uid=$3", legendaryPity, dropStreak, uid); err != nil {
		log.Printf("abyss pity/streak persist failed for %s: %v", uid, err)
	}
	b.abyssSetCelestialPity(uid, celestialPity)
	return labels
}

// escrowAbyssLoot persists one rolled drop into the run's loot escrow.
func (b *Bot) escrowAbyssLoot(uid, label string, g abyssLootGrant) bool {
	data, err := json.Marshal(g)
	if err != nil {
		log.Printf("abyss escrow marshal failed for %s: %v", uid, err)
		return false
	}
	if _, err := b.DB.Exec(
		"INSERT INTO abyss_escrow_loot (client_uid, item_type, label, item_data) VALUES ($1,$2,$3,$4)",
		uid, g.Type, label, data); err != nil {
		log.Printf("abyss escrow insert failed for %s: %v", uid, err)
		return false
	}
	return true
}

// applyAbyssEscrowLoot grants every escrowed item to the character and clears the
// escrow, returning the display labels of what was awarded. Called on bank.
func (b *Bot) applyAbyssEscrowLoot(uid string) []string {
	rows, err := b.DB.Query("SELECT id, label, item_data FROM abyss_escrow_loot WHERE client_uid=$1 ORDER BY id", uid)
	if err != nil {
		return nil
	}
	type pending struct {
		id    int64
		label string
		data  []byte
	}
	// Drain the cursor before issuing the per-item grant writes (which use the same
	// connection pool) to avoid an in-flight query conflict.
	var items []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.label, &p.data); err == nil {
			items = append(items, p)
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("abyss escrow drain failed for %s: %v", uid, err)
	}
	_ = rows.Close()

	var applied []string
	for _, p := range items {
		var g abyssLootGrant
		if err := json.Unmarshal(p.data, &g); err != nil {
			// Corrupt row — delete it so it can't wedge future banks.
			_, _ = b.DB.Exec("DELETE FROM abyss_escrow_loot WHERE id=$1", p.id)
			continue
		}
		if err := b.applyAbyssLootGrant(uid, g); err != nil {
			// Transient write failure — keep the escrow row so a later bank can
			// retry the grant instead of silently losing it.
			continue
		}
		// Delete each row as it is applied so a mid-loop failure can't double-grant.
		_, _ = b.DB.Exec("DELETE FROM abyss_escrow_loot WHERE id=$1", p.id)
		applied = append(applied, p.label)
	}
	return applied
}

// applyAbyssLootGrant replays a single escrowed grant through the live granters,
// reusing the same helpers (and their equip/auction/dedupe behaviour) as normal
// loot so escrowed items behave identically once awarded. A non-nil error means
// the grant did not land and the caller must keep its escrow row.
func (b *Bot) applyAbyssLootGrant(uid string, g abyssLootGrant) error {
	switch g.Type {
	case "gear":
		if g.Gear != nil {
			b.awardGearDrop(uid, *g.Gear)
		}
	case "cons":
		if g.ConsID != "" {
			b.grantConsumable(uid, g.ConsID, g.ConsDur)
		}
	case "skill":
		if g.Skill != nil {
			if _, ok := b.equipSkill(uid, *g.Skill); !ok {
				b.autoListUnwantedItems(uid, *g.Skill)
			}
		}
	case "ench":
		if g.Ench != nil {
			if _, ok := b.applyEnchantment(uid, *g.Ench); !ok {
				b.autoListUnwantedItems(uid, *g.Ench)
			}
		}
	case "artifact":
		if _, err := b.DB.Exec("UPDATE users SET artifact_mult=$2, artifact_name=$3, artifact_durability=$4 WHERE client_uid=$1",
			uid, g.ArtMult, g.ArtName, g.ArtDura); err != nil {
			return err
		}
	case "title":
		res, err := b.DB.Exec("UPDATE users SET title=$2, title_mult=$3, title_expires=NOW() + INTERVAL '7 days', title_source='abyss' WHERE client_uid=$1 AND (title IS NULL OR title_expires < NOW())",
			uid, g.TitleName, g.TitleMult)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			// An unexpired title is already equipped; don't clobber it, and don't lose
			// this drop — return an error so applyAbyssEscrowLoot keeps the escrow row
			// for a later bank (once the current title expires it will finally land).
			return fmt.Errorf("title slot occupied by an active title")
		}
	case "ultimate":
		b.grantAbyssUltimate(uid, g.UltID)
	case "unique":
		b.grantAbyssUnique(uid, g.UniqName, g.UniqRar, g.UniqPow)
	case "gold":
		if g.Gold > 0 {
			if _, err := b.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", g.Gold, uid); err != nil {
				return err
			}
		}
	case "mat":
		if err := b.grantMaterial(uid, g.MatID, g.MatN); err != nil {
			return err
		}
	case "tokens":
		if g.Tokens > 0 {
			if _, err := b.DB.Exec("UPDATE users SET abyss_tokens = abyss_tokens + $1 WHERE client_uid=$2", g.Tokens, uid); err != nil {
				return err
			}
		}
	case "goblin_token":
		if g.GoblinTokens > 0 {
			b.abyssAddGoblinTokens(uid, g.GoblinTokens)
		}
	}
	return nil
}

// grantAbyssUltimate awards an ultimate skill, activating it if the player runs
// fewer than maxActiveUltimates, and silently dropping exact duplicates (the
// escrow label already credited it).
func (b *Bot) grantAbyssUltimate(uid, ultID string) {
	if ultID == "" {
		return
	}
	var exists bool
	if err := b.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_ultimate_skills WHERE client_uid=$1 AND ultimate_id=$2)", uid, ultID).Scan(&exists); err != nil {
		log.Printf("abyss ultimate grant check failed for %s (%s): %v", uid, ultID, err)
		return
	}
	if exists {
		return
	}
	if _, err := b.DB.Exec("INSERT INTO user_ultimate_skills (client_uid, ultimate_id) VALUES ($1, $2)", uid, ultID); err != nil {
		log.Printf("abyss ultimate grant failed for %s (%s): %v", uid, ultID, err)
		return
	}
	if _, err := b.DB.Exec("UPDATE users SET ultimate_skills_count = ultimate_skills_count + 1 WHERE client_uid=$1", uid); err != nil {
		log.Printf("abyss ultimate count update failed for %s (%s): %v", uid, ultID, err)
	}
	_ = b.activateUltimateIfSlotFree(uid, ultID)
}

// grantAbyssUnique awards a unique item, ignoring exact duplicates.
func (b *Bot) grantAbyssUnique(uid, name string, rarity content.Rarity, power float64) {
	if name == "" {
		return
	}
	var exists bool
	if err := b.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_unique_items WHERE client_uid=$1 AND item_name=$2)", uid, name).Scan(&exists); err != nil {
		log.Printf("abyss unique grant check failed for %s (%s): %v", uid, name, err)
		return
	}
	if exists {
		return
	}
	if _, err := b.DB.Exec("INSERT INTO user_unique_items (client_uid, item_name, rarity, power) VALUES ($1, $2, $3, $4)", uid, name, rarity, power); err != nil {
		log.Printf("abyss unique grant failed for %s (%s): %v", uid, name, err)
		return
	}
	if _, err := b.DB.Exec("UPDATE users SET unique_items_count = unique_items_count + 1 WHERE client_uid=$1", uid); err != nil {
		log.Printf("abyss unique count update failed for %s (%s): %v", uid, name, err)
	}
}
