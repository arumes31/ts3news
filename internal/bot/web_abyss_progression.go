package bot

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	abyssRunFlagRestedCharges  = "rested_charges"
	abyssRunFlagCatchupCharges = "catchup_charges"
	abyssRunFlagVeteranTrack   = "veteran_track"
)

type abyssEntryProgression struct {
	RestedCharges int
	Returning     bool
	VeteranTrack  string
}

type abyssProgressTrackView struct {
	Name  string
	Desc  string
	Value int64
	Next  int64
}

var abyssVeteranTracks = []struct {
	Key   string
	Label string
	ID    int64
}{
	{Key: "iron", Label: "Iron Nerve · clear while at 50% HP or lower", ID: 1},
	{Key: "untouched", Label: "Untouched · clear at 90% HP or higher", ID: 2},
	{Key: "boss", Label: "Boss Oath · clear boss floors", ID: 3},
}

func normalizeAbyssVeteranTrack(key string) (string, int64) {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, track := range abyssVeteranTracks {
		if track.Key == key {
			return track.Key, track.ID
		}
	}
	return "", 0
}

func abyssVeteranTrackKey(id int64) string {
	for _, track := range abyssVeteranTracks {
		if track.ID == id {
			return track.Key
		}
	}
	return ""
}

// seedAbyssProgressionFlagsInTx grants a bounded number of progression boosts
// from the most recent completed run. The flags are run-scoped, so refreshes and
// restarts cannot duplicate them and a fresh player is not mistaken for a returner.
func seedAbyssProgressionFlagsInTx(tx *sql.Tx, uid, veteranTrack string, at time.Time) (abyssEntryProgression, error) {
	flags, err := loadAbyssRunFlagsInTx(tx, uid)
	if err != nil {
		return abyssEntryProgression{}, err
	}

	var lastRun sql.NullTime
	if err := tx.QueryRow("SELECT MAX(created_at) FROM abyss_runs WHERE client_uid=$1", uid).Scan(&lastRun); err != nil {
		return abyssEntryProgression{}, err
	}
	result := abyssEntryProgression{}
	if lastRun.Valid && at.After(lastRun.Time) {
		elapsed := at.Sub(lastRun.Time)
		if elapsed >= 24*time.Hour {
			result.RestedCharges = int(elapsed / (24 * time.Hour))
			if result.RestedCharges > 5 {
				result.RestedCharges = 5
			}
			flags[abyssRunFlagRestedCharges] = int64(result.RestedCharges)
		}
		if elapsed >= 14*24*time.Hour {
			result.Returning = true
			flags[abyssRunFlagCatchupCharges] = 10
		}
	}
	if key, id := normalizeAbyssVeteranTrack(veteranTrack); id > 0 {
		flags[abyssRunFlagVeteranTrack] = id
		result.VeteranTrack = key
	}
	if err := saveAbyssRunFlagsInTx(tx, uid, flags); err != nil {
		return abyssEntryProgression{}, err
	}
	return result, nil
}

func abyssProgressionXPPercent(flags map[string]int64) int {
	pct := 0
	if flags[abyssRunFlagRestedCharges] > 0 {
		pct += 20
	}
	if flags[abyssRunFlagCatchupCharges] > 0 {
		pct += 25
	}
	return pct
}

func abyssProgressionCounterKey(uid, track string) string {
	return "abyss_progress_" + track + "_" + uid
}

func abyssProgressionTier(value int64) (suffix string, next int64) {
	switch {
	case value >= 200:
		return "gold", 0
	case value >= 50:
		return "silver", 200
	case value >= 10:
		return "bronze", 50
	default:
		return "", 10
	}
}

func (b *Bot) recordAbyssProgression(uid, track string) {
	var value int64
	err := b.DB.QueryRow(`INSERT INTO app_meta (key, value) VALUES ($1, '1')
		ON CONFLICT (key) DO UPDATE SET value=(COALESCE(NULLIF(app_meta.value, ''), '0')::bigint + 1)::text
		RETURNING value::bigint`, abyssProgressionCounterKey(uid, track)).Scan(&value)
	if err != nil {
		return
	}
	for _, threshold := range []struct {
		N      int64
		Suffix string
	}{{10, "bronze"}, {50, "silver"}, {200, "gold"}} {
		if value >= threshold.N {
			b.awardAchievement(uid, "progress_"+track+"_"+threshold.Suffix)
		}
	}
}

func abyssVeteranQualifies(track string, depth, currentHP, maxHP int) bool {
	switch track {
	case "iron":
		return maxHP > 0 && currentHP*2 <= maxHP
	case "untouched":
		return maxHP > 0 && currentHP*10 >= maxHP*9
	case "boss":
		return depth > 0 && depth%abyssBossEvery == 0
	default:
		return false
	}
}

