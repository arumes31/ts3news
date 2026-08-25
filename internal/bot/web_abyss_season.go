package bot

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const (
	abyssSeasonWeeks = 10
	abyssSeasonWeek  = 7 * 24 * time.Hour
)

var abyssSeasonAnchor = time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)

type abyssSeasonDefinition struct {
	Key        string
	Name       string
	Icon       string
	Affinity   string
	Palette    string
	RewardWord string
	Tagline    string
}

var abyssSeasonDefinitions = []abyssSeasonDefinition{
	{Key: "frostbound_vigil", Name: "Frostbound Vigil", Icon: "❄️", Affinity: "frost", Palette: "winter", RewardWord: "Rime", Tagline: "Hold the line where the cold remembers every fallen delver."},
	{Key: "verdant_reawakening", Name: "Verdant Reawakening", Icon: "🌿", Affinity: "nature", Palette: "bloom", RewardWord: "Verdant", Tagline: "Ancient roots split the stone and reclaim the forgotten passages."},
	{Key: "starless_march", Name: "Starless March", Icon: "🌑", Affinity: "void", Palette: "void", RewardWord: "Starless", Tagline: "Walk by memory when the last light disappears below."},
	{Key: "ember_descent", Name: "Ember Descent", Icon: "🔥", Affinity: "fire", Palette: "ember", RewardWord: "Ember", Tagline: "The deep furnaces wake; fire-marked biomes now burn through every depth."},
}

var abyssSeasonRewardNames = []string{
	"Scout Sigil",
	"Footfall Trail",
	"Portrait Frame",
	"Victory Flourish",
	"Run Banner",
	"Companion Tint",
	"Vitality Frame",
	"Depth Crown",
	"Mount Trail",
	"Delver Regalia",
}

var abyssSeasonRewardKinds = []string{
	"Sigil", "Trail", "Frame", "Emote", "Banner",
	"Pet tint", "HUD frame", "Crown", "Mount trail", "Regalia",
}

var abyssSeasonWeekGoals = []int64{5, 8, 10, 12, 15, 18, 20, 22, 25, 30}

type abyssSeasonCampaign struct {
	abyssSeasonDefinition
	ID          string
	Start       time.Time
	End         time.Time
	CurrentWeek int
}

type abyssSeasonRewardView struct {
	Week      int
	Name      string
	Kind      string
	Goal      int64
	Progress  int64
	Percent   int
	Available bool
	Complete  bool
	Claimed   bool
	Current   bool
}

type abyssSeasonJourneyView struct {
	ID          string
	Name        string
	Icon        string
	Affinity    string
	Palette     string
	Tagline     string
	StartLabel  string
	EndLabel    string
	CurrentWeek int
	Weeks       []abyssSeasonRewardView
	Claimed     int
	LoadError   string
}

func abyssSeasonCampaignAt(at time.Time) abyssSeasonCampaign {
	at = at.UTC()
	elapsed := at.Sub(abyssSeasonAnchor)
	seasonLength := time.Duration(abyssSeasonWeeks) * abyssSeasonWeek
	seasonIndex := int64(elapsed / seasonLength)
	if elapsed < 0 && elapsed%seasonLength != 0 {
		seasonIndex--
	}
	definitionIndex := seasonIndex % int64(len(abyssSeasonDefinitions))
	if definitionIndex < 0 {
		definitionIndex += int64(len(abyssSeasonDefinitions))
	}
	start := abyssSeasonAnchor.Add(time.Duration(seasonIndex) * seasonLength)
	definition := abyssSeasonDefinitions[definitionIndex]
	return abyssSeasonCampaign{
		abyssSeasonDefinition: definition,
		ID:                    fmt.Sprintf("%s_%s", definition.Key, start.Format("20060102")),
		Start:                 start,
		End:                   start.Add(seasonLength),
		CurrentWeek:           int(at.Sub(start)/abyssSeasonWeek) + 1,
	}
}

func abyssSeasonCosmeticKey(campaign abyssSeasonCampaign, week int) string {
	return fmt.Sprintf("season_%s_week_%02d", campaign.ID, week)
}

