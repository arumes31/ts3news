package bot

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"

	"ts3news/internal/clientquery"
	"ts3news/internal/content"
	"ts3news/internal/leveling"
)

// XP-modifier tuning constants.
const (
	critChance         = 0.05
	critMult           = 3.0
	partyMult          = 1.25
	serverMultPerUser  = 0.05
	serverMultCap      = 2.0
	noGamePenalty      = 0.5
	dailyLoginXP       = 5
	lootBoxEveryLevels = 25
	lootBoxMin         = 50
	lootBoxMax         = 500
	slothGraceDays     = 7
	slothDailyDecay    = 0.02
	artifactChance     = 0.01
	titleChance        = 0.005
	gearChance         = 0.05
	consChance         = 0.1
	enchChance         = 0.02
	skillChance        = 0.05
	duraLossPerFight   = 1
	duraLossPenalty    = 3
	occupiedSlotRare   = 0.1
	deathXPPenalty     = 0.05 // 5% XP loss on death
)

type UserInCombat struct {
	UID         string
	Nickname    string
	CLID        int
	Level       int
	Stats       content.Stats
	Skills      []content.Skill
	CurrentHP   int
	RegenStacks int
	Pets        []*content.Mob
}

type activeUser struct {
	u       *UserInCombat
	effects []content.ItemEffect
}

// cycleContext holds per-cycle shared facts used by the XP modifiers.
type cycleContext struct {
	onlineNormal       int
	channelNormalCount map[int]int
	onlineNicks        map[string]bool
	today              time.Time
}

func (b *Bot) buildCycleContext(clients []clientquery.ClientInfo) cycleContext {
	online := map[string]bool{}
	chans := map[int]int{}
	normal := 0
	for _, cl := range clients {
		if cl.Type == 0 {
			normal++
			online[strings.ToLower(cl.Nickname)] = true
			if cl.CID >= 0 {
				chans[cl.CID]++
			}
		}
	}
	return cycleContext{
		onlineNormal:       normal,
		channelNormalCount: chans,
		onlineNicks:        online,
		today:              time.Now(),
	}
}

// processUserXP applies all XP gains for one user this cycle.
func (b *Bot) processUserXP(uid, nickname string, cid, base int, hasGame bool, ctx cycleContext) (*levelResult, []string, string) {
	var notes []string
	delta := 0

	if b.Cfg.EnableXPModifiers {
		b.ensureUserHasGear(uid)

		if b.dailyLoginDue(uid, ctx.today) {
			delta += dailyLoginXP
			notes = append(notes, fmt.Sprintf("daily login +%d", dailyLoginXP))
			b.setLastLogin(uid, ctx.today)
		}
	}

	stats, mult, mnotes := b.calculateTotalStats(uid, ctx.today)
	notes = append(notes, mnotes...)

	// Intelligence bonus
	if stats.INT > 0 {
		intMult := 1.0 + float64(stats.INT)/1000.0
		mult *= intMult
		notes = append(notes, fmt.Sprintf("INT bonus x%.3f", intMult))
	}

	// Flavour stats
	if stats.STN > 50 { notes = append(notes, "You smell terrible!") }
	if stats.CHA > 100 { notes = append(notes, "You are remarkably charming today.") }
	if stats.SHN > 50 { notes = append(notes, "You are literally glowing.") }

	awardMult := b.computeMiscMult(uid, nickname, cid, ctx)
	if !hasGame {
		mult *= noGamePenalty
		notes = append(notes, "no new game -50%")
	}
	award := 0
	if base > 0 {
		award = int(math.Round(float64(base) * mult * awardMult))
		if award < 1 { award = 1 }
	} else {
		// Penalty should NOT be subject to positive multipliers (streak, etc.)
		award = base // base is already negative here
		// Cap loss at 10% of total XP or a reasonable flat amount for low levels
		var curXP int
		_ = b.DB.QueryRow("SELECT xp FROM users WHERE client_uid=$1", uid).Scan(&curXP)
		maxLoss := -(10 + int(float64(curXP)*0.1))
		if award < maxLoss { award = maxLoss }
	}
	delta += award

	lr, err := b.awardXP(uid, nickname, delta)
	if err != nil {
		log.Printf("processUserXP: awardXP failed for %s: %v", nickname, err)
		return &levelResult{}, notes, ""
	}

	if b.Cfg.EnableXPModifiers {
		if box := lootBoxForCross(lr.OldLevel, lr.NewLevel); box > 0 {
			if lr2, err := b.awardXP(uid, nickname, box); err == nil {
				notes = append(notes, fmt.Sprintf("LOOT BOX +%d XP", box))
				lr = &levelResult{OldLevel: lr.OldLevel, NewLevel: lr2.NewLevel, TotalXP: lr2.TotalXP, Awarded: delta + box}
			}
		}
	}

	return lr, notes, ""
}

func (b *Bot) computeMiscMult(uid, nickname string, cid int, ctx cycleContext) float64 {
	if !b.Cfg.EnableXPModifiers {
		return 1.0
	}
	mult := 1.0
	if streak := b.updateStreak(uid, ctx.today); streakMultiplier(streak) > 1 {
		mult *= streakMultiplier(streak)
	}
	if rand.Float64() < critChance {
		mult *= critMult
	}
	if sv := serverMultiplier(ctx.onlineNormal); sv > 1 {
		mult *= sv
	}
	if cid >= 0 && ctx.channelNormalCount[cid] > 1 {
		mult *= partyMult
	}
	return mult
}

func (b *Bot) getPets(uid string) []*content.Mob {
	rows, err := b.DB.Query("SELECT name, mob_type, level, hp, max_hp, str, def, spd FROM user_pets WHERE client_uid = $1", uid)
	if err != nil { return nil }
	defer func() { _ = rows.Close() }()
	var out []*content.Mob
	for rows.Next() {
		var m content.Mob
		var mType string
		var maxHP int
		if err := rows.Scan(&m.Name, &mType, &m.Level, &m.Stats.HP, &maxHP, &m.Stats.STR, &m.Stats.DEF, &m.Stats.SPD); err == nil {
			m.Type = content.MobType(mType)
			out = append(out, &m)
		}
	}
	return out
}

func (b *Bot) savePet(uid string, m *content.Mob) {
	_, _ = b.DB.Exec(`INSERT INTO user_pets (client_uid, name, mob_type, level, hp, max_hp, str, def, spd) 
	                  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`, 
	                  uid, m.Name, string(m.Type), m.Level, m.Stats.HP, m.Stats.HP, m.Stats.STR, m.Stats.DEF, m.Stats.SPD)
}

