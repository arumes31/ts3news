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
	critChance         = 0.05 // 5% chance of a critical hit
	critMult           = 3.0  // critical hit multiplier
	partyMult          = 1.25 // +25% when a full party is online
	serverMultPerUser  = 0.05 // +5% per extra online user
	serverMultCap      = 2.0  // server multiplier ceiling
	noGamePenalty      = 0.5  // XP factor when there is no new game to poke
	dailyLoginXP       = 5    // flat XP for connecting once a day
	lootBoxEveryLevels = 25   // a loot box every N levels
	lootBoxMin         = 50
	lootBoxMax         = 500
	slothGraceDays     = 7    // days offline before the Sloth debuff applies
	slothDailyDecay    = 0.02 // 2% XP lost per offline day after the grace period
	artifactChance     = 0.01 // 1% chance per cycle to receive a corrupted artifact
	artifactDays       = 1    // artifact lasts ~1 day
	streakBronze       = 3
	streakSilver       = 5
	streakGold         = 7
)

// cycleContext holds per-cycle shared facts used by the XP modifiers.
type cycleContext struct {
	onlineNormal int             // number of online normal (voice) clients, incl. the bot
	onlineNicks  map[string]bool // lowercased nicknames currently online
	today        time.Time
}

func (b *Bot) buildCycleContext(clients []clientquery.ClientInfo) cycleContext {
	online := map[string]bool{}
	normal := 0
	for _, cl := range clients {
		if cl.Type == 0 {
			normal++
			online[strings.ToLower(cl.Nickname)] = true
		}
	}
	return cycleContext{onlineNormal: normal, onlineNicks: online, today: time.Now()}
}

// processUserXP applies all XP gains for one user this cycle: daily login bonus,
// the (multiplier-adjusted) main award, loot boxes, and rolling a new artifact.
// hasGame=false applies the no-game penalty. It returns the resulting level
// change, human-readable notes for the PM, and an optional artifact-announcement
// poke (empty if none was granted).
func (b *Bot) processUserXP(uid, nickname string, base int, hasGame bool, ctx cycleContext) (*levelResult, []string, string) {
	var notes []string
	delta := 0

	if b.Cfg.EnableXPModifiers && b.dailyLoginDue(uid, ctx.today) {
		delta += dailyLoginXP
		notes = append(notes, fmt.Sprintf("daily login +%d", dailyLoginXP))
		b.setLastLogin(uid, ctx.today)
	}

	mult, mnotes := b.computeAwardMult(uid, nickname, ctx)
	notes = append(notes, mnotes...)
	if !hasGame {
		mult *= noGamePenalty
		notes = append(notes, "no new game -50%")
	}
	award := int(math.Round(float64(base) * mult))
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
		artifactPoke = b.rollArtifact(uid, ctx.today)
	}
	return lr, notes, artifactPoke
}

// computeAwardMult returns the combined XP multiplier from streaks, crits, the
// server population, party bonus and any active artifact, plus display notes.
func (b *Bot) computeAwardMult(uid, nickname string, ctx cycleContext) (float64, []string) {
	if !b.Cfg.EnableXPModifiers {
		return 1.0, nil
	}
	mult := 1.0
	var notes []string

	if streak := b.updateStreak(uid, ctx.today); streakMultiplier(streak) > 1 {
		sm := streakMultiplier(streak)
		mult *= sm
		notes = append(notes, fmt.Sprintf("%dd streak x%g", streak, sm))
	}
	if rand.Float64() < critChance {
		mult *= critMult
		notes = append(notes, fmt.Sprintf("CRIT x%g", critMult))
	}
	if sv := serverMultiplier(ctx.onlineNormal); sv > 1 {
		mult *= sv
		notes = append(notes, fmt.Sprintf("server x%.2f", sv))
	}
	if b.partyAllOnline(nickname, ctx.onlineNicks) {
		mult *= partyMult
		notes = append(notes, fmt.Sprintf("party x%g", partyMult))
	}
	if am, label := b.activeGearMult(uid, ctx.today); am != 1 {
		mult *= am
		notes = append(notes, label...)
	}
	return mult, notes
}

func streakMultiplier(streak int) float64 {
	switch {
	case streak >= streakGold:
		return 2.0
	case streak >= streakSilver:
		return 1.5
	case streak >= streakBronze:
		return 1.25
	default:
		return 1.0
	}
}