func (b *Bot) abyssSeasonProgress(ctx context.Context, uid string, campaign abyssSeasonCampaign) ([abyssSeasonWeeks]int64, error) {
	var progress [abyssSeasonWeeks]int64
	rows, err := b.DB.QueryContext(ctx, `
		SELECT FLOOR(EXTRACT(EPOCH FROM (created_at - $2::timestamptz)) / 604800)::integer AS week_index,
		       COALESCE(SUM(floors_cleared), 0)::bigint
		FROM abyss_runs
		WHERE client_uid=$1 AND created_at >= $2 AND created_at < $3
		GROUP BY week_index`, uid, campaign.Start, campaign.End)
	if err != nil {
		return progress, fmt.Errorf("query season journey progress: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var weekIndex int
		var floors int64
		if err := rows.Scan(&weekIndex, &floors); err != nil {
			return progress, fmt.Errorf("scan season journey progress: %w", err)
		}
		if weekIndex >= 0 && weekIndex < len(progress) {
			progress[weekIndex] = floors
		}
	}
	if err := rows.Err(); err != nil {
		return progress, fmt.Errorf("iterate season journey progress: %w", err)
	}
	return progress, nil
}

func (b *Bot) abyssSeasonJourney(ctx context.Context, uid string, at time.Time, owned map[string]bool) (abyssSeasonJourneyView, error) {
	campaign := abyssSeasonCampaignAt(at)
	view := abyssSeasonJourneyView{
		ID: campaign.ID, Name: campaign.Name, Icon: campaign.Icon,
		Affinity: campaign.Affinity, Palette: campaign.Palette, Tagline: campaign.Tagline,
		StartLabel: campaign.Start.Format("02 Jan 2006"), EndLabel: campaign.End.Add(-time.Second).Format("02 Jan 2006"),
		CurrentWeek: campaign.CurrentWeek,
		Weeks:       make([]abyssSeasonRewardView, 0, abyssSeasonWeeks),
	}
	progress, err := b.abyssSeasonProgress(ctx, uid, campaign)
	if err != nil {
		view.LoadError = "Journey progress is temporarily unavailable."
	}
	for week := 1; week <= abyssSeasonWeeks; week++ {
		goal := abyssSeasonWeekGoals[week-1]
		floors := progress[week-1]
		claimed := owned[abyssSeasonCosmeticKey(campaign, week)]
		percent := int(floors * 100 / goal)
		if percent > 100 {
			percent = 100
		}
		view.Weeks = append(view.Weeks, abyssSeasonRewardView{
			Week: week, Name: campaign.RewardWord + " " + abyssSeasonRewardNames[week-1],
			Kind: abyssSeasonRewardKinds[week-1], Goal: goal, Progress: floors, Percent: percent,
			Available: err == nil && week <= campaign.CurrentWeek,
			Complete:  err == nil && floors >= goal,
			Claimed:   claimed, Current: week == campaign.CurrentWeek,
		})
		if claimed {
			view.Claimed++
		}
	}
	return view, err
}

func (s *WebServer) handleAbyssSeasonClaim(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Week int `json:"week"`
	}
	if readJSON(r, &req) != nil || req.Week < 1 || req.Week > abyssSeasonWeeks {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid season week"})
		return
	}
	now := time.Now().UTC()
	campaign := abyssSeasonCampaignAt(now)
	if req.Week > campaign.CurrentWeek {
		writeJSON(w, map[string]any{"ok": false, "error": "season week is not available yet"})
		return
	}
	progress, err := s.bot.abyssSeasonProgress(r.Context(), uid, campaign)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if progress[req.Week-1] < abyssSeasonWeekGoals[req.Week-1] {
		writeJSON(w, map[string]any{"ok": false, "error": "weekly journey objective is not complete"})
		return
	}
	key := abyssSeasonCosmeticKey(campaign, req.Week)
	result, err := s.bot.DB.ExecContext(r.Context(), `
		INSERT INTO abyss_shop_cosmetics (client_uid,cosmetic_key) VALUES ($1,$2)
		ON CONFLICT (client_uid,cosmetic_key) DO NOTHING`, uid, key)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	changed, err := result.RowsAffected()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	name := campaign.RewardWord + " " + abyssSeasonRewardNames[req.Week-1]
	writeJSON(w, map[string]any{
		"ok": true, "week": req.Week, "name": name, "claimed": true,
		"already_owned": changed == 0,
	})
}
