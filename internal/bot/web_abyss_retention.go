package bot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

const (
	abyssSeasonObjectivesPerWeek = 5
	abyssLoginCalendarDays       = 28
	abyssEndlessFirstRewardDepth = 125
	abyssEndlessRewardStep       = 25
	abyssEndlessMaxClaimDepth    = 10_000
)

var abyssSeasonObjectiveVerbs = []string{"Scout", "Chart", "Endure", "Break", "Conquer"}

type abyssLoginDayView struct {
	Day     int
	Date    string
	Weekday string
	Reward  string
	Claimed bool
	Current bool
	Future  bool
}

type abyssLoginCalendarView struct {
	CycleLabel   string
	Days         []abyssLoginDayView
	Claimed      int
	CurrentDay   int
	TodayClaimed bool
	TodayReward  string
	LoadError    string
}

type abyssWeeklyDigestView struct {
	Depth          int
	DepthTrend     int
	Gold           int64
	GoldPerDay     int64
	GoldTrend      int64
	Runs           int
	RunsTrend      int
	Floors         int
	FloorsTrend    int
	PreviousDepth  int
	PreviousGold   int64
	PreviousRuns   int
	PreviousFloors int
	NearRecord     string
	LoadError      string
}

type abyssEndlessRewardView struct {
	Depth     int
	Rank      int
	Name      string
	Key       string
	Available bool
	Owned     bool
}

type abyssEndlessView struct {
	Unlocked  bool
	BestDepth int
	Renown    int
	NextDepth int
	Rewards   []abyssEndlessRewardView
}

type abyssRetentionView struct {
	Login   abyssLoginCalendarView
	Digest  abyssWeeklyDigestView
	Endless abyssEndlessView
}

type abyssDigestMetrics struct {
	Depth          int
	Gold           int64
	Runs           int
	Floors         int
	PreviousDepth  int
	PreviousGold   int64
	PreviousRuns   int
	PreviousFloors int
}

type abyssLoginClaimResult struct {
	Gold           int64
	Tokens         int64
	RewardGold     int64
	RewardTokens   int64
	AlreadyClaimed bool
}

func enrichAbyssSeasonJournal(
	view *abyssSeasonJourneyView,
	campaign abyssSeasonCampaign,
	progress [abyssSeasonWeeks]int64,
	owned map[string]bool,
) {
	view.ObjectiveTotal = abyssSeasonWeeks * abyssSeasonObjectivesPerWeek
	view.Objectives = make([]abyssSeasonObjectiveView, 0, view.ObjectiveTotal)
	for week := 1; week <= abyssSeasonWeeks; week++ {
		weeklyGoal := abyssSeasonWeekGoals[week-1]
		floors := progress[week-1]
		for stage := 1; stage <= abyssSeasonObjectivesPerWeek; stage++ {
			goal := (weeklyGoal*int64(stage) + abyssSeasonObjectivesPerWeek - 1) /
				abyssSeasonObjectivesPerWeek
			percent := min(100, int(floors*100/max(goal, 1)))
			complete := floors >= goal
			view.Objectives = append(view.Objectives, abyssSeasonObjectiveView{
				ID:        fmt.Sprintf("w%02d-%02d", week, stage),
				Week:      week,
				Stage:     stage,
				Label:     fmt.Sprintf("%s %d season %s", abyssSeasonObjectiveVerbs[stage-1], goal, pluralWord(goal, "floor", "floors")),
				Goal:      goal,
				Progress:  min(floors, goal),
				Percent:   percent,
				Available: week <= campaign.CurrentWeek,
				Complete:  complete,
			})
			if complete {
				view.ObjectivesDone++
			}
		}
	}
	view.FinaleName = campaign.RewardWord + " Chronicle Mantle"
	view.FinaleUnlocked = view.ObjectivesDone == view.ObjectiveTotal
	view.FinaleClaimed = owned[abyssSeasonJournalFinaleKey(campaign)]
}

func abyssSeasonJournalFinaleKey(campaign abyssSeasonCampaign) string {
	return "season_" + campaign.ID + "_journal_finale"
}

func (b *Bot) abyssRetentionProgram(
	ctx context.Context,
	uid string,
	at time.Time,
	bestDepth int,
	owned map[string]bool,
) abyssRetentionView {
	claimedDates, loginErr := b.abyssLoginClaimedDates(ctx, uid, at)
	login := abyssLoginCalendarAt(at, claimedDates)
	if loginErr != nil {
		login.LoadError = "Daily calendar progress is temporarily unavailable."
	}
	digest, digestErr := b.abyssWeeklyDigest(ctx, uid, at, bestDepth)
	if digestErr != nil {
		digest.LoadError = "Weekly run digest is temporarily unavailable."
	}
	return abyssRetentionView{
		Login:   login,
		Digest:  digest,
		Endless: abyssEndlessProgram(bestDepth, owned),
	}
}

