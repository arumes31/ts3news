package bot

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"sort"
	"strings"
	"time"

	"ts3news/internal/clientquery"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
	"ts3news/internal/leveling"
)

// XP-modifier tuning constants.
const (
	critChance          = 0.05
	critMult            = 3.0
	partyMult           = 1.25
	serverMultPerUser   = 0.05
	serverMultCap       = 2.0
	noGamePenalty       = 0.5
	dailyLoginXP        = 5
	lootBoxEveryLevels  = 25
	lootBoxMin          = 50
	lootBoxMax          = 500
	slothGraceDays      = 7
	slothDailyDecay     = 0.02
	artifactChance      = 0.01
	titleChance         = 0.005
	gearChance          = 0.10
	consChance          = 0.1
	enchChance          = 0.02
	skillChance         = 0.05
	uniqueItemChance    = 0.01  // 1% chance per loot roll
	ultimateSkillChance = 0.005 // 0.5% chance per loot roll
	duraLossPerFight    = 1
	duraLossPenalty     = 3
	occupiedSlotRare    = 0.1
	deathXPPenalty      = 0.05 // 5% XP loss on death
)

type UserInCombat struct {
	UID           string
	Nickname      string
	CLID          int
	Level         int
	Stats         content.Stats
	Skills        []content.Skill
	UltimateSkill *content.UltimateSkill
	CurrentHP     int
	RegenStacks   int
	Gold          int64
	Pets          []*content.Mob
	Equipped      map[content.GearSlot]content.Gear
	Position      content.Position
	STRMod        float64
	DEFMod        float64
	SPDMod        float64
	LootFocus     string // Default "balanced", can be "gold" or "loot"
	FloorModifier string
	IsClone       bool // If true, DB updates are skipped (for co-op)
	// EscrowLoot suppresses inline loot application in the combat engine. The Abyss
	// sets this so drops are not granted mid-run; instead they are rolled into the
	// run's loot escrow (locked until banked, lost on death). See web_abyss_loot.go.
	EscrowLoot bool
}

