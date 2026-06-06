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
	duraLossPerFight   = 1 // Each fight loses 1 durability
	occupiedSlotRare   = 0.1 // 10% of normal chance if slot is occupied
)

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

	if b.Cfg.EnableXPModifiers {
		mob := content.SpawnRandomMob()
		combatXP, combatMsg := b.resolveCombat(stats, mob)
		delta += combatXP
		notes = append(notes, combatMsg)
		
		// Durability system: lose dura every fight
		b.applyDurabilityLoss(uid)
	}

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

	artifactPoke := ""
	if b.Cfg.EnableXPModifiers {
		artifactPoke = b.rollLootAndTitles(uid, ctx.today)
	}
	return lr, notes, artifactPoke
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
	if err := b.DB.QueryRow("SELECT last_poke_date, streak_days FROM users WHERE client_uid=$1", uid).Scan(&last, &streak); err != nil {
		return 0
	}
	if last.Valid && sameDay(last.Time, today) { return streak }
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
	return !(last.Valid && sameDay(last.Time, today))
}

func (b *Bot) setLastLogin(uid string, today time.Time) {
	_, _ = b.DB.Exec("UPDATE users SET last_login_date=$2 WHERE client_uid=$1", uid, today)
}

// ---- gear / artifacts / titles ----

func (b *Bot) ensureUserHasGear(uid string) {
	var count int
	if err := b.DB.QueryRow("SELECT COUNT(*) FROM user_gear WHERE client_uid = $1", uid).Scan(&count); err == nil && count == 0 {
		gear := content.RandomStarterGear()
		_, _ = b.DB.Exec("INSERT INTO user_gear (client_uid, slot, gear_id, durability) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING",
			uid, string(gear.Slot), gear.ID, gear.MaxDurability)
		log.Printf("mandatory loot: granted %s to %s", gear.Name, uid)
	}
}

func (b *Bot) applyDurabilityLoss(uid string) {
	// Lose dura on gear
	_, _ = b.DB.Exec("UPDATE user_gear SET durability = durability - $2 WHERE client_uid = $1", uid, duraLossPerFight)
	// Delete broken gear
	_, _ = b.DB.Exec("DELETE FROM user_gear WHERE client_uid = $1 AND durability <= 0", uid)
	
	// Lose dura on artifact
	_, _ = b.DB.Exec("UPDATE users SET artifact_durability = artifact_durability - $2 WHERE client_uid = $1 AND artifact_durability > 0", uid, duraLossPerFight)
	// Clear broken artifact
	_, _ = b.DB.Exec("UPDATE users SET artifact_mult=1, artifact_name=NULL, artifact_durability=0 WHERE client_uid=$1 AND artifact_durability <= 0 AND artifact_name IS NOT NULL", uid)
}

func (b *Bot) activeLootMult(uid string, today time.Time) (float64, content.Stats, []string) {
	mult := 1.0
	var stats content.Stats
	var notes []string

	// 1. Title (Time-based)
	var title sql.NullString
	var tMult sql.NullFloat64
	var tExp sql.NullTime
	if err := b.DB.QueryRow("SELECT title, title_mult, title_expires FROM users WHERE client_uid=$1", uid).Scan(&title, &tMult, &tExp); err == nil {
		if tExp.Valid && !today.After(tExp.Time) && title.Valid {
			mult *= tMult.Float64
			notes = append(notes, fmt.Sprintf("%s x%g", title.String, tMult.Float64))
			if t, ok := content.GetTitleByName(title.String); ok {
				stats = stats.Add(t.Stats)
			}
		} else if title.Valid {
			_, _ = b.DB.Exec("UPDATE users SET title=NULL, title_mult=NULL, title_expires=NULL WHERE client_uid=$1", uid)
		}
	}

	// 2. Corrupted Artifact (Durability-based)
	var aMult sql.NullFloat64
	var aName sql.NullString
	var aDura int
	if err := b.DB.QueryRow("SELECT artifact_mult, artifact_name, artifact_durability FROM users WHERE client_uid=$1", uid).Scan(&aMult, &aName, &aDura); err == nil {
		if aName.Valid && aName.String != "" && aDura > 0 {
			mult *= aMult.Float64
			notes = append(notes, fmt.Sprintf("%s x%g (%d dura)", aName.String, aMult.Float64, aDura))
			if art, ok := content.GetArtifactByName(aName.String); ok {
				stats = stats.Add(art.Stats)
			}
		}
	}

	// 3. 24 Gear Slots (Durability-based)
	rows, err := b.DB.Query("SELECT slot, gear_id, durability FROM user_gear WHERE client_uid = $1", uid)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var slot, gearID string
			var dura int
			if err := rows.Scan(&slot, &gearID, &dura); err == nil {
				if gear, ok := content.GetGearByID(gearID); ok {
					mult *= gear.XPMultiplier
					stats = stats.Add(gear.Stats)
					notes = append(notes, fmt.Sprintf("%s x%g (%d dura)", gear.Name, gear.XPMultiplier, dura))
				}
			}
		}
	}

	return mult, stats, notes
}

func (b *Bot) calculateTotalStats(uid string, today time.Time) (content.Stats, float64, []string) {
	var level int
	_ = b.DB.QueryRow("SELECT level FROM users WHERE client_uid=$1", uid).Scan(&level)
	base := content.Stats{
		HP: 50 + (level / 10), STR: 10 + (level / 50), DEF: 5 + (level / 100), SPD: 10 + (level / 50), LCK: level / 200,
	}
	mult, lootStats, notes := b.activeLootMult(uid, today)
	return base.Add(lootStats), mult, notes
}

