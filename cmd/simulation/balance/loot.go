package main

import (
	"fmt"
	"math/rand"
)

// SimLootDrop represents a single loot drop from combat.
type SimLootDrop struct {
	Type      string // "gear", "skill", "artifact", "consumable", "enchantment", "unique", "ultimate"
	Name      string
	Rarity    SimRarity
	ForPlayer int // player ID who receives it
}

// SimLootSystem handles loot drops and distribution.
type SimLootSystem struct {
	Drops []SimLootDrop
	AH    *SimAuctionHouse // Reference to auction house for listing unwanted loot
}

// RollLootForPlayer generates loot drops for a player after a victorious fight.
// Mirrors bot.rollLootForUser logic.
func (ls *SimLootSystem) RollLootForPlayer(rng *rand.Rand, player *SimPlayer, mobsKilled int, currentFight int, params SimParams) []SimLootDrop {
	var drops []SimLootDrop

	rolls := mobsKilled + 1 // At least 1 roll per fight, more for more kills
	if rolls > 5 {
		rolls = 5
	}

	for roll := 0; roll < rolls; roll++ {
		// Gear drop
		if rng.Float64() < params.GearDropChance {
			g := GenerateGear(rng, player.Level, params)
			if player.EquipGear(g) {
				player.TotalGearDrops++
				player.RecalculateStats(params)
				drops = append(drops, SimLootDrop{
					Type:      "gear",
					Name:      fmt.Sprintf("%s %s", g.Rarity, g.Name),
					Rarity:    g.Rarity,
					ForPlayer: player.ID,
				})
			} else {
				// Not an upgrade — list on AH if rare+
				if g.Rarity >= RarityRare {
					ls.AutoListGear(player, g, currentFight)
				}
			}
		}

		// Skill drop
		if rng.Float64() < params.SkillDropChance {
			skill := GenerateSkill(rng, player.Level)
			if len(player.Skills) < 6 { // Max 6 skills
				player.Skills = append(player.Skills, skill)
				player.TotalSkillDrops++
				drops = append(drops, SimLootDrop{
					Type:      "skill",
					Name:      fmt.Sprintf("%s %s", skill.Rarity, skill.Name),
					Rarity:    skill.Rarity,
					ForPlayer: player.ID,
				})
			}
		}

		// Consumable drop
		if rng.Float64() < params.ConsumableChance {
			cons := GenerateConsumable(rng, player.Level)
			player.Consumables = append(player.Consumables, cons)
			drops = append(drops, SimLootDrop{
				Type:      "consumable",
				Name:      cons.Name,
				Rarity:    RarityCommon,
				ForPlayer: player.ID,
			})
		}

		// Enchantment drop
		if rng.Float64() < params.EnchantmentChance {
			drops = append(drops, SimLootDrop{
				Type:      "enchantment",
				Name:      "Enchantment Shard",
				Rarity:    RollRarity(rng),
				ForPlayer: player.ID,
			})
			// Simplified: enchantments boost a random gear piece's XP mult
			for _, g := range player.Gear {
				if g.Durability > 0 && g.XPMult < 2.0 {
					g.XPMult += 0.05
					break
				}
			}
		}

		// Unique item drop
		if rng.Float64() < params.UniqueItemChance {
			adjectives := []string{"Ancient", "Cursed", "Blessed", "Radiant", "Shadow", "Eternal", "Void", "Celestial", "Infernal", "Frost"}
			nouns := []string{"Blade", "Shield", "Helm", "Armor", "Boots", "Gloves", "Ring", "Amulet", "Staff", "Bow"}
			name := adjectives[rng.Intn(len(adjectives))] + " " + nouns[rng.Intn(len(nouns))]
			if !player.UniqueItems[name] {
				player.UniqueItems[name] = true
				drops = append(drops, SimLootDrop{
					Type:      "unique",
					Name:      name,
					Rarity:    RarityRare,
					ForPlayer: player.ID,
				})
			}
		}

		// Ultimate skill drop
		if rng.Float64() < params.UltimateSkillChance {
			ult := GenerateUltimate(rng, player.Level)
			if player.UltimateSkill == nil || ult.Power > player.UltimateSkill.Power {
				player.UltimateSkill = ult
				drops = append(drops, SimLootDrop{
					Type:      "ultimate",
					Name:      ult.Name,
					Rarity:    ult.Rarity,
					ForPlayer: player.ID,
				})
			}
		}

		// Artifact drop
		if rng.Float64() < params.ArtifactChance {
			drops = append(drops, SimLootDrop{
				Type:      "artifact",
				Name:      "Mysterious Artifact",
				Rarity:    RollRarity(rng),
				ForPlayer: player.ID,
			})
			// Simplified: artifact gives a random item effect
			effects := []SimItemEffect{EffectThorns, EffectVampiric, EffectBerserk, EffectLucky, EffectCleanse, EffectPhoenix}
			player.ItemEffects = append(player.ItemEffects, effects[rng.Intn(len(effects))])
		}
	}

	ls.Drops = append(ls.Drops, drops...)
	return drops
}

// AutoListGear lists a gear item on the auction house if it's rare+.
func (ls *SimLootSystem) AutoListGear(player *SimPlayer, g *SimGear, currentFight int) {
	if ls.AH == nil || g.Rarity < RarityRare {
		return
	}
	ls.AH.AutoListUnwantedGear(player, g, currentFight)
}

// ApplyDurabilityLoss handles post-combat durability loss.
// Mirrors bot.applyDurabilityLoss in xp.go.
func ApplyDurabilityLoss(rng *rand.Rand, player *SimPlayer, defeat bool, params SimParams) {
	lossChance := params.DuraLossChance
	minLoss := params.DuraLossMin
	maxLoss := params.DuraLossMax
	if defeat {
		lossChance = params.DefeatDuraLossChance
		minLoss = params.DefeatDuraLossMin
		maxLoss = params.DefeatDuraLossMax
	}

	// Fragile doubles loss chance
	if player.HasEffect(EffectFragile) {
		lossChance *= 2.0
	}

	var brokenSlots []string

	for slot, g := range player.Gear {
		if g.Durability <= 0 {
			continue
		}
		if rng.Float64() < lossChance {
			itemLoss := minLoss + rng.Intn(maxLoss-minLoss+1)
			// Higher XP mult items lose more
			if g.XPMult > 1.0 {
				itemLoss += int((g.XPMult - 1.0) * 10)
			}
			g.Durability -= itemLoss
			if g.Durability <= 0 {
				brokenSlots = append(brokenSlots, slot)
			}
		}
	}

	// Remove broken gear
	for _, slot := range brokenSlots {
		delete(player.Gear, slot)
	}

	if len(brokenSlots) > 0 {
		player.RecalculateStats(params)
	}

	// Auto-repair: spend gold to repair damaged gear
	for _, g := range player.Gear {
		if g.Durability < g.MaxDur && g.Durability > 0 {
			repairNeeded := g.MaxDur - g.Durability
			repairCost := int64(repairNeeded) * params.RepairCostPerPoint
			if player.Gold >= repairCost {
				player.Gold -= repairCost
				g.Durability = g.MaxDur
			}
		}
	}
}