func serverMultiplier(onlineNormal int) float64 {
	humans := onlineNormal - 1 // exclude the bot's own client
	if humans < 1 {
		humans = 1
	}
	m := 1 + serverMultPerUser*float64(humans-1)
	if m > serverMultCap {
		m = serverMultCap
	}
	return m
}

func lootBoxForCross(oldLevel, newLevel int) int {
	if newLevel <= oldLevel {
		return 0
	}
	first := (oldLevel/lootBoxEveryLevels + 1) * lootBoxEveryLevels
	if first <= newLevel {
		return lootBoxMin + rand.Intn(lootBoxMax-lootBoxMin+1)
	}
	return 0
}

func (b *Bot) partyAllOnline(nickname string, online map[string]bool) bool {
	ln := strings.ToLower(nickname)
	for _, party := range b.parties {
		inParty := false
		for _, m := range party {
			if m == ln {
				inParty = true
				break
			}
		}
		if !inParty {
			continue
		}
		all := true
		for _, m := range party {
			if !online[m] {
				all = false
				break
			}
		}
		if all && len(party) > 1 {
			return true
		}
	}
	return false
}

// ---- streak / login state ----

func (b *Bot) updateStreak(uid string, today time.Time) int {
	var last sql.NullTime
	var streak int
	if err := b.DB.QueryRow("SELECT last_poke_date, streak_days FROM users WHERE client_uid=$1", uid).Scan(&last, &streak); err != nil {
		return 0
	}
	if last.Valid && sameDay(last.Time, today) {
		return streak // already counted today
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
	return !(last.Valid && sameDay(last.Time, today))
}

func (b *Bot) setLastLogin(uid string, today time.Time) {
	_, _ = b.DB.Exec("UPDATE users SET last_login_date=$2 WHERE client_uid=$1", uid, today)
}

// ---- gear / artifacts ----

func (b *Bot) activeGearMult(uid string, today time.Time) (float64, []string) {
	mult := 1.0
	var notes []string

	query := `SELECT 
		artifact_mult, artifact_name, artifact_expires,
		weapon_id, weapon_expires,
		armor_id, armor_expires,
		relic_id, relic_expires
		FROM users WHERE client_uid=$1`
		
	var artMult sql.NullFloat64
	var artName sql.NullString
	var artExpires sql.NullTime
	var wID, aID, rID sql.NullString
	var wExp, aExp, rExp sql.NullTime

	if err := b.DB.QueryRow(query, uid).Scan(
		&artMult, &artName, &artExpires,
		&wID, &wExp,
		&aID, &aExp,
		&rID, &rExp,
	); err != nil {
		return mult, notes
	}

	// Artifacts (from the original system)
	if artExpires.Valid && !today.After(artExpires.Time) && artMult.Valid {
		m := artMult.Float64
		mult *= m
		label := "artifact"
		if artName.Valid && artName.String != "" {
			label = artName.String
		}
		notes = append(notes, fmt.Sprintf("%s x%g", label, m))
	} else if artMult.Valid && artMult.Float64 != 1 {
		_, _ = b.DB.Exec("UPDATE users SET artifact_mult=1, artifact_name=NULL, artifact_expires=NULL WHERE client_uid=$1", uid)
	}

	// Check other gear slots
	checkSlot := func(idCol sql.NullString, expCol sql.NullTime, clearQuery string) {
		if !idCol.Valid || idCol.String == "" {
			return
		}
		if expCol.Valid && !today.After(expCol.Time) {
			if gear, ok := content.GetGearByID(idCol.String); ok {
				mult *= gear.XPMultiplier
				notes = append(notes, fmt.Sprintf("%s x%g", gear.Name, gear.XPMultiplier))
			}
		} else {
			// Expired
			_, _ = b.DB.Exec(clearQuery, uid)
		}
	}

	checkSlot(wID, wExp, "UPDATE users SET weapon_id=NULL, weapon_expires=NULL WHERE client_uid=$1")
	checkSlot(aID, aExp, "UPDATE users SET armor_id=NULL, armor_expires=NULL WHERE client_uid=$1")
	checkSlot(rID, rExp, "UPDATE users SET relic_id=NULL, relic_expires=NULL WHERE client_uid=$1")

	return mult, notes
}

// rollArtifact has a small chance to grant gear/artifacts (applied to
// future XP gains), returning a short announcement poke (<=100 chars) or "".
func (b *Bot) rollArtifact(uid string, today time.Time) string {
	if rand.Float64() >= artifactChance {
		return ""
	}

	// 50% chance for a corrupted artifact, 50% chance for other gear
	if rand.Float64() < 0.5 {
		a := content.RandomArtifact()
		expires := today.AddDate(0, 0, 1) // 1 day for corrupted
		_, _ = b.DB.Exec("UPDATE users SET artifact_mult=$2, artifact_name=$3, artifact_expires=$4 WHERE client_uid=$1",
			uid, a.Mult, a.Name, expires)

		verb := "boosts"
		if !a.IsBoon() {
			verb = "drains"
		}
		return clampRunes(fmt.Sprintf("🩸 Corrupted Artifact: %s (%s) %s your XP for 24h!", a.Name, a.Effect(), verb), 100)
	}

	gear := content.RandomGearDrop()
	expires := today.Add(gear.Duration)

	switch gear.Slot {
	case content.SlotWeapon:
		_, _ = b.DB.Exec("UPDATE users SET weapon_id=$2, weapon_expires=$3 WHERE client_uid=$1", uid, gear.ID, expires)
	case content.SlotArmor:
		_, _ = b.DB.Exec("UPDATE users SET armor_id=$2, armor_expires=$3 WHERE client_uid=$1", uid, gear.ID, expires)
	case content.SlotRelic:
		_, _ = b.DB.Exec("UPDATE users SET relic_id=$2, relic_expires=$3 WHERE client_uid=$1", uid, gear.ID, expires)
	}

	return clampRunes(fmt.Sprintf("🛡️ Loot Drop: %s! (%s) Equipped in %s slot.", gear.Name, gear.Description, gear.Slot), 100)
}

// ---- Sloth decay (inactivity penalty) ----

// slothDecay applies the Sloth debuff to users offline past the grace period: 2%
// XP lost per offline day. If a user's level drops they lose their level group.
func (b *Bot) slothDecay(c *clientquery.Client, today time.Time) {
	if !b.Cfg.EnableXPModifiers {
		return
	}
	cutoff := today.AddDate(0, 0, -slothGraceDays)
	rows, err := b.DB.Query(
		`SELECT client_uid, nickname, xp, level, group_level, cldbid, last_seen, last_decay_date
		 FROM users WHERE last_seen < $1`, cutoff)
	if err != nil {
		log.Printf("sloth: query failed: %v", err)
		return
	}
	type decayRow struct {
		uid, nick                       string
		xp, level, groupLevel, cldbid   int
		lastSeen                        time.Time
		lastDecay                       sql.NullTime
	}
	var batch []decayRow
	for rows.Next() {
		var d decayRow
		if err := rows.Scan(&d.uid, &d.nick, &d.xp, &d.level, &d.groupLevel, &d.cldbid, &d.lastSeen, &d.lastDecay); err == nil {
			batch = append(batch, d)
		}
	}
	rows.Close()

	decayed := 0
	for _, d := range batch {
		from := d.lastSeen.AddDate(0, 0, slothGraceDays) // decay starts after the grace period
		if d.lastDecay.Valid && d.lastDecay.Time.After(from) {
			from = d.lastDecay.Time
		}
		days := daysBetween(from, today)
		if days <= 0 {
			continue
		}
		factor := math.Pow(1-slothDailyDecay, float64(days))
		newXP := int(float64(d.xp) * factor)
		newLevel := leveling.LevelForXP(newXP)
		_, _ = b.DB.Exec("UPDATE users SET xp=$2, level=$3, last_decay_date=$4 WHERE client_uid=$1",
			d.uid, newXP, newLevel, today)
		decayed++

		// Lock: drop the level group if decay pushed them below its boundary.
		if newLevel < d.groupLevel && b.Cfg.XPServerGroups && d.cldbid > 0 {
			if sgid, ok := b.xpGroups[d.groupLevel]; ok {
				_ = c.ServerGroupDelClient(sgid, d.cldbid)
				b.maybeDeleteEmptyLevel(c, d.groupLevel, sgid)
			}
			_, _ = b.DB.Exec("UPDATE users SET group_level=0 WHERE client_uid=$1", d.uid)
			log.Printf("sloth: %s decayed to level %d (was group level %d) — group removed", d.nick, newLevel, d.groupLevel)
		}
	}
	if decayed > 0 {
		log.Printf("sloth: applied inactivity decay to %d offline user(s)", decayed)
	}
}

// ---- helpers ----

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
	if len(r) <= max {
		return s
	}
	return string(r[:max])
}