func (b *Bot) deletePet(uid, name string) {
	_, _ = b.DB.Exec("DELETE FROM user_pets WHERE client_uid = $1 AND name = $2", uid, name)
}

func (b *Bot) updatePetHP(uid, name string, hp int) {
	if hp <= 0 {
		b.deletePet(uid, name)
	} else {
		_, _ = b.DB.Exec("UPDATE user_pets SET hp = $3 WHERE client_uid = $1 AND name = $2", uid, name, hp)
	}
}

func (b *Bot) checkUserRevive(u *UserInCombat, logs *[]string) bool {
	if u.CurrentHP > 0 { return false }
	
	// 1. Check Consumables
	cons := b.getConsumables(u.UID)
	for _, c := range cons {
		if c.Type == content.ConsumableRevive {
			u.CurrentHP = u.Stats.HP / 2
			*logs = append(*logs, fmt.Sprintf("🔥 %s REVIVED (Item)!", u.Nickname))
			_, _ = b.DB.Exec("DELETE FROM user_consumables WHERE client_uid = $1 AND cons_id = $2", u.UID, c.ID)
			return true
		}
	}
	// 2. Check Item Effects (Phoenix)
	_, _, _, effects := b.activeLootMult(u.UID, time.Now())
	for _, eff := range effects {
		if eff == content.EffectPhoenix {
			u.CurrentHP = u.Stats.HP / 2
			*logs = append(*logs, fmt.Sprintf("✨ %s REVIVED (Phoenix)!", u.Nickname))
			return true
		}
	}
	return false
}

