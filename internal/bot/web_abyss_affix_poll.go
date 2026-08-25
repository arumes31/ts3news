package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const abyssWeekendVoteKeyPrefix = "abyss_weekend_affix_vote_"

type abyssWeekendAffixOption struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Votes    int    `json:"votes"`
	Selected bool   `json:"selected"`
}

type abyssWeekendAffixPoll struct {
	Week        string                    `json:"week"`
	Options     []abyssWeekendAffixOption `json:"options"`
	Winner      string                    `json:"winner"`
	WinnerLabel string                    `json:"winner_label"`
	VotingOpen  bool                      `json:"voting_open"`
	StartsAt    string                    `json:"starts_at"`
}

func abyssWeekendWeekKey(at time.Time) string {
	year, week := at.UTC().ISOWeek()
	return fmt.Sprintf("%04d-W%02d", year, week)
}

func abyssWeekendVoteKey(week, uid string) string {
	return abyssWeekendVoteKeyPrefix + week + "_" + uid
}

func abyssWeekendAffixOptions(at time.Time) []string {
	_, week := at.UTC().ISOWeek()
	start := week % len(abyssDailyMods)
	options := make([]string, 0, 3)
	for offset := 0; len(options) < 3; offset++ {
		key := abyssDailyMods[(start+offset*3)%len(abyssDailyMods)]
		known := false
		for _, existing := range options {
			known = known || existing == key
		}
		if !known {
			options = append(options, key)
		}
	}
	return options
}

func abyssWeekendVotingOpen(at time.Time) bool {
	weekday := at.UTC().Weekday()
	return weekday >= time.Monday && weekday <= time.Friday
}

func abyssWeekendStartsAt(at time.Time) time.Time {
	day := at.UTC().Truncate(24 * time.Hour)
	offset := (int(day.Weekday()) + 6) % 7
	monday := day.AddDate(0, 0, -offset)
	return monday.AddDate(0, 0, 5)
}

func abyssWeekendAffixPollBase(at time.Time) abyssWeekendAffixPoll {
	status := abyssWeekendAffixPoll{
		Week: abyssWeekendWeekKey(at), VotingOpen: abyssWeekendVotingOpen(at),
		StartsAt: abyssWeekendStartsAt(at).Format(time.RFC3339), WinnerLabel: "Awaiting votes",
	}
	for _, key := range abyssWeekendAffixOptions(at) {
		status.Options = append(status.Options, abyssWeekendAffixOption{Key: key, Label: abyssDailyAffixLabel(key)})
	}
	return status
}

func (b *Bot) abyssWeekendAffixPoll(uid string, at time.Time) (abyssWeekendAffixPoll, error) {
	status := abyssWeekendAffixPollBase(at)
	week := status.Week
	counts := make(map[string]int, len(status.Options))
	rows, err := b.DB.Query("SELECT value, COUNT(*) FROM app_meta WHERE key LIKE $1 GROUP BY value", abyssWeekendVoteKeyPrefix+week+"_%")
	if err != nil {
		return status, fmt.Errorf("load Abyss weekend affix tally: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var key string
		var votes int
		if err := rows.Scan(&key, &votes); err != nil {
			return status, fmt.Errorf("scan Abyss weekend affix tally: %w", err)
		}
		counts[key] = max(0, votes)
	}
	if err := rows.Err(); err != nil {
		return status, fmt.Errorf("iterate Abyss weekend affix tally: %w", err)
	}
	selected := ""
	if uid != "" {
		err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssWeekendVoteKey(week, uid)).Scan(&selected)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return status, fmt.Errorf("load Abyss weekend affix vote: %w", err)
		}
	}
	bestVotes := -1
	for index := range status.Options {
		status.Options[index].Votes = counts[status.Options[index].Key]
		status.Options[index].Selected = selected == status.Options[index].Key
		if status.Options[index].Votes > bestVotes {
			bestVotes = status.Options[index].Votes
			status.Winner = status.Options[index].Key
			status.WinnerLabel = status.Options[index].Label
		}
	}
	if bestVotes <= 0 {
		status.Winner = ""
		status.WinnerLabel = "Awaiting votes"
	}
	return status, nil
}

func (b *Bot) abyssWeekendAffixWinner(at time.Time) string {
	if b == nil || b.DB == nil {
		return ""
	}
	status, err := b.abyssWeekendAffixPoll("", at)
	if err != nil {
		return ""
	}
	return status.Winner
}

func applyAbyssWeekendAffix(calendar []abyssAffixCalendarDay, winner string) []abyssAffixCalendarDay {
	if winner == "" {
		return calendar
	}
	for index := range calendar {
		if calendar[index].Weekday == "Sat" || calendar[index].Weekday == "Sun" {
			calendar[index].Key = winner
			calendar[index].Label = abyssDailyAffixLabel(winner) + " · community vote"
		}
	}
	return calendar
}

func (s *WebServer) handleAbyssWeekendAffixVote(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Affix string `json:"affix"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	now := time.Now().UTC()
	if !abyssWeekendVotingOpen(now) {
		writeJSON(w, map[string]any{"ok": false, "error": "weekend voting is closed"})
		return
	}
	allowed := false
	for _, key := range abyssWeekendAffixOptions(now) {
		allowed = allowed || req.Affix == key
	}
	if !allowed {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid weekend affix"})
		return
	}
	week := abyssWeekendWeekKey(now)
	if _, err := s.bot.DB.ExecContext(r.Context(), `INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssWeekendVoteKey(week, uid), req.Affix); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	status, err := s.bot.abyssWeekendAffixPoll(uid, now)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "poll": status})
}