type activeUser struct {
	u           *UserInCombat
	effects     []content.ItemEffect
	lastSkillID string
	Stunned     bool // scripted boss-phase stun: skips this user's next turn
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
			notes = append(notes, i18n.T("bot.flavour.daily_login", dailyLoginXP))
			b.setLastLogin(uid, ctx.today)
		}
	}

	stats, mult, _, mnotes := b.calculateTotalStats(uid, ctx.today)
	notes = append(notes, mnotes...)

	// Intelligence bonus
	if stats.INT > 0 {
		intMult := 1.0 + float64(stats.INT)/1000.0
		mult *= intMult
		notes = append(notes, i18n.T("bot.flavour.int_bonus", intMult))
	}

	// Flavour stats
	if stats.STN > 50 {
		notes = append(notes, i18n.T("bot.flavour.smell_terrible"))
	}
	if stats.CHA > 100 {
		notes = append(notes, i18n.T("bot.flavour.charming"))
	}
	if stats.SHN > 50 {
		notes = append(notes, i18n.T("bot.flavour.glowing"))
	}

	awardMult := b.computeMiscMult(uid, nickname, cid, ctx)
	if !hasGame {
		mult *= noGamePenalty
		notes = append(notes, i18n.T("bot.flavour.no_game_penalty"))
	}
	award := 0
	if base > 0 {
		award = int(math.Round(float64(base) * mult * awardMult))
		if award < 1 {
			award = 1
		}
	} else {
		// Penalty should NOT be subject to positive multipliers (streak, etc.)
		award = base // base is already negative here
		var curXP, curLevel int
		_ = b.DB.QueryRow("SELECT xp, level FROM users WHERE client_uid=$1", uid).Scan(&curXP, &curLevel)
		
		baseXP := leveling.XPForLevel(curLevel)
		levelProgress := curXP - baseXP
		
		maxLoss := -levelProgress
		if maxLoss > -10 {
			maxLoss = -10 // minimum 10 xp loss
		}
		
		if award < maxLoss {
			award = maxLoss
		}
	}
	delta += award

	lr, err := b.awardXP(uid, nickname, delta)
	if err != nil {
		log.Printf("processUserXP: awardXP failed for %s: %v", nickname, err)
		return &levelResult{}, notes, ""
	}

	if b.Cfg.EnableXPModifiers {
		if lr.NewLevel/lootBoxEveryLevels > lr.OldLevel/lootBoxEveryLevels {
			// Milestone Reward!
			g := content.RandomGearDrop()
			if g.Rarity < content.RarityEpic {
				g.Rarity = content.RarityEpic
			}
			_ = b.awardGearDrop(uid, g)
			
			c := content.RandomConsumable()
			_, _ = b.DB.Exec("INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", uid, c.ID, c.Duration)
			
			notes = append(notes, fmt.Sprintf("🎁 Level %d Milestone Reached! You found a %s and a %s!", (lr.NewLevel/lootBoxEveryLevels)*lootBoxEveryLevels, g.Name, c.Name))
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
	// #nosec G404
	if rand.Float64() < critChance { // #nosec G404
		mult *= critMult
	}
	if sv := serverMultiplier(ctx.onlineNormal); sv > 1 {
		mult *= sv
	}
	if cid >= 0 && ctx.channelNormalCount[cid] > 1 {
		mult *= partyMult
	}

	// Group size XP penalty: Solo players get 100% XP. Groups of 2-4 get a 10% penalty.
	// Groups of 5+ get an additional 5% penalty per extra member (min 50%).
	groupSize := ctx.channelNormalCount[cid]
	if groupSize >= 2 {
		var groupPenalty float64
		if groupSize <= 4 {
			groupPenalty = 0.9 // 10% penalty for small groups
		} else {
			groupPenalty = 0.9 - float64(groupSize-4)*0.05
			if groupPenalty < 0.5 {
				groupPenalty = 0.5
			}
		}
		mult *= groupPenalty
	}

	return mult
}

func (b *Bot) getPets(uid string) []*content.Mob {
	rows, err := b.DB.Query("SELECT name, mob_type, level, hp, max_hp, str, def, spd FROM user_pets WHERE client_uid = $1", uid)
	if err != nil {
		return nil
	}
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
	if u.CurrentHP > 0 {
		return false
	}

	// 1. Check Consumables
	cons := b.getConsumables(u.UID)
	for _, c := range cons {
		if c.Type == content.ConsumableRevive {
			u.CurrentHP = u.Stats.HP / 2
			*logs = append(*logs, i18n.T("bot.combat.revived_item", u.Nickname, c.ID))
			_, _ = b.DB.Exec("DELETE FROM user_consumables WHERE client_uid = $1 AND cons_id = $2", u.UID, c.ID)
			return true
		}
	}
	// 2. Check Item Effects (Phoenix)
	_, _, _, _, effects := b.activeLootMult(u.UID, time.Now())
	for _, eff := range effects {
		if eff == content.EffectPhoenix {
			u.CurrentHP = u.Stats.HP / 2
			*logs = append(*logs, i18n.T("bot.combat.revived_phoenix", u.Nickname))
			return true
		}
	}
	return false
}

func getElementMult(attacker, defender content.Element) float64 {
	// Fire > Air > Earth > Water > Fire
	switch attacker {
	case content.ElementFire:
		if defender == content.ElementAir {
			return 2.0
		}
		if defender == content.ElementWater {
			return 0.5
		}
	case content.ElementAir:
		if defender == content.ElementEarth {
			return 2.0
		}
		if defender == content.ElementFire {
			return 0.5
		}
	case content.ElementEarth:
		if defender == content.ElementWater {
			return 2.0
		}
		if defender == content.ElementAir {
			return 0.5
		}
	case content.ElementWater:
		if defender == content.ElementFire {
			return 2.0
		}
		if defender == content.ElementEarth {
			return 0.5
		}
	}
	return 1.0
}

type LootResult struct {
	UID  string
	Note string
	Poke string
}

// ambushDamageCapPct bounds how much of a player's max HP a surprise round (mobs
// acting before any player has moved) may strip. An ambush can wound but must
// never be a guaranteed kill — combined with the "never below 1 HP during the
// ambush" clamp in mobTurn, every player is guaranteed to act at least once. This
// applies to all combat modes (the channel cycle and the Abyss alike).
const ambushDamageCapPct = 0.5

// bossResistCap is the most boss damage a character can ever shrug off via the
// auto-scaling boss resistance below.
const bossResistCap = 0.80

// bossResist is an automatic, derived "boss resistance" that mitigates damage from
// boss-tier enemies, scaling with the character's level and gear score. Bosses
// should be a war of attrition every player gets to fight, not a coin-flip
// one-shot. It is purely derived (no gear slot / no DB column) so it grows with
// progression on its own, and is hard-capped so bosses still hurt. Applies in all
// combat modes.
func bossResist(level, gearScore int) float64 {
	r := float64(level)*0.0006 + float64(gearScore)*0.00002
	if r < 0 {
		r = 0
	}
	if r > bossResistCap {
		r = bossResistCap
	}
	return r
}

func (b *Bot) resolveChannelCombat(users []UserInCombat, initialMobs []*content.Mob, avgLvl int, diffFactor float64, zone content.Zone) ([]string, int, bool, []LootResult) {
	var logs []string
	var loots []LootResult
	victory := false
	var totalUserDamage, totalMobDamage, totalRewardXP int

	isAbyss := false
	for _, uc := range users {
		if uc.EscrowLoot {
			isAbyss = true
			break
		}
	}

	floorMod := ""
	for _, uc := range users {
		if uc.FloorModifier != "" {
			floorMod = uc.FloorModifier
			break
		}
	}

	// Determine number of waves (1-3)
	// #nosec G404
	waves := 1
	// #nosec G404
	if rand.Float64() < 0.2 {
		waves = 2
	}
	// #nosec G404
	if rand.Float64() < 0.05 {
		waves = 3
	}

	// Later waves spawn at the *encounter's* level, not the party's. In normal
	// combat the initial mobs are already spawned at avgLvl so this is a no-op, but
	// the Abyss spawns a decoupled, depth-scaled mob level — without this, waves 2/3
	// would ignore that and spawn at the (much higher) player level, undoing the
	// Abyss mob-level decoupling and producing instant-kill follow-up waves.
	spawnLvl := avgLvl
	if len(initialMobs) > 0 {
		sum := 0
		for _, m := range initialMobs {
			sum += m.Level
		}
		spawnLvl = sum / len(initialMobs)
		if spawnLvl < 1 {
			spawnLvl = 1
		}
	}

	phaseOnce := make(map[string]bool)
	activeUsers := make([]activeUser, len(users))
	for i := range users {
		_, _, _, _, effects := b.activeLootMult(users[i].UID, time.Now())
		activeUsers[i] = activeUser{u: &users[i], effects: effects}
		activeUsers[i].u.STRMod = 1.0
		activeUsers[i].u.DEFMod = 1.0
		activeUsers[i].u.SPDMod = 1.0
	}

	for w := 1; w <= waves; w++ {
		var currentMobs []*content.Mob
		if w == 1 {
			// Deep copy initial mobs
			currentMobs = make([]*content.Mob, len(initialMobs))
			for i, m := range initialMobs {
				currentMobs[i] = m.Clone()
				currentMobs[i].STRMod = 1.0
				currentMobs[i].DEFMod = 1.0
				currentMobs[i].SPDMod = 1.0
				// Some manually-built mobs (e.g. Abyss scripted bosses) ship without
				// MaxHP set; seed it from base HP so phase-percentage checks are valid.
				if currentMobs[i].MaxHP <= 0 {
					currentMobs[i].MaxHP = currentMobs[i].Stats.HP
					currentMobs[i].CurrentHP = currentMobs[i].MaxHP
				}
				if floorMod == "enraged" {
					currentMobs[i].Effects = append(currentMobs[i].Effects, content.EffectEnraged)
				}
			}
		} else {
			// Spawn new wave
			logs = append(logs, i18n.T("bot.combat.wave_approaches", w))
			newMobs := content.SpawnMobGroup(spawnLvl, zone, diffFactor*zone.Difficulty, len(users), w == 3)
			currentMobs = make([]*content.Mob, len(newMobs))
			for i := range newMobs {
				currentMobs[i] = (&newMobs[i]).Clone()
				currentMobs[i].STRMod = 1.0
				currentMobs[i].DEFMod = 1.0
				currentMobs[i].SPDMod = 1.0
				if floorMod == "enraged" {
					currentMobs[i].Effects = append(currentMobs[i].Effects, content.EffectEnraged)
				}
				initialMobs = append(initialMobs, currentMobs[i]) // track for rewards
			}
		}

		for _, m := range currentMobs {
			totalRewardXP += m.RewardXP
		}

		// Initialize wave header (rarity-coloured enemy names + wave countdown)
		mobCounts := make(map[string]int)
		mobTypes := make(map[string]content.MobType)
		totalEnemyCR := 0
		for _, m := range currentMobs {
			dn := m.DisplayName()
			mobCounts[dn]++
			mobTypes[dn] = m.Type
			totalEnemyCR += m.Score()
		}
		// Iterate in sorted key order so the wave header is stable across runs
		// (Go map iteration order is randomized).
		mobNames := make([]string, 0, len(mobCounts))
		for name := range mobCounts {
			mobNames = append(mobNames, name)
		}
		sort.Strings(mobNames)
		var enemyNames []string
		for _, name := range mobNames {
			count := mobCounts[name]
			display := colorMobName(name, mobTypes[name])
			if count > 1 {
				enemyNames = append(enemyNames, i18n.T("bot.combat.enemy_count", count, display))
			} else {
				enemyNames = append(enemyNames, display)
			}
		}
		// On the opening wave, set the scene with a deterministic zone lore line.
		if w == 1 {
			if lore := zoneLore(zone.Name); lore != "" {
				logs = append(logs, lore)
			}
		}
		logs = append(logs, i18n.T("bot.combat.wave_header", w, totalEnemyCR, strings.Join(enemyNames, ", "), waves))

		// Reset SPD for any stunned mobs from previous round/waves
		for _, m := range currentMobs {
			if m.Stats.SPD == 0 {
				m.Stats.SPD = 10
			}
		}

		// Fight the wave
		waveVictory := false
		// #nosec G404
		playerStarts := rand.IntN(2) == 0 // #nosec G404
		if !playerStarts {
			logs = append(logs, i18n.T("bot.combat.ambush"))
		}

		maxRounds := 10
		enrageRound := 8
		if isAbyss {
			maxRounds = 40
			enrageRound = 30
		}

		for r := 1; r <= maxRounds; r++ {
			intensify := 1.0 + float64(r-1)*0.15
			fatigueMult := 1.0
			if r > 5 {
				fatigueMult = 1.0 - float64(r-5)*0.1
				if fatigueMult < 0.1 {
					fatigueMult = 0.1
				}
			}
			healPenalty := 1.0
			if r >= 6 {
				healPenalty = 1.0 - float64(r-5)*0.2
			}
			if healPenalty < 0 {
				healPenalty = 0
			}
			// Scripted Boss Phases and Soft-Enrage (Item #63, #69, #70)
			for _, m := range currentMobs {
				if m.Stats.HP > 0 {
					// Check soft-enrage past round 8
					if r > enrageRound && m.Type == content.MobBoss {
						hasEnraged := false
						for _, eff := range m.Effects {
							if eff == content.EffectEnraged {
								hasEnraged = true
								break
							}
						}
						if !hasEnraged {
							m.Effects = append(m.Effects, content.EffectEnraged)
							logs = append(logs, fmt.Sprintf("[color=#f44336]⏳ The Abyss closes in! %s becomes ENRAGED! (Double damage)[/color]", m.Name))
						}
					}

					// Custom Boss Scripted Phases — use the live combat HP field
					// (m.Stats.HP, which damage decrements) against MaxHP, not the
					// spawn-time CurrentHP snapshot which never updates here.
					hpPct := float64(m.Stats.HP) / float64(m.MaxHP)
					if m.Name == "Gorgoroth the Firelord" {
						if hpPct < 0.50 {
							hasEnraged := false
							for _, eff := range m.Effects {
								if eff == content.EffectEnraged {
									hasEnraged = true
									break
								}
							}
							if !hasEnraged {
								m.Effects = append(m.Effects, content.EffectEnraged)
								m.Element = content.ElementFire
								logs = append(logs, "[color=#ff3333]🔥 Gorgoroth bellows in fury as his blood boils, wrapping himself in roaring flames! (Gains Enraged & Element shifted to Fire)[/color]")
							}
						}
					} else if m.Name == "Malakor the Voidweaver" {
						if hpPct < 0.50 {
							hasArmored := false
							for _, eff := range m.Effects {
								if eff == content.EffectArmored {
									hasArmored = true
									break
								}
							}
							if !hasArmored {
								// Purge debuffs only; preserve beneficial effects (Regen, etc.)
								var kept []content.MobEffect
								for _, eff := range m.Effects {
									if eff == content.EffectRegen || eff == content.EffectArmored {
										kept = append(kept, eff)
									}
								}
								m.Effects = append(kept, content.EffectArmored)
								logs = append(logs, "[color=#9c27b0]🔮 Malakor wraps himself in a shimmering Void Barrier! (Active debuffs purged, gains Armored)[/color]")
							}
						}
					} else if m.Name == "Azazoth the Slumbering Eye" {
						if hpPct < 0.50 {
							if !phaseOnce["azazoth_stun"] {
								phaseOnce["azazoth_stun"] = true
								for i := range activeUsers {
									activeUsers[i].Stunned = true
								}
								logs = append(logs, "[color=#ffeb3b]👁️ Azazoth opens his slumbering eye, releasing a hypnotic pulse that dazes all delvers! (Skip next turn)[/color]")
							}
						}
					} else if m.Name == "Abyssus, Heart of the Void" {
						if hpPct < 0.75 {
							hasEnraged := false
							for _, eff := range m.Effects {
								if eff == content.EffectEnraged {
									hasEnraged = true
									break
								}
							}
							if !hasEnraged {
								m.Effects = append(m.Effects, content.EffectEnraged)
								m.Element = content.ElementFire
								logs = append(logs, "[color=#ff3333]🔥 Abyssus bellows in fury, wrapping himself in roaring flames! (Gains Enraged & Element shifted to Fire)[/color]")
							}
						}
						if hpPct < 0.50 {
							hasArmored := false
							for _, eff := range m.Effects {
								if eff == content.EffectArmored {
									hasArmored = true
									break
								}
							}
							if !hasArmored {
								m.Effects = append(m.Effects, content.EffectArmored)
								logs = append(logs, "[color=#9c27b0]🔮 Abyssus channels Void shield! (Gains Armored)[/color]")
							}
						}
						if hpPct < 0.25 {
							if !phaseOnce["abyssus_stun"] {
								phaseOnce["abyssus_stun"] = true
								for i := range activeUsers {
									activeUsers[i].Stunned = true
								}
								logs = append(logs, "[color=#ffeb3b]👁️ Abyssus releases a cataclysmic sleep shockwave! (All delvers stunned)[/color]")
							}
						}
					}
				}
			}

			b.applyEffects(activeUsers, currentMobs, zone, r, intensify, healPenalty, &logs)

			if playerStarts {
				b.userTurn(activeUsers, &currentMobs, zone, intensify*fatigueMult, healPenalty, &logs, &totalUserDamage, &totalMobDamage, avgLvl, diffFactor, users, &loots)
				if len(b.getAliveMobs(currentMobs)) == 0 {
					waveVictory = true
					break
				}
				b.mobTurn(activeUsers, currentMobs, zone, intensify, &logs, &totalMobDamage, &totalUserDamage, r, false)
			} else {
				// The opening round of an enemy-first wave is the ambush: soften it so
				// it can't one-shot a player before they ever act.
				b.mobTurn(activeUsers, currentMobs, zone, intensify, &logs, &totalMobDamage, &totalUserDamage, r, r == 1)
				aliveUsers := 0
				for _, u := range users {
					if u.CurrentHP > 0 {
						aliveUsers++
					}
				}
				if aliveUsers == 0 {
					break
				}
				b.userTurn(activeUsers, &currentMobs, zone, intensify*fatigueMult, healPenalty, &logs, &totalUserDamage, &totalMobDamage, avgLvl, diffFactor, users, &loots)
				if len(b.getAliveMobs(currentMobs)) == 0 {
					waveVictory = true
					break
				}
			}

			for _, au := range activeUsers {
				if au.u.UltimateSkill != nil && au.u.UltimateSkill.CurrentCooldown > 0 {
					au.u.UltimateSkill.CurrentCooldown--
				}
			}

			aliveUsers := 0
			for _, u := range users {
				if u.CurrentHP > 0 {
					aliveUsers++
				}
			}
			if aliveUsers == 0 {
				break
			}
		}

		if !waveVictory {
			victory = false
			break
		}
		if w == waves {
			victory = true
		}
	}

	var finalAwardedXP int
	logs, finalAwardedXP, victory = b.distributeRewards(users, activeUsers, victory, totalUserDamage, totalMobDamage, totalRewardXP, initialMobs, nil, zone, logs, avgLvl)
	return logs, finalAwardedXP, victory, loots
}

func (b *Bot) applyEffects(activeUsers []activeUser, mobs []*content.Mob, zone content.Zone, round int, intensify, healPenalty float64, logs *[]string) {
	doubleHazards := false
	for _, au := range activeUsers {
		if strings.Contains(au.u.FloorModifier, "double_hazards") {
			doubleHazards = true
			break
		}
	}
	for _, eff := range zone.Effects {
		if eff.Type == content.ZoneHazard {
			dmg := int(eff.Power * 25 * intensify)
			if doubleHazards {
				dmg *= 2
			}
			if dmg < 1 {
				dmg = 1
			}
			for i := range activeUsers {
				u := activeUsers[i].u
				hasCleanse := false
				for _, ueff := range activeUsers[i].effects {
					if ueff == content.EffectCleanse {
						hasCleanse = true
						break
					}
				}
				if hasCleanse {
					if round == 1 {
						*logs = append(*logs, i18n.T("bot.combat.hazard_cleansed", u.Nickname, eff.Name))
					}
					continue
				}
				u.CurrentHP -= dmg
				if u.CurrentHP <= 0 {
					u.CurrentHP = 0
					if !b.checkUserRevive(u, logs) {
						*logs = append(*logs, i18n.T("bot.combat.hazard_slain", u.Nickname, eff.Name))
					}
				}
			}
			for _, m := range mobs {
				m.Stats.HP -= dmg
			}
			if round == 1 {
				*logs = append(*logs, i18n.T("bot.combat.hazard_active", eff.Name))
			}
		}
	}

	for i := range mobs {
		m := mobs[i]
		if m.Stats.HP <= 0 {
			continue
		}
		// Improvement 4: Status Effect Stacking
		poisonStacks := 0
		regenStacks := 0
		for _, eff := range m.Effects {
			if eff == content.EffectPoisoned {
				poisonStacks++
			}
			if eff == content.EffectRegen {
				regenStacks++
			}
		}

		if poisonStacks > 0 {
			delta := int(float64(m.Stats.HP/20) * float64(poisonStacks) * intensify)
			if delta < 1 {
				delta = 1
			}
			m.Stats.HP -= delta
			if round%3 == 0 {
				*logs = append(*logs, i18n.T("bot.combat.poison_damage", m.Name, delta, poisonStacks))
			}
		}
		if regenStacks > 0 {
			delta := int(float64(m.Stats.HP/20) * float64(regenStacks) * healPenalty)
			if delta < 1 {
				delta = 1
			}
			m.Stats.HP += delta
		}
	}

	for _, au := range activeUsers {
		u := au.u
		if u.CurrentHP <= 0 {
			continue
		}

		// Improvement 40: Scaling Consumables (Auto-use healing if < 50% HP)
		if u.CurrentHP < u.Stats.HP/2 && healPenalty > 0 {
			cons := b.getConsumables(u.UID)
			for _, c := range cons {
				if c.Type == content.ConsumableHealing {
					healAmt := int(float64(u.Stats.HP) * c.EffectValue)
					u.CurrentHP += healAmt
					if u.CurrentHP > u.Stats.HP {
						u.CurrentHP = u.Stats.HP
					}
					*logs = append(*logs, i18n.T("bot.combat.consumable_used", u.Nickname, c.Name, healAmt, c.EffectValue*100))
					// Consume the item
					_, _ = b.DB.Exec("DELETE FROM user_consumables WHERE ctid IN (SELECT ctid FROM user_consumables WHERE client_uid = $1 AND cons_id = $2 LIMIT 1)", u.UID, c.ID)
					break // Only use one potion per round
				}
			}
		}

		// Passive Regen Stacks
		if u.RegenStacks > 0 {
			heal := int(float64(u.RegenStacks*2) * healPenalty)
			u.CurrentHP += heal
			if u.CurrentHP > u.Stats.HP {
				u.CurrentHP = u.Stats.HP
			}
		}
		// Pets Regen
		for _, p := range u.Pets {
			if p.Stats.HP > 0 {
				p.Stats.HP += int(float64(p.Level*2) * healPenalty)
			}
		}
	}
}

// randomLootEligibleUser picks a random non-clone user to receive kill loot so
// co-op clones never trigger DB-persisting loot rolls (gear, gold, consumables,
// pity). Returns nil if no eligible recipient exists.
func randomLootEligibleUser(users []UserInCombat) *UserInCombat {
	var eligible []*UserInCombat
	for i := range users {
		if !users[i].IsClone {
			eligible = append(eligible, &users[i])
		}
	}
	if len(eligible) == 0 {
		return nil
	}
	// #nosec G404 -- non-cryptographic loot recipient selection
	return eligible[rand.IntN(len(eligible))]
}

func (b *Bot) userTurn(activeUsers []activeUser, mobs *[]*content.Mob, zone content.Zone, intensify, healPenalty float64, logs *[]string, totalUserDamage, totalMobDamage *int, avgLvl int, diffFactor float64, originalUsers []UserInCombat, loots *[]LootResult) {
	for i := range activeUsers {
		au := &activeUsers[i]
		u := au.u
		if u.CurrentHP <= 0 {
			continue
		}
		// Scripted boss-phase stun: consume the flag and skip this turn.
		if au.Stunned {
			au.Stunned = false
			*logs = append(*logs, i18n.T("bot.combat.stunned", u.Nickname))
			continue
		}
		if u.Stats.SPD == 0 {
			u.Stats.SPD = 10
			*logs = append(*logs, i18n.T("bot.combat.stunned", u.Nickname))
			continue
		}

		// Zone Buff check
		uSTR := int(float64(u.Stats.STR) * u.STRMod)
		for _, eff := range zone.Effects {
			if eff.Type == content.ZoneBuff {
				uSTR = int(float64(uSTR) * (1.0 + eff.Power))
			}
		}

		// Momentum check (from simulation): 10% chance for 10% STR boost
		// #nosec G404
		if rand.Float64() < 0.1 {
			uSTR = int(float64(uSTR) * 1.1)
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
			if eff == content.EffectVampiric {
				lifesteal += 5
			}
		}

		// #nosec G404
		if multiStrike > 0 && rand.IntN(100) < multiStrike { // #nosec G404
			extraHits = 2
			*logs = append(*logs, i18n.T("bot.combat.double_attack", u.Nickname))
		}

		for h := 0; h < extraHits; h++ {
			aliveMobs := b.getAliveMobs(*mobs)
			if len(aliveMobs) == 0 {
				break
			}
			// #nosec G404
			target := aliveMobs[rand.IntN(len(aliveMobs))] // #nosec G404

			dmgMult := 1.0
			ignoreDef := 0.0
			for _, eff := range au.effects {
				if eff == content.EffectBerserk && u.CurrentHP < u.Stats.HP/2 {
					dmgMult += 0.2
				}
				if eff == content.EffectFragile {
					dmgMult += 0.3
				}
			}

			// #nosec G404
			if len(u.Skills) > 0 && rand.Float64() < 0.3 { // #nosec G404
				// #nosec G404
				s := u.Skills[rand.IntN(len(u.Skills))] // #nosec G404
				dmgMult *= s.Power
				ignoreDef = s.IgnoreDef
				*logs = append(*logs, i18n.T("bot.combat.skill_activation", u.Nickname, s.Name))

				// Combo System (Improvement 6)
				if au.lastSkillID != "" && au.lastSkillID == s.ID {
					dmgMult *= 1.25
					*logs = append(*logs, i18n.T("bot.combat.combo", s.Name))
				}
				au.lastSkillID = s.ID

				// #nosec G404
				if s.StunChance > 0 && rand.Float64() < s.StunChance { // #nosec G404
					*logs = append(*logs, i18n.T("bot.combat.stunned", target.Name))
					target.Stats.SPD = 0
				}
			} else {
				au.lastSkillID = "" // Reset combo if no skill used
			}

			// Elemental System (Improvement 1)
			// Determine user's active element from MainHand
			userElement := content.ElementPhysical
			if mh, ok := u.Equipped[content.SlotMainHand]; ok {
				userElement = mh.Element
			}
			elementMult := getElementMult(userElement, target.Element)
			if elementMult > 1.0 {
				*logs = append(*logs, i18n.T("bot.combat.element_effective", userElement, target.Element))
			} else if elementMult < 1.0 {
				*logs = append(*logs, i18n.T("bot.combat.element_weak", userElement, target.Element))
			}
			dmgMult *= elementMult

			// Position Bonus (Improvement 2)
			if u.Position == content.PositionBackline {
				dmgMult *= 1.10 // 10% damage bonus for backline
			}

			// Ultimate Skill activation
			if u.UltimateSkill != nil && u.UltimateSkill.CurrentCooldown == 0 {
				dmgMult *= u.UltimateSkill.Power
				*logs = append(*logs, i18n.T("bot.combat.ultimate_activation", u.UltimateSkill.Name))
				u.UltimateSkill.CurrentCooldown = u.UltimateSkill.CooldownRounds
			}

			effDef := float64(target.Stats.DEF) * target.DEFMod * (1.0 - ignoreDef)
			dmg := int((float64(uSTR)*dmgMult - effDef) * intensify)

			// Percentage-Based Damage Floor (15% of STR) to prevent DEF stalemates
			minDmg := int(float64(uSTR) * 0.15 * intensify)
			if dmg < minDmg {
				dmg = minDmg
			}
			if dmg < 1 {
				dmg = 1
			}

			// Daily affix: Execute — strikes land 50% harder on targets below 30% HP.
			if strings.Contains(u.FloorModifier, "execute") && target.MaxHP > 0 && target.Stats.HP*10 < target.MaxHP*3 {
				dmg = dmg * 3 / 2
			}

			target.Stats.HP -= dmg
			*totalUserDamage += dmg

			// Chain Attack Logic for groups (3+ players)
			// #nosec G404
			if len(originalUsers) >= 3 && rand.Float64() < 0.3 { // #nosec G404
				others := b.getAliveMobs(*mobs)
				if len(others) > 1 {
					var chainTarget *content.Mob
					for _, xm := range others {
						if xm != target {
							chainTarget = xm
							break
						}
					}
					if chainTarget != nil {
						chainDmg := dmg / 2
						if chainDmg < 1 {
							chainDmg = 1
						}
						chainTarget.Stats.HP -= chainDmg
						*totalUserDamage += chainDmg
					}
				}
			}

			// Mind Control Logic (Scale with level)
			if mindControlLevel > 0 && len(u.Pets) < mindControlLevel && target.Stats.HP > 0 && float64(target.Stats.HP) < float64(target.Level*20)*0.2 {
				// #nosec G404
				if rand.Float64() < 0.5 { // #nosec G404
					*logs = append(*logs, i18n.T("bot.combat.captive", target.Name))
					u.Pets = append(u.Pets, target)
					b.savePet(u.UID, target)
					target.Stats.HP = target.Level * 10
					newMobs := []*content.Mob{}
					for _, xm := range *mobs {
						if xm != target {
							newMobs = append(newMobs, xm)
						}
					}
					*mobs = newMobs
				}
			}

			if lifesteal > 0 {
				heal := int(float64(dmg) * float64(lifesteal) / 100.0 * healPenalty)
				if heal > 0 {
					u.CurrentHP += heal
					if u.CurrentHP > u.Stats.HP {
						u.CurrentHP = u.Stats.HP
					}
				}
			}

			if target.Stats.HP <= 0 {
				*logs = append(*logs, i18n.T("bot.combat.defeated", target.Name, u.Nickname))
				// Weekly affix: Bloodlust heals the slayer for 20% of max HP on a kill.
				if strings.Contains(u.FloorModifier, "bloodlust") {
					u.CurrentHP += u.Stats.HP / 5
					if u.CurrentHP > u.Stats.HP {
						u.CurrentHP = u.Stats.HP
					}
				}
				// Award loot for every mob defeated, regardless of final outcome.
				// Clones (co-op helpers) are excluded so loot never persists for them.
				if winner := randomLootEligibleUser(originalUsers); winner != nil {
					b.awardCombatLoot(winner, *target, zone, logs, loots)
				}
				b.handleDeathEffects(target, mobs, logs, avgLvl, diffFactor, activeUsers)
			}
			if len(b.getAliveMobs(*mobs)) == 0 {
				break
			}
		}

		// Pet Attack (Silent damage)
		for _, p := range u.Pets {
			if p.Stats.HP <= 0 {
				continue
			}

			// Betrayal check (3% chance)
			// #nosec G404
			if rand.Float64() < 0.03 { // #nosec G404
				// #nosec G404
				targetAU := activeUsers[rand.IntN(len(activeUsers))] // #nosec G404
				target := targetAU.u
				if target.CurrentHP > 0 {
					pdmg := int(float64(p.Stats.STR-target.Stats.DEF) * intensify)
					if pdmg < 1 {
						pdmg = 1
					}
					target.CurrentHP -= pdmg
					*logs = append(*logs, i18n.T("bot.combat.rogue_pet_bite", p.Name, target.Nickname, pdmg))
					*totalMobDamage += pdmg
					if target.CurrentHP <= 0 {
						target.CurrentHP = 0
						if !b.checkUserRevive(target, logs) {
							*logs = append(*logs, i18n.T("bot.combat.slain_by_pet", target.Nickname, p.Name))
						}
					}
					continue
				}
			}

			aliveMobs := b.getAliveMobs(*mobs)
			if len(aliveMobs) == 0 {
				break
			}
			// #nosec G404
			ptarget := aliveMobs[rand.IntN(len(aliveMobs))] // #nosec G404
			pdmg := int(float64(p.Stats.STR-ptarget.Stats.DEF) * intensify)
			if pdmg < 1 {
				pdmg = 1
			}
			ptarget.Stats.HP -= pdmg
			*totalUserDamage += pdmg
			if ptarget.Stats.HP <= 0 {
				*logs = append(*logs, i18n.T("bot.combat.killed_by_pet", ptarget.Name, p.Name))
				// Clones (co-op helpers) are excluded so loot never persists for them.
				if winner := randomLootEligibleUser(originalUsers); winner != nil {
					b.awardCombatLoot(winner, *ptarget, zone, logs, loots)
				}
				b.handleDeathEffects(ptarget, mobs, logs, avgLvl, diffFactor, activeUsers)
			}
		}

		if len(b.getAliveMobs(*mobs)) == 0 {
			break
		}
	}
}

func (b *Bot) mobTurn(activeUsers []activeUser, mobs []*content.Mob, zone content.Zone, intensify float64, logs *[]string, totalMobDamage, totalUserDamage *int, round int, ambush bool) {
	// Ambush softening (all modes): track a per-target damage budget so a surprise
	// round can strip at most ambushDamageCapPct of each player's max HP, and clamp
	// it below so the ambush can never land the killing blow. Guarantees every
	// player gets to act at least once. Keyed by *UserInCombat so co-op partners
	// each get their own budget.
	var ambushBudget map[*UserInCombat]int
	if ambush {
		ambushBudget = make(map[*UserInCombat]int, len(activeUsers))
		for _, au := range activeUsers {
			budget := int(float64(au.u.Stats.HP) * ambushDamageCapPct)
			if budget < 1 {
				budget = 1
			}
			ambushBudget[au.u] = budget
		}
	}
	for _, m := range mobs {
		if m.Stats.HP <= 0 || m.Stats.SPD == 0 {
			if m.Stats.SPD == 0 {
				m.Stats.SPD = 10
			} // recover
			continue
		}

		// Positional Combat: Prioritize Frontline (Improvement 2)
		var potentialTargets []activeUser
		for _, au := range activeUsers {
			if au.u.CurrentHP > 0 && au.u.Position == content.PositionFrontline {
				potentialTargets = append(potentialTargets, au)
			}
		}
		// If no frontline, target anyone
		if len(potentialTargets) == 0 {
			for _, au := range activeUsers {
				if au.u.CurrentHP > 0 {
					potentialTargets = append(potentialTargets, au)
				}
			}
		}

		if len(potentialTargets) == 0 {
			continue
		}

		// #nosec G404
		targetAU := potentialTargets[rand.IntN(len(potentialTargets))] // #nosec G404
		target := targetAU.u

		// Physical Evasion for Backline
		if target.Position == content.PositionBackline && m.Element == content.ElementPhysical {
			// #nosec G404
			if rand.Float64() < 0.5 { // 50% extra miss chance for physical mobs vs backline
				*logs = append(*logs, i18n.T("bot.combat.stealth_shadow", target.Nickname, m.Name))
				continue
			}
		}

		// Task 60: Stealth check - skip first round mob attacks
		hasStealth := false
		for _, eff := range targetAU.effects {
			if eff == content.EffectStealth {
				hasStealth = true
				break
			}
		}
		if round == 1 && hasStealth {
			continue
		}

		// Task 63: Parry check - 10% chance to take 0 damage and counter
		hasParry := false
		for _, eff := range targetAU.effects {
			if eff == content.EffectParry {
				hasParry = true
				break
			}
		}
		// #nosec G404
		if hasParry && rand.IntN(100) < 10 { // #nosec G404
			*logs = append(*logs, i18n.T("bot.combat.parried", target.Nickname, m.Name))
			counterDmg := int(float64(target.Stats.STR) * 0.5 * intensify)
			if counterDmg < 1 {
				counterDmg = 1
			}
			m.Stats.HP -= counterDmg
			*totalUserDamage += counterDmg
			continue
		}

		// #nosec G404
		// Dodge check - capped at 25%
		dodgeChance := target.Stats.DGE
		if dodgeChance > 25 {
			dodgeChance = 25
		}
		if rand.IntN(100) < dodgeChance { // #nosec G404
			continue
		} // #nosec G404

		dmgMult := 1.0
		// #nosec G404
		if len(m.Spells) > 0 && rand.Float64() < 0.2 { // #nosec G404
			// #nosec G404
			s := m.Spells[rand.IntN(len(m.Spells))] // #nosec G404
			dmgMult = s.Power
			*logs = append(*logs, i18n.T("bot.combat.cast_spell", m.Name, s.Name))
		}

		// Elemental System (Improvement 1)
		targetElement := content.ElementPhysical
		// Determine user's defensive element from Chest/OffHand
		if ch, ok := target.Equipped[content.SlotChest]; ok {
			targetElement = ch.Element
		}
		elementMult := getElementMult(m.Element, targetElement)
		dmgMult *= elementMult

		// Treasure Goblin Flee Logic
		if m.Type == content.MobTreasureGoblin && round >= 3 {
			// #nosec G404
			if rand.Float64() < 0.3 {
				*logs = append(*logs, i18n.T("bot.combat.goblin_flee"))
				m.Stats.HP = 0 // Remove from combat
				continue
			}
		}

		mSTR := int(float64(m.Stats.STR) * m.STRMod)
		// Zone Debuff check
		for _, eff := range zone.Effects {
			if eff.Type == content.ZoneDebuff {
				mSTR = int(float64(mSTR) * (1.0 - eff.Power))
			}
		}

		for _, eff := range m.Effects {
			switch eff {
			case content.EffectEnraged:
				mSTR = int(float64(mSTR) * 1.5)
			case content.EffectWeakened:
				mSTR = int(float64(mSTR) * 0.5)
			}
		}

		dmg := int((float64(mSTR)*dmgMult - float64(target.Stats.DEF)*target.DEFMod) * intensify)

		// Frontline Defense Bonus (Improvement 2)
		if target.Position == content.PositionFrontline {
			dmg = int(float64(dmg) * 0.9) // 10% damage reduction for frontline
		}

		// Percentage-Based Damage Floor (15% of STR)
		minDmg := int(float64(mSTR) * 0.15 * intensify)
		if dmg < minDmg {
			dmg = minDmg
		}
		if dmg < 1 {
			dmg = 1
		}

		for _, eff := range m.Effects {
			// #nosec G404
			if eff == content.EffectBlinded && rand.Float64() < 0.5 {
				dmg = 0
			} // #nosec G404
		}

		// Boss resistance: shrink hits from boss-tier enemies by the target's
		// auto-scaling resistance so a boss can't erase a full-HP, well-geared
		// character in a single blow. Trash mobs are unaffected.
		if m.Type == content.MobBoss || m.Type == content.MobLegendary {
			if resist := bossResist(target.Level, target.Stats.Score()); resist > 0 {
				dmg = int(float64(dmg) * (1.0 - resist))
				if dmg < 0 {
					dmg = 0
				}
			}
		}

		// Ambush cap: limit total surprise-round damage to this target and never let
		// it reduce them below 1 HP, so a dense enemy group can't erase a full-HP
		// character before their first action.
		if ambush {
			if budget, ok := ambushBudget[target]; ok {
				if dmg > budget {
					dmg = budget
				}
				ambushBudget[target] = budget - dmg
			}
			if dmg >= target.CurrentHP {
				dmg = target.CurrentHP - 1
			}
			if dmg < 0 {
				dmg = 0
			}
		}

		// Daily affix: Iron Skin shaves 30% off every hit a delver takes.
		if strings.Contains(target.FloorModifier, "iron_skin") {
			dmg = dmg * 7 / 10
		}

		target.CurrentHP -= dmg
		*totalMobDamage += dmg

		// Daily affix: Vampiric mobs heal for 15% of the damage they deal.
		if dmg > 0 && strings.Contains(target.FloorModifier, "vampiric_mobs") {
			m.Stats.HP += dmg * 15 / 100
			if m.MaxHP > 0 && m.Stats.HP > m.MaxHP {
				m.Stats.HP = m.MaxHP
			}
		}

		// Check Revival
		if target.CurrentHP <= 0 {
			if !b.checkUserRevive(target, logs) {
				*logs = append(*logs, i18n.T("bot.combat.slain_by_mob", target.Nickname, m.Name))
			}
		}

		for _, eff := range targetAU.effects {
			if eff == content.EffectThorns && dmg > 0 {
				reflect := dmg / 10
				if reflect < 1 {
					reflect = 1
				}
				m.Stats.HP -= reflect
				*totalUserDamage += reflect
			}
		}
	}
}

func (b *Bot) distributeRewards(users []UserInCombat, activeUsers []activeUser, victory bool, totalUserDamage, totalMobDamage, totalRewardXP int, initialMobs []*content.Mob, mobs []*content.Mob, zone content.Zone, logs []string, avgLvl int) ([]string, int, bool) {
	// Summarize Combat — centred header plus visual damage-share bars.
	totalDamage := totalUserDamage + totalMobDamage
	logs = append(logs, hr())
	logs = append(logs, centerHeader(i18n.T("bot.combat.summary_title")))
	logs = append(logs, i18n.T("bot.combat.summary_party", colorHeal(totalUserDamage), damageBar(totalUserDamage, totalDamage)))
	logs = append(logs, i18n.T("bot.combat.summary_mobs", colorDmg(totalMobDamage), damageBar(totalMobDamage, totalDamage)))

	// Update pity, quests, consumables AND persistent stats
	for i := range users {
		u := &users[i]
		// Save pets state
		if !u.IsClone {
			for _, p := range u.Pets {
				b.updatePetHP(u.UID, p.Name, p.Stats.HP)
			}
		}

		finalXP := 0
		if victory {
			if !u.IsClone {
				_, _ = b.DB.Exec("UPDATE users SET consecutive_losses = 0 WHERE client_uid = $1", u.UID)
				b.updateQuest(u.UID, "mobs_killed", len(initialMobs))
			}

			// Regen Stacks logic
			hasRegEffect := false
			_, _, _, _, effects := b.activeLootMult(u.UID, time.Now())
			for _, eff := range effects {
				if eff == content.EffectRegenStack {
					hasRegEffect = true
				}
			}
			if hasRegEffect {
				u.RegenStacks++
			}

		} else {
			if !u.IsClone {
				_, _ = b.DB.Exec("UPDATE users SET consecutive_losses = consecutive_losses + 1 WHERE client_uid = $1", u.UID)
				// Death Penalty: 25% of the XP required for the current level
				var curXP, curLevel int
				_ = b.DB.QueryRow("SELECT xp, level FROM users WHERE client_uid=$1", u.UID).Scan(&curXP, &curLevel)

				baseXP := leveling.XPForLevel(curLevel)
				nextXP := leveling.XPForLevel(curLevel + 1)
				levelSize := nextXP - baseXP

				penalty := int(float64(levelSize) * 0.25)
				if penalty < 10 {
					penalty = 10
				}

				finalXP -= penalty

				// Increase jackpot by 1% of lost XP value
				b.incrementJackpot("global", int64(penalty))

				logs = append(logs, deathPenaltyLine(u.Nickname, penalty))
			}
			u.CurrentHP = 0   // dead
			u.RegenStacks = 0 // lose stacks on death
		}

		// Gold Drop logic
		goldDrop := 0
		if victory {
			// Economic Inflation (Improvement 44)
			var totalGold int64
			_ = b.DB.QueryRow("SELECT SUM(gold) FROM users").Scan(&totalGold)
			inflationMult := 1.0
			if totalGold > 10000000 { // 10M Gold threshold
				inflationMult = 1.0 / (1.0 + float64(totalGold-10000000)/5000000.0)
			}

			for _, m := range initialMobs {
				baseGold := m.Level * 2
				switch m.Type {
				case content.MobBoss, content.MobLegendary:
					baseGold = m.Level * 10
				case content.MobTreasureGoblin:
					baseGold = m.Level * 25
				}
				// #nosec G404
				goldDrop += int(float64(baseGold) * (0.8 + rand.Float64()*0.4) * inflationMult)
			}
			
			// First Win of the Day Bonus
			if !u.IsClone {
				var lastWin sql.NullTime
				_ = b.DB.QueryRow("SELECT last_win FROM users WHERE client_uid=$1", u.UID).Scan(&lastWin)

				if !lastWin.Valid || lastWin.Time.Before(time.Now().Add(-24*time.Hour)) {
					// 5x Gold and XP multiplier for First Win
					goldDrop *= 5
					finalXP *= 5
					logs = append(logs, "🌟 FIRST WIN OF THE DAY! (5x Gold & XP)")
					_, _ = b.DB.Exec("UPDATE users SET last_win=NOW() WHERE client_uid=$1", u.UID)
				}
			}

			// VIP Gold Bonus
			vip, _ := b.getVIP(u.UID)
			if vip.Bonus > 0 {
				goldDrop = int(float64(goldDrop) * (1.0 + float64(vip.Bonus)/100.0))
			}

			u.Gold += int64(goldDrop)
		}

		if !u.IsClone {
			// Save ultimate skill cooldown state
			if u.UltimateSkill != nil {
				_, _ = b.DB.Exec("UPDATE users SET ultimate_cooldown = $2 WHERE client_uid = $1", u.UID, u.UltimateSkill.CurrentCooldown)
			}

			_, _ = b.DB.Exec("UPDATE users SET current_hp = $2, regen_stacks = $3, gold = users.gold + $4 WHERE client_uid = $1", u.UID, u.CurrentHP, u.RegenStacks, int64(goldDrop))

			_, _ = b.DB.Exec("UPDATE user_consumables SET remaining_fights = remaining_fights - 1 WHERE client_uid = $1", u.UID)
			_, _ = b.DB.Exec("DELETE FROM user_consumables WHERE client_uid = $1 AND remaining_fights < 0", u.UID)
		}

		if finalXP > 0 {
			// Improvement 24: Dynamic Level Scaling
			// Penalize high level players in low level zones
			if u.Level > avgLvl+20 {
				penalty := float64(u.Level-(avgLvl+20)) * 0.1
				if penalty > 1.0 {
					penalty = 1.0
				}
				finalXP = int(float64(finalXP) * (1.0 - penalty))
				if finalXP < 0 {
					finalXP = 0
				}
			}

			// Apply gear XP multipliers to combat rewards
			mult, _, _, _, _ := b.activeLootMult(u.UID, time.Now())
			if mult > 1.0 {
				finalXP = int(float64(finalXP) * mult)
			}
		}
		if finalXP != 0 && !u.IsClone {
			_, _ = b.awardXP(u.UID, "", finalXP)
		}
	}

	if victory {
		logs = append(logs, i18n.T("bot.combat.victory", len(initialMobs), zone.Name))

		// 1% chance for global jackpot on victory
		// #nosec G404
		if rand.Float64() < 0.01 {
			// Find a winner among participants
			// #nosec G404
			winner := users[rand.IntN(len(users))]
			jackpot := b.claimJackpot(winner.UID, "global")
			if jackpot > 0 {
				logs = append(logs, "🔥 GLOBAL JACKPOT WIN! "+winner.Nickname+" won "+FormatGold(jackpot)+"!")
			}
		}

		return logs, totalRewardXP / realUserCount(users), true
	}
	logs = append(logs, i18n.T("bot.combat.defeat", zone.Name))
	return logs, -totalRewardXP / (2 * realUserCount(users)), false
}

// realUserCount counts non-clone participants. Co-op clones must not dilute the
// combat XP share, since only real delvers actually receive it. Always >= 1.
func realUserCount(users []UserInCombat) int {
	n := 0
	for i := range users {
		if !users[i].IsClone {
			n++
		}
	}
	if n < 1 {
		n = 1
	}
	return n
}

func (b *Bot) getAliveMobs(mobs []*content.Mob) []*content.Mob {
	var out []*content.Mob
	for _, m := range mobs {
		if m.Stats.HP > 0 {
			out = append(out, m)
		}
	}
	return out
}

func (b *Bot) handleDeathEffects(m *content.Mob, mobs *[]*content.Mob, logs *[]string, avgLvl int, diffFactor float64, users []activeUser) {
	if m.DeathEffect == nil {
		return
	}

	*logs = append(*logs, i18n.T("bot.combat.death_trigger", m.Name, m.DeathEffect.Type, m.DeathEffect.Name))

	switch m.DeathEffect.Type {
	case content.DeathSummon:
		count := 1
		if m.Type == content.MobCommon {
			count = 3
		} // Trash mobs summon hordes
		for i := 0; i < count; i++ {
			// Summoned mobs are lower tier
			lvl := avgLvl - 5
			if lvl < 1 {
				lvl = 1
			}
			newMob := content.SpawnMob(lvl, false, diffFactor*0.7)
			newMob.Name = "Summoned " + newMob.Name
			*mobs = append(*mobs, &newMob)
		}
		*logs = append(*logs, i18n.N("bot.combat.reinforcements", count))

	case content.DeathExplosion:
		dmg := m.Level * 10
		*logs = append(*logs, i18n.T("bot.combat.explosion_damage", dmg))
		for i := range users {
			target := users[i].u
			if target.CurrentHP <= 0 {
				continue
			}
			target.CurrentHP -= dmg
			if target.CurrentHP <= 0 {
				target.CurrentHP = 0
				if !b.checkUserRevive(target, logs) {
					*logs = append(*logs, i18n.T("bot.combat.slain_by_explosion", target.Nickname))
				}
			}
		}

	case content.DeathCurse:
		for i := range users {
			users[i].u.Stats.STR -= 10
			users[i].u.Stats.DEF -= 5
		}
		*logs = append(*logs, i18n.T("bot.combat.curse_weakens"))

	case content.DeathXP:
		*logs = append(*logs, i18n.T("bot.combat.bonus_xp_pulse"))

	case content.DeathLoot:
		*logs = append(*logs, i18n.T("bot.combat.shiny_items"))
	}
}

func (b *Bot) getConsumables(uid string) []content.Consumable {
	rows, err := b.DB.Query("SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid = $1", uid)
	if err != nil {
		return nil
	}
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
	case streak >= 7:
		return 2.0
	case streak >= 5:
		return 1.5
	case streak >= 3:
		return 1.25
	default:
		return 1.0
	}
}