func (b *Bot) resolveChannelCombat(users []UserInCombat, initialMobs []*content.Mob, avgLvl int, diffFactor float64, zone content.Zone) ([]string, int, bool) {
	var logs []string
	mobs := initialMobs

	// 1. Battle Header (What we fighting)
	var partyNames []string
	totalPartyGS := 0
	for _, u := range users {
		gs := u.Stats.Score()
		totalPartyGS += gs
		partyNames = append(partyNames, fmt.Sprintf("%s (%d)", u.Nickname, gs))
	}
	
	mobCounts := make(map[string]int)
	totalEnemyCR := 0
	for _, m := range mobs {
		mobCounts[m.DisplayName()]++
		totalEnemyCR += m.Score()
	}
	var enemyNames []string
	for name, count := range mobCounts {
		if count > 1 {
			enemyNames = append(enemyNames, fmt.Sprintf("%dx %s", count, name))
		} else {
			enemyNames = append(enemyNames, name)
		}
	}
	logs = append(logs, fmt.Sprintf("⚔️ BATTLE [GS:%d VS CR:%d]", totalPartyGS, totalEnemyCR))
	logs = append(logs, fmt.Sprintf("🛡️ %s VS %s", strings.Join(partyNames, ", "), strings.Join(enemyNames, ", ")))

	var activeUsers []activeUser
	for i := range users {
		_, _, _, effects := b.activeLootMult(users[i].UID, time.Now())
		activeUsers = append(activeUsers, activeUser{u: &users[i], effects: effects})
	}

	// Apply consumables to users before fight
	for _, au := range activeUsers {
		u := au.u
		cons := b.getConsumables(u.UID)
		for _, c := range cons {
			if c.Type == content.ConsumableBuff {
				if c.ID == "P3" { u.Stats.STR += c.EffectValue; logs = append(logs, fmt.Sprintf("🛡️ %s is buffed by %s!", u.Nickname, c.Name)) }
				if c.ID == "P4" { u.Stats.DEF += c.EffectValue; logs = append(logs, fmt.Sprintf("🛡️ %s is buffed by %s!", u.Nickname, c.Name)) }
			}
		}
	}

	// Pity system
	totalLosses := 0
	for _, u := range users {
		var l int
		_ = b.DB.QueryRow("SELECT consecutive_losses FROM users WHERE client_uid=$1", u.UID).Scan(&l)
		totalLosses += l
	}
	avgLosses := 0.0
	if len(users) > 0 { avgLosses = float64(totalLosses) / float64(len(users)) }
	pityBuff := 1.0 + (avgLosses * 0.2)
	if pityBuff > 1.0 {
		logs = append(logs, fmt.Sprintf("⚠️ Combat Pity active: Stats boosted by %.0f%%!", (pityBuff-1.0)*100))
	}

	for i := range users {
		u := &users[i]
		u.Stats.HP = int(float64(u.Stats.HP) * pityBuff)
		u.Stats.STR = int(float64(u.Stats.STR) * pityBuff)
		u.Stats.DEF = int(float64(u.Stats.DEF) * pityBuff)
	}

	totalRewardXP := 0
	for _, m := range mobs { totalRewardXP += m.RewardXP }

	// Log mob effects
	for _, m := range mobs {
		for _, eff := range m.Effects {
			logs = append(logs, fmt.Sprintf("❕ %s is %s!", m.Name, eff))
		}
	}

	victory := false
	var totalUserDamage, totalMobDamage int

	for r := 1; r <= 10; r++ { // Reduced to 10 rounds max for speed
		// Escalating Intensity: Damage increases by 15% per round to prevent stalls
		intensify := 1.0 + float64(r-1)*0.15
		
		// Healing Exhaustion: Reduced healing after round 5
		healPenalty := 1.0
		if r > 5 { healPenalty = 1.0 - float64(r-5)*0.2 }
		if healPenalty < 0 { healPenalty = 0 }

		// 1. Round Start Effects (Regen/Poison/Pets/Hazards)
		for _, eff := range zone.Effects {
			if eff.Type == content.ZoneHazard {
				dmg := int(eff.Power * 25 * intensify)
				if dmg < 1 { dmg = 1 }
				for i := range activeUsers {
					activeUsers[i].u.CurrentHP -= dmg
				}
				for _, m := range mobs {
					m.Stats.HP -= dmg
				}
				if r == 1 { logs = append(logs, fmt.Sprintf("⛈️ %s Hazard is active!", eff.Name)) }
			}
		}

		for i := range mobs {
			m := mobs[i]
			if m.Stats.HP <= 0 { continue }
			for _, eff := range m.Effects {
				switch eff {
				case content.EffectPoisoned: 
					delta := int(float64(m.Stats.HP/20) * intensify)
					if delta < 1 { delta = 1 }
					m.Stats.HP -= delta
				case content.EffectRegen: 
					delta := int(float64(m.Stats.HP/20) * healPenalty)
					if delta < 1 { delta = 1 }
					m.Stats.HP += delta
				}
			}
		}
		
		for _, au := range activeUsers {
			u := au.u
			if u.CurrentHP <= 0 { continue }
			// Passive Regen Stacks
			if u.RegenStacks > 0 {
				heal := int(float64(u.RegenStacks * 2) * healPenalty)
				u.CurrentHP += heal
				if u.CurrentHP > u.Stats.HP { u.CurrentHP = u.Stats.HP }
			}
			// Pets Regen
			for _, p := range u.Pets {
				if p.Stats.HP > 0 {
					p.Stats.HP += int(float64(p.Level * 2) * healPenalty)
				}
			}
		}

		// 2. User Turn
		for _, au := range activeUsers {
			u := au.u
			if u.CurrentHP <= 0 { continue }

			// Zone Buff check
			uSTR := u.Stats.STR
			for _, eff := range zone.Effects {
				if eff.Type == content.ZoneBuff {
					uSTR = int(float64(uSTR) * (1.0 + eff.Power))
				}
			}

			var lifesteal int
			var multiStrike int
			var mindControlLevel int
			var extraHits = 1

			var tName sql.NullString
			_ = b.DB.QueryRow("SELECT title FROM users WHERE client_uid=$1", u.UID).Scan(&tName)
			if tName.Valid {
				if t, ok := content.GetTitleByName(tName.String); ok {
					lifesteal = t.Lifesteal
					multiStrike = t.MultiStrike
				}
			}
			
			// Calculate Mind Control Level
			rows, _ := b.DB.Query("SELECT gear_id FROM user_gear WHERE client_uid = $1", u.UID)
			if rows != nil {
				for rows.Next() {
					var gid string
					if err := rows.Scan(&gid); err == nil {
						if g, ok := content.GetGearByID(gid); ok && g.Special == content.EffectMindControl {
							mindControlLevel += int(g.Rarity) + 1
						}
					}
				}
				_ = rows.Close()
			}
			for _, s := range u.Skills {
				if s.Special == content.EffectMindControl {
					mindControlLevel += int(s.Rarity) + 1
				}
			}

			for _, eff := range au.effects {
				if eff == content.EffectVampiric { lifesteal += 5 }
			}

			if multiStrike > 0 && rand.Intn(100) < multiStrike {
				extraHits = 2
				logs = append(logs, fmt.Sprintf("⚔️ %s double attack!", u.Nickname))
			}

			for h := 0; h < extraHits; h++ {
				aliveMobs := b.getAliveMobs(mobs)
				if len(aliveMobs) == 0 { break }
				target := aliveMobs[rand.Intn(len(aliveMobs))]
				
				dmgMult := 1.0
				ignoreDef := 0.0
				for _, eff := range au.effects {
					if eff == content.EffectBerserk && u.CurrentHP < u.Stats.HP/2 { dmgMult += 0.2 }
					if eff == content.EffectFragile { dmgMult += 0.3 }
				}

				if len(u.Skills) > 0 && rand.Float64() < 0.3 {
					s := u.Skills[rand.Intn(len(u.Skills))]
					dmgMult *= s.Power
					ignoreDef = s.IgnoreDef
					logs = append(logs, fmt.Sprintf("✨ %s: %s!", u.Nickname, s.Name))
					if s.StunChance > 0 && rand.Float64() < s.StunChance {
						logs = append(logs, fmt.Sprintf("💫 %s STUNNED!", target.Name))
						target.Stats.SPD = 0
					}
				}

				effDef := float64(target.Stats.DEF) * (1.0 - ignoreDef)
				dmg := int((float64(uSTR)*dmgMult - effDef) * intensify)
				
				// Percentage-Based Damage Floor (15% of STR) to prevent DEF stalemates
				minDmg := int(float64(uSTR) * 0.15 * intensify)
				if dmg < minDmg { dmg = minDmg }
				if dmg < 1 { dmg = 1 }

				target.Stats.HP -= dmg
				totalUserDamage += dmg

				// Chain Attack Logic for groups (3+ players)
				if len(users) >= 3 && rand.Float64() < 0.3 {
					others := b.getAliveMobs(mobs)
					if len(others) > 1 {
						var chainTarget *content.Mob
						for _, xm := range others { if xm != target { chainTarget = xm; break } }
						if chainTarget != nil {
							chainDmg := dmg / 2
							if chainDmg < 1 { chainDmg = 1 }
							chainTarget.Stats.HP -= chainDmg
							totalUserDamage += chainDmg
						}
					}
				}

				// Mind Control Logic (Scale with level)
				if mindControlLevel > 0 && len(u.Pets) < mindControlLevel && target.Stats.HP > 0 && float64(target.Stats.HP) < float64(target.Level*20)*0.2 { 
					if rand.Float64() < 0.5 {
						logs = append(logs, fmt.Sprintf("🌀 Captive: %s!", target.Name))
						u.Pets = append(u.Pets, target)
						b.savePet(u.UID, target)
						target.Stats.HP = target.Level * 10
						newMobs := []*content.Mob{}
						for _, xm := range mobs { if xm != target { newMobs = append(newMobs, xm) } }
						mobs = newMobs
					}
				}

				if lifesteal > 0 {
					heal := int(float64(dmg) * float64(lifesteal) / 100.0 * healPenalty)
					if heal > 0 { 
						u.CurrentHP += heal
						if u.CurrentHP > u.Stats.HP { u.CurrentHP = u.Stats.HP }
					}
				}

				if target.Stats.HP <= 0 {
					logs = append(logs, fmt.Sprintf("☠️ %s defeated by %s!", target.Name, u.Nickname))
					// Award loot for every mob defeated, regardless of final outcome
					winner := users[rand.Intn(len(users))]
					if note := b.rollLootForUser(winner.UID, *target, zone.Difficulty); note != "" {
						logs = append(logs, fmt.Sprintf("🎁 %s looted %s: %s", winner.Nickname, target.Name, note))
					}
					b.handleDeathEffects(target, &mobs, &logs, avgLvl, diffFactor, activeUsers)
				}
				if len(b.getAliveMobs(mobs)) == 0 { break }
			}
			
			// Pet Attack (Silent damage)
			for _, p := range u.Pets {
				if p.Stats.HP <= 0 { continue }

				// Betrayal check (3% chance)
				if rand.Float64() < 0.03 {
					targetAU := activeUsers[rand.Intn(len(activeUsers))]
					target := targetAU.u
					if target.CurrentHP > 0 {
						pdmg := int(float64(p.Stats.STR - target.Stats.DEF) * intensify)
						if pdmg < 1 { pdmg = 1 }
						target.CurrentHP -= pdmg
						logs = append(logs, fmt.Sprintf("⚠️ Rogue Pet %s bit %s for %d!", p.Name, target.Nickname, pdmg))
						totalMobDamage += pdmg
						b.checkUserRevive(target, &logs)
						continue
					}
				}

				aliveMobs := b.getAliveMobs(mobs)
				if len(aliveMobs) == 0 { break }
				ptarget := aliveMobs[rand.Intn(len(aliveMobs))]
				pdmg := int(float64(p.Stats.STR - ptarget.Stats.DEF) * intensify)
				if pdmg < 1 { pdmg = 1 }
				ptarget.Stats.HP -= pdmg
				totalUserDamage += pdmg
				if ptarget.Stats.HP <= 0 {
					logs = append(logs, fmt.Sprintf("☠️ %s killed by pet %s!", ptarget.Name, p.Name))
					winner := users[rand.Intn(len(users))]
					if note := b.rollLootForUser(winner.UID, *ptarget, zone.Difficulty); note != "" {
						logs = append(logs, fmt.Sprintf("🎁 %s looted %s: %s", winner.Nickname, ptarget.Name, note))
					}
					b.handleDeathEffects(ptarget, &mobs, &logs, avgLvl, diffFactor, activeUsers)
				}
			}
			
			if len(b.getAliveMobs(mobs)) == 0 { break }
		}
		if len(b.getAliveMobs(mobs)) == 0 { victory = true; break }

		// 3. Mob Turn
		for _, m := range mobs {
			if m.Stats.HP <= 0 || m.Stats.SPD == 0 { 
				if m.Stats.SPD == 0 { m.Stats.SPD = 10 } // recover
				continue 
			}
			
			targetAU := activeUsers[rand.Intn(len(activeUsers))]
			target := targetAU.u
			if target.CurrentHP <= 0 { continue }
			
			if rand.Intn(100) < target.Stats.DGE { continue }

			dmgMult := 1.0
			if len(m.Spells) > 0 && rand.Float64() < 0.2 {
				s := m.Spells[rand.Intn(len(m.Spells))]
				dmgMult = s.Power
				logs = append(logs, fmt.Sprintf("🔥 %s cast %s!", m.Name, s.Name))
			}

			mSTR := m.Stats.STR
			// Zone Debuff check
			for _, eff := range zone.Effects {
				if eff.Type == content.ZoneDebuff {
					mSTR = int(float64(mSTR) * (1.0 - eff.Power))
				}
			}

			for _, eff := range m.Effects {
				switch eff {
				case content.EffectEnraged: mSTR = int(float64(mSTR) * 1.5)
				case content.EffectWeakened: mSTR = int(float64(mSTR) * 0.5)
				}
			}

			dmg := int((float64(mSTR)*dmgMult - float64(target.Stats.DEF)) * intensify)
			
			// Percentage-Based Damage Floor (10% of STR)
			minDmg := int(float64(mSTR) * 0.10 * intensify)
			if dmg < minDmg { dmg = minDmg }
			if dmg < 1 { dmg = 1 }
			
			for _, eff := range m.Effects {
				if eff == content.EffectBlinded && rand.Float64() < 0.5 { dmg = 0 }
			}

			target.CurrentHP -= dmg
			totalMobDamage += dmg

			// Check Revival
			if target.CurrentHP <= 0 {
				if !b.checkUserRevive(target, &logs) {
					logs = append(logs, fmt.Sprintf("💀 %s was slain by %s!", target.Nickname, m.Name))
				}
			}

			for _, eff := range targetAU.effects {
				if eff == content.EffectThorns && dmg > 0 {
					reflect := dmg / 10
					if reflect < 1 { reflect = 1 }
					m.Stats.HP -= reflect
					totalUserDamage += reflect
				}
			}
		}
		
		aliveUsers := 0
		for _, u := range users { if u.CurrentHP > 0 { aliveUsers++ } }
		if aliveUsers == 0 { victory = false; break }
	}

	// Summarize Combat
	logs = append(logs, fmt.Sprintf("📊 Battle Summary: Party %d dmg vs Mobs %d dmg.", totalUserDamage, totalMobDamage))

	// Update pity, quests, consumables AND persistent stats
	for i := range users {
		u := &users[i]
		// Save pets state
		for _, p := range u.Pets {
			b.updatePetHP(u.UID, p.Name, p.Stats.HP)
		}

		finalXP := 0
		if victory {
			_, _ = b.DB.Exec("UPDATE users SET consecutive_losses = 0 WHERE client_uid = $1", u.UID)
			b.updateQuest(u.UID, "mobs_killed", len(initialMobs))
			
			// Regen Stacks logic
			hasRegEffect := false
			_, _, _, effects := b.activeLootMult(u.UID, time.Now())
			for _, eff := range effects { if eff == content.EffectRegenStack { hasRegEffect = true } }
			if hasRegEffect { u.RegenStacks++ }
			
		} else {
			_, _ = b.DB.Exec("UPDATE users SET consecutive_losses = consecutive_losses + 1 WHERE client_uid = $1", u.UID)
			// Death Penalty
			var curXP int
			_ = b.DB.QueryRow("SELECT xp FROM users WHERE client_uid=$1", u.UID).Scan(&curXP)
			penalty := int(float64(curXP) * deathXPPenalty)
			if penalty < 10 { penalty = 10 }
			finalXP -= penalty
			u.CurrentHP = 0 // dead
			u.RegenStacks = 0 // lose stacks on death
		}
		
		_, _ = b.DB.Exec("UPDATE users SET current_hp = $2, regen_stacks = $3 WHERE client_uid = $1", u.UID, u.CurrentHP, u.RegenStacks)
		
		_, _ = b.DB.Exec("UPDATE user_consumables SET remaining_fights = remaining_fights - 1 WHERE client_uid = $1", u.UID)
		_, _ = b.DB.Exec("DELETE FROM user_consumables WHERE client_uid = $1 AND remaining_fights < 0", u.UID)
		
		if finalXP != 0 {
			_, _ = b.awardXP(u.UID, "", finalXP)
		}
	}

	if victory {
		logs = append(logs, fmt.Sprintf("🏁 VICTORY! Party defeated all %d mobs in %s.", len(mobs), zone.Name))
		return logs, totalRewardXP / len(users), true
	}
	logs = append(logs, fmt.Sprintf("🏁 DEFEAT! Party was overrun in %s.", zone.Name))
	return logs, -totalRewardXP / (2 * len(users)), false
}