// advanceAbyssProgression consumes per-floor boosts and advances challenge
// counters only after a floor was successfully persisted as cleared.
func (b *Bot) advanceAbyssProgression(uid string, depth, currentHP, maxHP int, modifier string, weekly bool) {
	flags := b.loadRunFlags(uid)
	if flags[abyssRunFlagRestedCharges] > 0 {
		flags[abyssRunFlagRestedCharges]--
	}
	if flags[abyssRunFlagCatchupCharges] > 0 {
		flags[abyssRunFlagCatchupCharges]--
	}
	_ = b.saveRunFlags(uid, flags)

	if modifier == "fragile_cache" && maxHP > 0 && currentHP*2 >= maxHP {
		b.recordAbyssProgression(uid, "cachekeeper")
	}
	if depth > 0 && depth%abyssBossEvery == 0 {
		b.recordAbyssProgression(uid, "bossbreaker")
	}
	if weekly {
		b.recordAbyssProgression(uid, "expeditioner")
		b.incrementCommunityExpedition()
	}
	if track := abyssVeteranTrackKey(flags[abyssRunFlagVeteranTrack]); abyssVeteranQualifies(track, depth, currentHP, maxHP) {
		b.recordAbyssProgression(uid, "veteran_"+track)
	}
}

func (b *Bot) abyssProgressionViews(uid string) []abyssProgressTrackView {
	defs := []struct {
		Key, Name, Desc string
	}{
		{"cachekeeper", "Cachekeeper", "Preserve Fragile Caches above 50% HP"},
		{"bossbreaker", "Bossbreaker", "Clear boss floors"},
		{"expeditioner", "Expeditioner", "Clear Weekly Expedition floors"},
		{"veteran_iron", "Iron Nerve", "Clear while at 50% HP or lower"},
		{"veteran_untouched", "Untouched", "Clear at 90% HP or higher"},
		{"veteran_boss", "Boss Oath", "Clear boss floors with the track active"},
		{"coordinator", "Synchronized", "Have every living party member ready before the timer"},
		{"party_combo", "Chain Reaction", "Coordinate two skills on the same target"},
		{"ghost_party", "Ghost Chaser", "Beat an asynchronous party replay's round count"},
	}
	values := make(map[string]int64, len(defs))
	args := make([]any, len(defs))
	keyToTrack := make(map[string]string, len(defs))
	for i, def := range defs {
		key := abyssProgressionCounterKey(uid, def.Key)
		args[i] = key
		keyToTrack[key] = def.Key
	}
	rows, err := b.DB.Query(`SELECT key, value FROM app_meta WHERE key IN ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, args...)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var key, raw string
			if rows.Scan(&key, &raw) == nil {
				value, _ := strconv.ParseInt(raw, 10, 64)
				values[keyToTrack[key]] = value
			}
		}
	}
	views := make([]abyssProgressTrackView, 0, len(defs))
	for _, def := range defs {
		value := values[def.Key]
		_, next := abyssProgressionTier(value)
		views = append(views, abyssProgressTrackView{Name: def.Name, Desc: def.Desc, Value: value, Next: next})
	}
	return views
}

func abyssProgressionAchievementName(code string) string {
	if !strings.HasPrefix(code, "progress_") {
		return ""
	}
	rest := strings.TrimPrefix(code, "progress_")
	tier := ""
	for _, suffix := range []string{"_bronze", "_silver", "_gold"} {
		if strings.HasSuffix(rest, suffix) {
			tier = strings.TrimPrefix(suffix, "_")
			rest = strings.TrimSuffix(rest, suffix)
			break
		}
	}
	labels := map[string]string{
		"cachekeeper": "Cachekeeper", "bossbreaker": "Bossbreaker", "expeditioner": "Expeditioner",
		"veteran_iron": "Iron Nerve", "veteran_untouched": "Untouched", "veteran_boss": "Boss Oath",
		"coordinator": "Synchronized", "party_combo": "Chain Reaction", "ghost_party": "Ghost Chaser",
	}
	label := labels[rest]
	if label == "" || tier == "" {
		return ""
	}
	return fmt.Sprintf("%s (%s)", label, strings.ToUpper(tier[:1])+tier[1:])
}

func abyssSanctuaryStage(upgrades map[string]int) (int, string) {
	total := 0
	for _, level := range upgrades {
		total += level
	}
	switch {
	case total >= 7:
		return 3, "Radiant Refuge"
	case total >= 4:
		return 2, "Fortified Haven"
	case total >= 1:
		return 1, "Kindled Camp"
	default:
		return 0, "Forgotten Alcove"
	}
}

// abyssPermanentBonus returns 1+bonus with diminishing returns after capAt.
func abyssPermanentBonus(rawBonus, capAt float64) float64 {
	return 1 + softCap(rawBonus, capAt)
}
