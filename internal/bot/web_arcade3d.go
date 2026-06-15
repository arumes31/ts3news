package bot

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ts3news/internal/content"
)

// arcade3dGoldCap bounds the gold a single skill-game run can award, since the
// score is reported by the (untrusted) client.
const arcade3dGoldCap = 1500

// arcade3dScoreRewardTier defines a score threshold and its bonus gold reward.
// Tiers provide incremental bonuses for achieving higher scores, encouraging replayability.
var arcade3dScoreRewardTiers = []struct {
	Threshold int64
	BonusGold int
}{
	{100, 10},
	{500, 25},
	{1000, 50},
	{2000, 100},
	{5000, 250},
}

// arcade3dGearTier defines score thresholds for improved gear drop chances.
var arcade3dGearTiers = []struct {
	Threshold  int64
	BaseChance int
	CapChance  int
}{
	{0, 10, 40},    // Base tier: 10% base, 40% cap
	{500, 15, 50},  // Intermediate: 15% base, 50% cap
	{1500, 20, 60}, // Advanced: 20% base, 60% cap
	{3000, 25, 70}, // Expert: 25% base, 70% cap
}

// validGameIDPattern ensures game identifiers match expected format (alphanumeric with underscores/hyphens).
var validGameIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// arcade3dRecentRewards tracks recent reward claims to prevent double-spending.
// Key: "uid:game:score", Value: timestamp
var arcade3dRecentRewards = make(map[string]time.Time)

// mini3DGame is one entry in the 3D games hub.
type mini3DGame struct {
	File  string
	Title string
	Icon  string
	Desc  string
}

// arcade3dGames lists every standalone Three.js game served under /play/. The
// order here is the order shown on the hub.
var arcade3dGames = []mini3DGame{
	{"game_01_3d_snake.html", "3D Snake", "🐍", "Grow without biting your tail"},
	{"game_02_pong.html", "3D Pong", "🏓", "Rally past the CPU paddle"},
	{"game_03_space_invaders.html", "Voxel Invaders", "👾", "Blast the descending swarm"},
	{"game_04_frogger.html", "Frogger", "🐸", "Hop across the busy lanes"},
	{"game_05_endless_runner.html", "Endless Runner", "🏃", "Switch lanes and jump"},
	{"game_06_meteor_dodger.html", "Meteor Dodger", "☄️", "Survive the meteor storm"},
	{"game_07_3d_tetris.html", "3D Tetris", "🧱", "Clear lines, don't top out"},
	{"game_08_maze_chaser.html", "Maze Chaser", "👻", "Grab pellets, dodge the ghost"},
	{"game_09_asteroids.html", "Asteroids", "🪨", "Thrust, rotate, shoot, split"},
	{"game_10_breakout.html", "Breakout", "🧨", "Bounce the ball, smash bricks"},
	{"game_11_missile_command.html", "Missile Command", "🚀", "Intercept incoming missiles"},
	{"game_12_tunnel_flyer.html", "Tunnel Flyer", "🐦", "Flap through the gaps"},
	{"game_13_tower_stack.html", "Tower Stack", "🏗️", "Drop and align the blocks"},
	{"game_14_block_dodger.html", "Block Dodger", "🟦", "Weave past falling blocks"},
	{"game_15_whack_a_mole.html", "Whack-a-Mole", "🔨", "Bonk every mole in time"},
	{"game_16_air_hockey.html", "Air Hockey", "🥅", "Out-score the CPU mallet"},
	{"game_17_tank_battle.html", "Tank Battle", "🛡️", "Hunt enemy tanks, dodge shells"},
	{"game_18_helix_drop.html", "Helix Drop", "🌀", "Spin the tower, drop through gaps"},
	{"game_19_galaxy_shooter.html", "Galaxy Shooter", "🌌", "Auto-fire through the waves"},
	{"game_20_cube_runner.html", "Cube Runner", "🔺", "Weave the neon field"},
	{"game_21_gem_collector.html", "Gem Collector", "💎", "Grab gems, skip the bombs"},
	{"game_22_laser_defense.html", "Laser Defense", "🟢", "Aim the turret, defend the core"},
	{"game_23_star_dodger.html", "Star Dodger", "✨", "Drift clear of the comets"},
	{"game_24_color_catch.html", "Colour Catch", "🎨", "Catch only your colour"},
	{"game_25_platform_jumper.html", "Platform Jumper", "⬆️", "Bounce ever higher"},
	{"game_26_lunar_lander.html", "Lunar Lander", "🌙", "Touch down softly on the pad"},
	{"game_27_crossy_road.html", "Crossy Road", "🚗", "Hop forward as far as you can"},
	{"game_28_light_cycles.html", "Light Cycles", "🏍️", "Out-manoeuvre the rival trail"},
	{"game_29_ring_flyer.html", "Ring Flyer", "💍", "Thread the floating rings"},
	{"game_30_simon_memory.html", "Simon Memory", "🧠", "Repeat the growing sequence"},
	{"game_31_tft_battler.html", "TFT Auto-Battler", "♟️", "Draft, position, auto-fight"},
}

