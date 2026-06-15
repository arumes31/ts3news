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

// arcade3dLeaderRow is one ranked player on a 3D game score leaderboard.
type arcade3dLeaderRow struct {
	Rank        int
	Nickname    string
	HighScore   int64
	GoldEarned  int64
	TimesPlayed int
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

// top3DScores returns up to limit players ranked by highest score (ties broken
// by total gold earned) for the given 3D game since the given time.
func (b *Bot) top3DScores(game string, since time.Time, limit int) []arcade3dLeaderRow {
	rows, err := b.DB.Query(
		`SELECT COALESCE(NULLIF(u.nickname, ''), 'Adventurer') AS nick,
		        MAX(a.score)        AS high_score,
		        COALESCE(SUM(a.gold_awarded), 0) AS gold_earned,
		        COUNT(*)            AS times_played
		   FROM arcade3d_scores a
		   LEFT JOIN users u ON u.client_uid = a.client_uid
		  WHERE a.game = $1 AND a.created_at >= $2
		  GROUP BY a.client_uid, u.nickname
		  ORDER BY high_score DESC, gold_earned DESC
		  LIMIT $3`,
		game, since, limit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []arcade3dLeaderRow
	rank := 1
	for rows.Next() {
		var r arcade3dLeaderRow
		if err := rows.Scan(&r.Nickname, &r.HighScore, &r.GoldEarned, &r.TimesPlayed); err != nil {
			continue
		}
		r.Rank = rank
		rank++
		out = append(out, r)
	}
	return out
}

// arcade3dLeaderboards bundles the three time-windowed score rankings for a
// single 3D game.
type arcade3dLeaderboards struct {
	Day     []arcade3dLeaderRow
	Month   []arcade3dLeaderRow
	AllTime []arcade3dLeaderRow
}

// get3DGameLeaderboards returns a map of game ID → leaderboard data for all
// tracked 3D games. Each game includes Day, Month, and AllTime rankings.
func (b *Bot) get3DGameLeaderboards() map[string]arcade3dLeaderboards {
	games := b.get3DGameList()
	result := make(map[string]arcade3dLeaderboards, len(games))
	const top = 10
	now := time.Now()
	for _, g := range games {
		result[g.ID] = arcade3dLeaderboards{
			Day:     b.top3DScores(g.ID, now.AddDate(0, 0, -1), top),
			Month:   b.top3DScores(g.ID, now.AddDate(0, 0, -30), top),
			AllTime: b.top3DScores(g.ID, time.Unix(0, 0), top),
		}
	}
	return result
}

// game3DInfo holds the display metadata for a tracked 3D game.
type game3DInfo struct {
	ID   string // machine-readable game identifier (e.g., "snake")
	Name string // human-readable display name (e.g., "3D Snake")
}

// get3DGameList returns the list of all tracked 3D games with their display
// names. The game IDs are derived from the arcade3dGames list in web_arcade3d.go.
func (b *Bot) get3DGameList() []game3DInfo {
	return []game3DInfo{
		{"game_01_3d_snake", "3D Snake"},
		{"game_02_pong", "3D Pong"},
		{"game_03_space_invaders", "Voxel Invaders"},
		{"game_04_frogger", "Frogger"},
		{"game_05_endless_runner", "Endless Runner"},
		{"game_06_meteor_dodger", "Meteor Dodger"},
		{"game_07_3d_tetris", "3D Tetris"},
		{"game_08_maze_chaser", "Maze Chaser"},
		{"game_09_asteroids", "Asteroids"},
		{"game_10_breakout", "Breakout"},
		{"game_11_missile_command", "Missile Command"},
		{"game_12_tunnel_flyer", "Tunnel Flyer"},
		{"game_13_tower_stack", "Tower Stack"},
		{"game_14_block_dodger", "Block Dodger"},
		{"game_15_whack_a_mole", "Whack-a-Mole"},
		{"game_16_air_hockey", "Air Hockey"},
		{"game_17_tank_battle", "Tank Battle"},
		{"game_18_helix_drop", "Helix Drop"},
		{"game_19_galaxy_shooter", "Galaxy Shooter"},
		{"game_20_cube_runner", "Cube Runner"},
		{"game_21_gem_collector", "Gem Collector"},
		{"game_22_laser_defense", "Laser Defense"},
		{"game_23_star_dodger", "Star Dodger"},
		{"game_24_color_catch", "Colour Catch"},
		{"game_25_platform_jumper", "Platform Jumper"},
		{"game_26_lunar_lander", "Lunar Lander"},
		{"game_27_crossy_road", "Crossy Road"},
		{"game_28_light_cycles", "Light Cycles"},
		{"game_29_ring_flyer", "Ring Flyer"},
		{"game_30_simon_memory", "Simon Memory"},
		{"game_31_tft_battler", "TFT Auto-Battler"},
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
	s.render(w, "leaderboards", map[string]any{
		"Title":           "Leaderboards",
		"Nav":             "leaderboards",
		"U":               u,
		"ArcadeLeaders":   s.bot.gameLeaderboards("arcade"),
		"TFTLeaders":      s.bot.gameLeaderboards("tft"),
		"Arcade3DLeaders": s.bot.get3DGameLeaderboards(),
		"Game3DList":      s.bot.get3DGameList(),
	})
}
