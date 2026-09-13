package bot

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"

	"ts3news/internal/content"
)

const maxArcadeBet = 100_000_000

// wheelSegments are the multipliers of the 12-segment fortune wheel (×bet).
// Shared with the client (the canvas draws WHEEL.length slices) so the rendered
// wheel matches the server outcome. Tuned to ~0.96 RTP — a single high segment
// on a small wheel would otherwise make the game player-favoured.
var wheelSegments = []float64{0, 0, 0, 0, 1.5, 0, 0, 2, 0, 0, 3, 5}

// arcadeOutcome is the result of one arcade round. The typed animation fields let
// the front-end play a graphic that lands on the server-decided result.
type arcadeOutcome struct {
	OK         bool     `json:"ok"`
	Error      string   `json:"error,omitempty"`
	Game       string   `json:"game"`
	Bet        int64    `json:"bet"`
	Payout     int64    `json:"payout"` // gross returned (0 = lost the bet)
	Net        int64    `json:"net"`    // all gold credited minus bet
	BasePayout int64    `json:"base_payout"`
	Rebate     int64    `json:"rebate"`
	Win        bool     `json:"win"`
	Detail     string   `json:"detail"`
	Gold       int64    `json:"gold"`
	Symbols    []string `json:"symbols,omitempty"`  // slots
	Roll       int      `json:"roll,omitempty"`     // dice
	Side       string   `json:"side,omitempty"`     // coinflip
	Card       int      `json:"card,omitempty"`     // highlow
	Segment    int      `json:"segment"`            // wheel (index into wheelSegments)
	Mult       float64  `json:"mult,omitempty"`     // wheel/payout multiplier
	Chest      int      `json:"chest,omitempty"`    // vault treasure position (1–3)
	Chance     int      `json:"chance,omitempty"`   // expedition success threshold (1–100)
	GearWon    string   `json:"gear_won,omitempty"` // gear looted on a win

	JackpotWin    bool  `json:"jackpot_win,omitempty"`
	JackpotAmount int64 `json:"jackpot_amount,omitempty"`
	NewJackpot    int64 `json:"new_jackpot,omitempty"`
}

func (s *WebServer) handleArcadePage(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}
	vip, pts := s.bot.getVIP(uid)
	s.render(w, "arcade", map[string]any{
		"Title": "Arcade", "Nav": "arcade", "U": u,
		"WheelJSON":    jsonJS(wheelSegments),
		"VIP":          vip,
		"VIPPoints":    pts,
		"RoundAccount": fmt.Sprintf("%x", sha256.Sum256([]byte(uid))),
		"JackpotSlots": s.bot.getJackpot("global"),
		"CanDaily":     s.bot.canSpinDaily(uid),
	})
}

