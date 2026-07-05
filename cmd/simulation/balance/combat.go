package main

import (
	"fmt"
	"math/rand"
)

// SimCombatResult holds the outcome of a full combat encounter.
type SimCombatResult struct {
	Victory           bool
	Waves             int
	TotalRounds       int
	MobsKilled        int
	XPGained          float64
	XPPenalty         float64
	GoldGained        int64
	PlayerHPRemaining []int
	LootDrops         []SimLootDrop
	DamageDealt       int64
	DamageReceived    int64
}

// combatPlayer is a mutable combat state for a player during a fight.
type combatPlayer struct {
	p           *SimPlayer
	hp          int
	str         int
	def         int
	lastSkillID string
	petHealCD   int
}

// ResolveCombat runs a full multi-wave combat encounter.
// Mirrors bot.resolveChannelCombat in xp.go.
func ResolveCombat(rng *rand.Rand, players []*SimPlayer, avgLvl int, difficulty float64, params SimParams, logs *[]string) SimCombatResult {
	result := SimCombatResult{}

	// Determine number of waves
	waves := 1
	if rng.Float64() < params.Wave2Chance {
		waves = 2
	}
	if rng.Float64() < params.Wave3Chance {
		waves = 3
	}
	result.Waves = waves

	// Roll ambush
	playerStarts := rng.Intn(2) == 0
	if !playerStarts {
		*logs = append(*logs, "⚠️ AMBUSH! Enemies attack first!")
	}

	// Build combat player states
	partyBonus := PartyBonus(len(players))
	cp := make([]combatPlayer, len(players))
	for i, p := range players {
		pm := 1.0 + p.PityStack
		cp[i] = combatPlayer{
			p:   p,
			hp:  p.CurrentHP,
			str: int(float64(p.Stats.STR) * partyBonus * pm),
			def: int(float64(p.Stats.DEF) * partyBonus * pm),
		}
	}

	var allMobs []*SimMob // track for rewards

	for w := 1; w <= waves; w++ {
		var currentMobs []*SimMob

		if w == 1 {
			mobGroup := SpawnMobGroup(rng, avgLvl, difficulty, len(players), params)
			currentMobs = make([]*SimMob, len(mobGroup))
			for i := range mobGroup {
				m := mobGroup[i]
				currentMobs[i] = &m
				allMobs = append(allMobs, currentMobs[i])
			}
		} else {
			*logs = append(*logs, fmt.Sprintf("📢 WAVE %d APPROACHES!", w))
			mobGroup := SpawnMobGroup(rng, avgLvl, difficulty, len(players), params)
			currentMobs = make([]*SimMob, len(mobGroup))
			for i := range mobGroup {
				m := mobGroup[i]
				currentMobs[i] = &m
				allMobs = append(allMobs, currentMobs[i])
			}
		}

		// Reset stunned mobs
		for _, m := range currentMobs {
			if m.Stats.SPD == 0 {
				m.Stats.SPD = 10
			}
		}

		// Fight the wave
		waveVictory := false
		for r := 1; r <= params.MaxRounds; r++ {
			intensify := 1.0 + float64(r-1)*params.IntensifyPerRound
			fatigueMult := 1.0
			if r > params.FatigueStartRound {
				fatigueMult = 1.0 - float64(r-params.FatigueStartRound)*params.FatiguePerRound
				if fatigueMult < params.FatigueFloor {
					fatigueMult = params.FatigueFloor
				}
			}
			healPenalty := 1.0
			if r > 10 {
				healPenalty = 1.0 - float64(r-10)*0.2
				if healPenalty < 0 {
					healPenalty = 0
				}
			}

			// Apply mob effects (poison, regen)
			ApplyMobEffects(currentMobs, r, intensify, logs)

			// Apply zone-like hazard damage (simplified: 10% chance of environmental damage)
			if rng.Float64() < 0.15 {
				hazardDmg := int(float64(avgLvl) * 0.5 * intensify)
				if hazardDmg < 1 {
					hazardDmg = 1
				}
				for i := range cp {
					if cp[i].hp > 0 {
						// Cleanse check
						if cp[i].p.HasEffect(EffectCleanse) {
							continue
						}
						cp[i].hp -= hazardDmg
						if cp[i].hp < 0 {
							cp[i].hp = 0
						}
					}
				}
				for _, m := range currentMobs {
					if m.HP > 0 {
						m.HP -= hazardDmg
					}
				}
			}

			if playerStarts {
				// Player turn
				playerTurn(rng, cp, currentMobs, players, intensify*fatigueMult, healPenalty, params, logs, &result)
				if len(GetAliveMobs(currentMobs)) == 0 {
					waveVictory = true
					break
				}
				// Mob turn
				mobTurn(rng, cp, currentMobs, players, intensify, fatigueMult, params, logs, &result, r)
			} else {
				// Mob turn first
				mobTurn(rng, cp, currentMobs, players, intensify, fatigueMult, params, logs, &result, r)
				// Check if all players dead
				anyAlive := false
				for i := range cp {
					if cp[i].hp > 0 {
						anyAlive = true
						break
					}
				}
				if !anyAlive {
					break
				}
				// Player turn
				playerTurn(rng, cp, currentMobs, players, intensify*fatigueMult, healPenalty, params, logs, &result)
				if len(GetAliveMobs(currentMobs)) == 0 {
					waveVictory = true
					break
				}
			}

			// Cooldown ticks
			for i := range cp {
				if cp[i].hp > 0 && cp[i].p.UltimateSkill != nil && cp[i].p.UltimateSkill.CurrentCooldown > 0 {
					cp[i].p.UltimateSkill.CurrentCooldown--
				}
				if cp[i].petHealCD > 0 {
					cp[i].petHealCD--
				}
			}

			// Check player survival
			anyAlive := false
			for i := range cp {
				if cp[i].hp > 0 {
					anyAlive = true
					break
				}
			}
			if !anyAlive {
				break
			}

			result.TotalRounds = r
		}

		if !waveVictory {
			result.Victory = false
			break
		}
		if w == waves {
			result.Victory = true
		}
	}

	// Count mobs killed
	for _, m := range allMobs {
		if m.HP <= 0 {
			result.MobsKilled++
		}
	}

	// Calculate rewards
	totalRewardXP := 0
	totalRewardGold := int64(0)
	for _, m := range allMobs {
		totalRewardXP += m.RewardXP
		totalRewardGold += m.RewardGold
	}

	if result.Victory {
		result.XPGained = float64(totalRewardXP) / float64(len(players))
		result.GoldGained = totalRewardGold / int64(len(players))
	} else {
		result.XPPenalty = (float64(totalRewardXP) / float64(len(players))) * params.DeathXPPenalty
	}

	// Update player HP and stats
	for i, p := range players {
		p.CurrentHP = cp[i].hp
		result.PlayerHPRemaining = append(result.PlayerHPRemaining, cp[i].hp)

		if result.Victory {
			p.ConsecutiveWins++
			p.ConsecutiveLosses = 0
			p.PityStack = 0
			p.TotalWins++
			p.Gold += result.GoldGained
			p.TotalGoldEarned += result.GoldGained
		} else {
			p.ConsecutiveWins = 0
			p.ConsecutiveLosses++
			p.PityStack = clampFloat(p.PityStack+params.PityPerLoss, 0, params.PityCap)
			p.TotalLosses++
			p.CurrentHP = 0
		}
		p.TotalFights++
	}

	// Recalculate stats to undo any temporary combat mutations (e.g., DeathCurse)
	for _, p := range players {
		savedHP := p.CurrentHP
		p.RecalculateStats(params)
		p.CurrentHP = savedHP
	}

	return result
}

