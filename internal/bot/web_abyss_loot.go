package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"

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
	if fortune := b.loadAbyssStats(uid).UpFortune; fortune > 0 {
		qualityMult *= 1.0 + float64(fortune)*0.06
		lootFindBonus += float64(fortune) * 0.04
	}
	rareScale := lootRarityScale(mob.Level)

	// Check if user has Lucky Coin equipped
	equipped := b.getEquippedItems(uid)
	hasLuckyCoin := false
	if _, hasCoin := equipped[content.SlotTrinket1]; hasCoin && equipped[content.SlotTrinket1].ID == "ABYSS_LUCKY_COIN" {
		hasLuckyCoin = true
	}

	// Dynamic Scaling: load active run depth
	run := b.loadAbyssRun(uid)
	scale := 1.0
	if run.Active && run.Depth > 0 {
		scale = 1.0 + float64(run.Depth)*0.02 // +2% stats per floor depth
	}

	// Load legendary pity
	var legendaryPity int
	_ = b.DB.QueryRow("SELECT legendary_pity FROM users WHERE client_uid=$1", uid).Scan(&legendaryPity)

	var labels []string
	add := func(label string, g abyssLootGrant) {
		if b.escrowAbyssLoot(uid, label, g) {
			labels = append(labels, label)
		}
	}

	// Bosses and legendaries always seal a guaranteed consumable.
	if mob.Type == content.MobBoss || mob.Type == content.MobLegendary {
		c := content.RandomConsumable()
		add(i18n.T("bot.loot.item", c.Name, c.ID), abyssLootGrant{Type: "cons", ConsID: c.ID, ConsDur: c.Duration})
	}

	// Unique Boss Relics (5% chance)
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
		add(fmt.Sprintf("✨ Unique Relic: %s [Legendary]", relicName), abyssLootGrant{
			Type: "unique", UniqName: relicName, UniqRar: content.RarityLegendary, UniqPow: 15.0,
		})
	}

	for i := 0; i < count; i++ {
		// #nosec G404 - loot rolls are not security-sensitive
		r := rand.Float64() - lootFindBonus

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
			add(fmt.Sprintf("💰 %d gold", gold), abyssLootGrant{Type: "gold", Gold: gold})
			continue
		}

		switch {
		case r < ultimateSkillChance*qualityMult*rareScale:
			us := content.RandomUltimateSkill()
			add(fmt.Sprintf("🌟 Ultimate: %s", us.Name), abyssLootGrant{Type: "ultimate", UltID: us.ID})
		case r < titleChance*qualityMult*rareScale:
			t := content.RandomTitle()
			add(fmt.Sprintf("🏷️ Title: %s", t.Name), abyssLootGrant{Type: "title", TitleName: t.Name, TitleMult: t.XPMultiplier})
		case r < uniqueItemChance*qualityMult*rareScale:
			ui := content.RandomUniqueItem()
			add(fmt.Sprintf("✨ %s [%s]", ui.Name, ui.Rarity.String()), abyssLootGrant{Type: "unique", UniqName: ui.Name, UniqRar: ui.Rarity, UniqPow: ui.Power})
		case r < artifactChance*qualityMult*rareScale:
			a := content.RandomArtifact()
			add(fmt.Sprintf("🔮 Artifact: %s", a.Name), abyssLootGrant{Type: "artifact", ArtName: a.Name, ArtMult: a.Mult, ArtDura: a.MaxDurability})
		case r < enchChance*qualityMult*rareScale:
			ench := content.RandomEnchantment()
			ench.Stats.STR = int(float64(ench.Stats.STR) * zoneDifficulty)
			ench.Stats.SPD = int(float64(ench.Stats.SPD) * zoneDifficulty)
			add(fmt.Sprintf("💠 Enchant: %s", ench.Name), abyssLootGrant{Type: "ench", Ench: &ench})
		case r < skillChance*qualityMult:
			s := content.RandomSkill()
			s.Power *= zoneDifficulty
			add(fmt.Sprintf("📘 Skill: %s", s.Name), abyssLootGrant{Type: "skill", Skill: &s})
		case r < consChance*qualityMult:
			c := content.RandomConsumable()
			add(i18n.T("bot.loot.item", c.Name, c.ID), abyssLootGrant{Type: "cons", ConsID: c.ID, ConsDur: c.Duration})
		case r < gearChance*qualityMult || legendaryPity >= 40:
			var g content.Gear
			if legendaryPity >= 40 {
				g = content.RandomAbyssGearDrop()
				g.Rarity = content.RarityLegendary
				legendaryPity = 0
			} else {
				g = content.RandomGearDrop()
				if rand.Float64() < 0.20 {
					g = content.RandomAbyssGearDrop()
				}
			}

			// Apply Dynamic Item Scaling
			g.Stats.HP = int(float64(g.Stats.HP) * zoneDifficulty * scale)
			g.Stats.STR = int(float64(g.Stats.STR) * zoneDifficulty * scale)
			g.Stats.DEF = int(float64(g.Stats.DEF) * zoneDifficulty * scale)
			g.Stats.SPD = int(float64(g.Stats.SPD) * zoneDifficulty * scale)

			// 20% chance to drop Unidentified
			if rand.Float64() < 0.20 {
				g.Unidentified = true
			}

			// Sockets & Gemstones: Epic+ items roll with 1-3 sockets
			if g.Rarity >= content.RarityEpic {
				g.Sockets = 1 + rand.IntN(3)
			}

			// Eldritch Gear Tier: 5% chance Legendary gear drops as Eldritch (Mythic rarity, +50% stats)
			if g.Rarity == content.RarityLegendary && rand.Float64() < 0.05 {
				g.Rarity = content.RarityMythic
				g.Eldritch = true
				g.Stats = g.Stats.Scaled(1.5)
			}

			// Cursed Weapons: 10% chance Epic+ weapon (MainHand, OffHand, Ranged) drops as Cursed (+50% stats, but -2% HP/turn)
			isWeapon := g.Slot == content.SlotMainHand || g.Slot == content.SlotOffHand || g.Slot == content.SlotRanged
			if isWeapon && g.Rarity >= content.RarityEpic && rand.Float64() < 0.10 {
				g.Cursed = true
				g.Stats = g.Stats.Scaled(1.5)
			}

			// Update legendary pity
			if g.Rarity >= content.RarityLegendary {
				legendaryPity = 0
			} else {
				legendaryPity++
			}

			label := fmt.Sprintf("%s [s:%s] (gs:%d R:[color=%s]%s[/color])", g.Name, string(g.Slot), g.Stats.Score(), g.Rarity.Color(), g.Rarity.String())
			if g.Unidentified {
				label = fmt.Sprintf("Unidentified %s [s:%s] (R:[color=%s]%s[/color])", string(g.Slot), string(g.Slot), g.Rarity.Color(), g.Rarity.String())
			}

			add(label, abyssLootGrant{Type: "gear", Gear: &g})
		default:
			// 100% drop guarantee → a common gear or a small potion.
			if rand.Float64() < 0.7 {
				g := content.RandomStarterGear()
				// Sockets / unidentified checks on common starter gear too
				if rand.Float64() < 0.20 {
					g.Unidentified = true
				}
				legendaryPity++
				label := fmt.Sprintf("%s [s:%s] (%s)", g.Name, string(g.Slot), g.Rarity.String())
				if g.Unidentified {
					label = fmt.Sprintf("Unidentified %s [s:%s] (%s)", string(g.Slot), string(g.Slot), g.Rarity.String())
				}
				add(label, abyssLootGrant{Type: "gear", Gear: &g})
			} else {
				add(i18n.T("bot.loot.small_health_potion"), abyssLootGrant{Type: "cons", ConsID: "P1", ConsDur: 0})
			}
		}
	}

	_, _ = b.DB.Exec("UPDATE users SET legendary_pity=$1 WHERE client_uid=$2", legendaryPity, uid)
	return labels
}