func (b *Bot) getAliveMobs(mobs []*content.Mob) []*content.Mob {
	var out []*content.Mob
	for _, m := range mobs { if m.Stats.HP > 0 { out = append(out, m) } }
	return out
}

func (b *Bot) handleDeathEffects(m *content.Mob, mobs *[]*content.Mob, logs *[]string, avgLvl int, diffFactor float64, users []activeUser) {
	if m.DeathEffect == nil { return }
	
	*logs = append(*logs, fmt.Sprintf("⚠️ %s triggers %s: %s!", m.Name, m.DeathEffect.Type, m.DeathEffect.Name))

	switch m.DeathEffect.Type {
	case content.DeathSummon:
		count := 1
		if m.Type == content.MobCommon { count = 3 } // Trash mobs summon hordes
		for i := 0; i < count; i++ {
			// Summoned mobs are lower tier
			lvl := avgLvl - 5
			if lvl < 1 { lvl = 1 }
			newMob := content.SpawnMob(lvl, false, diffFactor * 0.7)
			newMob.Name = "Summoned " + newMob.Name
			*mobs = append(*mobs, &newMob)
		}
		*logs = append(*logs, fmt.Sprintf("📢 %d reinforcements have arrived!", count))

	case content.DeathExplosion:
		dmg := m.Level * 10
		for i := range users {
			users[i].u.CurrentHP -= dmg
			if users[i].u.CurrentHP < 0 { users[i].u.CurrentHP = 0 }
		}
		*logs = append(*logs, fmt.Sprintf("💥 Explosion dealt %d damage to everyone!", dmg))

	case content.DeathCurse:
		for i := range users {
			users[i].u.Stats.STR -= 10
			users[i].u.Stats.DEF -= 5
		}
		*logs = append(*logs, "🥀 A dark curse weakens the party!")

	case content.DeathXP:
		*logs = append(*logs, "✨ A pulse of pure energy provides bonus XP!")

	case content.DeathLoot:
		*logs = append(*logs, "💰 Shiny items scatter across the floor!")
	}
}

