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
	duraLossPerFight   = 1
	duraLossPenalty    = 3
	occupiedSlotRare   = 0.1
)

type UserInCombat struct {
	UID      string
	Nickname string
	CLID     int
	Level    int
	Stats    content.Stats
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
	award := int(math.Round(float64(base) * mult * awardMult))
	if award < 1 {
		award = 1
	}
	delta += award

	lr, err := b.awardXP(uid, nickname, delta)
	if err != nil {
		log.Printf("processUserXP: awardXP failed for %s: %v", nickname, err)
		return nil, notes, ""
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

func (b *Bot) resolveChannelCombat(users []UserInCombat, mobs []content.Mob) ([]string, int, bool) {
	var logs []string
	totalUserHP := 0
	for _, u := range users { totalUserHP += u.Stats.HP }
	totalMobHP := 0
	for _, m := range mobs { totalMobHP += m.Stats.HP }
	totalRewardXP := 0
	for _, m := range mobs { totalRewardXP += m.RewardXP }

	// Log mob effects
	for _, m := range mobs {
		for _, eff := range m.Effects {
			logs = append(logs, fmt.Sprintf("%s is %s!", m.Name, eff))
		}
	}

	victory := false
	for r := 0; r < 20; r++ {
		// round start effects
		for i := range mobs {
			m := &mobs[i]
			if m.Stats.HP <= 0 { continue }
			for _, eff := range m.Effects {
				switch eff {
				case content.EffectPoisoned:
					dmg := m.Stats.HP / 20
					if dmg < 1 { dmg = 1 }
					m.Stats.HP -= dmg
					totalMobHP -= dmg
				case content.EffectRegen:
					heal := m.Stats.HP / 20
					m.Stats.HP += heal
					totalMobHP += heal
				}
			}
		}

		// User turn
		for _, u := range users {
			target := &mobs[rand.Intn(len(mobs))]
			if target.Stats.HP <= 0 { continue }
			
			// Critical check
			dmgMult := 1.0
			isCrit := rand.Intn(100) < u.Stats.CRT
			if isCrit { dmgMult = 2.0 }

			dmg := int(float64(u.Stats.STR-target.Stats.DEF) * dmgMult)
			if dmg < 1 { dmg = 1 }

			// Mob Blinded
			for _, eff := range target.Effects {
				if eff == content.EffectBlinded && rand.Float64() < 0.2 { // Blinded mobs take more dmg or miss? Let's say user hit harder
					dmg = int(float64(dmg) * 1.5)
				}
			}

			target.Stats.HP -= dmg
			totalMobHP -= dmg
		}
		if totalMobHP <= 0 { victory = true; break }

		// Mob turn
		for _, m := range mobs {
			if m.Stats.HP <= 0 { continue }
			target := &users[rand.Intn(len(users))]
			
			// Dodge check
			if rand.Intn(100) < target.Stats.DGE {
				continue
			}

			mSTR := m.Stats.STR
			mDEF := m.Stats.DEF
			for _, eff := range m.Effects {
				switch eff {
				case content.EffectEnraged: mSTR = int(float64(mSTR) * 1.5)
				case content.EffectArmored: mDEF = int(float64(mDEF) * 1.5)
				case content.EffectWeakened: mSTR = int(float64(mSTR) * 0.5)
				case content.EffectFleet: // already scales SPD which isn't used in this simple round loop but could be
				}
			}

			dmg := mSTR - target.Stats.DEF
			if dmg < 1 { dmg = 1 }

			// Mob Blinded miss chance
			for _, eff := range m.Effects {
				if eff == content.EffectBlinded && rand.Float64() < 0.5 {
					dmg = 0
				}
			}

			target.Stats.HP -= dmg
			totalUserHP -= dmg
		}
		if totalUserHP <= 0 { victory = false; break }
	}

	if victory {
		logs = append(logs, fmt.Sprintf("VICTORY! Party defeated a group of %d mobs.", len(mobs)))
		return logs, totalRewardXP / len(users), true
	}
	logs = append(logs, "DEFEAT! Party was overrun by mobs.")
	return logs, -totalRewardXP / len(users), false
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
	return !(last.Valid && sameDay(last.Time, today))
}

func (b *Bot) setLastLogin(uid string, today time.Time) {
	_, _ = b.DB.Exec("UPDATE users SET last_login_date=$2 WHERE client_uid=$1", uid, today)
}

func (b *Bot) ensureUserHasGear(uid string) {
	var count int
	if err := b.DB.QueryRow("SELECT COUNT(*) FROM user_gear WHERE client_uid = $1", uid).Scan(&count); err == nil && count == 0 {
		gear := content.RandomStarterGear()
		_, _ = b.DB.Exec("INSERT INTO user_gear (client_uid, slot, gear_id, durability) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
			uid, string(gear.Slot), gear.ID, gear.MaxDurability)
	}
}

func (b *Bot) applyDurabilityLoss(uid string, defeat bool) {
	// Stamina reduces dura loss chance
	var stats content.Stats
	stats, _, _ = b.calculateTotalStats(uid, time.Now())
	if rand.Intn(100) < stats.STA { return } // Saved by stamina!

	loss := duraLossPerFight
	if defeat { loss = duraLossPenalty }
	_, _ = b.DB.Exec("UPDATE user_gear SET durability = durability - $2 WHERE client_uid = $1", uid, loss)
	_, _ = b.DB.Exec("DELETE FROM user_gear WHERE client_uid = $1 AND durability <= 0", uid)
	_, _ = b.DB.Exec("UPDATE users SET artifact_durability = artifact_durability - $2 WHERE client_uid = $1 AND artifact_durability > 0", uid, loss)
	_, _ = b.DB.Exec("UPDATE users SET artifact_mult=1, artifact_name=NULL, artifact_durability=0 WHERE client_uid=$1 AND artifact_durability <= 0 AND artifact_name IS NOT NULL", uid)
}

func (b *Bot) calculateTotalStats(uid string, today time.Time) (content.Stats, float64, []string) {
	var level int
	_ = b.DB.QueryRow("SELECT level FROM users WHERE client_uid=$1", uid).Scan(&level)
	base := content.Stats{
		HP: 100 + level*5, STR: 10 + level, DEF: 5 + level/2, SPD: 10 + level, LCK: level/5,
		INT: level/10, STA: level/10, CRT: 5 + level/50, DGE: 5 + level/50,
	}
	mult, lootStats, notes := b.activeLootMult(uid, today)
	return base.Add(lootStats), mult, notes
}

func (b *Bot) activeLootMult(uid string, today time.Time) (float64, content.Stats, []string) {
	mult := 1.0
	var stats content.Stats
	var notes []string
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
			if art, ok := content.GetArtifactByName(aName.String); ok { stats = stats.Add(art.Stats) }
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
					note := fmt.Sprintf("%s x%g (%d dura)", gear.Name, gear.XPMultiplier, dura)
					if enchID.Valid && enchID.String != "" {
						if ench, ok := content.GetEnchantmentByID(enchID.String); ok {
							stats = stats.Add(ench.Stats)
							note = fmt.Sprintf("[%s] %s", ench.Name, note)
						}
					}
					notes = append(notes, note)
				}
			}
		}
	}
	return mult, stats, notes
}

