package bot

import (
	"errors"
	"io"
	"net/http"
)

const (
	abyssRunFlagTokenAnte   = "token_ante"
	abyssRunFlagRiskDialPct = "risk_dial_pct"
)

func abyssTokenAnteValid(tokens int) bool {
	switch tokens {
	case 0, 5, 10, 20:
		return true
	default:
		return false
	}
}

func abyssRiskDialValid(percent int) bool {
	return percent >= -20 && percent <= 50 && percent%10 == 0
}

func applyAbyssStakeReward(reward int64, tokenAnte, riskDialPct int) int64 {
	if reward <= 0 {
		return 0
	}
	if !abyssTokenAnteValid(tokenAnte) {
		tokenAnte = 0
	}
	if !abyssRiskDialValid(riskDialPct) {
		riskDialPct = 0
	}
	return reward * int64(100+tokenAnte) / 100 * int64(100+riskDialPct) / 100
}

func abyssRiskDialMultiplier(percent int) float64 {
	if !abyssRiskDialValid(percent) {
		return 1
	}
	return float64(100+percent) / 100
}

func abyssRiskWithDial(base, percent int) int {
	if !abyssRiskDialValid(percent) {
		percent = 0
	}
	return min(100, max(0, base*(100+percent)/100))
}

func (b *Bot) abyssRunRiskPct(uid string, depth int, tier abyssTier) int {
	flags := b.loadRunFlags(uid)
	return abyssRiskWithDial(abyssRiskPct(depth, tier, b.abyssPlayerCR(uid)), int(flags[abyssRunFlagRiskDialPct]))
}

func (b *Bot) abyssRunSurvivalChance(uid string, depth int, tier abyssTier) int {
	return 100 - b.abyssRunRiskPct(uid, depth, tier)
}

// handleAbyssRiskDial changes the next floor's symmetric danger/reward modifier.
// The per-player lock keeps the validation and flag write ordered with descends.
func (s *WebServer) handleAbyssRiskDial(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

	var req struct {
		Percent int `json:"percent"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if !abyssRiskDialValid(req.Percent) {
		writeJSON(w, map[string]any{"ok": false, "error": "risk dial must be between -20% and +50% in 10% steps"})
		return
	}
	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "risk can be changed only during a live run"})
		return
	}
	flags := s.bot.loadRunFlags(uid)
	flags[abyssRunFlagRiskDialPct] = int64(req.Percent)
	if err := s.bot.saveRunFlags(uid, flags); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	tier, _ := abyssTierByKey(run.Tier)
	writeJSON(w, map[string]any{
		"ok": true, "percent": req.Percent,
		"risk": s.bot.abyssRunRiskPct(uid, run.Depth+1, tier),
		"msg":  "Next-floor danger and cache reward adjusted together.",
	})
}