func (b *Bot) getConsumables(uid string) []content.Consumable {
	rows, err := b.DB.Query("SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid = $1", uid)
	if err != nil { return nil }
	defer func() { _ = rows.Close() }()
	var out []content.Consumable
	for rows.Next() {
		var id string
		var rem int
		if err := rows.Scan(&id, &rem); err == nil {
			if c, ok := content.GetConsumableByID(id); ok {
				c.Duration = rem
				out = append(out, c)
			}
		}
	}
	return out
}

func (b *Bot) updateQuest(uid, qType string, progress int) {
	_, _ = b.DB.Exec(`INSERT INTO user_quests (client_uid, quest_type, progress, total_earned) 
	                  VALUES ($1, $2, $3, $3) 
	                  ON CONFLICT (client_uid, quest_type) 
	                  DO UPDATE SET progress = user_quests.progress + $3, total_earned = user_quests.total_earned + $3`, 
	                  uid, qType, progress)
}

func streakMultiplier(streak int) float64 {
	switch {
	case streak >= 7: return 2.0
	case streak >= 5: return 1.5
	case streak >= 3: return 1.25
	default: return 1.0
	}
}

func serverMultiplier(onlineNormal int) float64 {
	humans := onlineNormal - 1
	if humans < 1 { humans = 1 }
	m := 1 + serverMultPerUser*float64(humans-1)
	if m > serverMultCap { m = serverMultCap }
	return m
}

func lootBoxForCross(oldLevel, newLevel int) int {
	if newLevel <= oldLevel { return 0 }
	first := (oldLevel/lootBoxEveryLevels + 1) * lootBoxEveryLevels
	if first <= newLevel { return lootBoxMin + rand.Intn(lootBoxMax-lootBoxMin+1) }
	return 0
}

func (b *Bot) updateStreak(uid string, today time.Time) int {
	var last sql.NullTime
	var streak int
	if err := b.DB.QueryRow("SELECT last_poke_date, streak_days FROM users WHERE client_uid=$1", uid).Scan(&last, &streak); err != nil { return 0 }
	if last.Valid && sameDay(last.Time, today) { return streak }
	if last.Valid && sameDay(last.Time, today.AddDate(0, 0, -1)) { streak++ } else { streak = 1 }
	_, _ = b.DB.Exec("UPDATE users SET streak_days=$2, last_poke_date=$3 WHERE client_uid=$1", uid, streak, today)
	return streak
}

func (b *Bot) dailyLoginDue(uid string, today time.Time) bool {
	var last sql.NullTime
	if err := b.DB.QueryRow("SELECT last_login_date FROM users WHERE client_uid=$1", uid).Scan(&last); err != nil { return false }
	return !last.Valid || !sameDay(last.Time, today)
}

func (b *Bot) setLastLogin(uid string, today time.Time) {
	_, _ = b.DB.Exec("UPDATE users SET last_login_date=$2 WHERE client_uid=$1", uid, today)
}