func serverMultiplier(onlineNormal int) float64 {
	humans := onlineNormal - 1
	if humans < 1 {
		humans = 1
	}
	// Simulation-tuned base: 1.5x for any human presence
	m := 1.5 + serverMultPerUser*float64(humans-1)
	if m > serverMultCap {
		m = serverMultCap
	}
	return m
}

func (b *Bot) updateStreak(uid string, today time.Time) int {
	var last sql.NullTime
	var streak int
	if err := b.DB.QueryRow("SELECT last_poke_date, streak_days FROM users WHERE client_uid=$1", uid).Scan(&last, &streak); err != nil {
		return 0
	}
	if last.Valid && sameDay(last.Time, today) {
		return streak
	}
	if last.Valid && sameDay(last.Time, today.AddDate(0, 0, -1)) {
		streak++
	} else {
		streak = 1
	}
	_, _ = b.DB.Exec("UPDATE users SET streak_days=$2, last_poke_date=$3 WHERE client_uid=$1", uid, streak, today)
	return streak
}

func (b *Bot) dailyLoginDue(uid string, today time.Time) bool {
	var last sql.NullTime
	if err := b.DB.QueryRow("SELECT last_login_date FROM users WHERE client_uid=$1", uid).Scan(&last); err != nil {
		return false
	}
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
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	equipped := make(map[string]bool)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err == nil {
			equipped[s] = true
		}
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

func (b *Bot) applyDurabilityLoss(uid string, defeat bool) []string {
	var warnings []string
	var stats content.Stats
	var effects []content.ItemEffect
	_, stats, _, _, effects = b.activeLootMult(uid, time.Now())

	// Check for repair consumables and apply before durability loss
	consRows, err := b.DB.Query("SELECT cons_id FROM user_consumables WHERE client_uid = $1 AND cons_id IN ('P6','P7')", uid)
	if err == nil {
		var repairIDs []string
		for consRows.Next() {
			var cid string
			if err := consRows.Scan(&cid); err == nil {
				repairIDs = append(repairIDs, cid)
			}
		}
		_ = consRows.Close()

		for _, cid := range repairIDs {
			var repairAmt int
			switch cid {
			case "P6":
				repairAmt = 30
			case "P7":
				repairAmt = 75
			}

			// Only repair if there's actually broken gear
			var brokenCount int
			_ = b.DB.QueryRow("SELECT COUNT(*) FROM user_gear WHERE client_uid = $1 AND durability < max_durability", uid).Scan(&brokenCount)
			if brokenCount > 0 {
				// Apply repair to all damaged gear (spread evenly)
				_, _ = b.DB.Exec("UPDATE user_gear SET durability = LEAST(durability + $2, max_durability) WHERE client_uid = $1 AND durability < max_durability", uid, repairAmt/brokenCount)
				// Also repair artifact
				_, _ = b.DB.Exec("UPDATE users SET artifact_durability = LEAST(artifact_durability + 15, 30) WHERE client_uid = $1 AND artifact_durability > 0 AND artifact_durability < 30", uid)
				// Consume one repair kit
				_, _ = b.DB.Exec("DELETE FROM user_consumables WHERE ctid IN (SELECT ctid FROM user_consumables WHERE client_uid = $1 AND cons_id = $2 LIMIT 1)", uid, cid)
			}
		}
	}

	// Fragile check
	lossMult := 1
	for _, eff := range effects {
		if eff == content.EffectFragile {
			lossMult = 2
		}
	}

	// #nosec G404
	if rand.IntN(100) < stats.STA {
		return warnings
	} // #nosec G404

	baseLoss := duraLossPerFight * lossMult
	if defeat {
		baseLoss = duraLossPenalty * lossMult
	}

	// XP items lose durability faster: the higher the XP multiplier, the faster they decay.
	// This ensures powerful XP-boosting gear is a tradeoff (more XP but shorter lifespan).
	grows, gerr := b.DB.Query("SELECT gear_id, enchantment_id FROM user_gear WHERE client_uid = $1", uid)
	if gerr == nil {
		type gearLoss struct {
			gearID string
			loss   int
		}
		var losses []gearLoss
		for grows.Next() {
			var gearID string
			var enchID sql.NullString
			if grows.Scan(&gearID, &enchID) == nil {
				itemLoss := baseLoss
				if gear, ok := content.GetGearByID(gearID); ok {
					// Gear with XP multiplier > 1.0 loses extra durability proportional to the bonus
					if gear.XPMultiplier > 1.0 {
						xpPenalty := int((gear.XPMultiplier - 1.0) * 10) // e.g. 1.30x = +3 extra loss
						itemLoss += xpPenalty
					}
				}
				if enchID.Valid && enchID.String != "" {
					if ench, ok := content.GetEnchantmentByID(enchID.String); ok {
						// Enchantments with XP multiplier also increase durability loss
						if ench.XPMultiplier > 1.0 {
							xpPenalty := int((ench.XPMultiplier - 1.0) * 10)
							itemLoss += xpPenalty
						}
					}
				}
				losses = append(losses, gearLoss{gearID: gearID, loss: itemLoss})
			}
		}
		_ = grows.Close()

		// Apply individual durability losses
		for _, gl := range losses {
			var oldDura int
			_ = b.DB.QueryRow("SELECT durability FROM user_gear WHERE client_uid = $1 AND gear_id = $2", uid, gl.gearID).Scan(&oldDura)
			
			_, _ = b.DB.Exec("UPDATE user_gear SET durability = durability - $2 WHERE client_uid = $1 AND gear_id = $3", uid, gl.loss, gl.gearID)
			
			if gl.loss > baseLoss*2 && gl.loss >= 10 {
				if gear, ok := content.GetGearByID(gl.gearID); ok {
					warnings = append(warnings, fmt.Sprintf("⚠️ Your %s took heavy damage (-%d durability)!", gear.Name, gl.loss))
				}
			}
			
			if oldDura > 0 && oldDura-gl.loss <= 0 {
				if gear, ok := content.GetGearByID(gl.gearID); ok {
					warnings = append(warnings, fmt.Sprintf("💥 Your %s shattered into pieces!", gear.Name))
				}
			} else if oldDura > 10 && oldDura-gl.loss <= 10 {
				if gear, ok := content.GetGearByID(gl.gearID); ok {
					warnings = append(warnings, fmt.Sprintf("⚠️ Your %s is badly damaged and will break soon!", gear.Name))
				}
			}
		}
	} else {
		// Fallback: uniform loss if query fails
		_, _ = b.DB.Exec("UPDATE user_gear SET durability = durability - $2 WHERE client_uid = $1", uid, baseLoss)
	}

	_, _ = b.DB.Exec("DELETE FROM user_gear WHERE client_uid = $1 AND durability <= 0", uid)
	
	// Artifact break check
	var oldArtDura int
	var artName sql.NullString
	_ = b.DB.QueryRow("SELECT artifact_durability, artifact_name FROM users WHERE client_uid = $1", uid).Scan(&oldArtDura, &artName)
	
	_, _ = b.DB.Exec("UPDATE users SET artifact_durability = artifact_durability - $2 WHERE client_uid = $1 AND artifact_durability > 0", uid, baseLoss)
	
	if oldArtDura > 0 && oldArtDura-baseLoss <= 0 && artName.Valid && artName.String != "" {
		warnings = append(warnings, fmt.Sprintf("💥 Your %s shattered into pieces!", artName.String))
	} else if oldArtDura > 10 && oldArtDura-baseLoss <= 10 && artName.Valid && artName.String != "" {
		warnings = append(warnings, fmt.Sprintf("⚠️ Your %s is badly damaged and will break soon!", artName.String))
	}
	
	_, _ = b.DB.Exec("UPDATE users SET artifact_mult=1, artifact_name=NULL, artifact_durability=0 WHERE client_uid=$1 AND artifact_durability <= 0 AND artifact_name IS NOT NULL", uid)
	
	return warnings
}

func (b *Bot) calculateTotalStats(uid string, today time.Time) (content.Stats, float64, int, []string) {
	var level, prestige int
	_ = b.DB.QueryRow("SELECT level, prestige FROM users WHERE client_uid=$1", uid).Scan(&level, &prestige)
	base := content.Stats{
		HP: 100 + level*5, STR: 10 + level, DEF: 5 + level/2, SPD: 10 + level, LCK: level / 5,
		INT: level / 10, STA: level / 10, CRT: 5 + level/50, DGE: 5 + level/50,
	}

	// Apply Prestige Bonus
	if prestige > 0 {
		pMult := 1.0 + (float64(prestige) * prestigeStatBonus)
		base.HP = int(float64(base.HP) * pMult)
		base.STR = int(float64(base.STR) * pMult)
		base.DEF = int(float64(base.DEF) * pMult)
		base.SPD = int(float64(base.SPD) * pMult)
	}

	mult, lootStats, gearScore, notes, effects := b.activeLootMult(uid, today)
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

	return totalStats, mult, gearScore, notes
}

func (b *Bot) activeLootMult(uid string, today time.Time) (float64, content.Stats, int, []string, []content.ItemEffect) {
	mult := 1.0
	var stats content.Stats
	var notes []string
	var effects []content.ItemEffect
	var gearScore int

	var title sql.NullString
	var tMult sql.NullFloat64
	var tExp sql.NullTime
	if err := b.DB.QueryRow("SELECT title, title_mult, title_expires FROM users WHERE client_uid=$1", uid).Scan(&title, &tMult, &tExp); err == nil {
		if tExp.Valid && !today.After(tExp.Time) && title.Valid {
			mult *= tMult.Float64
			notes = append(notes, i18n.T("bot.loot.multiplier_simple", title.String, tMult.Float64))
			if t, ok := content.GetTitleByName(title.String); ok {
				stats = stats.Add(t.Stats)
			}
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
			notes = append(notes, i18n.T("bot.loot.multiplier_with_dur", aName.String, aMult.Float64, int(aDura)))
			if art, ok := content.GetArtifactByName(aName.String); ok {
				stats = stats.Add(art.Stats)
				gearScore += art.Stats.Score()
				if art.Special != content.EffectNone {
					effects = append(effects, art.Special)
				}
			}
		}
	}
	// Calculate gear XP multiplier
	// Only Rare+ items provide XP bonuses (Common/Uncommon have 1.0-1.05x)
	// Max possible from gear: 30 slots × 1.30x = ~2600x (capped by rarity distribution)
	abyssSetPieces := 0
	rows, err := b.DB.Query("SELECT gear_id, durability, enchantment_id FROM user_gear WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var gearID string
			var dura int
			var enchID sql.NullString
			if err := rows.Scan(&gearID, &dura, &enchID); err == nil {
				if content.IsAbyssGearID(gearID) {
					abyssSetPieces++
				}
				if gear, ok := content.GetGearByID(gearID); ok {
					// Define which slots can have high XP multipliers (more than 20%)
					highXPSlots := map[content.GearSlot]bool{
						content.SlotMainHand: true,
						content.SlotChest:    true,
						content.SlotHead:     true,
						content.SlotLegs:     true,
						content.SlotFeet:     true,
						content.SlotFinger1:  true,
					}

					// Apply XP multiplier based on slot and rarity restrictions
					xpMultiplier := 1.0
					if gear.Rarity >= content.RarityRare {
						if highXPSlots[gear.Slot] {
							// High XP slots can have full multiplier
							xpMultiplier = gear.XPMultiplier
						} else {
							// Other slots limited to max 1-2% XP bonus
							if gear.XPMultiplier > 1.02 {
								xpMultiplier = 1.02
							} else {
								xpMultiplier = gear.XPMultiplier
							}
						}
						mult *= xpMultiplier
					}

					// Only show gear with XP multiplier in notes, but without durability
					if xpMultiplier > 1.0 {
						notes = append(notes, i18n.T("bot.loot.multiplier_simple", gear.Name, xpMultiplier))
					}

					stats = stats.Add(gear.Stats)
					gearScore += gear.Stats.Score()
					if gear.Special != content.EffectNone {
						effects = append(effects, gear.Special)
					}

					if enchID.Valid && enchID.String != "" {
						if ench, ok := content.GetEnchantmentByID(enchID.String); ok {
							// Apply doubled stats at runtime (Unstable Enchantments mechanic)
							eStats := ench.Stats
							eStats.STR *= 2
							eStats.SPD *= 2
							stats = stats.Add(eStats)
							gearScore += eStats.Score()
							mult *= ench.XPMultiplier // Apply enchantment XP penalty
							if ench.Special != content.EffectNone {
								effects = append(effects, ench.Special)
							}

							eName := ench.Name
							if ench.Special != content.EffectNone {
								eName = i18n.T("bot.loot.pool_prefix", ench.Special, eName)
							}
							notes = append(notes, i18n.T("bot.loot.pool_prefix_xp", eName, gear.Name, ench.XPMultiplier))
						}
					}
				}
			}
		}
	}

	// Abyss set bonus: equipping multiple ABYSS_ pieces grants cumulative 2/4/6-piece
	// stat tiers, applied here so the set works everywhere the character fights.
	if bonus, reached := content.AbyssSetBonus(abyssSetPieces); reached > 0 {
		stats = stats.Add(bonus)
		gearScore += bonus.Score()
		notes = append(notes, fmt.Sprintf("🕳️ Abyss Set (%d pieces)", reached))
	}

	// Skills also provide effects
	srows, err := b.DB.Query("SELECT skill_id FROM user_skills WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = srows.Close() }()
		for srows.Next() {
			var sid string
			if err := srows.Scan(&sid); err == nil {
				if s, ok := content.GetSkillByID(sid); ok {
					if s.Special != content.EffectNone {
						effects = append(effects, s.Special)
					}
				}
			}
		}
	}

	// Ultimate Skill also provides effect
	var ultimateID sql.NullString
	if err := b.DB.QueryRow("SELECT ultimate_skill_id FROM users WHERE client_uid = $1", uid).Scan(&ultimateID); err == nil {
		if ultimateID.Valid && ultimateID.String != "" {
			if us, ok := content.GetUltimateSkillByID(ultimateID.String); ok {
				if us.Special != content.EffectNone {
					effects = append(effects, us.Special)
				}
			}
		}
	}

	// Apply active elixir buffs
	crows, err := b.DB.Query("SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid = $1 AND remaining_fights > 0", uid)
	if err == nil {
		defer func() { _ = crows.Close() }()
		for crows.Next() {
			var cid string
			var rem int
			if err := crows.Scan(&cid, &rem); err == nil {
				switch cid {
				case "strength_elixir":
					stats.STR += 15
					notes = append(notes, i18n.T("bot.loot.multiplier_simple", "Strength Elixir", 1.0))
				case "iron_skin_brew":
					stats.DEF += 10
					notes = append(notes, i18n.T("bot.loot.multiplier_simple", "Iron Skin Brew", 1.0))
				}
			}
		}
	}

	return mult, stats, gearScore, notes, effects
}

