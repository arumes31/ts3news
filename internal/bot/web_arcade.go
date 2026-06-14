package bot

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
)

const maxArcadeBet = 100000

// wheelSegments are the multipliers of the 8-segment fortune wheel (×bet).
// Shared with the client so the rendered wheel matches the server outcome.
var wheelSegments = []float64{0, 0, 1.5, 0, 2, 0, 3, 10}

// arcadeOutcome is the result of one arcade round. The typed animation fields let
// the front-end play a graphic that lands on the server-decided result.
type arcadeOutcome struct {
	OK      bool     `json:"ok"`
	Error   string   `json:"error,omitempty"`
	Game    string   `json:"game"`
	Bet     int64    `json:"bet"`
	Payout  int64    `json:"payout"` // gross returned (0 = lost the bet)
	Net     int64    `json:"net"`    // payout - bet
	Win     bool     `json:"win"`
	Detail  string   `json:"detail"`
	Gold    int64    `json:"gold"`
	Symbols []string `json:"symbols,omitempty"` // slots
	Roll    int      `json:"roll,omitempty"`    // dice
	Side    string   `json:"side,omitempty"`    // coinflip
	Card    int      `json:"card,omitempty"`    // highlow
	Segment int      `json:"segment"`           // wheel (index into wheelSegments)
	Mult    float64  `json:"mult,omitempty"`    // wheel/payout multiplier
}

func (s *WebServer) handleArcadePage(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}
	segJSON, _ := json.Marshal(wheelSegments)
	s.render(w, "arcade", map[string]any{
		"Title": "Arcade", "Nav": "arcade", "U": u,
		"WheelJSON": string(segJSON),
	})
}

func (s *WebServer) handleArcadeAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Game   string `json:"game"`
		Bet    int64  `json:"bet"`
		Choice string `json:"choice"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, arcadeOutcome{OK: false, Error: "bad request"})
		return
	}
	if req.Bet <= 0 || req.Bet > maxArcadeBet {
		writeJSON(w, arcadeOutcome{OK: false, Error: "invalid bet"})
		return
	}

	// Atomically take the bet (fails if the user can't afford it).
	res, err := s.bot.DB.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", req.Bet, uid)
	if err != nil {
		writeJSON(w, arcadeOutcome{OK: false, Error: "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, arcadeOutcome{OK: false, Error: "not enough gold"})
		return
	}

	// #nosec G404 -- non-cryptographic arcade RNG
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	out := playArcade(rng, req.Game, req.Bet, req.Choice)
	if !out.OK {
		_, _ = s.bot.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", req.Bet, uid) // refund
		writeJSON(w, out)
		return
	}

	// Credit the payout and read the resulting balance atomically (via RETURNING)
	// so no concurrent operation can change gold between the update and the read.
	var gold int64
	if out.Payout > 0 {
		_ = s.bot.DB.QueryRow("UPDATE users SET gold = gold + $1 WHERE client_uid=$2 RETURNING gold", out.Payout, uid).Scan(&gold)
	} else {
		_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	}
	out.Gold = gold
	out.Net = out.Payout - out.Bet
	// A push (e.g. dice rolling 4) returns the bet so Payout > 0 but Net == 0; only
	// a positive Net is an actual win.
	out.Win = out.Net > 0
	writeJSON(w, out)
}

// playArcade dispatches to the individual games. Each carries a small house edge.
func playArcade(rng *rand.Rand, game string, bet int64, choice string) arcadeOutcome {
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
	default:
		return arcadeOutcome{OK: false, Error: "unknown game"}
	}
	return out
}

var slotSymbols = []string{"🍒", "🍋", "🔔", "⭐", "💎", "7️⃣"}

// playSlots spins 5 reels. 3/4/5 of a kind pay 2x/5x/25x.
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
		return reels, bet * 25, "JACKPOT! 5 of a kind ×25"
	case best == 4:
		return reels, bet * 5, "4 of a kind ×5"
	case best == 3:
		return reels, bet * 2, "3 of a kind ×2"
	default:
		return reels, 0, "No match"
	}
}

// playDice rolls 1-6: 5–6 pays 2.5x, 4 pushes, 1–3 loses.
func playDice(rng *rand.Rand, bet int64) (int, int64, string) {
	roll := rng.IntN(6) + 1
	switch {
	case roll >= 5:
		return roll, bet * 5 / 2, "Rolled " + itoa(roll) + " — win ×2.5"
	case roll == 4:
		return roll, bet, "Rolled 4 — push"
	default:
		return roll, 0, "Rolled " + itoa(roll) + " — loss"
	}
}

// playCoinflip is a near-even flip (1.95x) on the chosen side.
func playCoinflip(rng *rand.Rand, bet int64, choice string) (string, int64, string) {
	if choice != "heads" && choice != "tails" {
		choice = "heads"
	}
	flip := "heads"
	if rng.IntN(2) == 0 {
		flip = "tails"
	}
	if flip == choice {
		return flip, bet * 195 / 100, flip + " — you win ×1.95"
	}
	return flip, 0, flip + " — you lose"
}

// playWheel spins the 8-segment wheel (house edge ~5%).
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
		return card, bet * 19 / 10, "Drew " + itoa(card) + " — win ×1.9"
	}
	return card, 0, "Drew " + itoa(card) + " — loss"
}