func (b *Bot) ensureUserHasGear(uid string) {
	var count int
	// Count gear, skills, and check artifact
	_ = b.DB.QueryRow(`
		SELECT 
			(SELECT COUNT(*) FROM user_gear WHERE client_uid = $1) + 
			(SELECT COUNT(*) FROM user_skills WHERE client_uid = $1) + 
			(CASE WHEN artifact_name IS NOT NULL AND artifact_durability > 0 THEN 1 ELSE 0 END)
		FROM users WHERE client_uid = $1`, uid).Scan(&count)

	if count > 5 {
		return
	}

	// Get currently equipped slots
	rows, err := b.DB.Query("SELECT slot FROM user_gear WHERE client_uid = $1", uid)
	if err != nil { return }
	defer func() { _ = rows.Close() }()
	
	equipped := make(map[string]bool)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil { equipped[s] = true }
	}

	// Fill ALL empty slots with Novice gear
	for _, slot := range content.AllSlots {
		slotStr := string(slot)
		if !equipped[slotStr] {
			gearID := fmt.Sprintf("B_%s", slotStr)
			if gear, ok := content.GetGearByID(gearID); ok {
				_, _ = b.DB.Exec("INSERT INTO user_gear (client_uid, slot, gear_id, durability) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
					uid, slotStr, gear.ID, gear.MaxDurability)
			}
		}
	}

	// Give 2 Random "Better" Items (Head Start)
	for i := 0; i < 2; i++ {
		g := content.RandomGearDrop()
		// Only give if it's actually an improvement or filling a low-tier slot
		if b.shouldEquip(uid, g) {
			_, _ = b.DB.Exec(`INSERT INTO user_gear (client_uid, slot, gear_id, durability) 
			                  VALUES ($1, $2, $3, $4) 
			                  ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = $3, durability = $4`, 
			                  uid, string(g.Slot), g.ID, g.MaxDurability)
		}
	}

	// Also give Novice Skills if empty
	var skillCount int
	_ = b.DB.QueryRow("SELECT COUNT(*) FROM user_skills WHERE client_uid = $1", uid).Scan(&skillCount)
	if skillCount == 0 {
		_, _ = b.DB.Exec("INSERT INTO user_skills (client_uid, slot, skill_id) VALUES ($1, 1, 'S0_1'), ($1, 2, 'S0_2')", uid)
	}
}

func (b *Bot) applyDurabilityLoss(uid string, defeat bool) {
	var stats content.Stats
	var effects []content.ItemEffect
	_, stats, _, effects = b.activeLootMult(uid, time.Now())
	
	// Fragile check
	lossMult := 1
	for _, eff := range effects {
		if eff == content.EffectFragile { lossMult = 2 }
	}

	if rand.Intn(100) < stats.STA { return }

	loss := duraLossPerFight * lossMult
	if defeat { loss = duraLossPenalty * lossMult }
	_, _ = b.DB.Exec("UPDATE user_gear SET durability = durability - $2 WHERE client_uid = $1", uid, loss)
	_, _ = b.DB.Exec("DELETE FROM user_gear WHERE client_uid = $1 AND durability <= 0", uid)
	_, _ = b.DB.Exec("UPDATE users SET artifact_durability = artifact_durability - $2 WHERE client_uid = $1 AND artifact_durability > 0", uid, loss)
	_, _ = b.DB.Exec("UPDATE users SET artifact_mult=1, artifact_name=NULL, artifact_durability=0 WHERE client_uid=$1 AND artifact_durability <= 0 AND artifact_name IS NOT NULL", uid)
}

func (b *Bot) calculateTotalStats(uid string, today time.Time) (content.Stats, float64, []string) {
	var level, prestige int
	_ = b.DB.QueryRow("SELECT level, prestige FROM users WHERE client_uid=$1", uid).Scan(&level, &prestige)
	base := content.Stats{
		HP: 100 + level*5, STR: 10 + level, DEF: 5 + level/2, SPD: 10 + level, LCK: level/5,
		INT: level/10, STA: level/10, CRT: 5 + level/50, DGE: 5 + level/50,
	}
	
	// Apply Prestige Bonus
	if prestige > 0 {
		pMult := 1.0 + (float64(prestige) * prestigeStatBonus)
		base.HP = int(float64(base.HP) * pMult)
		base.STR = int(float64(base.STR) * pMult)
		base.DEF = int(float64(base.DEF) * pMult)
		base.SPD = int(float64(base.SPD) * pMult)
	}

	mult, lootStats, notes, effects := b.activeLootMult(uid, today)
	totalStats := base.Add(lootStats)

	// Apply effects to stats
	for _, eff := range effects {
		switch eff {
		case content.EffectLucky:
			totalStats.LCK = int(float64(totalStats.LCK) * 1.1)
		case content.EffectQuick:
			totalStats.SPD = int(float64(totalStats.SPD) * 1.1)
		case content.EffectBulwark:
			totalStats.DEF = int(float64(totalStats.DEF) * 1.1)
		case content.EffectRadiant:
			mult *= 1.1
		}
	}

	return totalStats, mult, notes
}

func (b *Bot) activeLootMult(uid string, today time.Time) (float64, content.Stats, []string, []content.ItemEffect) {
	mult := 1.0
	var stats content.Stats
	var notes []string
	var effects []content.ItemEffect

	var title sql.NullString
	var tMult sql.NullFloat64
	var tExp sql.NullTime
	if err := b.DB.QueryRow("SELECT title, title_mult, title_expires FROM users WHERE client_uid=$1", uid).Scan(&title, &tMult, &tExp); err == nil {
		if tExp.Valid && !today.After(tExp.Time) && title.Valid {
			mult *= tMult.Float64
			notes = append(notes, fmt.Sprintf("%s x%g", title.String, tMult.Float64))
			if t, ok := content.GetTitleByName(title.String); ok { stats = stats.Add(t.Stats) }
		} else if title.Valid {
			_, _ = b.DB.Exec("UPDATE users SET title=NULL, title_mult=NULL, title_expires=NULL WHERE client_uid=$1", uid)
		}
	}
	var aMult sql.NullFloat64
	var aName sql.NullString
	var aDura int
	if err := b.DB.QueryRow("SELECT artifact_mult, artifact_name, artifact_durability FROM users WHERE client_uid=$1", uid).Scan(&aMult, &aName, &aDura); err == nil {
		if aName.Valid && aName.String != "" && aDura > 0 {
			mult *= aMult.Float64
			notes = append(notes, fmt.Sprintf("%s x%g (%d dura)", aName.String, aMult.Float64, aDura))
			if art, ok := content.GetArtifactByName(aName.String); ok { 
				stats = stats.Add(art.Stats) 
				if art.Special != content.EffectNone { effects = append(effects, art.Special) }
			}
		}
	}
	rows, err := b.DB.Query("SELECT gear_id, durability, enchantment_id FROM user_gear WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var gearID string
			var dura int
			var enchID sql.NullString
			if err := rows.Scan(&gearID, &dura, &enchID); err == nil {
				if gear, ok := content.GetGearByID(gearID); ok {
					mult *= gear.XPMultiplier
					stats = stats.Add(gear.Stats)
					if gear.Special != content.EffectNone { effects = append(effects, gear.Special) }
					
					note := fmt.Sprintf("%s x%g (%d dura)", gear.Name, gear.XPMultiplier, dura)
					if gear.Special != content.EffectNone { note = fmt.Sprintf("[%s] %s", gear.Special, note) }

					if enchID.Valid && enchID.String != "" {
						if ench, ok := content.GetEnchantmentByID(enchID.String); ok {
							stats = stats.Add(ench.Stats)
							mult *= ench.XPMultiplier // Apply enchantment XP penalty
							if ench.Special != content.EffectNone { effects = append(effects, ench.Special) }
							
							eName := ench.Name
							if ench.Special != content.EffectNone { eName = fmt.Sprintf("%s (%s)", eName, ench.Special) }
							note = fmt.Sprintf("[%s] %s (x%g XP)", eName, note, ench.XPMultiplier)
						}
					}
					notes = append(notes, note)
				}
			}
		}
	}

	// Skills also provide effects
	srows, err := b.DB.Query("SELECT skill_id FROM user_skills WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = srows.Close() }()
		for srows.Next() {
			var sid string
			if err := srows.Scan(&sid); err == nil {
				if s, ok := content.GetSkillByID(sid); ok {
					if s.Special != content.EffectNone { effects = append(effects, s.Special) }
				}
			}
		}
	}

	return mult, stats, notes, effects
}