func abyssLoginCalendarAt(at time.Time, claimedDates map[string]bool) abyssLoginCalendarView {
	today := utcDay(at)
	elapsedDays := int(today.Sub(abyssSeasonAnchor).Hours() / 24)
	currentIndex := elapsedDays % abyssLoginCalendarDays
	if currentIndex < 0 {
		currentIndex += abyssLoginCalendarDays
	}
	cycleStart := today.AddDate(0, 0, -currentIndex)
	view := abyssLoginCalendarView{
		CycleLabel: cycleStart.Format("02 Jan") + " — " +
			cycleStart.AddDate(0, 0, abyssLoginCalendarDays-1).Format("02 Jan 2006"),
		Days:       make([]abyssLoginDayView, 0, abyssLoginCalendarDays),
		CurrentDay: currentIndex + 1,
	}
	for index := range abyssLoginCalendarDays {
		day := cycleStart.AddDate(0, 0, index)
		date := day.Format("2006-01-02")
		gold, tokens := abyssLoginReward(index + 1)
		reward := fmt.Sprintf("%dg", gold)
		if tokens > 0 {
			reward = fmt.Sprintf("%d %s", tokens, pluralWord(tokens, "token", "tokens"))
		}
		claimed := claimedDates[date]
		current := day.Equal(today)
		view.Days = append(view.Days, abyssLoginDayView{
			Day: index + 1, Date: date, Weekday: day.Format("Mon"), Reward: reward,
			Claimed: claimed, Current: current, Future: day.After(today),
		})
		if claimed {
			view.Claimed++
		}
		if current {
			view.TodayClaimed = claimed
			view.TodayReward = reward
		}
	}
	return view
}

func abyssLoginReward(day int) (int64, int64) {
	day = min(max(day, 1), abyssLoginCalendarDays)
	if day%7 == 0 {
		return 0, int64(day / 7)
	}
	return int64(400 + day*100), 0
}

