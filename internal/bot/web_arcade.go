package bot

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"

	"ts3news/internal/content"
)

const maxArcadeBet = 100000

// wheelSegments are the multipliers of the 12-segment fortune wheel (×bet).
// Shared with the client (the canvas draws WHEEL.length slices) so the rendered
// wheel matches the server outcome. Tuned to ~0.96 RTP — a single high segment
// on a small wheel would otherwise make the game player-favoured.
var wheelSegments = []float64{0, 0, 0, 0, 1.5, 0, 0, 2, 0, 0, 3, 5}

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
	Symbols []string `json:"symbols,omitempty"`  // slots
	Roll    int      `json:"roll,omitempty"`     // dice
	Side    string   `json:"side,omitempty"`     // coinflip
	Card    int      `json:"card,omitempty"`     // highlow
	Segment int      `json:"segment"`            // wheel (index into wheelSegments)
	Mult    float64  `json:"mult,omitempty"`     // wheel/payout multiplier
	GearWon string   `json:"gear_won,omitempty"` // gear looted on a win

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
		"JackpotSlots": s.bot.getJackpot("slots"),
		"CanDaily":     s.bot.canSpinDaily(uid),
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

	// Award VIP points: 1 per 10 gold wagered
	s.bot.addVIPPoints(uid, int(req.Bet/10))

	// #nosec G404 -- non-cryptographic arcade RNG
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	out := playArcade(rng, req.Game, req.Bet, req.Choice)
	if !out.OK {
		_, _ = s.bot.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", req.Bet, uid) // refund
		writeJSON(w, out)
		return
	}

	// Progressive Jackpot logic (Triggerable in every game)
	isJackpot := false
	switch req.Game {
	case "slots":
		isJackpot = out.Detail == "JACKPOT! 5 of a kind ×88"
	case "dice":
		isJackpot = out.Payout >= req.Bet*5 // High multiplier win
	case "coin":
		// Coin flip has small multipliers, maybe consecutive wins?
		// For now, let's just make it a random chance on win
		if out.Win && rng.IntN(100) < 2 { isJackpot = true }
	default:
		// Generic random chance for other games
		if out.Win && rng.IntN(100) < 1 { isJackpot = true }
	}

	if isJackpot {
		jackpot := s.bot.claimJackpot(uid, "global")
		if jackpot > 0 {
			out.JackpotWin = true
			out.JackpotAmount = jackpot
			out.Payout += jackpot
			out.Detail = "🔥 GLOBAL JACKPOT WIN! " + out.Detail
		}
	} else if out.Payout < req.Bet {
		// Increment jackpot by 1% of lost value (net loss)
		lost := req.Bet - out.Payout
		s.bot.incrementJackpot("global", lost)
	}

	out.NewJackpot = s.bot.getJackpot("global")

	// Progressive Jackpot logic (Slots only for legacy compatibility, but uses global now)
	if req.Game == "slots" {
		out.NewJackpot = s.bot.getJackpot("global")
	}

	// Credit the payout and read the resulting balance atomically (via RETURNING)
	// so no concurrent operation can change gold between the update and the read.
	var gold int64
	if out.Payout > 0 {
		_ = s.bot.DB.QueryRow("UPDATE users SET gold = gold + $1 WHERE client_uid=$2 RETURNING gold", out.Payout, uid).Scan(&gold)
	} else {
		// Apply VIP loss-back if applicable
		vip, _ := s.bot.getVIP(uid)
		if vip.Rebate > 0 {
			rebate := req.Bet * int64(vip.Rebate) / 100
			if rebate > 0 {
				_ = s.bot.DB.QueryRow("UPDATE users SET gold = gold + $1 WHERE client_uid=$2 RETURNING gold", rebate, uid).Scan(&gold)
				out.Detail += fmt.Sprintf(" (VIP loss-back: +%d gold)", rebate)
			} else {
				_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
			}
		} else {
			_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
		}
	}
	out.Net = out.Payout - out.Bet
	// A push (e.g. dice rolling 4) returns the bet so Payout > 0 but Net == 0; only
	// a positive Net is an actual win.
	out.Win = out.Net > 0

	// Winning rounds have a chance to also drop a gear piece.
	if out.Win {
		// #nosec G404 -- non-cryptographic drop roll
		if rng.IntN(100) < 15 {
			g := content.RandomArcadeGearDrop()
			result := s.bot.awardGearDrop(uid, g)
			out.GearWon = result.Prefix + result.ItemName
		}
	}

	out.Gold = gold
	s.bot.recordGameResult(uid, "arcade", out.Win, out.Net)
	writeJSON(w, out)
}

func (s *WebServer) handleDailySpinAPI(w http.ResponseWriter, _ *http.Request, uid string) {
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
		_, _ = s.bot.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", gold, uid)
	case roll < 95:
		g := content.RandomArcadeGearDrop()
		result := s.bot.awardGearDrop(uid, g)
		gear = result.ItemName
		reward = result.Prefix + result.ItemName
	default:
		gold = 2500
		reward = "JACKPOT! Looted 2500 gold!"
		_, _ = s.bot.DB.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", gold, uid)
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
		return card, bet * 2, "Drew " + itoa(card) + " — win ×2"
	}
	return card, 0, "Drew " + itoa(card) + " — loss"
}

func mulBet(bet int64, m float64) int64 { return int64(float64(bet) * m) }