func (b *Bot) rollLootForUser(uid string, mob content.Mob, zoneDifficulty float64) string {
	var results []string
	count := 1
	if mob.Type == content.MobBoss { count = 2 }
	if mob.Type == content.MobLegendary { count = 4 }

	// Double Loot Title check
	var tName sql.NullString
	_ = b.DB.QueryRow("SELECT title FROM users WHERE client_uid=$1", uid).Scan(&tName)
	
	// Effect check
	_, _, _, effects := b.activeLootMult(uid, time.Now())
	lootFindBonus := 0.0
	for _, eff := range effects {
		if eff == content.EffectTreasureHunter { lootFindBonus += 0.05 }
	}

	// Loot Quality Multiplier: Higher difficulty = better chance for Rares
	qualityMult := zoneDifficulty
	if qualityMult < 1.0 { qualityMult = 1.0 }

	if tName.Valid {
		if t, ok := content.GetTitleByName(tName.String); ok && t.DoubleLoot {
			count *= 2
		}
	}

	for i := 0; i < count; i++ {
		r := rand.Float64() - lootFindBonus
		lootFound := false
		if r < titleChance*qualityMult {
			t := content.RandomTitle()
			_, _ = b.DB.Exec("UPDATE users SET title=$2, title_mult=$3, title_expires=NOW() + INTERVAL '7 days' WHERE client_uid=$1", uid, t.Name, t.XPMultiplier)
			results = append(results, "Title: "+t.Name)
			lootFound = true
		} else if r < artifactChance*qualityMult {
			a := content.RandomArtifact()
			// Scale Artifact stats with zone difficulty
			a.Stats.HP = int(float64(a.Stats.HP) * zoneDifficulty)
			a.Stats.STR = int(float64(a.Stats.STR) * zoneDifficulty)
			a.Stats.DEF = int(float64(a.Stats.DEF) * zoneDifficulty)
			_, _ = b.DB.Exec("UPDATE users SET artifact_mult=$2, artifact_name=$3, artifact_durability=$4 WHERE client_uid=$1", uid, a.Mult, a.Name, a.MaxDurability)
			results = append(results, "Artifact: "+a.Name)
			lootFound = true
		} else if r < gearChance*qualityMult {
			g := content.RandomGearDrop()
			// Scale Gear stats with zone difficulty
			g.Stats.HP = int(float64(g.Stats.HP) * zoneDifficulty)
			g.Stats.STR = int(float64(g.Stats.STR) * zoneDifficulty)
			g.Stats.DEF = int(float64(g.Stats.DEF) * zoneDifficulty)
			g.Stats.SPD = int(float64(g.Stats.SPD) * zoneDifficulty)

			if b.shouldEquip(uid, g) {
				_, _ = b.DB.Exec(`INSERT INTO user_gear (client_uid, slot, gear_id, durability) VALUES ($1, $2, $3, $4) ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = $3, durability = $4`, uid, string(g.Slot), g.ID, g.MaxDurability)
				results = append(results, "Equipped: "+g.Name)
			} else {
				// Disenchant
				xp := 1 + int(g.Rarity)*2
				_, _ = b.awardXP(uid, "", xp)
				results = append(results, fmt.Sprintf("Disenchanted %s (+%d XP)", g.Name, xp))
			}
			lootFound = true
		} else if r < skillChance*qualityMult {
			s := content.RandomSkill()
			// Scale Skill power with zone difficulty
			s.Power *= zoneDifficulty
			if slot, ok := b.equipSkill(uid, s); ok {
				results = append(results, fmt.Sprintf("Learned %s (Slot %d)", s.Name, slot))
			} else {
				// Disenchant
				xp := 2 + int(s.Rarity)*3
				_, _ = b.awardXP(uid, "", xp)
				results = append(results, fmt.Sprintf("Disenchanted %s (+%d XP)", s.Name, xp))
			}
			lootFound = true
		} else if r < consChance*qualityMult {
			c := content.RandomConsumable()
			_, _ = b.DB.Exec("INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", uid, c.ID, c.Duration)
			results = append(results, "Item: "+c.Name)
			lootFound = true
		} else if r < enchChance*qualityMult {
			ench := content.RandomEnchantment()
			// Scale Enchantment stats with zone difficulty
			ench.Stats.STR = int(float64(ench.Stats.STR) * zoneDifficulty)
			ench.Stats.SPD = int(float64(ench.Stats.SPD) * zoneDifficulty)

			if slot, ok := b.applyEnchantment(uid, ench); ok {
				results = append(results, fmt.Sprintf("Enchanted %s with %s", slot, ench.Name))
			} else {
				// Disenchant
				xp := 3 + int(ench.Rarity)*5
				_, _ = b.awardXP(uid, "", xp)
				results = append(results, fmt.Sprintf("Disenchanted %s (+%d XP)", ench.Name, xp))
			}
			lootFound = true
		}

		// 100% Drop Guarantee: If nothing else found, drop a Common item or Scrap
		if !lootFound {
			if rand.Float64() < 0.7 {
				// Drop a basic common gear (Trash/Scrap fallback)
				g := content.RandomStarterGear()
				if b.shouldEquip(uid, g) {
					_, _ = b.DB.Exec(`INSERT INTO user_gear (client_uid, slot, gear_id, durability) VALUES ($1, $2, $3, $4) ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = $3, durability = $4`, uid, string(g.Slot), g.ID, g.MaxDurability)
					results = append(results, "Found: "+g.Name)
				} else {
					results = append(results, "Looted Scrap (+1 XP)")
					_, _ = b.awardXP(uid, "", 1)
				}
			} else {
				results = append(results, "Item: Small Health Potion")
				_, _ = b.DB.Exec("INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, 'P1', 0) ON CONFLICT DO NOTHING", uid)
			}
		}
	}
	if len(results) > 0 { return "🎁 Loot: " + strings.Join(results, ", ") }
	return ""
}