func (s *WebServer) handleArcadeAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Game      string `json:"game"`
		Bet       int64  `json:"bet"`
		Choice    string `json:"choice"`
		RequestID string `json:"request_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, arcadeOutcome{OK: false, Error: "bad request"})
		return
	}
	if req.Bet < 100 || req.Bet > maxArcadeBet || req.Bet%100 != 0 {
		writeJSON(w, arcadeOutcome{OK: false, Error: "invalid bet"})
		return
	}
	// Reject invalid games and choices before debiting gold or awarding VIP points.
	if !validArcadeChoice(req.Game, req.Choice) {
		writeJSON(w, arcadeOutcome{OK: false, Error: "invalid game or choice"})
		return
	}

	out, err := s.bot.settleArcade(r.Context(), uid, req.Game, req.Choice, req.Bet, req.RequestID)
	if err != nil {
		writeJSON(w, arcadeOutcome{Error: err.Error()})
		return
	}
	writeJSON(w, out)
}

func (s *WebServer) handleDailySpinAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONStatus(w, http.StatusMethodNotAllowed, map[string]any{"ok": false, "error": "POST only"})
		return
	}
	if !s.bot.attemptDailySpin(uid) {
		writeJSON(w, map[string]any{"ok": false, "error": "already spun today"})
		return
	}
	// #nosec G404
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

	var reward string
	var gold int64
	var gear string

	roll := rng.IntN(100)
	switch {
	case roll < 70:
		gold = int64(100 + rng.IntN(400))
		reward = fmt.Sprintf("Looted %d gold!", gold)
		_, _ = s.bot.DB.Exec("/* economy:bot.WebServer.handleDailySpinAPI */ UPDATE users SET gold = gold + $1 WHERE client_uid=$2", gold, uid)
	case roll < 95:
		g := content.RandomArcadeGearDrop()
		result := s.bot.awardGearDrop(uid, g)
		gear = result.ItemName
		reward = result.Prefix + result.ItemName
	default:
		gold = 2500
		reward = "JACKPOT! Looted 2500 gold!"
		_, _ = s.bot.DB.Exec("/* economy:bot.WebServer.handleDailySpinAPI */ UPDATE users SET gold = gold + $1 WHERE client_uid=$2", gold, uid)
	}

	var newGold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&newGold)

	writeJSON(w, map[string]any{
		"ok": true, "reward": reward, "gold": gold, "gear": gear,
		"new_gold": newGold,
	})
}

// playArcade dispatches to the individual games. Each carries a small house edge.
func playArcade(rng *rand.Rand, game string, bet int64, choice string) arcadeOutcome {
	if !validArcadeChoice(game, choice) {
		return arcadeOutcome{OK: false, Error: "invalid game or choice"}
	}
	out := arcadeOutcome{OK: true, Game: game, Bet: bet}
	switch game {
	case "slots":
		out.Symbols, out.Payout, out.Detail = playSlots(rng, bet)
	case "dice":
		out.Roll, out.Payout, out.Detail = playDice(rng, bet)
	case "coinflip":
		out.Side, out.Payout, out.Detail = playCoinflip(rng, bet, choice)
	case "wheel":
		out.Segment, out.Mult, out.Payout, out.Detail = playWheel(rng, bet)
	case "highlow":
		out.Card, out.Payout, out.Detail = playHighLow(rng, bet, choice)
	case "vault":
		out.Chest = rng.IntN(3) + 1
		if itoa(out.Chest) == choice {
			out.Payout = bet * 285 / 100
			out.Detail = "Treasure found in chest " + choice + " — win ×2.85"
		} else {
			out.Detail = "Empty chest — treasure was in chest " + itoa(out.Chest)
		}
	case "expedition":
		chance, multiplier := expeditionRisk(choice)
		out.Chance, out.Mult = chance, multiplier
		out.Roll = rng.IntN(100) + 1
		if out.Roll <= chance {
			out.Payout = mulBet(bet, multiplier)
			out.Detail = "Expedition survived — win ×" + ftoa(multiplier)
		} else {
			out.Detail = "Expedition lost — the depths claimed your wager"
		}
	default:
		return arcadeOutcome{OK: false, Error: "unknown game"}
	}
	return out
}

func validArcadeChoice(game, choice string) bool {
	switch game {
	case "slots", "dice", "wheel":
		return choice == ""
	case "coinflip":
		return choice == "heads" || choice == "tails"
	case "highlow":
		return choice == "high" || choice == "low"
	case "vault":
		return choice == "1" || choice == "2" || choice == "3"
	case "expedition":
		return choice == "scout" || choice == "delve" || choice == "abyss"
	default:
		return false
	}
}

// Each route returns 96% of wagers on average before integer rounding and bonuses.
func expeditionRisk(choice string) (int, float64) {
	switch choice {
	case "scout":
		return 80, 1.2
	case "delve":
		return 48, 2
	case "abyss":
		return 24, 4
	default:
		return 0, 0
	}
}

var slotSymbols = []string{"🍒", "🍋", "🔔", "⭐", "💎", "7️⃣"}

// playSlots spins 5 reels. 3/4/5 of a kind pay 3x/16x/88x.
func playSlots(rng *rand.Rand, bet int64) ([]string, int64, string) {
	reels := make([]string, 5)
	counts := map[string]int{}
	for i := range reels {
		sym := slotSymbols[rng.IntN(len(slotSymbols))]
		reels[i] = sym
		counts[sym]++
	}
	best := 0
	for _, c := range counts {
		if c > best {
			best = c
		}
	}
	switch {
	case best >= 5:
		return reels, bet * 88, "JACKPOT! 5 of a kind ×88"
	case best == 4:
		return reels, bet * 16, "4 of a kind ×16"
	case best == 3:
		return reels, bet * 3, "3 of a kind ×3"
	default:
		return reels, 0, "No match"
	}
}

// playDice rolls 1-6: 5–6 pays 2.4x, 4 pushes, 1–3 loses.
func playDice(rng *rand.Rand, bet int64) (int, int64, string) {
	roll := rng.IntN(6) + 1
	switch {
	case roll >= 5:
		return roll, mulBet(bet, 2.4), "Rolled " + itoa(roll) + " — win ×2.4"
	case roll == 4:
		return roll, bet, "Rolled 4 — push"
	default:
		return roll, 0, "Rolled " + itoa(roll) + " — loss"
	}
}

// playCoinflip is a near-even flip (1.93x) on the chosen side.
func playCoinflip(rng *rand.Rand, bet int64, choice string) (string, int64, string) {
	if choice != "heads" && choice != "tails" {
		choice = "heads"
	}
	flip := "heads"
	if rng.IntN(2) == 0 {
		flip = "tails"
	}
	if flip == choice {
		return flip, bet * 193 / 100, flip + " — you win ×1.93"
	}
	return flip, 0, flip + " — you lose"
}

// playWheel spins the 12-segment wheel (house edge ~4%).
func playWheel(rng *rand.Rand, bet int64) (int, float64, int64, string) {
	seg := rng.IntN(len(wheelSegments))
	mult := wheelSegments[seg]
	pay := int64(float64(bet) * mult)
	if pay == 0 {
		return seg, mult, 0, "Landed on a blank"
	}
	return seg, mult, pay, "Won ×" + ftoa(mult)
}

// playHighLow draws a card 1-13; player bets high (>7) or low (<7); 7 loses.
func playHighLow(rng *rand.Rand, bet int64, choice string) (int, int64, string) {
	if choice != "high" && choice != "low" {
		choice = "high"
	}
	card := rng.IntN(13) + 1
	win := (choice == "high" && card > 7) || (choice == "low" && card < 7)
	if win {
		return card, bet * 208 / 100, "Drew " + itoa(card) + " — win ×2.08"
	}
	return card, 0, "Drew " + itoa(card) + " — loss"
}

func mulBet(bet int64, m float64) int64 { return int64(float64(bet) * m) }
