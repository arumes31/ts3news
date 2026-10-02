package bot

import "time"

type riftChallengeView struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	Criteria     string   `json:"criteria"`
	MinHazards   int      `json:"min_hazards"`
	MinEnemies   int      `json:"min_enemies"`
	Difficulties []string `json:"difficulties"`
}

func riftChallenge(now time.Time) riftChallengeView {
	now = now.UTC()
	_, key := abyssDailyChallengeAt(now)
	return riftChallengeFor(key)
}

func riftChallengeFor(key string) riftChallengeView {
	label := abyssDailyAffixLabel(key)
	criteria := "Requires hazard zones in combat rooms"
	minHazards := 1
	minEnemies := 0
	targetDifficulties := []string{"Wayfarer", "Veteran", "Champion", "Mythic"}
	switch key {
	case "double_hazards":
		criteria = "Requires at least 2 hazard zones across the route"
		minHazards = 2
	case "enraged_mobs", "vampiric_mobs":
		criteria = "Requires at least 15 defenders across the mission"
		minEnemies = 15
	case "glass_cannon", "execute":
		criteria = "Requires Veteran, Champion, or Mythic difficulty"
		targetDifficulties = []string{"Veteran", "Champion", "Mythic"}
	case "gold_rush":
		criteria = "Requires Champion or Mythic difficulty"
		targetDifficulties = []string{"Champion", "Mythic"}
	case "iron_skin", "bloodlust", "zero_durability_loss":
		criteria = "Compatible with tactical cover layouts"
		minHazards = 0
	}
	return riftChallengeView{
		Key:          key,
		Label:        label,
		Criteria:     criteria,
		MinHazards:   minHazards,
		MinEnemies:   minEnemies,
		Difficulties: targetDifficulties,
	}
}
