//go:build e2e

package bot

import (
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
)

// Test-only wallets use the real game engines without production gold, loot or VIP effects.
func registerArcadePlaytest(mux *http.ServeMux, server *WebServer) {
	type wallet struct {
		gold  int64
		daily bool
		rng   *rand.Rand
	}
	var mu sync.Mutex
	wallets := map[string]*wallet{}
	nextID := 0
	getWallet := func(w http.ResponseWriter, r *http.Request) *wallet {
		if cookie, err := r.Cookie("arcade_playtest"); err == nil {
			if current := wallets[cookie.Value]; current != nil {
				return current
			}
		}
		nextID++
		id := strconv.Itoa(nextID)
		current := &wallet{gold: 25000, daily: true, rng: rand.New(rand.NewPCG(7, 42))}
		wallets[id] = current
		http.SetCookie(w, &http.Cookie{Name: "arcade_playtest", Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
		return current
	}
	mux.HandleFunc("/arcade", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		current := getWallet(w, r)
		if err := server.tmpl.ExecuteTemplate(w, "arcade", map[string]any{
			"Title": "Arcade", "Nav": "arcade", "EnableAbyss": true,
			"U":   &webUser{UID: "arcade-e2e", Nickname: "Arcade Tester", Gold: current.gold},
			"VIP": map[string]any{"Name": "Bronze"}, "VIPPoints": 120,
			"WheelJSON": jsonJS(wheelSegments), "JackpotSlots": int64(50000), "CanDaily": current.daily,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/api/arcade/play", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		current := getWallet(w, r)
		var request struct {
			Game   string
			Bet    int64
			Choice string
		}
		if readJSON(r, &request) != nil || request.Bet < 1 || request.Bet > maxArcadeBet || !validArcadeChoice(request.Game, request.Choice) {
			writeJSON(w, arcadeOutcome{Error: "invalid game, wager or choice"})
			return
		}
		if request.Bet > current.gold {
			writeJSON(w, arcadeOutcome{Error: "not enough gold"})
			return
		}
		outcome := playArcade(current.rng, request.Game, request.Bet, request.Choice)
		outcome.Net = outcome.Payout - outcome.Bet
		outcome.Win = outcome.Net > 0
		current.gold += outcome.Net
		outcome.Gold = current.gold
		writeJSON(w, outcome)
	})
	mux.HandleFunc("/api/arcade/daily-spin", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		current := getWallet(w, r)
		if !current.daily {
			writeJSON(w, map[string]any{"ok": false, "error": "Daily tribute already claimed"})
			return
		}
		current.daily = false
		current.gold += 500
		writeJSON(w, map[string]any{"ok": true, "reward": "Looted 500 gold!", "new_gold": current.gold})
	})
}
