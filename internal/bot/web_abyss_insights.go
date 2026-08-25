package bot

import (
	"encoding/json"
	"math"
	"sort"
	"time"

	"ts3news/internal/content"
)

type abyssCountView struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

type abyssTierRateView struct {
	Tier    string `json:"tier"`
	Wins    int    `json:"wins"`
	Runs    int    `json:"runs"`
	Percent int    `json:"percent"`
}

type abyssBestiaryProgressView struct {
	Family  string `json:"family"`
	Seen    int    `json:"seen"`
	Target  int    `json:"target"`
	Percent int    `json:"percent"`
}

type abyssForgeROIView struct {
	Actions       int     `json:"actions"`
	MaterialSpent int64   `json:"material_spent"`
	CRGained      float64 `json:"cr_gained"`
}

type abyssPrestigePreviewView struct {
	Current    int `json:"current"`
	Next       int `json:"next"`
	CurrentPct int `json:"current_pct"`
	NextPct    int `json:"next_pct"`
}

type abyssRunInsightsView struct {
	Runs           []abyssHistoryRow           `json:"runs"`
	DeathCauses    []abyssCountView            `json:"death_causes"`
	TierRates      []abyssTierRateView         `json:"tier_rates"`
	Bestiary       []abyssBestiaryProgressView `json:"bestiary"`
	ForgeROI       abyssForgeROIView           `json:"forge_roi"`
	SessionEscrow  int64                       `json:"session_escrow"`
	SessionSeconds int64                       `json:"session_seconds"`
	SessionGoldPH  int64                       `json:"session_gold_per_hour"`
	Prestige       abyssPrestigePreviewView    `json:"prestige"`
}

var abyssDeathReasonLabels = map[string]string{
	"defeat":        "Combat defeat",
	"revive_failed": "Failed revival",
	"conceded":      "Conceded",
	"timeout":       "Downed timeout",
	"legacy":        "Earlier defeat",
}

var abyssBestiaryFamilyTargets = map[string]int{
	string(content.MobCommon):         12,
	string(content.MobEliteMinion):    8,
	string(content.MobElite):          8,
	string(content.MobMiniboss):       7,
	string(content.MobBoss):           7,
	string(content.MobLegendary):      4,
	string(content.MobTreasureGoblin): 4,
	abyssLegacyBestiaryFamily:         50,
}

func (b *Bot) abyssRunInsights(uid string, run abyssRun, history []abyssHistoryRow, bestiary []abyssBestiaryRow, prestige int) abyssRunInsightsView {
	view := abyssRunInsightsView{
		Runs:          history,
		SessionEscrow: run.Escrow,
		Prestige: abyssPrestigePreviewView{
			Current: prestige, Next: prestige + 1,
			CurrentPct: prestige * 5, NextPct: (prestige + 1) * 5,
		},
	}
	if run.Active && !run.StartedAt.IsZero() {
		view.SessionSeconds = max(int64(time.Since(run.StartedAt).Seconds()), 0)
		if view.SessionSeconds >= 15 {
			view.SessionGoldPH = run.Escrow * 3600 / view.SessionSeconds
		}
	}
	view.DeathCauses = b.abyssDeathCauseViews(uid)
	view.TierRates = b.abyssTierRateViews(uid)
	view.Bestiary = abyssBestiaryProgressViews(bestiary)
	view.ForgeROI = b.abyssForgeROIWeek(uid)
	return view
}

func (b *Bot) abyssDeathCauseViews(uid string) []abyssCountView {
	rows, err := b.DB.Query(`SELECT end_reason, COUNT(*) FROM abyss_runs
		WHERE client_uid=$1 AND victory=FALSE GROUP BY end_reason ORDER BY COUNT(*) DESC, end_reason`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []abyssCountView
	for rows.Next() {
		var reason string
		var count int
		if rows.Scan(&reason, &count) != nil {
			continue
		}
		label := abyssDeathReasonLabels[reason]
		if label == "" {
			label = "Other defeat"
		}
		out = append(out, abyssCountView{Key: reason, Label: label, Count: count})
	}
	return out
}

func (b *Bot) abyssTierRateViews(uid string) []abyssTierRateView {
	rows, err := b.DB.Query(`SELECT tier, COUNT(*) FILTER (WHERE victory), COUNT(*)
		FROM abyss_runs WHERE client_uid=$1 GROUP BY tier ORDER BY tier`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []abyssTierRateView
	for rows.Next() {
		var rate abyssTierRateView
		if rows.Scan(&rate.Tier, &rate.Wins, &rate.Runs) != nil {
			continue
		}
		if rate.Runs > 0 {
			rate.Percent = int(math.Round(float64(rate.Wins) * 100 / float64(rate.Runs)))
		}
		out = append(out, rate)
	}
	return out
}

func abyssBestiaryProgressViews(rows []abyssBestiaryRow) []abyssBestiaryProgressView {
	seen := make(map[string]int)
	for _, row := range rows {
		seen[row.Family]++
	}
	families := make([]string, 0, len(seen))
	for family := range seen {
		families = append(families, family)
	}
	sort.Strings(families)
	out := make([]abyssBestiaryProgressView, 0, len(families))
	for _, family := range families {
		target := abyssBestiaryFamilyTargets[family]
		if target < seen[family] {
			target = seen[family]
		}
		if target == 0 {
			target = 1
		}
		out = append(out, abyssBestiaryProgressView{
			Family: family, Seen: seen[family], Target: target,
			Percent: min(100, int(math.Round(float64(seen[family])*100/float64(target)))),
		})
	}
	return out
}

func (b *Bot) abyssForgeROIWeek(uid string) abyssForgeROIView {
	rows, err := b.DB.Query(`SELECT before_state, after_state FROM abyss_forge_mutation_audit
		WHERE client_uid=$1 AND succeeded=TRUE AND created_at >= NOW() - INTERVAL '7 days'
		ORDER BY id DESC LIMIT 200`, uid)
	if err != nil {
		return abyssForgeROIView{}
	}
	defer func() { _ = rows.Close() }()
	type snapshot struct {
		Item      content.Gear     `json:"item"`
		Materials map[string]int64 `json:"materials"`
	}
	var out abyssForgeROIView
	for rows.Next() {
		var beforeJSON, afterJSON []byte
		if rows.Scan(&beforeJSON, &afterJSON) != nil {
			continue
		}
		var before, after snapshot
		if json.Unmarshal(beforeJSON, &before) != nil || json.Unmarshal(afterJSON, &after) != nil {
			continue
		}
		out.Actions++
		for material, oldCount := range before.Materials {
			if spent := oldCount - after.Materials[material]; spent > 0 {
				out.MaterialSpent += spent
			}
		}
		if gain := after.Item.CombatRating() - before.Item.CombatRating(); gain > 0 {
			out.CRGained += gain
		}
	}
	return out
}