func (b *Bot) resolveCombat(userStats content.Stats, mob content.Mob) (int, string) {
	uHP, mHP := userStats.HP, mob.Stats.HP
	for r := 0; r < 10; r++ {
		uDmg := userStats.STR - mob.Stats.DEF
		if uDmg < 1 { uDmg = 1 }
		mHP -= uDmg
		if mHP <= 0 { return mob.RewardXP, fmt.Sprintf("Victory! You slew %s (%s).", mob.Name, mob.Type) }
		mDmg := mob.Stats.STR - userStats.DEF
		if mDmg < 1 { mDmg = 1 }
		uHP -= mDmg
		if uHP <= 0 { return -mob.RewardXP, fmt.Sprintf("Defeat! You were crushed by %s (%s).", mob.Name, mob.Type) }
	}
	return 0, fmt.Sprintf("Draw! You and %s (%s) both retreated.", mob.Name, mob.Type)
}

func (b *Bot) rollLootAndTitles(uid string, today time.Time) string {
	r := rand.Float64()
	if r < titleChance {
		t := content.RandomTitle()
		expires := today.AddDate(0, 0, 7)
		_, _ = b.DB.Exec("UPDATE users SET title=$2, title_mult=$3, title_expires=$4 WHERE client_uid=$1", uid, t.Name, t.XPMultiplier, expires)
		return clampRunes(fmt.Sprintf("👑 RARE TITLE: You are now known as '%s'! (x%g XP for 7d)", t.Name, t.XPMultiplier), 100)
	}
	
	if r < artifactChance {
		a := content.RandomArtifact()
		_, _ = b.DB.Exec("UPDATE users SET artifact_mult=$2, artifact_name=$3, artifact_durability=$4 WHERE client_uid=$1", uid, a.Mult, a.Name, a.MaxDurability)
		return clampRunes(fmt.Sprintf("🩸 Corrupted Artifact: %s (%s) assigned! (%d dura)", a.Name, a.Effect(), a.MaxDurability), 100)
	}

	if r < gearChance {
		gear := content.RandomGearDrop()
		
		// Check if slot is occupied
		var exists bool
		_ = b.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_gear WHERE client_uid=$1 AND slot=$2)", uid, string(gear.Slot)).Scan(&exists)
		if exists && rand.Float64() >= occupiedSlotRare {
			return "" // Very rare to find gear for occupied slot
		}

		_, _ = b.DB.Exec("INSERT INTO user_gear (client_uid, slot, gear_id, durability) VALUES ($1, $2, $3, $4) ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id = $3, durability = $4",
			uid, string(gear.Slot), gear.ID, gear.MaxDurability)
		return clampRunes(fmt.Sprintf("🛡️ Loot Drop: %s! (%s) Equipped in %s slot.", gear.Name, gear.Description, gear.Slot), 100)
	}
	return ""
}

func (b *Bot) slothDecay(c *clientquery.Client, today time.Time) {
	if !b.Cfg.EnableXPModifiers { return }
	cutoff := today.AddDate(0, 0, -slothGraceDays)
	rows, err := b.DB.Query(`SELECT client_uid, nickname, xp, level, group_level, cldbid, last_seen, last_decay_date FROM users WHERE last_seen < $1`, cutoff)
	if err != nil { return }
	type decayRow struct {
		uid, nick string
		xp, level, groupLevel, cldbid int
		lastSeen time.Time
		lastDecay sql.NullTime
	}
	var batch []decayRow
	for rows.Next() {
		var d decayRow
		if err := rows.Scan(&d.uid, &d.nick, &d.xp, &d.level, &d.groupLevel, &d.cldbid, &d.lastSeen, &d.lastDecay); err == nil {
			batch = append(batch, d)
		}
	}
	_ = rows.Close()
	for _, d := range batch {
		from := d.lastSeen.AddDate(0, 0, slothGraceDays)
		if d.lastDecay.Valid && d.lastDecay.Time.After(from) { from = d.lastDecay.Time }
		days := daysBetween(from, today)
		if days <= 0 { continue }
		factor := math.Pow(1-slothDailyDecay, float64(days))
		newXP := int(float64(d.xp) * factor)
		newLevel := leveling.LevelForXP(newXP)
		_, _ = b.DB.Exec("UPDATE users SET xp=$2, level=$3, last_decay_date=$4 WHERE client_uid=$1", d.uid, newXP, newLevel, today)
		if newLevel < d.groupLevel && b.Cfg.XPServerGroups && d.cldbid > 0 {
			if sgid, ok := b.xpGroups[d.groupLevel]; ok {
				_ = c.ServerGroupDelClient(sgid, d.cldbid)
				b.maybeDeleteEmptyLevel(c, d.groupLevel, sgid)
			}
			_, _ = b.DB.Exec("UPDATE users SET group_level=0 WHERE client_uid=$1", d.uid)
		}
	}
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
func dayFloor(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
func daysBetween(from, to time.Time) int {
	return int(dayFloor(to).Sub(dayFloor(from)).Hours() / 24)
}
func clampRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max { return s }
	return string(r[:max])
}