func isKnownGame(file string) bool {
	for _, g := range arcade3dGames {
		if g.File == file {
			return true
		}
	}
	return false
}

// handleArcade3DHub renders the grid of playable 3D games.
func (s *WebServer) handleArcade3DHub(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}
	s.render(w, "arcade3d", map[string]any{
		"Title": "Arcade 3D",
		"Nav":   "games",
		"U":     u,
		"Games": arcade3dGames,
	})
}

// handleArcade3DPlay serves a single embedded game by name (behind auth so the
// reward API can attribute the run to the player via their session cookie).
func (s *WebServer) handleArcade3DPlay(w http.ResponseWriter, r *http.Request, uid string) {
	name := strings.TrimPrefix(r.URL.Path, "/play/")
	if !isKnownGame(name) {
		http.NotFound(w, r)
		return
	}
	b, err := webAssets.ReadFile("webassets/games/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(b)
}

// sanitizeGameID validates and sanitizes a game identifier to prevent SQL injection.
// Returns the sanitized game ID or empty string if invalid.
func sanitizeGameID(game string) string {
	game = strings.TrimSpace(game)
	if len(game) == 0 || len(game) > 50 {
		return ""
	}
	if !validGameIDPattern.MatchString(game) {
		return ""
	}
	return game
}

// calculateArcade3DGold computes gold reward from score with tier bonuses.
// Formula: base = min(score/2 + 20, 1500) + tier bonuses
// Returns the total gold awarded.
func calculateArcade3DGold(score int64) int {
	// Base gold calculation: score/2 + 20, capped at arcade3dGoldCap
	baseGold := int(score/2) + 20
	if baseGold > arcade3dGoldCap {
		baseGold = arcade3dGoldCap
	}

	// Add tier bonuses for achieving higher score thresholds
	bonusGold := 0
	for _, tier := range arcade3dScoreRewardTiers {
		if score >= tier.Threshold {
			bonusGold += tier.BonusGold
		}
	}

	return baseGold + bonusGold
}

// calculateArcade3DGearChance computes gear drop chance based on score tiers.
// Formula: chance = min(score/scale + base, cap)% where scale/base/cap vary by tier
// Returns the drop chance as a percentage (0-100).
func calculateArcade3DGearChance(score int64) int {
	// Find the appropriate tier for this score
	tier := arcade3dGearTiers[0] // default to base tier
	for _, t := range arcade3dGearTiers {
		if score >= t.Threshold {
			tier = t
		}
	}

	// Calculate chance: base + score scaling, capped at tier maximum
	// Scale factor of 50 provides gradual increase within each tier
	chance := tier.BaseChance + int(score/50)
	if chance > tier.CapChance {
		chance = tier.CapChance
	}
	return chance
}

// recordArcade3DResult inserts a game result into the arcade3d_scores table.
// Returns the gear won (if any) as a string.
func (s *WebServer) recordArcade3DResult(uid, game string, score int64, goldAwarded int, gearWon string) error {
	_, err := s.bot.DB.Exec(
		`INSERT INTO arcade3d_scores (client_uid, game, score, gold_awarded, gear_won)
		 VALUES ($1, $2, $3, $4, $5)`,
		uid, game, score, goldAwarded, gearWon,
	)
	return err
}

// isDuplicateReward checks if this exact score was recently claimed by this user for this game.
// Prevents double-spending of the same score submission.
func isDuplicateReward(uid, game string, score int64) bool {
	key := fmt.Sprintf("%s:%s:%d", uid, game, score)
	if ts, exists := arcade3dRecentRewards[key]; exists {
		// Allow re-submission after 5 minutes
		if time.Since(ts) < 5*time.Minute {
			return true
		}
		// Clean up old entry
		delete(arcade3dRecentRewards, key)
	}
	return false
}

// markRewardClaimed records that this score has been claimed for reward purposes.
func markRewardClaimed(uid, game string, score int64) {
	key := fmt.Sprintf("%s:%s:%d", uid, game, score)
	arcade3dRecentRewards[key] = time.Now()

	// Periodically clean up old entries (simple approach: limit map size)
	if len(arcade3dRecentRewards) > 10000 {
		cutoff := time.Now().Add(-10 * time.Minute)
		for k, ts := range arcade3dRecentRewards {
			if ts.Before(cutoff) {
				delete(arcade3dRecentRewards, k)
			}
		}
	}
}

// handleArcade3DReward converts a reported game score into gold (with tier bonuses)
// and a score-scaled gear drop chance, then records the result to arcade3d_scores table.
// Expects POST with JSON body: {"game": "snake", "score": 1500}
func (s *WebServer) handleArcade3DReward(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Game  string `json:"game"`
		Score int64  `json:"score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	// Validate game identifier - sanitize to prevent SQL injection
	game := sanitizeGameID(req.Game)
	if game == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid game identifier"})
		return
	}

	// Validate score is non-negative
	if req.Score < 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "score must be non-negative"})
		return
	}

	// Rate limiting: prevent double-spending the same score
	if isDuplicateReward(uid, game, req.Score) {
		writeJSON(w, map[string]any{"ok": false, "error": "reward already claimed for this score"})
		return
	}

	// Calculate gold reward with tier bonuses
	// Formula: base = min(score/2 + 20, 1500) + sum of tier bonuses
	gold := calculateArcade3DGold(req.Score)

	// Award gold to user
	var bal int64
	if err := s.bot.DB.QueryRow(
		"UPDATE users SET gold = gold + $1 WHERE client_uid=$2 RETURNING gold", gold, uid,
	).Scan(&bal); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "no character"})
		return
	}

	// Calculate gear drop chance based on score tier
	// Higher scores unlock better tiers with higher base chances and caps
	chance := calculateArcade3DGearChance(req.Score)

	// Roll for gear drop
	// #nosec G404 -- non-cryptographic drop roll
	gear := ""
	if rand.IntN(100) < chance {
		g := content.RandomGearDrop()
		result := s.bot.awardGearDrop(uid, g)
		gear = result.Prefix + result.ItemName
	}

	// Record result in arcade3d_scores table for per-game leaderboards
	if err := s.recordArcade3DResult(uid, game, req.Score, gold, gear); err != nil {
		// Log error but don't fail the request - gold was already awarded
		// The gear_won field will be empty in the record
		_ = s.recordArcade3DResult(uid, game, req.Score, gold, "")
	}

	// Mark this reward as claimed to prevent double-spending
	markRewardClaimed(uid, game, req.Score)

	writeJSON(w, map[string]any{
		"ok":       true,
		"gold_won": gold,
		"gear":     gear,
		"gold":     bal,
		"chance":   chance, // Include chance for transparency
	})
}