// playerTurn resolves all player actions for one round.
// Mirrors bot.userTurn in xp.go.
func playerTurn(rng *rand.Rand, cp []combatPlayer, mobs []*SimMob, players []*SimPlayer, intensify, healPenalty float64, params SimParams, logs *[]string, result *SimCombatResult) {
	for i := range cp {
		if cp[i].hp <= 0 {
			continue
		}

		p := cp[i].p

		// Auto-use healing consumable if HP < 50%
		if cp[i].hp < p.MaxHP/2 {
			for ci, c := range p.Consumables {
				if c.Type == ConsumableHealing {
					healAmt := int(float64(p.MaxHP) * c.EffectValue)
					cp[i].hp += healAmt
					if cp[i].hp > p.MaxHP {
						cp[i].hp = p.MaxHP
					}
					*logs = append(*logs, fmt.Sprintf("🧪 %s used %s: +%d HP!", p.PlayerDisplay(), c.Name, healAmt))
					p.Consumables[ci].Remaining--
					if p.Consumables[ci].Remaining <= 0 {
						p.Consumables = append(p.Consumables[:ci], p.Consumables[ci+1:]...)
					}
					break
				}
			}
		}

		// Compute effective STR
		uSTR := cp[i].str

		// Momentum: 10% chance for 10% STR boost
		if rng.Float64() < params.MomentumChance {
			uSTR = int(float64(uSTR) * 1.1)
		}

		// Lifesteal and multi-strike from effects
		lifesteal := p.Lifesteal()
		multiStrike := p.MultiStrike()
		extraHits := 1

		if multiStrike > 0 && rng.Intn(100) < multiStrike {
			extraHits = 2
			*logs = append(*logs, fmt.Sprintf("⚔️ %s double attack!", p.PlayerDisplay()))
		}

		for h := 0; h < extraHits; h++ {
			aliveMobs := GetAliveMobs(mobs)
			if len(aliveMobs) == 0 {
				break
			}
			target := aliveMobs[rng.Intn(len(aliveMobs))]

			dmgMult := 1.0
			ignoreDef := 0.0

			// Berserk: +20% damage when below half HP
			if p.HasEffect(EffectBerserk) && cp[i].hp < p.MaxHP/2 {
				dmgMult += 0.2
			}
			// Fragile: +30% damage
			if p.HasEffect(EffectFragile) {
				dmgMult += 0.3
			}

			var dmg int

			// Skill usage: 30% chance
			if len(p.Skills) > 0 && rng.Float64() < params.SkillProcChance {
				skill := p.Skills[rng.Intn(len(p.Skills))]
				dmgMult *= skill.Power
				ignoreDef = skill.IgnoreDef

				// Combo bonus
				comboBonus := 1.0
				if cp[i].lastSkillID != "" && cp[i].lastSkillID == skill.ID {
					comboBonus = params.ComboBonus
					dmgMult *= comboBonus
				}
				cp[i].lastSkillID = skill.ID

				effDef := float64(ComputeEffectiveMobDEF(target)) * (1.0 - ignoreDef)
				dmg = int((float64(uSTR)*dmgMult - effDef) * intensify)
				minDmg := int(float64(uSTR) * 0.15 * intensify)
				if dmg < minDmg {
					dmg = minDmg
				}
				if dmg < 1 {
					dmg = 1
				}

				skillMsg := fmt.Sprintf("✨ %s: %s deals %d dmg", p.PlayerDisplay(), skill.Name, dmg)
				if comboBonus > 1.0 {
					skillMsg += " (COMBO!)"
				}
				*logs = append(*logs, skillMsg)

				// Stun
				if skill.StunChance > 0 && rng.Float64() < skill.StunChance {
					*logs = append(*logs, fmt.Sprintf("💫 %s STUNNED!", target.Name))
					target.Stats.SPD = 0
				}

				// Skill heal
				if skill.HealPct > 0 {
					heal := int(float64(p.MaxHP) * skill.HealPct * healPenalty)
					if heal > 0 {
						cp[i].hp += heal
						if cp[i].hp > p.MaxHP {
							cp[i].hp = p.MaxHP
						}
					}
				}

				target.HP -= dmg
				result.DamageDealt += int64(dmg)
			} else {
				cp[i].lastSkillID = "" // Reset combo

				// Regular attack with element
				userElement := p.Element
				if mh, ok := p.Gear["MainHand"]; ok {
					userElement = mh.Element
				}
				elementMult := GetElementMult(userElement, target.Element)
				dmgMult *= elementMult

				// Backline bonus
				if p.Position == PositionBackline {
					dmgMult *= 1.10
				}

				// Ultimate skill
				if p.UltimateSkill != nil && p.UltimateSkill.CurrentCooldown == 0 {
					ultMult := p.UltimateSkill.Power
					if bonus := p.TreeBonus.Pct["ult_damage"]; bonus > 0 {
						ultMult *= (1.0 + bonus)
					}
					dmgMult *= ultMult
					effDef := float64(ComputeEffectiveMobDEF(target)) * (1.0 - ignoreDef)
					dmg = int((float64(uSTR)*dmgMult - effDef) * intensify)
					minDmg := int(float64(uSTR) * 0.15 * intensify)
					if dmg < minDmg {
						dmg = minDmg
					}
					if dmg < 1 {
						dmg = 1
					}
					*logs = append(*logs, fmt.Sprintf("🌟 ULTIMATE: %s deals %d dmg!", p.UltimateSkill.Name, dmg))
					
					cooldownVal := p.UltimateSkill.CooldownRounds
					if red := p.TreeBonus.Pct["ult_cooldown"]; red > 0 {
						cooldownVal = int(float64(cooldownVal) * (1.0 - red))
						if cooldownVal < 2 {
							cooldownVal = 2
						}
					}
					p.UltimateSkill.CurrentCooldown = cooldownVal
				} else {
					effDef := float64(ComputeEffectiveMobDEF(target)) * (1.0 - ignoreDef)
					dmg = int((float64(uSTR)*dmgMult - effDef) * intensify)
					minDmg := int(float64(uSTR) * 0.15 * intensify)
					if dmg < minDmg {
						dmg = minDmg
					}
					if dmg < 1 {
						dmg = 1
					}
				}

				target.HP -= dmg
				result.DamageDealt += int64(dmg)
			}

			// Chain attack for groups of 3+
			if len(players) >= 3 && rng.Float64() < params.ChainAttackChance {
				others := GetAliveMobs(mobs)
				if len(others) > 1 {
					var chainTarget *SimMob
					for _, xm := range others {
						if xm != target {
							chainTarget = xm
							break
						}
					}
					if chainTarget != nil {
						chainDmg := int(float64(uSTR) * params.ChainDamagePct * intensify)
						if chainDmg < 1 {
							chainDmg = 1
						}
						chainTarget.HP -= chainDmg
						result.DamageDealt += int64(chainDmg)
					}
				}
			}

			// Mind control: capture mob below 20% threshold
			mindControlLvl := p.MindControlLevel()
			if mindControlLvl > 0 && len(p.Pets) < mindControlLvl && target.HP > 0 &&
				float64(target.HP) < 0.2*float64(target.MaxHP) {
				if rng.Float64() < 0.5 {
					*logs = append(*logs, fmt.Sprintf("🌀 Captive: %s!", target.Name))
					p.Pets = append(p.Pets, &SimPet{
						Name:  target.Name,
						Level: target.Level,
						Stats: SimStats{HP: target.Level * 10, STR: target.Stats.STR / 2, DEF: target.Stats.DEF / 2, SPD: target.Stats.SPD},
					})
					target.HP = 0 // Remove from combat
				}
			}

			// Lifesteal
			if lifesteal > 0 && dmg > 0 {
				heal := int(float64(dmg) * float64(lifesteal) / 100.0 * healPenalty)
				if heal > 0 {
					cp[i].hp += heal
					if cp[i].hp > p.MaxHP {
						cp[i].hp = p.MaxHP
					}
				}
			}

			// Check mob death
			if target.HP <= 0 {
				*logs = append(*logs, fmt.Sprintf("☠️ %s defeated!", target.Name))
				ApplyDeathEffects(target, &mobs, players, avgLevel(players), 1.0, rng, params, logs)
			}

			if len(GetAliveMobs(mobs)) == 0 {
				break
			}
		}

		// Pet attacks
		for petIdx, pet := range p.Pets {
			if pet.Stats.HP <= 0 {
				continue
			}
			aliveMobs := GetAliveMobs(mobs)
			if len(aliveMobs) == 0 {
				break
			}

			// Betrayal chance
			betrayalChance := 0.03
			if red := p.TreeBonus.Pct["pet_betrayal_reduce"]; red > 0 {
				betrayalChance -= red
				if betrayalChance < 0 {
					betrayalChance = 0
				}
			}
			if rng.Float64() < betrayalChance {
				targetIdx := rng.Intn(len(cp))
				if cp[targetIdx].hp > 0 {
					pdmg := maxInt(1, pet.Stats.STR-cp[targetIdx].def)
					cp[targetIdx].hp -= pdmg
					if cp[targetIdx].hp < 0 {
						cp[targetIdx].hp = 0
					}
					*logs = append(*logs, fmt.Sprintf("⚠️ Rogue Pet %s bit %s for %d!", pet.Name, cp[targetIdx].p.PlayerDisplay(), pdmg))
					result.DamageReceived += int64(pdmg)
				}
				continue
			}

			// pet2 healspell logic
			if petIdx == 1 && cp[i].petHealCD == 0 {
				var bestTarget *combatPlayer
				lowestHPPct := 1.0
				for k := range cp {
					if cp[k].hp > 0 && cp[k].hp < cp[k].p.MaxHP {
						pct := float64(cp[k].hp) / float64(cp[k].p.MaxHP)
						if pct < lowestHPPct {
							lowestHPPct = pct
							bestTarget = &cp[k]
						}
					}
				}
				if bestTarget != nil {
					healAmt := int(float64(bestTarget.p.MaxHP) * 0.15) + pet.Level * 3
					if healAmt < 10 {
						healAmt = 10
					}
					bestTarget.hp += healAmt
					if bestTarget.hp > bestTarget.p.MaxHP {
						healAmt -= (bestTarget.hp - bestTarget.p.MaxHP)
						bestTarget.hp = bestTarget.p.MaxHP
					}
					cp[i].petHealCD = 2
					*logs = append(*logs, fmt.Sprintf("✨ %s's Pet %s casts a Healing Spell on %s, restoring %d HP! (2-round cooldown)", p.PlayerDisplay(), pet.Name, bestTarget.p.PlayerDisplay(), healAmt))
					continue
				}
			}

			targetMob := aliveMobs[rng.Intn(len(aliveMobs))]
			petDmgMult := 1.0
			if bonus := p.TreeBonus.Pct["pet_damage_pct"]; bonus > 0 {
				petDmgMult += bonus
			}
			baseDmg := maxInt(1, pet.Stats.STR-ComputeEffectiveMobDEF(targetMob))
			pdmg := maxInt(1, int(float64(baseDmg)*petDmgMult*intensify))
			targetMob.HP -= pdmg
			result.DamageDealt += int64(pdmg)
			if targetMob.HP <= 0 {
				*logs = append(*logs, fmt.Sprintf("🐾 %s killed %s!", pet.Name, targetMob.Name))
				ApplyDeathEffects(targetMob, &mobs, players, avgLevel(players), 1.0, rng, params, logs)
			}
		}

		if len(GetAliveMobs(mobs)) == 0 {
			break
		}
	}
}