func (b *Bot) rollLootForUser(uid string, mob content.Mob) string {
	var results []string
	count := 1
	if mob.Type == content.MobBoss { count = 2 }
	if mob.Type == content.MobLegendary { count = 4 }
	for i := 0; i < count; i++ {
		r := rand.Float64()
		if r < titleChance {
			t := content.RandomTitle()
			_, _ = b.DB.Exec("UPDATE users SET title=$2, title_mult=$3, title_expires=NOW() + INTERVAL '7 days' WHERE client_uid=$1", uid, t.Name, t.XPMultiplier)
			results = append(results, "Title: "+t.Name)
		} else if r < artifactChance {
			a := content.RandomArtifact()
			_, _ = b.DB.Exec("UPDATE users SET artifact_mult=$2, artifact_name=$3, artifact_durability=$4 WHERE client_uid=$1", uid, a.Mult, a.Name, a.MaxDurability)
			results = append(results, "Artifact: "+a.Name)
		} else if r < gearChance {
			g := content.RandomGearDrop()
			if b.shouldEquip(uid, g) {
				_, _ = b.DB.Exec(`INSERT INTO user_gear (client_uid, slot, gear_id, durability) VALUES ($1, $2, $3, $4) ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = $3, durability = $4`, uid, string(g.Slot), g.ID, g.MaxDurability)
				results = append(results, "Equipped: "+g.Name)
			} else {
				results = append(results, "Found: "+g.Name)
			}
		} else if r < consChance {
			c := content.RandomConsumable()
			_, _ = b.DB.Exec("INSERT INTO user_consumables (client_uid, cons_id, remaining_fights) VALUES ($1, $2, $3) ON CONFLICT DO NOTHING", uid, c.ID, c.Duration)
			results = append(results, "Item: "+c.Name)
		} else if r < enchChance {
			ench := content.RandomEnchantment()
			if slot, ok := b.applyEnchantment(uid, ench); ok {
				results = append(results, fmt.Sprintf("Enchanted %s with %s", slot, ench.Name))
			}
		}
	}
	if len(results) > 0 { return "🎁 Loot: " + strings.Join(results, ", ") }
	return ""
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
	_, _ = b.DB.Exec("UPDATE user_gear SET enchantment_id = $3 WHERE client_uid = $1 AND slot = $2", uid, target.slot, ench.ID)
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
	_, err = b.DB.Exec(`INSERT INTO users (client_uid, nickname, xp, level, last_seen) VALUES ($1, $2, $3, $4, NOW()) ON CONFLICT (client_uid) DO UPDATE SET xp = $3, level = $4, nickname = $2, last_seen = NOW()`, uid, nickname, total, newLevel)
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
