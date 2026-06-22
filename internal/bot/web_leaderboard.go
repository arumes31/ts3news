package bot

import (
	"net/http"
	"time"
)

// leaderRow is one ranked player on a "most wins" leaderboard.
type leaderRow struct {
	Rank     int
	Nickname string
	Wins     int
	NetGold  int64
}

// leaderboards bundles the three time-windowed rankings shown on the arcade and
// auto-battler pages.
type leaderboards struct {
	Day     []leaderRow
	Month   []leaderRow
	AllTime []leaderRow
}

// recordGameResult logs one play of a game for the leaderboards. Best-effort:
// failures never block the game outcome.
func (b *Bot) recordGameResult(uid, game string, won bool, net int64) {
	_, _ = b.DB.Exec(
		"INSERT INTO game_results (client_uid, game, won, net) VALUES ($1,$2,$3,$4)",
		uid, game, won, net)
}

// topWinners returns up to limit players ranked by win count (ties broken by net
// gold) for the given game since since.
func (b *Bot) topWinners(game string, since time.Time, limit int) []leaderRow {
	rows, err := b.DB.Query(
		`SELECT COALESCE(NULLIF(u.nickname, ''), 'Adventurer') AS nick,
		        COUNT(*) FILTER (WHERE g.won)        AS wins,
		        COALESCE(SUM(g.net), 0)              AS net
		   FROM game_results g
		   LEFT JOIN users u ON u.client_uid = g.client_uid
		  WHERE g.game = $1 AND g.created_at >= $2
		  GROUP BY g.client_uid, u.nickname
		 HAVING COUNT(*) FILTER (WHERE g.won) > 0
		  ORDER BY wins DESC, net DESC
		  LIMIT $3`,
		game, since, limit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []leaderRow
	rank := 1
	for rows.Next() {
		var r leaderRow
		if err := rows.Scan(&r.Nickname, &r.Wins, &r.NetGold); err != nil {
			continue
		}
		r.Rank = rank
		rank++
		out = append(out, r)
	}
	return out
}

// gameLeaderboards assembles the 1-day / 30-day / all-time rankings for a game.
func (b *Bot) gameLeaderboards(game string) leaderboards {
	const top = 10
	now := time.Now()
	return leaderboards{
		Day:     b.topWinners(game, now.AddDate(0, 0, -1), top),
		Month:   b.topWinners(game, now.AddDate(0, 0, -30), top),
		AllTime: b.topWinners(game, time.Unix(0, 0), top),
	}
}

// handleLeaderboardsPage renders the standalone /leaderboards page showing all
// leaderboard categories: arcade, TFT battle, and 3D games.
func (s *WebServer) handleLeaderboardsPage(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}
	s.render(w, "leaderboards-page", map[string]any{
		"Title":         "Leaderboards",
		"Nav":           "leaderboards",
		"U":             u,
		"ArcadeLeaders": s.bot.gameLeaderboards("arcade"),
	})
}