func (b *Bot) rollLootForUser(uid string, mob content.Mob, zoneDifficulty float64, focus string) (string, string) {
	var results []string
	var pokes []string
	count := 1
	if mob.Type == content.MobBoss {
		count = 3
		// Guaranteed consumable
		c := content.RandomConsumable()
		b.grantConsumable(uid, c.ID, c.Duration)
		results = append(results, i18n.T("bot.loot.item", c.Name, c.ID))
	}
	if mob.Type == content.MobLegendary {
		count = 5
		c := content.RandomConsumable()
		b.grantConsumable(uid, c.ID, c.Duration)
		results = append(results, i18n.T("bot.loot.item", c.Name, c.ID))
	}
	if mob.Type == content.MobTreasureGoblin {
		count = 2
	}

	// Double Loot Title check
	var tName sql.NullString
	_ = b.DB.QueryRow("SELECT title FROM users WHERE client_uid=$1", uid).Scan(&tName)

	vip, _ := b.getVIP(uid)

	// Effect check
	_, _, _, _, effects := b.activeLootMult(uid, time.Now())
	lootFindBonus := 0.0
	for _, eff := range effects {
		if eff == content.EffectTreasureHunter {
			lootFindBonus += 0.05
		}
	}

	// Loot Quality Multiplier: Higher difficulty = better chance for Rares
	qualityMult := zoneDifficulty
	if qualityMult < 1.0 {
		qualityMult = 1.0
	}

	if focus == "loot" {
		qualityMult *= 1.2
		lootFindBonus += 0.50
	}

	// Low-level / low-difficulty content drops fewer high-rarity items: the rare
	// tiers (ultimate, title, unique, artifact, enchant) are scaled down until the
	// fought level catches up. Common gear/consumable odds are left untouched.
	rareScale := lootRarityScale(mob.Level)

	if tName.Valid {
		if t, ok := content.GetTitleByName(tName.String); ok && t.DoubleLoot {
			count *= 2
		}
	}

	var ultPity, artPity int
	_ = b.DB.QueryRow("SELECT ultimate_pity, artifact_pity FROM users WHERE client_uid=$1", uid).Scan(&ultPity, &artPity)
	ultDropped, artDropped := false, false

	for i := 0; i < count; i++ {
		// #nosec G404
		r := rand.Float64() - lootFindBonus // #nosec G404

		// Gold focus is handled before the generic treasure-goblin branch so the
		// richer goblin payout below is actually reachable (the goblin branch's
		// own `continue` would otherwise skip it).
		if focus == "gold" {
			// If gold focus, skip item rolls but always award a gold bonus.
			// Treasure goblins get an even richer payout.
			if mob.Type == content.MobTreasureGoblin {
				gold := int64(1000 + rand.IntN(2000))
				if vip.Bonus > 0 {
					gold = int64(float64(gold) * (1.0 + float64(vip.Bonus)/100.0))
				}
				_, _ = b.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid = $2", gold, uid)
				results = append(results, fmt.Sprintf("💰 %d gold", gold))
			} else {
				// Standard gold reward for non-goblin mobs in gold-focus mode
				baseGold := int64(10 + rand.IntN(mob.RewardXP/2+10))
				if vip.Bonus > 0 {
					baseGold = int64(float64(baseGold) * (1.0 + float64(vip.Bonus)/100.0))
				}
				_, _ = b.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid = $2", baseGold, uid)
				results = append(results, fmt.Sprintf("💰 %d gold", baseGold))
			}
			continue
		}

		if mob.Type == content.MobTreasureGoblin {
			gold := int64(1000 + rand.IntN(2000)) // #nosec G404 - non-cryptographic gold calculation
			if vip.Bonus > 0 {
				gold = int64(float64(gold) * (1.0 + float64(vip.Bonus)/100.0))
			}
			_, _ = b.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid = $2", gold, uid)
			results = append(results, fmt.Sprintf("💰 %d gold", gold))
			continue
		}

		lootFound := false
		// ... rest of loop ...
		// Checks ordered by ascending threshold so smaller chances are evaluated first
		// Thresholds: title=0.005, ultimateSkill=0.005, uniqueItem=0.01, artifact=0.01, ench=0.02, skill=0.05, cons=0.1, gear=0.10
		
		effUltChance := ultimateSkillChance*qualityMult*rareScale + float64(ultPity)*0.001 // 0.1% extra per pity point
		effArtChance := artifactChance*qualityMult*rareScale + float64(artPity)*0.002 // 0.2% extra per pity point

		if r < effUltChance {
			// Ultimate skill drop (0.5%)
			us := content.RandomUltimateSkill()
			var exists bool
			_ = b.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_ultimate_skills WHERE client_uid=$1 AND ultimate_id=$2)", uid, us.ID).Scan(&exists)
			if !exists {
				_, _ = b.DB.Exec("INSERT INTO user_ultimate_skills (client_uid, ultimate_id) VALUES ($1, $2)", uid, us.ID)
				_, _ = b.DB.Exec("UPDATE users SET ultimate_skills_count = ultimate_skills_count + 1 WHERE client_uid=$1", uid)
				var currentUltimate sql.NullString
				_ = b.DB.QueryRow("SELECT ultimate_skill_id FROM users WHERE client_uid=$1", uid).Scan(&currentUltimate)
				if !currentUltimate.Valid {
					_, _ = b.DB.Exec("UPDATE users SET ultimate_skill_id=$2, ultimate_cooldown=0 WHERE client_uid=$1", uid, us.ID)
					results = append(results, i18n.T("bot.loot.ultimate_equipped", us.Name))
				} else {
					results = append(results, i18n.T("bot.loot.ultimate_collected", us.Name))
				}
				if us.Rarity >= content.RarityLegendary {
					pokes = append(pokes, i18n.T("bot.loot.major_ultimate", us.Name))
				}
			} else {
				// Duplicate Ultimates -> List on AH
				b.autoListUnwantedItems(uid, us)
				results = append(results, i18n.T("bot.loot.duplicate_ultimate_ah", us.Name))
			}
			lootFound = true
			ultDropped = true
		} else if r < titleChance*qualityMult*rareScale {
			t := content.RandomTitle()
			_, _ = b.DB.Exec("UPDATE users SET title=$2, title_mult=$3, title_expires=NOW() + INTERVAL '7 days' WHERE client_uid=$1", uid, t.Name, t.XPMultiplier)
			results = append(results, i18n.T("bot.loot.title", t.Name, t.Name))
			lootFound = true
		} else if r < uniqueItemChance*qualityMult*rareScale {
			// Unique item drop (1%)
			ui := content.RandomUniqueItem()
			var exists bool
			_ = b.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_unique_items WHERE client_uid=$1 AND item_name=$2)", uid, ui.Name).Scan(&exists)
			if !exists {
				_, _ = b.DB.Exec("INSERT INTO user_unique_items (client_uid, item_name, rarity, power) VALUES ($1, $2, $3, $4)", uid, ui.Name, ui.Rarity, ui.Power)
				_, _ = b.DB.Exec("UPDATE users SET unique_items_count = unique_items_count + 1 WHERE client_uid=$1", uid)
				results = append(results, i18n.T("bot.loot.unique", ui.Name, ui.Name, ui.Rarity.String()))
				if ui.Rarity >= content.RarityLegendary {
					pokes = append(pokes, i18n.T("bot.loot.unique_drop", ui.Name))
				}
			} else {
				// Duplicate Uniques -> List on AH
				b.autoListUnwantedItems(uid, ui)
				results = append(results, i18n.T("bot.loot.duplicate_unique_ah", ui.Name))
			}
			lootFound = true
		} else if r < effArtChance {
			a := content.RandomArtifact()
			a.Stats.HP = int(float64(a.Stats.HP) * zoneDifficulty)
			a.Stats.STR = int(float64(a.Stats.STR) * zoneDifficulty)
			a.Stats.DEF = int(float64(a.Stats.DEF) * zoneDifficulty)
			_, _ = b.DB.Exec("UPDATE users SET artifact_mult=$2, artifact_name=$3, artifact_durability=$4 WHERE client_uid=$1", uid, a.Mult, a.Name, a.MaxDurability)
			results = append(results, i18n.T("bot.loot.artifact", a.Name, a.Name))
			pokes = append(pokes, i18n.T("bot.loot.artifact_found", a.Name))
			lootFound = true
			artDropped = true
		} else if r < enchChance*qualityMult*rareScale {
			ench := content.RandomEnchantment()
			ench.Stats.STR = int(float64(ench.Stats.STR) * zoneDifficulty)
			ench.Stats.SPD = int(float64(ench.Stats.SPD) * zoneDifficulty)
			if slot, ok := b.applyEnchantment(uid, ench); ok {
				results = append(results, i18n.T("bot.loot.enchanted", slot, ench.Name, ench.Name))
			} else {
				// Unwanted Enchantments -> List on AH
				b.autoListUnwantedItems(uid, ench)
				results = append(results, i18n.T("bot.loot.unwanted_enchant_ah", ench.Name))
			}
			lootFound = true
		} else if r < skillChance*qualityMult {
			s := content.RandomSkill()
			s.Power *= zoneDifficulty
			if slot, ok := b.equipSkill(uid, s); ok {
				results = append(results, i18n.T("bot.loot.learned_skill", s.Name, s.Name, slot))
			} else {
				// Unwanted Skills -> List on AH
				b.autoListUnwantedItems(uid, s)
				results = append(results, i18n.T("bot.loot.unwanted_skill_ah", s.Name))
			}
			lootFound = true
		} else if r < consChance*qualityMult {
			c := content.RandomConsumable()
			b.grantConsumable(uid, c.ID, c.Duration)
			results = append(results, i18n.T("bot.loot.item", c.Name, c.ID))
			lootFound = true
		} else if r < gearChance*qualityMult {
			g := content.RandomGearDrop()
			if b.loadAbyssRun(uid).Active && rand.Float64() < 0.20 {
				g = content.RandomAbyssGearDrop()
			}
			g.Stats.HP = int(float64(g.Stats.HP) * zoneDifficulty)
			g.Stats.STR = int(float64(g.Stats.STR) * zoneDifficulty)
			g.Stats.DEF = int(float64(g.Stats.DEF) * zoneDifficulty)
			g.Stats.SPD = int(float64(g.Stats.SPD) * zoneDifficulty)

			result := b.awardGearDrop(uid, g)
			results = append(results, fmt.Sprintf("%s%s [s:%s] (gs:%d CR:%.1f R:[color=%s]%s[/color])",
				result.Prefix, g.Name, string(g.Slot), g.Stats.Score(), g.CombatRating(), g.Rarity.Color(), g.Rarity.String()))

			if result.Action == "equipped" && g.Rarity >= content.RarityLegendary {
				pokes = append(pokes, i18n.T("bot.loot.legendary_equipped", g.Name))
			}
			lootFound = true
		}

		// 100% Drop Guarantee: If nothing else found, drop a Common item
		if !lootFound && focus != "gold" {
			// #nosec G404
			if rand.Float64() < 0.7 { // #nosec G404
				// Drop a basic common gear
				g := content.RandomStarterGear()
				if b.shouldEquip(uid, g) {
					_, _ = b.DB.Exec(`INSERT INTO user_gear (client_uid, slot, gear_id, durability) VALUES ($1, $2, $3, $4) ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = $3, durability = $4`, uid, string(g.Slot), g.ID, g.MaxDurability)
					results = append(results, i18n.T("bot.loot.found", g.Name, string(g.Slot), g.Stats.Score(), g.CombatRating(), g.Rarity.String()))
				} else {
					b.autoListUnwantedItems(uid, g)
					results = append(results, i18n.T("bot.loot.listed_ah", g.Name, string(g.Slot), g.Rarity.Color(), g.Rarity.String()))
				}
			} else {
				results = append(results, i18n.T("bot.loot.small_health_potion"))
				_, _ = b.DB.Exec("INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, 'P1', 0) ON CONFLICT DO NOTHING", uid)
			}
		}
	}
	
	// Gold-focus rolls skip every item roll, so they must not advance pity (which
	// would otherwise inflate ultimate/artifact odds for free).
	if focus != "gold" {
		if ultDropped {
			ultPity = 0
		} else {
			ultPity += count
		}
		if artDropped {
			artPity = 0
		} else {
			artPity += count
		}
		_, _ = b.DB.Exec("UPDATE users SET ultimate_pity=$2, artifact_pity=$3 WHERE client_uid=$1", uid, ultPity, artPity)
	}
	
	resStr := ""
	if len(results) > 0 {
		resStr = strings.Join(results, ", ")
	}
	pokeStr := ""
	if len(pokes) > 0 {
		pokeStr = strings.Join(pokes, " ")
	}
	return resStr, pokeStr
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
	if err != nil {
		return 0, false
	}
	defer func() { _ = rows.Close() }()

	slots := make(map[int]string)
	for rows.Next() {
		var s int
		var id string
		if err := rows.Scan(&s, &id); err == nil {
			slots[s] = id
		}
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
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []content.Skill
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			if s, ok := content.GetSkillByID(id); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func (b *Bot) getUltimateSkill(uid string) *content.UltimateSkill {
	var ultimateID sql.NullString
	var cooldown int
	err := b.DB.QueryRow("SELECT ultimate_skill_id, ultimate_cooldown FROM users WHERE client_uid=$1", uid).Scan(&ultimateID, &cooldown)
	if err != nil || !ultimateID.Valid {
		return nil
	}
	if us, ok := content.GetUltimateSkillByID(ultimateID.String); ok {
		us.CurrentCooldown = cooldown
		return &us
	}
	return nil
}

func (b *Bot) applyEnchantment(uid string, ench content.Enchantment) (string, bool) {
	rows, err := b.DB.Query("SELECT slot, enchantment_id FROM user_gear WHERE client_uid = $1", uid)
	if err != nil {
		return "", false
	}
	defer func() { _ = rows.Close() }()
	type slotInfo struct{ slot, enchID string }
	var slots []slotInfo
	for rows.Next() {
		var s slotInfo
		var e sql.NullString
		if err := rows.Scan(&s.slot, &e); err == nil {
			if e.Valid {
				s.enchID = e.String
			}
			slots = append(slots, s)
		}
	}
	if len(slots) == 0 {
		return "", false
	}
	// #nosec G404
	target := slots[rand.IntN(len(slots))] // #nosec G404

	// Improvement 39: Unstable Enchantments
	// #nosec G404
	if rand.Float64() < 0.05 {
		// 5% chance to break item
		_, _ = b.DB.Exec("DELETE FROM user_gear WHERE client_uid = $1 AND slot = $2", uid, target.slot)
		return target.slot, false
	}

	// 95% chance for success + double stats boost
	ench.Stats.STR *= 2
	ench.Stats.SPD *= 2

	if target.enchID != "" {
		if cur, ok := content.GetEnchantmentByID(target.enchID); ok {
			if ench.Rarity < cur.Rarity {
				return "", false
			}
		}
	}
	_, _ = b.DB.Exec("UPDATE user_gear SET enchantment_id = $3, durability = durability + $4 WHERE client_uid = $1 AND slot = $2", uid, target.slot, ench.ID, ench.DuraBonus)
	return target.slot, true
}

func (b *Bot) shouldEquip(uid string, newGear content.Gear) bool {
	var currentID string
	err := b.DB.QueryRow("SELECT gear_id FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, string(newGear.Slot)).Scan(&currentID)
	if err == sql.ErrNoRows {
		return true
	}
	if cur, ok := content.GetGearByID(currentID); ok {
		// Prioritize XP Multiplier first for faster progression
		if newGear.XPMultiplier > cur.XPMultiplier {
			return true
		}
		// Equip if higher rarity OR if CombatRating is better (replaces stale gear with fresh durability)
		return newGear.Rarity > cur.Rarity || newGear.CombatRating() > cur.CombatRating()
	}
	return true
}

func (b *Bot) awardXP(uid, nickname string, awarded int) (*levelResult, error) {
	var curXP, curLevel int
	err := b.DB.QueryRow("SELECT xp, level FROM users WHERE client_uid = $1", uid).Scan(&curXP, &curLevel)
	if err == sql.ErrNoRows {
		curXP, curLevel = 0, 1
	} else if err != nil {
		return nil, err
	}
	total := curXP + awarded
	if total < 0 {
		total = 0
	}
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
	if err != nil {
		return
	}
	type decayRow struct {
		uid, nick string
		xp, level int
	}
	var batch []decayRow
	for rows.Next() {
		var d decayRow
		if err := rows.Scan(&d.uid, &d.nick, &d.xp, &d.level); err == nil {
			batch = append(batch, d)
		}
	}
	_ = rows.Close()
	for _, d := range batch {
		newXP := int(float64(d.xp) * (1.0 - slothDailyDecay))
		if newXP < 0 {
			newXP = 0
		}
		newLevel := leveling.LevelForXP(newXP)
		_, _ = b.DB.Exec("UPDATE users SET xp=$2, level=$3 WHERE client_uid=$1", d.uid, newXP, newLevel)
	}
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