// escrowAbyssLoot persists one rolled drop into the run's loot escrow.
func (b *Bot) escrowAbyssLoot(uid, label string, g abyssLootGrant) bool {
	data, err := json.Marshal(g)
	if err != nil {
		return false
	}
	_, err = b.DB.Exec(
		"INSERT INTO abyss_escrow_loot (client_uid, item_type, label, item_data) VALUES ($1,$2,$3,$4)",
		uid, g.Type, label, data)
	return err == nil
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
	_ = rows.Close()

	var applied []string
	for _, p := range items {
		var g abyssLootGrant
		if err := json.Unmarshal(p.data, &g); err != nil {
			// Corrupt row — delete it so it can't wedge future banks.
			_, _ = b.DB.Exec("DELETE FROM abyss_escrow_loot WHERE id=$1", p.id)
			continue
		}
		b.applyAbyssLootGrant(uid, g)
		// Delete each row as it is applied so a mid-loop failure can't double-grant.
		_, _ = b.DB.Exec("DELETE FROM abyss_escrow_loot WHERE id=$1", p.id)
		applied = append(applied, p.label)
	}
	return applied
}

// applyAbyssLootGrant replays a single escrowed grant through the live granters,
// reusing the same helpers (and their equip/auction/dedupe behaviour) as normal
// loot so escrowed items behave identically once awarded.
func (b *Bot) applyAbyssLootGrant(uid string, g abyssLootGrant) {
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
		_, _ = b.DB.Exec("UPDATE users SET artifact_mult=$2, artifact_name=$3, artifact_durability=$4 WHERE client_uid=$1",
			uid, g.ArtMult, g.ArtName, g.ArtDura)
	case "title":
		_, _ = b.DB.Exec("UPDATE users SET title=$2, title_mult=$3, title_expires=NOW() + INTERVAL '7 days' WHERE client_uid=$1",
			uid, g.TitleName, g.TitleMult)
	case "ultimate":
		b.grantAbyssUltimate(uid, g.UltID)
	case "unique":
		b.grantAbyssUnique(uid, g.UniqName, g.UniqRar, g.UniqPow)
	case "gold":
		if g.Gold > 0 {
			_, _ = b.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", g.Gold, uid)
		}
	}
}