func utcDay(at time.Time) time.Time {
	utc := at.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func abyssLoginClaimPrefix(uid string) string {
	digest := sha256.Sum256([]byte(uid))
	return "abyss_login_" + hex.EncodeToString(digest[:8]) + "_"
}

func abyssLoginClaimKey(uid string, day time.Time) string {
	return abyssLoginClaimPrefix(uid) + utcDay(day).Format("2006-01-02")
}

func (b *Bot) abyssLoginClaimedDates(
	ctx context.Context,
	uid string,
	at time.Time,
) (map[string]bool, error) {
	claimed := make(map[string]bool, abyssLoginCalendarDays)
	calendar := abyssLoginCalendarAt(at, map[string]bool{})
	firstDate := calendar.Days[0].Date
	lastDate := calendar.Days[len(calendar.Days)-1].Date
	prefix := abyssLoginClaimPrefix(uid)
	rows, err := b.DB.QueryContext(
		ctx,
		"SELECT key FROM app_meta WHERE key >= $1 AND key <= $2",
		prefix+firstDate,
		prefix+lastDate,
	)
	if err != nil {
		return claimed, fmt.Errorf("querying Abyss login calendar: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return claimed, fmt.Errorf("scanning Abyss login calendar: %w", err)
		}
		if len(key) >= len("2006-01-02") {
			date := key[len(key)-len("2006-01-02"):]
			if _, err := time.Parse("2006-01-02", date); err == nil {
				claimed[date] = true
			}
		}
	}
	if err := rows.Err(); err != nil {
		return claimed, fmt.Errorf("iterating Abyss login calendar: %w", err)
	}
	return claimed, nil
}

func (b *Bot) abyssWeeklyDigest(
	ctx context.Context,
	uid string,
	at time.Time,
	bestDepth int,
) (abyssWeeklyDigestView, error) {
	weekStart := utcDay(at).AddDate(0, 0, -(int(at.UTC().Weekday())+6)%7)
	previousStart := weekStart.AddDate(0, 0, -7)
	var metrics abyssDigestMetrics
	err := b.DB.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(depth) FILTER (WHERE created_at >= $2),0),
		       COALESCE(SUM(gold_banked) FILTER (WHERE created_at >= $2),0),
		       COUNT(*) FILTER (WHERE created_at >= $2),
		       COALESCE(SUM(floors_cleared) FILTER (WHERE created_at >= $2),0),
		       COALESCE(MAX(depth) FILTER (WHERE created_at < $2),0),
		       COALESCE(SUM(gold_banked) FILTER (WHERE created_at < $2),0),
		       COUNT(*) FILTER (WHERE created_at < $2),
		       COALESCE(SUM(floors_cleared) FILTER (WHERE created_at < $2),0)
		FROM abyss_runs
		WHERE client_uid=$1 AND created_at >= $3`, uid, weekStart, previousStart).
		Scan(
			&metrics.Depth,
			&metrics.Gold,
			&metrics.Runs,
			&metrics.Floors,
			&metrics.PreviousDepth,
			&metrics.PreviousGold,
			&metrics.PreviousRuns,
			&metrics.PreviousFloors,
		)
	if err != nil {
		return abyssWeeklyDigestView{}, fmt.Errorf("querying Abyss weekly digest: %w", err)
	}
	elapsedDays := max(1, int(utcDay(at).Sub(weekStart).Hours()/24)+1)
	return abyssWeeklyDigestFromMetrics(metrics, bestDepth, elapsedDays), nil
}

func abyssWeeklyDigestFromMetrics(
	metrics abyssDigestMetrics,
	bestDepth int,
	elapsedDays int,
) abyssWeeklyDigestView {
	view := abyssWeeklyDigestView{
		Depth: metrics.Depth, DepthTrend: metrics.Depth - metrics.PreviousDepth,
		Gold: metrics.Gold, GoldPerDay: metrics.Gold / int64(max(elapsedDays, 1)),
		GoldTrend: metrics.Gold - metrics.PreviousGold,
		Runs:      metrics.Runs, RunsTrend: metrics.Runs - metrics.PreviousRuns,
		Floors: metrics.Floors, FloorsTrend: metrics.Floors - metrics.PreviousFloors,
		PreviousDepth: metrics.PreviousDepth, PreviousGold: metrics.PreviousGold,
		PreviousRuns: metrics.PreviousRuns, PreviousFloors: metrics.PreviousFloors,
	}
	distance := bestDepth - metrics.Depth
	switch {
	case metrics.Depth > 0 && distance <= 0:
		view.NearRecord = "Personal record pace"
	case metrics.Depth > 0 && distance <= 3:
		view.NearRecord = fmt.Sprintf("%d %s from your record", distance, pluralWord(distance, "floor", "floors"))
	default:
		view.NearRecord = "Build the next record attempt"
	}
	return view
}

func pluralWord[T int | int64](value T, singular, plural string) string {
	if value == 1 {
		return singular
	}
	return plural
}

func abyssEndlessProgram(bestDepth int, owned map[string]bool) abyssEndlessView {
	view := abyssEndlessView{
		Unlocked:  bestDepth >= 100,
		BestDepth: bestDepth,
		Renown:    max(0, (bestDepth-100)/5),
		Rewards:   make([]abyssEndlessRewardView, 0, 6),
	}
	start := abyssEndlessFirstRewardDepth
	if bestDepth >= 200 {
		start = max(start, (bestDepth/abyssEndlessRewardStep-2)*abyssEndlessRewardStep)
	}
	for index := range 6 {
		depth := start + index*abyssEndlessRewardStep
		rank := (depth - 100) / abyssEndlessRewardStep
		key := abyssEndlessCosmeticKey(depth)
		view.Rewards = append(view.Rewards, abyssEndlessRewardView{
			Depth: depth, Rank: rank,
			Name: abyssEndlessRewardName(depth),
			Key:  key, Available: bestDepth >= depth, Owned: owned[key],
		})
		if view.NextDepth == 0 && bestDepth < depth {
			view.NextDepth = depth
		}
	}
	return view
}

func abyssEndlessCosmeticKey(depth int) string {
	return fmt.Sprintf("endless_depth_%04d", depth)
}

func abyssEndlessRewardName(depth int) string {
	names := []string{"Banner", "Footfall", "Aura", "Portrait"}
	rank := (depth - 100) / abyssEndlessRewardStep
	return fmt.Sprintf("Endless %s · Rank %d", names[(rank-1)%len(names)], rank)
}

func validAbyssEndlessRewardDepth(depth int) bool {
	return depth >= abyssEndlessFirstRewardDepth &&
		depth <= abyssEndlessMaxClaimDepth &&
		depth%abyssEndlessRewardStep == 0
}

func (b *Bot) claimAbyssCosmetic(ctx context.Context, uid, key string) (bool, error) {
	result, err := b.DB.ExecContext(ctx, `
		INSERT INTO abyss_shop_cosmetics (client_uid,cosmetic_key) VALUES ($1,$2)
		ON CONFLICT (client_uid,cosmetic_key) DO NOTHING`, uid, key)
	if err != nil {
		return false, fmt.Errorf("claiming Abyss cosmetic: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("reading Abyss cosmetic claim result: %w", err)
	}
	return changed > 0, nil
}

func (s *WebServer) handleAbyssSeasonJournalFinale(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	campaign := abyssSeasonCampaignAt(time.Now().UTC())
	progress, err := s.bot.abyssSeasonProgress(r.Context(), uid, campaign)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "journal progress is unavailable"})
		return
	}
	view := abyssSeasonJourneyView{}
	enrichAbyssSeasonJournal(&view, campaign, progress, map[string]bool{})
	if !view.FinaleUnlocked {
		writeJSON(w, map[string]any{"ok": false, "error": "complete all 50 season objectives first"})
		return
	}
	claimed, err := s.bot.claimAbyssCosmetic(r.Context(), uid, abyssSeasonJournalFinaleKey(campaign))
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "could not claim the finale cosmetic"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "name": view.FinaleName, "already_owned": !claimed,
	})
}

func (s *WebServer) handleAbyssLoginClaim(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	result, err := s.claimAbyssLogin(r.Context(), uid, time.Now().UTC())
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "could not claim today's expedition cache"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "gold": result.Gold, "tokens": result.Tokens,
		"reward_gold": result.RewardGold, "reward_tokens": result.RewardTokens,
		"already_claimed": result.AlreadyClaimed,
	})
}

func (s *WebServer) claimAbyssLogin(
	ctx context.Context,
	uid string,
	at time.Time,
) (abyssLoginClaimResult, error) {
	calendar := abyssLoginCalendarAt(at, map[string]bool{})
	rewardGold, rewardTokens := abyssLoginReward(calendar.CurrentDay)
	result := abyssLoginClaimResult{RewardGold: rewardGold, RewardTokens: rewardTokens}
	tx, err := s.bot.DB.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("starting Abyss login claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	claim, err := tx.ExecContext(ctx, `
		INSERT INTO app_meta (key,value) VALUES ($1,$2)
		ON CONFLICT (key) DO NOTHING`, abyssLoginClaimKey(uid, at), fmt.Sprintf("day:%d", calendar.CurrentDay))
	if err != nil {
		return result, fmt.Errorf("recording Abyss login claim: %w", err)
	}
	changed, err := claim.RowsAffected()
	if err != nil {
		return result, fmt.Errorf("reading Abyss login claim result: %w", err)
	}
	if changed == 0 {
		result.AlreadyClaimed = true
		rewardGold = 0
		rewardTokens = 0
		result.RewardGold = 0
		result.RewardTokens = 0
	}
	if err := tx.QueryRowContext(ctx, `
		UPDATE users
		SET gold=gold+$1, abyss_tokens=abyss_tokens+$2
		WHERE client_uid=$3
		RETURNING gold,abyss_tokens`, rewardGold, rewardTokens, uid).
		Scan(&result.Gold, &result.Tokens); err != nil {
		return result, fmt.Errorf("granting Abyss login reward: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("committing Abyss login claim: %w", err)
	}
	return result, nil
}

func (s *WebServer) handleAbyssEndlessClaim(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var request struct {
		Depth int `json:"depth"`
	}
	if readJSON(r, &request) != nil || !validAbyssEndlessRewardDepth(request.Depth) {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid endless reward depth"})
		return
	}
	var bestDepth int
	if err := s.bot.DB.QueryRowContext(
		r.Context(),
		"SELECT abyss_best_depth FROM users WHERE client_uid=$1",
		uid,
	).Scan(&bestDepth); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "could not verify endless progress"})
		return
	}
	if bestDepth < request.Depth {
		writeJSON(w, map[string]any{"ok": false, "error": "reach the required endless depth first"})
		return
	}
	key := abyssEndlessCosmeticKey(request.Depth)
	claimed, err := s.bot.claimAbyssCosmetic(r.Context(), uid, key)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "could not claim the endless cosmetic"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "depth": request.Depth, "name": abyssEndlessRewardName(request.Depth), "already_owned": !claimed,
	})
}
