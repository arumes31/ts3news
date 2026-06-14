package bot

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/leveling"
)

// Exchange rates (per the design): spend 1 gold to gain 3 XP; spend 2 XP to gain
// 1 gold.
const (
	goldToXPRate  = 3 // XP gained per gold spent
	xpToGoldRatio = 2 // XP spent per gold gained
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

func itoa(n int) string      { return strconv.Itoa(n) }
func ftoa(f float64) string  { return strconv.FormatFloat(f, 'g', -1, 64) }

// dayseed yields a stable seed that rotates once per day (shop refresh).
func dayseed() int64 { return time.Now().UTC().Unix() / 86400 }

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
}

func todayStock() []shopItemView {
	stock := content.ShopStock(dayseed(), shopStockSize)
	out := make([]shopItemView, 0, len(stock))
	for _, g := range stock {
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
	s.render(w, "shop", map[string]any{
		"Title":         "Shop",
		"Nav":           "shop",
		"U":             u,
		"Stock":         todayStock(),
		"GoldToXPRate":  goldToXPRate,
		"XPToGoldRatio": xpToGoldRatio,
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
		if gold < req.Amount {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
			return
		}
		gainXP := int(req.Amount) * goldToXPRate
		gold -= req.Amount
		xp += gainXP
		detail = "Spent " + FormatGold(req.Amount) + " gold for +" + itoa(gainXP) + " XP"
	case "xp_to_gold":
		spend := req.Amount - (req.Amount % xpToGoldRatio) // round down to a multiple
		if spend <= 0 || int64(xp) < spend {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough XP"})
			return
		}
		gainGold := spend / xpToGoldRatio
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

	// Only items in today's stock are purchasable (and at the server's price).
	var chosen *shopItemView
	for _, it := range todayStock() {
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
