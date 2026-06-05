package games

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type Game struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Worth     string `json:"worth"`      // e.g. "$29.99", or "N/A" for always-free games
	URL       string `json:"open_giveaway_url"`
	Platforms string `json:"platforms"`  // e.g. "PC, Steam" / "PC, Epic Games Store"
	EndDate   string `json:"end_date"`   // e.g. "2026-06-11 23:59:00", or "N/A"
	Type      string `json:"type"`       // e.g. "Game"
}

// FetchGiveaways returns all current game giveaways from GamerPower.
func FetchGiveaways() ([]Game, error) {
	resp, err := http.Get("https://www.gamerpower.com/api/giveaways?type=game&sort-by=date")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var games []Game
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, err
	}
	return games, nil
}

// IsLimitedTimePaidGiveaway reports whether this is a normally-paid Steam or Epic
// game that is free only for a limited time — i.e. a real "was €X, now free"
// deal, not an always-free / free-to-play title.
func (g Game) IsLimitedTimePaidGiveaway() bool {
	return g.isSteamOrEpic() && g.isNormallyPaid() && g.isLimitedTime()
}

// Store returns the friendly store name ("Steam" or "Epic Games"), or "" if neither.
func (g Game) Store() string {
	p := strings.ToLower(g.Platforms)
	switch {
	case strings.Contains(p, "steam"):
		return "Steam"
	case strings.Contains(p, "epic games"):
		return "Epic Games"
	default:
		return ""
	}
}

func (g Game) isSteamOrEpic() bool { return g.Store() != "" }

func (g Game) isNormallyPaid() bool {
	w := strings.TrimSpace(g.Worth)
	if w == "" || strings.EqualFold(w, "N/A") {
		return false
	}
	// Treat "$0.00" / "0" as not paid.
	digits := strings.NewReplacer("$", "", "€", "", ".", "", ",", "", " ", "").Replace(w)
	return strings.Trim(digits, "0") != ""
}

func (g Game) isLimitedTime() bool {
	e := strings.TrimSpace(g.EndDate)
	if e == "" || strings.EqualFold(e, "N/A") {
		return false
	}
	// If the end date is parseable and already in the past, it is no longer offered.
	if t, err := time.Parse("2006-01-02 15:04:05", e); err == nil {
		return t.After(time.Now())
	}
	return true
}
