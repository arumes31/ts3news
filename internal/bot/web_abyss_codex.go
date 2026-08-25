package bot

import (
	"sort"
	"time"
)

type abyssBestiaryRow struct {
	MobName       string
	Family        string
	Kills         int
	FirstKillAt   time.Time
	LastKillAt    time.Time
	FirstKillISO  string
	LastKillISO   string
	Milestone     string
	NextMilestone int
	KillsToNext   int
	Mastered      bool
}

type abyssBestiaryKill struct {
	MobName string
	Family  string
}

const abyssLegacyBestiaryFamily = "Legacy encounters"

func (b *Bot) loadAbyssBestiary(uid string) []abyssBestiaryRow {
	rows, err := b.DB.Query(
		`SELECT mob_name, mob_family, kills, first_kill_at, last_kill_at
		 FROM abyss_bestiary WHERE client_uid = $1
		 ORDER BY mob_family, kills DESC, mob_name`,
		uid,
	)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []abyssBestiaryRow
	for rows.Next() {
		var row abyssBestiaryRow
		if err := rows.Scan(&row.MobName, &row.Family, &row.Kills, &row.FirstKillAt, &row.LastKillAt); err != nil {
			continue
		}
		if row.Family == "" {
			row.Family = abyssLegacyBestiaryFamily
		}
		row.FirstKillISO = row.FirstKillAt.UTC().Format(time.RFC3339)
		row.LastKillISO = row.LastKillAt.UTC().Format(time.RFC3339)
		row.Milestone, row.NextMilestone, row.KillsToNext = abyssBestiaryMilestone(row.Kills)
		row.Mastered = row.Kills >= 100
		out = append(out, row)
	}
	return out
}

func abyssBestiaryMilestone(kills int) (string, int, int) {
	milestones := []struct {
		Kills int
		Name  string
	}{
		{1, "Discovered"}, {10, "Tracked"}, {25, "Studied"}, {50, "Hunted"},
		{100, "Mastered"}, {250, "Nemesis"}, {500, "Legend"}, {1000, "Mythic"},
	}
	current := "Undiscovered"
	for _, milestone := range milestones {
		if kills < milestone.Kills {
			return current, milestone.Kills, milestone.Kills - max(kills, 0)
		}
		current = milestone.Name
	}
	return current, 0, 0
}

func (b *Bot) recordAbyssKills(uid string, kills []abyssBestiaryKill) {
	for _, kill := range kills {
		_, _ = b.DB.Exec(
			`INSERT INTO abyss_bestiary (client_uid, mob_name, mob_family, kills, first_kill_at, last_kill_at)
			 VALUES ($1, $2, $3, 1, NOW(), NOW())
			 ON CONFLICT (client_uid, mob_name)
			 DO UPDATE SET kills = abyss_bestiary.kills + 1,
			               mob_family = EXCLUDED.mob_family,
			               last_kill_at = NOW()`,
			uid, kill.MobName, kill.Family,
		)
	}
}

type abyssAchievementView struct {
	Code      string
	Name      string
	Condition string
	Earned    bool
}

var abyssAchievementCatalog = []abyssAchievementView{
	{Code: "depth_10", Condition: "Reach depth 10 in a single run"},
	{Code: "depth_25", Condition: "Reach depth 25 in a single run"},
	{Code: "depth_50", Condition: "Reach depth 50 in a single run"},
	{Code: "depth_100", Condition: "Reach depth 100 in a single run"},
	{Code: "boss_1", Condition: "Defeat your first floor boss"},
	{Code: "boss_25", Condition: "Defeat 25 bosses across all runs"},
	{Code: "boss_100", Condition: "Defeat 100 bosses across all runs"},
	{Code: "bank_1m", Condition: "Bank 1,000,000 gold across all runs"},
	{Code: "bank_10m", Condition: "Bank 10,000,000 gold across all runs"},
	{Code: "bestiary_25", Condition: "Defeat 25 distinct monster species"},
	{Code: "bestiary_50", Condition: "Defeat 50 distinct monster species"},
	{Code: "bestiary_complete", Condition: "Complete the 50-species Abyss codex"},
	{Code: "prestige_1", Condition: "Prestige the Abyss once"},
	{Code: "hardcore_depth_10", Condition: "Reach depth 10 in Hardcore mode"},
	{Code: "perfect_run", Condition: "Bank a run after taking no damage"},
}

func allAbyssAchievementViews() []abyssAchievementView {
	views := append([]abyssAchievementView(nil), abyssAchievementCatalog...)
	views = append(views, abyssPactAchievementViews()...)
	for _, track := range abyssProgressTrackDefs {
		for _, tier := range []struct {
			Suffix string
			Count  int
		}{{"bronze", 10}, {"silver", 50}, {"gold", 200}} {
			code := "progress_" + track.Key + "_" + tier.Suffix
			views = append(views, abyssAchievementView{
				Code: code, Condition: "Complete " + itoa(tier.Count) + " " + track.Name + " objectives",
			})
		}
	}
	for i := range views {
		views[i].Name = abyssAchievementName(views[i].Code)
	}
	return views
}

func (b *Bot) abyssAchievementViews(uid string) []abyssAchievementView {
	earned := make(map[string]bool)
	rows, err := b.DB.Query("SELECT code FROM abyss_achievements WHERE client_uid=$1", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil
		}
		earned[code] = true
	}
	if rows.Err() != nil {
		return nil
	}
	views := allAbyssAchievementViews()
	known := make(map[string]bool, len(views))
	for i := range views {
		known[views[i].Code] = true
		views[i].Earned = earned[views[i].Code]
	}
	var legacy []string
	for code := range earned {
		if !known[code] {
			legacy = append(legacy, code)
		}
	}
	sort.Strings(legacy)
	for _, code := range legacy {
		views = append(views, abyssAchievementView{
			Code: code, Name: abyssAchievementName(code), Condition: "Legacy achievement", Earned: true,
		})
	}
	return views
}