// grantAbyssUltimate awards an ultimate skill, equipping it if the player has none
// and silently dropping exact duplicates (the escrow label already credited it).
func (b *Bot) grantAbyssUltimate(uid, ultID string) {
	if ultID == "" {
		return
	}
	var exists bool
	_ = b.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_ultimate_skills WHERE client_uid=$1 AND ultimate_id=$2)", uid, ultID).Scan(&exists)
	if exists {
		return
	}
	_, _ = b.DB.Exec("INSERT INTO user_ultimate_skills (client_uid, ultimate_id) VALUES ($1, $2)", uid, ultID)
	_, _ = b.DB.Exec("UPDATE users SET ultimate_skills_count = ultimate_skills_count + 1 WHERE client_uid=$1", uid)
	var cur sql.NullString
	_ = b.DB.QueryRow("SELECT ultimate_skill_id FROM users WHERE client_uid=$1", uid).Scan(&cur)
	if !cur.Valid {
		_, _ = b.DB.Exec("UPDATE users SET ultimate_skill_id=$2, ultimate_cooldown=0 WHERE client_uid=$1", uid, ultID)
	}
}

// grantAbyssUnique awards a unique item, ignoring exact duplicates.
func (b *Bot) grantAbyssUnique(uid, name string, rarity content.Rarity, power float64) {
	if name == "" {
		return
	}
	var exists bool
	_ = b.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_unique_items WHERE client_uid=$1 AND item_name=$2)", uid, name).Scan(&exists)
	if exists {
		return
	}
	_, _ = b.DB.Exec("INSERT INTO user_unique_items (client_uid, item_name, rarity, power) VALUES ($1, $2, $3, $4)", uid, name, rarity, power)
	_, _ = b.DB.Exec("UPDATE users SET unique_items_count = unique_items_count + 1 WHERE client_uid=$1", uid)
}
