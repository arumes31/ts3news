package bot

import (
	"fmt"
	"strconv"
	"time"
)

const (
	abyssWeeklyTalentXPPerFloor = 10
	abyssTalentXPPerPoint       = 100
	abyssWeeklyTalentXPCap      = 500
	abyssAchievementPointCap    = 50
	abyssTierMasteryFloorGoal   = 100
	abyssTierMasteryPointReward = 5
)

type abyssProgressionPointRewards struct {
	Points  int
	Sources []abyssTreePointSource
}

func abyssWeeklyTalentXPKey(uid string, at time.Time) string {
	return "abyss_weekly_talent_xp_" + abyssCurrentWeek(at) + "_" + uid
}

func buildAbyssProgressionPointRewards(
	achievements int,
	weeklyXP int,
	tierFloors map[string]int64,
) abyssProgressionPointRewards {
	achievements = min(max(achievements, 0), abyssAchievementPointCap)
	weeklyXP = min(max(weeklyXP, 0), abyssWeeklyTalentXPCap)
	achievementSource := abyssTreePointSource{
		Key: "achievements", Label: "Achievement talent points", Earned: achievements,
		Progress: achievements, Target: min(abyssAchievementPointCap, achievements+1),
		NextReward: 1, NextLabel: "Earn another Abyss achievement",
	}
	if achievements >= abyssAchievementPointCap {
		achievementSource.NextReward = 0
		achievementSource.NextLabel = "Achievement talent-point cap reached"
	}
	weeklySource := abyssTreePointSource{
		Key: "weekly_talent_xp", Label: "Weekly challenge talent XP", Earned: weeklyXP / abyssTalentXPPerPoint,
		Progress: weeklyXP % abyssTalentXPPerPoint, Target: abyssTalentXPPerPoint,
		NextReward: 1, NextLabel: fmt.Sprintf("Earn %d talent XP per Weekly Expedition floor", abyssWeeklyTalentXPPerFloor),
	}
	if weeklyXP >= abyssWeeklyTalentXPCap {
		weeklySource.Progress = abyssTalentXPPerPoint
		weeklySource.NextReward = 0
		weeklySource.NextLabel = "Weekly talent-XP cap reached"
	}
	result := abyssProgressionPointRewards{
		Points:  achievements + weeklyXP/abyssTalentXPPerPoint,
		Sources: []abyssTreePointSource{achievementSource, weeklySource},
	}
	for _, tierKey := range abyssTierOrder {
		tier := abyssTiers[tierKey]
		floors := max(tierFloors[tier.Key], 0)
		earned := 0
		if floors >= abyssTierMasteryFloorGoal {
			earned = abyssTierMasteryPointReward
			result.Points += earned
		}
		source := abyssTreePointSource{
			Key: "tier_" + tier.Key, Label: tier.Name + " tier mastery", Earned: earned,
			Progress: int(min(floors, abyssTierMasteryFloorGoal)), Target: abyssTierMasteryFloorGoal,
			NextReward: abyssTierMasteryPointReward, NextLabel: "Clear floors in completed " + tier.Name + " runs",
		}
		if earned > 0 {
			source.NextReward = 0
			source.NextLabel = tier.Name + " tier mastery complete"
		}
		result.Sources = append(result.Sources, source)
	}
	return result
}

func (b *Bot) abyssProgressionPointRewards(uid string, at time.Time) abyssProgressionPointRewards {
	var achievements int
	_ = b.DB.QueryRow("SELECT COUNT(*) FROM abyss_achievements WHERE client_uid=$1", uid).Scan(&achievements)

	weeklyXP := 0
	var rawWeeklyXP string
	if b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssWeeklyTalentXPKey(uid, at)).Scan(&rawWeeklyXP) == nil {
		weeklyXP, _ = strconv.Atoi(rawWeeklyXP)
	}

	tierFloors := make(map[string]int64, len(abyssTiers))
	rows, err := b.DB.Query(`SELECT tier, COALESCE(SUM(floors_cleared), 0)
		FROM abyss_runs WHERE client_uid=$1 AND victory=TRUE GROUP BY tier`, uid)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var tier string
			var floors int64
			if rows.Scan(&tier, &floors) == nil {
				tierFloors[tier] = floors
			}
		}
	}
	return buildAbyssProgressionPointRewards(achievements, weeklyXP, tierFloors)
}

func (b *Bot) awardAbyssWeeklyTalentXP(uid string, at time.Time) {
	_, _ = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = LEAST($3, COALESCE(NULLIF(app_meta.value, ''), '0')::integer + $2)::text`,
		abyssWeeklyTalentXPKey(uid, at), abyssWeeklyTalentXPPerFloor, abyssWeeklyTalentXPCap)
}

func abyssNewPlayerXPPercent(lifetimeFloors int64) int {
	if lifetimeFloors >= 0 && lifetimeFloors < 10 {
		return 100
	}
	return 0
}