// mobTurn resolves all mob actions for one round.
// Mirrors bot.mobTurn in xp.go.
func mobTurn(rng *rand.Rand, cp []combatPlayer, mobs []*SimMob, _ []*SimPlayer, intensify, fatigueMult float64, params SimParams, logs *[]string, result *SimCombatResult, round int) {
	for _, m := range mobs {
		if m.HP <= 0 {
			continue
		}
		// Stunned check
		if m.Stats.SPD == 0 {
			m.Stats.SPD = 10 // recover next round
			continue
		}

		// Target selection: prioritize frontline
		var potentialTargets []int
		for i := range cp {
			if cp[i].hp > 0 && cp[i].p.Position == PositionFrontline {
				potentialTargets = append(potentialTargets, i)
			}
		}
		if len(potentialTargets) == 0 {
			for i := range cp {
				if cp[i].hp > 0 {
					potentialTargets = append(potentialTargets, i)
				}
			}
		}
		if len(potentialTargets) == 0 {
			continue
		}

		targetIdx := potentialTargets[rng.Intn(len(potentialTargets))]
		target := cp[targetIdx]
		targetPlayer := target.p

		// Backline evasion: 50% miss for physical mobs vs backline
		if targetPlayer.Position == PositionBackline && m.Element == ElementPhysical {
			if rng.Float64() < 0.5 {
				*logs = append(*logs, fmt.Sprintf("💨 %s slipped into the shadows! %s missed.", targetPlayer.PlayerDisplay(), m.Name))
				continue
			}
		}

		// Stealth: skip first round
		if round == 1 && targetPlayer.HasEffect(EffectStealth) {
			continue
		}

		// Parry: 10% chance to counter
		if targetPlayer.HasEffect(EffectParry) && rng.Intn(100) < 10 {
			*logs = append(*logs, fmt.Sprintf("🛡️ %s PARRIED %s's attack and countered!", targetPlayer.PlayerDisplay(), m.Name))
			counterDmg := maxInt(1, int(float64(target.str)*0.5*intensify))
			m.HP -= counterDmg
			result.DamageDealt += int64(counterDmg)
			continue
		}

		// Dodge check — capped at 25%
		dodgeChance := targetPlayer.Stats.DGE
		if dodgeChance > params.DodgeCap {
			dodgeChance = params.DodgeCap
		}
		if rng.Intn(100) < dodgeChance {
			continue
		}

		// Spell cast: 20% chance
		dmgMult := 1.0
		if len(m.Spells) > 0 && rng.Float64() < params.MobSpellChance {
			spell := m.Spells[rng.Intn(len(m.Spells))]
			dmgMult = spell.Power
			*logs = append(*logs, fmt.Sprintf("🔥 %s cast %s!", m.Name, spell.Name))
		}

		// Elemental multiplier
		targetElement := ElementPhysical
		if ch, ok := targetPlayer.Gear["Chest"]; ok {
			targetElement = ch.Element
		}
		elementMult := GetElementMult(m.Element, targetElement)
		dmgMult *= elementMult

		// Compute effective mob STR with effects
		mSTR := ComputeEffectiveMobSTR(m, fatigueMult)

		// Zone debuff (simplified: 10% chance of 10% debuff)
		if rng.Float64() < 0.1 {
			mSTR = int(float64(mSTR) * 0.9)
		}

		dmg := int((float64(mSTR)*dmgMult - float64(target.def)) * intensify)

		// Frontline defense: 10% damage reduction
		if targetPlayer.Position == PositionFrontline {
			dmg = int(float64(dmg) * 0.9)
		}

		// Minimum damage: 25% of STR
		minDmg := int(float64(mSTR) * 0.25 * intensify)
		if dmg < minDmg {
			dmg = minDmg
		}
		if dmg < 1 {
			dmg = 1
		}

		// Blinded: 50% miss
		for _, eff := range m.Effects {
			if eff == MobEffectBlinded && rng.Float64() < 0.5 {
				dmg = 0
			}
		}

		if dmg > 0 {
			cp[targetIdx].hp -= dmg
			if cp[targetIdx].hp < 0 {
				cp[targetIdx].hp = 0
			}
			result.DamageReceived += int64(dmg)
			*logs = append(*logs, fmt.Sprintf("💢 %s hits %s for %d!", m.Name, targetPlayer.PlayerDisplay(), dmg))
		}

		// Check player death + revive
		if cp[targetIdx].hp <= 0 {
			// Check revive consumable
			revived := false
			for ci, c := range targetPlayer.Consumables {
				if c.Type == ConsumableRevive {
					cp[targetIdx].hp = targetPlayer.MaxHP / 2
					*logs = append(*logs, fmt.Sprintf("🔥 %s REVIVED [item:%s]!", targetPlayer.PlayerDisplay(), c.Name))
					targetPlayer.Consumables[ci].Remaining--
					if targetPlayer.Consumables[ci].Remaining <= 0 {
						targetPlayer.Consumables = append(targetPlayer.Consumables[:ci], targetPlayer.Consumables[ci+1:]...)
					}
					revived = true
					break
				}
			}
			// Check Phoenix effect
			if !revived && targetPlayer.HasEffect(EffectPhoenix) {
				cp[targetIdx].hp = targetPlayer.MaxHP / 2
				*logs = append(*logs, fmt.Sprintf("✨ %s REVIVED [item:phoenix]!", targetPlayer.PlayerDisplay()))
				revived = true
			}
			if !revived {
				*logs = append(*logs, fmt.Sprintf("💀 %s was slain by %s!", targetPlayer.PlayerDisplay(), m.Name))
			}
		}

		// Thorns: reflect 10% damage
		if targetPlayer.HasEffect(EffectThorns) && dmg > 0 {
			reflect := maxInt(1, dmg/10)
			m.HP -= reflect
			result.DamageDealt += int64(reflect)
		}
	}
}

// avgLevel computes the average level of a party.
func avgLevel(players []*SimPlayer) int {
	if len(players) == 0 {
		return 1
	}
	total := 0
	for _, p := range players {
		total += p.Level
	}
	return total / len(players)
}
