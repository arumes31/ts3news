package bot

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"strings"

	"ts3news/internal/content"
)

// arcade3dGoldCap bounds the gold a single skill-game run can award, since the
// score is reported by the (untrusted) client.
const arcade3dGoldCap = 1500

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

// handleArcade3DReward converts a reported game score into gold (capped) and a
// score-scaled gear drop, then records the play for the leaderboards.
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
	if req.Score < 0 {
		req.Score = 0
	}

	// Score → gold, with a per-run cap to bound the client-reported value.
	gold := req.Score/2 + 20
	if gold > arcade3dGoldCap {
		gold = arcade3dGoldCap
	}
	var bal int64
	if err := s.bot.DB.QueryRow(
		"UPDATE users SET gold = gold + $1 WHERE client_uid=$2 RETURNING gold", gold, uid,
	).Scan(&bal); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "no character"})
		return
	}

	// Gear drop chance rises with score, capped at 60%.
	chance := int(req.Score/40) + 10
	if chance > 60 {
		chance = 60
	}
	gear := ""
	// #nosec G404 -- non-cryptographic drop roll
	if rand.IntN(100) < chance {
		g := content.RandomGearDrop()
		if _, err := s.bot.DB.Exec(
			"INSERT INTO user_inventory (client_uid, gear_id, durability) VALUES ($1,$2,$3)", uid, g.ID, g.MaxDurability,
		); err == nil {
			gear = g.Rarity.String() + " " + g.Name
		}
	}

	s.bot.recordGameResult(uid, "arcade3d", req.Score > 0, gold)
	writeJSON(w, map[string]any{"ok": true, "gold_won": gold, "gear": gear, "gold": bal})
}
