package bot

import (
	"fmt"
	"time"
)

type abyssAffixCalendarDay struct {
	Date     string `json:"date"`
	Weekday  string `json:"weekday"`
	Key      string `json:"key"`
	Label    string `json:"label"`
	Today    bool   `json:"today"`
	Upcoming bool   `json:"upcoming"`
}

type abyssPactFeaturedView struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Week  string `json:"week"`
}

type abyssPactSynergy struct {
	PactKey  string  `json:"pact_key"`
	AffixKey string  `json:"affix_key"`
	Label    string  `json:"label"`
	Bonus    float64 `json:"bonus"`
}

type abyssPactRewardLine struct {
	Key              string  `json:"key"`
	Label            string  `json:"label"`
	BaseBonusPct     float64 `json:"base_bonus_pct"`
	MasteryBonusPct  float64 `json:"mastery_bonus_pct"`
	FeaturedBonusPct float64 `json:"featured_bonus_pct"`
	TotalBonusPct    float64 `json:"total_bonus_pct"`
	Mastered         bool    `json:"mastered"`
	Featured         bool    `json:"featured"`
}

type abyssPactRewardBreakdown struct {
	Lines            []abyssPactRewardLine `json:"lines"`
	Synergies        []abyssPactSynergy    `json:"synergies"`
	BaseBonusPct     float64               `json:"base_bonus_pct"`
	MasteryBonusPct  float64               `json:"mastery_bonus_pct"`
	FeaturedBonusPct float64               `json:"featured_bonus_pct"`
	SynergyBonusPct  float64               `json:"synergy_bonus_pct"`
	MysteryBonusPct  float64               `json:"mystery_bonus_pct"`
	TotalBonusPct    float64               `json:"total_bonus_pct"`
	Multiplier       float64               `json:"multiplier"`
	Featured         abyssPactFeaturedView `json:"featured"`
}

var abyssPactSynergyCatalog = []abyssPactSynergy{
	{PactKey: "anemic", AffixKey: "bloodlust", Label: "Bloodlust + Anemic", Bonus: 0.05},
}

func abyssDailyChallengeAt(at time.Time) (int64, string) {
	at = at.UTC()
	seed := int64(at.Year()*1000 + at.YearDay())
	return seed, abyssDailyMods[seed%int64(len(abyssDailyMods))]
}

func abyssDailyAffixLabel(key string) string {
	labels := map[string]string{
		"double_hazards":       "Doubled Hazards",
		"zero_durability_loss": "Tempered Gear",
		"enraged_mobs":         "Enraged Host",
		"glass_cannon":         "Glass Cannon",
		"gold_rush":            "Gold Rush",
		"iron_skin":            "Iron Skin",
		"bloodlust":            "Bloodlust",
		"execute":              "Execution",
		"vampiric_mobs":        "Vampiric Host",
	}
	if label := labels[key]; label != "" {
		return label
	}
	return key
}

func abyssAffixCalendar(at time.Time) []abyssAffixCalendarDay {
	today := at.UTC().Truncate(24 * time.Hour)
	weekdayOffset := (int(today.Weekday()) + 6) % 7
	monday := today.AddDate(0, 0, -weekdayOffset)
	days := make([]abyssAffixCalendarDay, 0, 7)
	for offset := range 7 {
		day := monday.AddDate(0, 0, offset)
		_, key := abyssDailyChallengeAt(day)
		days = append(days, abyssAffixCalendarDay{
			Date: day.Format("2006-01-02"), Weekday: day.Format("Mon"), Key: key,
			Label: abyssDailyAffixLabel(key), Today: day.Equal(today), Upcoming: day.After(today),
		})
	}
	return days
}

func abyssFeaturedPactAt(at time.Time) abyssPactFeaturedView {
	year, week := at.UTC().ISOWeek()
	index := (year*100 + week) % len(abyssPactCatalog)
	pact := abyssPactCatalog[index]
	return abyssPactFeaturedView{
		Key: pact.Key, Label: pact.Label,
		Week: fmt.Sprintf("%04d-W%02d", year, week),
	}
}

func abyssPactRewardBreakdownAt(pacts []string, mastery map[string]int, dailyAffix string, at time.Time) abyssPactRewardBreakdown {
	return abyssPactRewardBreakdownForRunAt(pacts, mastery, dailyAffix, at, false)
}

func abyssPactRewardBreakdownForRunAt(pacts []string, mastery map[string]int, dailyAffix string, at time.Time, mystery bool) abyssPactRewardBreakdown {
	featured := abyssFeaturedPactAt(at)
	breakdown := abyssPactRewardBreakdown{Featured: featured, Multiplier: 1}
	selected := make(map[string]bool, len(pacts))
	for _, key := range pacts {
		selected[key] = true
	}
	for _, pact := range abyssPactCatalog {
		if !selected[pact.Key] {
			continue
		}
		base := pact.Reward
		masteryBonus := 0.0
		mastered := mastery[pact.Key] >= abyssPactMasteryRuns
		if mastered {
			masteryBonus = base * 0.05
		}
		featuredBonus := 0.0
		isFeatured := pact.Key == featured.Key
		if isFeatured {
			featuredBonus = base + masteryBonus
		}
		lineTotal := base + masteryBonus + featuredBonus
		breakdown.Lines = append(breakdown.Lines, abyssPactRewardLine{
			Key: pact.Key, Label: pact.Label, BaseBonusPct: pct1(base), MasteryBonusPct: pct1(masteryBonus),
			FeaturedBonusPct: pct1(featuredBonus), TotalBonusPct: pct1(lineTotal), Mastered: mastered, Featured: isFeatured,
		})
		breakdown.BaseBonusPct += base * 100
		breakdown.MasteryBonusPct += masteryBonus * 100
		breakdown.FeaturedBonusPct += featuredBonus * 100
		breakdown.Multiplier += lineTotal
	}
	for _, synergy := range abyssPactSynergyCatalog {
		if selected[synergy.PactKey] && dailyAffix == synergy.AffixKey {
			breakdown.Synergies = append(breakdown.Synergies, synergy)
			breakdown.SynergyBonusPct += synergy.Bonus * 100
			breakdown.Multiplier += synergy.Bonus
		}
	}
	if mystery {
		breakdown.MysteryBonusPct = abyssMysteryPactReward * 100
		breakdown.Multiplier += abyssMysteryPactReward
	}
	breakdown.BaseBonusPct = pctNumber1(breakdown.BaseBonusPct)
	breakdown.MasteryBonusPct = pctNumber1(breakdown.MasteryBonusPct)
	breakdown.FeaturedBonusPct = pctNumber1(breakdown.FeaturedBonusPct)
	breakdown.SynergyBonusPct = pctNumber1(breakdown.SynergyBonusPct)
	breakdown.TotalBonusPct = pctNumber1((breakdown.Multiplier - 1) * 100)
	return breakdown
}

func pct1(value float64) float64 { return pctNumber1(value * 100) }

func pctNumber1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

// abyssPactBankTokenGrant converts a conservative share of pact risk into
// tokens. Five hundred floor-percentage points produce one token, with one
// token guaranteed for any completed pact run and a cap of one per floor.
func abyssPactBankTokenGrant(floors int, bonusPct float64) int {
	if floors <= 0 || bonusPct <= 0 {
		return 0
	}
	tokens := int(float64(floors) * bonusPct / 500)
	return min(max(tokens, 1), floors)
}
