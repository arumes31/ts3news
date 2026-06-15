package bot

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/leveling"
)

// Exchange rates: spend 10 gold to gain 1 XP; spend 10 XP to gain 5 gold
// (i.e. 2 XP per gold).
const (
	goldPerXP     = 10 // gold spent to gain 1 XP
	xpPerGold     = 2  // XP spent to gain 1 gold (10 XP → 5 gold)
	shopStockSize = 12
)

// gearPrice is the fair buy price of a gear piece, scaled by combat power and
// rarity. Vendor/sell value is half this.
func gearPrice(g content.Gear) int64 {
	p := int64(g.CombatRating()*12+float64(g.Stats.Score())*6) * (int64(g.Rarity) + 1)
	if p < 25 {
		p = 25
	}
	return p
}

func itoa(n int) string     { return strconv.Itoa(n) }
func ftoa(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }

// Shop stock rotates on windows whose length is a deterministic, pseudo-random
// value between shopMinHours and shopMaxHours. All players share one rotation.
const (
	shopMinHours   = 1
	shopMaxHours   = 6
	shopAnchorUnix = 1735689600 // 2025-01-01T00:00:00Z — fixed rotation origin
)

// shopWindowDuration returns the length (in seconds) of shop window idx, in
// [shopMinHours, shopMaxHours] hours, derived deterministically from idx.
func shopWindowDuration(idx int64) int64 {
	// #nosec G115 -- idx is always non-negative (window index)
	h := uint64(idx)*0x9E3779B97F4A7C15 + 0x123456789
	h ^= h >> 29
	span := uint64(shopMaxHours - shopMinHours + 1)
	hours := int64(h%span) + shopMinHours
	return hours * 3600
}

// shopWindow returns the current rotation window's stock seed and end time by
// walking fixed-origin windows until the one containing now.
func shopWindow(now time.Time) (seed int64, endsAt time.Time) {
	start := int64(shopAnchorUnix)
	nowU := now.Unix()
	if nowU < start {
		return 0, time.Unix(start+shopWindowDuration(0), 0).UTC()
	}
	var idx int64
	for {
		dur := shopWindowDuration(idx)
		if start+dur > nowU {
			return idx, time.Unix(start+dur, 0).UTC()
		}
		start += dur
		idx++
	}
}

type shopItemView struct {
	ID          string
	Name        string
	Slot        string
	Icon        string
	Rarity      string
	RarityColor string
	CR          float64
	Score       int
	Price       int64
	IsUpgrade   bool
}

func stockForSeed(seed int64, equippedGear map[string]content.Gear) []shopItemView {
	stock := content.ShopStock(seed, shopStockSize)
	out := make([]shopItemView, 0, len(stock))
	for _, g := range stock {
		isUpgrade := false
		if curr, ok := equippedGear[string(g.Slot)]; !ok {
			isUpgrade = true
		} else if g.CombatRating() > curr.CombatRating() && g.Stats.Score() > curr.Stats.Score() {
			isUpgrade = true
		}

		out = append(out, shopItemView{
			ID:          g.ID,
			Name:        g.Name,
			Slot:        string(g.Slot),
			Icon:        content.SlotIcon(g.Slot),
			Rarity:      g.Rarity.String(),
			RarityColor: g.Rarity.Color(),
			CR:          g.CombatRating(),
			Score:       g.Stats.Score(),
			Price:       gearPrice(g),
			IsUpgrade:   isUpgrade,
		})
	}
	return out
}

func (s *WebServer) handleShopPage(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}
	
	// Load player's equipped gear to determine upgrades
	equippedGear := make(map[string]content.Gear)
	rows, err := s.bot.DB.Query("SELECT slot, gear_id FROM user_gear WHERE client_uid=$1", uid)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var slot, gearID string
			if err := rows.Scan(&slot, &gearID); err == nil {
				if cg, ok := content.GetGearByID(gearID); ok {
					equippedGear[slot] = cg
				}
			}
		}
	}

	seed, endsAt := shopWindow(time.Now())
	refreshIn := int(time.Until(endsAt).Seconds())
	if refreshIn < 0 {
		refreshIn = 0
	}
	s.render(w, "shop", map[string]any{
		"Title":     "Shop",
		"Nav":       "shop",
		"U":         u,
		"Stock":     stockForSeed(seed, equippedGear),
		"RefreshIn": refreshIn,
		"GoldPerXP": goldPerXP,
		"XPPerGold": xpPerGold,
	})
	}

// handleExchangeAPI converts between gold and XP at the fixed rates.
func (s *WebServer) handleExchangeAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Direction string `json:"direction"` // "gold_to_xp" | "xp_to_gold"
		Amount    int64  `json:"amount"`    // amount of the input resource to spend
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid amount"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "tx"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gold int64
	var xp int
	if err := tx.QueryRow("SELECT gold, xp FROM users WHERE client_uid=$1", uid).Scan(&gold, &xp); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "no character"})
		return
	}

	var detail string
	switch req.Direction {
	case "gold_to_xp":
		spend := req.Amount - (req.Amount % goldPerXP) // only whole XP, no wasted gold
		if spend <= 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "need at least " + itoa(goldPerXP) + " gold"})
			return
		}
		if gold < spend {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
			return
		}
		gainXP := int(spend / goldPerXP)
		gold -= spend
		xp += gainXP
		detail = "Spent " + FormatGold(spend) + " gold for +" + itoa(gainXP) + " XP"
	case "xp_to_gold":
		spend := req.Amount - (req.Amount % xpPerGold) // round down to a multiple
		if spend <= 0 || int64(xp) < spend {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough XP"})
			return
		}
		gainGold := spend / xpPerGold
		xp -= int(spend)
		gold += gainGold
		detail = "Spent " + itoa(int(spend)) + " XP for +" + FormatGold(gainGold) + " gold"
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "bad direction"})
		return
	}

	newLevel := leveling.LevelForXP(xp)
	if _, err := tx.Exec("UPDATE users SET gold=$1, xp=$2, level=$3 WHERE client_uid=$4", gold, xp, newLevel, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "save"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "commit"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "detail": detail, "gold": gold, "xp": xp, "level": newLevel,
		"level_name": leveling.LevelName(newLevel),
	})
}

// handleBuyAPI buys a gear item from today's rotating stock for its fair price
// and places it in the inventory.
func (s *WebServer) handleBuyAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	// Only items in the current rotation are purchasable (at the server's price).
	seed, _ := shopWindow(time.Now())
	var chosen *shopItemView
	for _, it := range stockForSeed(seed, nil) {
		if it.ID == req.ID {
			c := it
			chosen = &c
			break
		}
	}
	if chosen == nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not in stock"})
		return
	}
	g, ok := content.GetGearByID(chosen.ID)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown gear"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "tx"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", chosen.Price, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	if _, err := tx.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability) VALUES ($1, $2, $3)", uid, g.ID, g.MaxDurability); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "inventory"})
		return
	}
	// Read the post-purchase balance inside the transaction to avoid a race with
	// other concurrent operations between commit and a separate query.
	var gold int64
	if err := tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "gold"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "bought": g.Name, "gold": gold})
}