func (b *Bot) equipSkill(uid string, newSkill content.Skill) (int, bool) {
	// Check for Title-based extra slots
	extraSlots := 0
	var tName sql.NullString
	_ = b.DB.QueryRow("SELECT title FROM users WHERE client_uid=$1", uid).Scan(&tName)
	if tName.Valid {
		if t, ok := content.GetTitleByName(tName.String); ok {
			extraSlots = t.ExtraSkills
		}
	}
	maxSlots := 5 + extraSlots

	// Find slot to replace (empty first, then lowest rarity)
	rows, err := b.DB.Query("SELECT slot, skill_id FROM user_skills WHERE client_uid = $1", uid)
	if err != nil { return 0, false }
	defer func() { _ = rows.Close() }()
	
	slots := make(map[int]string)
	for rows.Next() {
		var s int
		var id string
		if err := rows.Scan(&s, &id); err == nil { slots[s] = id }
	}

	// 1. Empty slot
	for i := 1; i <= maxSlots; i++ {
		if _, ok := slots[i]; !ok {
			_, _ = b.DB.Exec("INSERT INTO user_skills (client_uid, slot, skill_id) VALUES ($1, $2, $3)", uid, i, newSkill.ID)
			return i, true
		}
	}

	// 2. Replace lowest rarity if new one is better
	for i := 1; i <= maxSlots; i++ {
		if curID := slots[i]; curID != "" {
			if cur, ok := content.GetSkillByID(curID); ok {
				if newSkill.Rarity > cur.Rarity {
					_, _ = b.DB.Exec("UPDATE user_skills SET skill_id = $3 WHERE client_uid = $1 AND slot = $2", uid, i, newSkill.ID)
					return i, true
				}
			}
		}
	}

	return 0, false
}

func (b *Bot) getSkills(uid string) []content.Skill {
	rows, err := b.DB.Query("SELECT skill_id FROM user_skills WHERE client_uid = $1", uid)
	if err != nil { return nil }
	defer func() { _ = rows.Close() }()
	var out []content.Skill
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			if s, ok := content.GetSkillByID(id); ok { out = append(out, s) }
		}
	}
	return out
}

func (b *Bot) applyEnchantment(uid string, ench content.Enchantment) (string, bool) {
	rows, err := b.DB.Query("SELECT slot, enchantment_id FROM user_gear WHERE client_uid = $1", uid)
	if err != nil { return "", false }
	defer func() { _ = rows.Close() }()
	type slotInfo struct { slot, enchID string }
	var slots []slotInfo
	for rows.Next() {
		var s slotInfo
		var e sql.NullString
		if err := rows.Scan(&s.slot, &e); err == nil {
			if e.Valid { s.enchID = e.String }
			slots = append(slots, s)
		}
	}
	if len(slots) == 0 { return "", false }
	target := slots[rand.Intn(len(slots))]
	if target.enchID != "" {
		if cur, ok := content.GetEnchantmentByID(target.enchID); ok {
			if ench.Rarity < cur.Rarity { return "", false }
		}
	}
	_, _ = b.DB.Exec("UPDATE user_gear SET enchantment_id = $3, durability = durability + $4 WHERE client_uid = $1 AND slot = $2", uid, target.slot, ench.ID, ench.DuraBonus)
	return target.slot, true
}

func (b *Bot) shouldEquip(uid string, newGear content.Gear) bool {
	var currentID string
	err := b.DB.QueryRow("SELECT gear_id FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, string(newGear.Slot)).Scan(&currentID)
	if err == sql.ErrNoRows { return true }
	if cur, ok := content.GetGearByID(currentID); ok {
		return newGear.Rarity > cur.Rarity || newGear.Stats.Score() > cur.Stats.Score()
	}
	return true
}

func (b *Bot) awardXP(uid, nickname string, awarded int) (*levelResult, error) {
	var curXP, curLevel int
	err := b.DB.QueryRow("SELECT xp, level FROM users WHERE client_uid = $1", uid).Scan(&curXP, &curLevel)
	if err == sql.ErrNoRows { curXP, curLevel = 0, 1 } else if err != nil { return nil, err }
	total := curXP + awarded
	if total < 0 { total = 0 }
	newLevel := leveling.LevelForXP(total)
	
	if nickname != "" {
		_, err = b.DB.Exec(`INSERT INTO users (client_uid, nickname, xp, level, last_seen) VALUES ($1, $2, $3, $4, NOW()) ON CONFLICT (client_uid) DO UPDATE SET xp = $3, level = $4, nickname = $2, last_seen = NOW()`, uid, nickname, total, newLevel)
	} else {
		_, err = b.DB.Exec(`UPDATE users SET xp = $2, level = $3, last_seen = NOW() WHERE client_uid = $1`, uid, total, newLevel)
	}
	return &levelResult{OldLevel: curLevel, NewLevel: newLevel, TotalXP: total, Awarded: awarded}, err
}

func (b *Bot) slothDecay(c *clientquery.Client, today time.Time) {
	cutoff := today.AddDate(0, 0, -slothGraceDays)
	rows, err := b.DB.Query(`SELECT client_uid, nickname, xp, level, last_seen FROM users WHERE last_seen < $1`, cutoff)
	if err != nil { return }
	type decayRow struct {
		uid, nick string
		xp, level int
	}
	var batch []decayRow
	for rows.Next() {
		var d decayRow
		if err := rows.Scan(&d.uid, &d.nick, &d.xp, &d.level); err == nil { batch = append(batch, d) }
	}
	_ = rows.Close()
	for _, d := range batch {
		newXP := int(float64(d.xp) * (1.0 - slothDailyDecay))
		if newXP < 0 { newXP = 0 }
		newLevel := leveling.LevelForXP(newXP)
		_, _ = b.DB.Exec("UPDATE users SET xp=$2, level=$3 WHERE client_uid=$1", d.uid, newXP, newLevel)
	}
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
