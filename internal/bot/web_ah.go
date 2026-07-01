package bot

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ts3news/internal/content"
)

type ahListingView struct {
	ID        string
	ItemType  string
	ItemID    string
	Icon      string
	Name      string
	Price     int64
	Seller    string
	Listed    string
	Mine      bool
	IsUpgrade bool
}

// ahIcon returns a slot-matched icon for a gear listing, or a type icon otherwise.
func ahIcon(itemType, itemID string) string {
	if itemType == "gear" {
		if g, ok := content.GetGearByID(itemID); ok {
			return content.SlotIcon(g.Slot)
		}
		return "💎"
	}
	switch itemType {
	case "skill":
		return "✨"
	case "ultimate", "ultimate_skill":
		return "🌟"
	case "unique", "unique_item":
		return "💠"
	case "enchantment":
		return "🔰"
	case "artifact":
		return "🏺"
	default:
		return "📦"
	}
}

type ahHistoryView struct {
	Name  string
	Price int64
	Role  string // "Bought" or "Sold"
	Other string // counterparty nickname
	When  string
}

func (s *WebServer) handleAHPage(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}

	searchQuery := r.URL.Query().Get("q")
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	limit := 20
	offset := (page - 1) * limit

	// Load player's equipped gear to determine upgrades
	equippedGear := make(map[string]content.Gear)
	rows, err := s.bot.DB.Query("SELECT slot, gear_id FROM user_gear WHERE client_uid=$1", uid)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var slot, gearID string
			if err := rows.Scan(&slot, &gearID); err == nil {
				if cg, ok := content.GetGearByID(gearID); ok {
					equippedGear[slot] = cg
				}
			}
		}
	}

	activeListings := s.bot.ahActiveListings(uid, equippedGear, searchQuery, limit, offset)
	totalCount := s.bot.ahActiveListingsCount(searchQuery)
	totalPages := (totalCount + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	s.render(w, "ah", map[string]any{
		"Title":       "Auction House",
		"Nav":         "ah",
		"U":           u,
		"Active":      activeListings,
		"Mine":        s.bot.ahMyListings(uid),
		"History":     s.bot.ahHistory(uid, 20),
		"Sellable":    s.bot.inventoryItems(uid),
		"SearchQuery": searchQuery,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"TotalCount":  totalCount,
		"PrevPage":    page - 1,
		"NextPage":    page + 1,
	})
}

func (b *Bot) ahActiveListings(uid string, equippedGear map[string]content.Gear, search string, limit, offset int) []ahListingView {
	var rows *sql.Rows
	var err error
	if search != "" {
		rows, err = b.DB.Query(`
			SELECT a.id, a.item_type, a.item_id, a.item_name, a.price, a.listed_at, COALESCE(u.nickname,'?'), a.seller_uid
			FROM auction_house a LEFT JOIN users u ON u.client_uid = a.seller_uid
			WHERE a.sold_at IS NULL AND a.expires_at > NOW() AND a.item_name ILIKE $1
			ORDER BY a.price ASC LIMIT $2 OFFSET $3`, "%"+search+"%", limit, offset)
	} else {
		rows, err = b.DB.Query(`
			SELECT a.id, a.item_type, a.item_id, a.item_name, a.price, a.listed_at, COALESCE(u.nickname,'?'), a.seller_uid
			FROM auction_house a LEFT JOIN users u ON u.client_uid = a.seller_uid
			WHERE a.sold_at IS NULL AND a.expires_at > NOW()
			ORDER BY a.price ASC LIMIT $1 OFFSET $2`, limit, offset)
	}
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []ahListingView
	for rows.Next() {
		var v ahListingView
		var t time.Time
		var seller string
		if err := rows.Scan(&v.ID, &v.ItemType, &v.ItemID, &v.Name, &v.Price, &t, &v.Seller, &seller); err != nil {
			continue
		}
		v.Icon = ahIcon(v.ItemType, v.ItemID)
		v.Listed = t.Format("Jan 02")
		v.Mine = seller == uid

		// Determine if upgrade
		if v.ItemType == "gear" {
			if cg, ok := content.GetGearByID(v.ItemID); ok {
				if curr, ok := equippedGear[string(cg.Slot)]; !ok {
					v.IsUpgrade = true
				} else if cg.CombatRating() > curr.CombatRating() && cg.Stats.Score() > curr.Stats.Score() {
					v.IsUpgrade = true
				}
			}
		}

		out = append(out, v)
	}
	return out
}

func (b *Bot) ahActiveListingsCount(search string) int {
	var count int
	var err error
	if search != "" {
		err = b.DB.QueryRow(`
			SELECT COUNT(*)
			FROM auction_house
			WHERE sold_at IS NULL AND expires_at > NOW() AND item_name ILIKE $1`, "%"+search+"%").Scan(&count)
	} else {
		err = b.DB.QueryRow(`
			SELECT COUNT(*)
			FROM auction_house
			WHERE sold_at IS NULL AND expires_at > NOW()`).Scan(&count)
	}
	if err != nil {
		return 0
	}
	return count
}

func (b *Bot) ahMyListings(uid string) []ahListingView {
	rows, err := b.DB.Query(`
		SELECT id, item_type, item_id, item_name, price, listed_at
		FROM auction_house
		WHERE seller_uid=$1 AND sold_at IS NULL AND expires_at > NOW()
		ORDER BY listed_at DESC`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []ahListingView
	for rows.Next() {
		var v ahListingView
		var t time.Time
		if err := rows.Scan(&v.ID, &v.ItemType, &v.ItemID, &v.Name, &v.Price, &t); err != nil {
			continue
		}
		v.Icon = ahIcon(v.ItemType, v.ItemID)
		v.Listed = t.Format("Jan 02")
		v.Mine = true
		out = append(out, v)
	}
	return out
}

func (b *Bot) ahHistory(uid string, limit int) []ahHistoryView {
	rows, err := b.DB.Query(`
		SELECT a.item_name, a.price, a.sold_at, a.seller_uid, a.buyer_uid,
		       COALESCE(sb.nickname,'?'), COALESCE(bu.nickname,'vendor')
		FROM auction_house a
		LEFT JOIN users sb ON sb.client_uid = a.seller_uid
		LEFT JOIN users bu ON bu.client_uid = a.buyer_uid
		WHERE a.sold_at IS NOT NULL AND (a.seller_uid=$1 OR a.buyer_uid=$1)
		ORDER BY a.sold_at DESC LIMIT $2`, uid, limit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []ahHistoryView
	for rows.Next() {
		var v ahHistoryView
		var t time.Time
		var seller, buyer *string
		var sellerNick, buyerNick string
		if err := rows.Scan(&v.Name, &v.Price, &t, &seller, &buyer, &sellerNick, &buyerNick); err != nil {
			continue
		}
		v.When = t.Format("Jan 02 15:04")
		if seller != nil && *seller == uid {
			v.Role = "Sold"
			v.Other = buyerNick
		} else {
			v.Role = "Bought"
			v.Other = sellerNick
		}
		out = append(out, v)
	}
	return out
}

// handleAHBuyAPI buys an active listing; the item lands in the buyer's inventory.
func (s *WebServer) handleAHBuyAPI(w http.ResponseWriter, r *http.Request, uid string) {
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

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "tx"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var itemType, itemID, name, sellerUID string
	var dataJSON []byte
	var price int64
	var durability sql.NullInt64
	err = tx.QueryRow(`
		SELECT item_type, item_id, item_name, item_data, price, seller_uid, durability
		FROM auction_house
		WHERE id=$1 AND sold_at IS NULL AND expires_at > NOW() FOR UPDATE`, req.ID).
		Scan(&itemType, &itemID, &name, &dataJSON, &price, &sellerUID, &durability)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "listing unavailable"})
		return
	}
	if sellerUID == uid {
		writeJSON(w, map[string]any{"ok": false, "error": "cannot buy your own listing"})
		return
	}

	// Deduct buyer gold.
	res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", price, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	// Mark sold, pay seller.
	if _, err := tx.Exec("UPDATE auction_house SET buyer_uid=$1, sold_at=NOW() WHERE id=$2", uid, req.ID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "sold"})
		return
	}
	if _, err := tx.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", price, sellerUID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "pay"})
		return
	}
	// Deliver gear into the buyer's inventory, preserving the listing's durability.
	if itemType == "gear" {
		var g content.Gear
		if err := json.Unmarshal(dataJSON, &g); err == nil {
			dur := g.MaxDurability
			if durability.Valid {
				dur = int(durability.Int64)
			}
			if _, err := tx.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability) VALUES ($1, $2, $3)", uid, g.ID, dur); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "deliver"})
				return
			}
		}
	}
	// Read the post-purchase balance inside the transaction to avoid a race between
	// commit and a separate query.
	var gold int64
	if err := tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "gold"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "bought": name, "gold": gold})
}

// handleAHListAPI lists an inventory gear piece on the auction house.
func (s *WebServer) handleAHListAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InvID int64 `json:"inv_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.InvID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid request"})
		return
	}

	var gid string
	var dur int
	if err := s.bot.DB.QueryRow("SELECT gear_id, durability FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gid, &dur); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	g, ok := content.GetGearByID(gid)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown gear"})
		return
	}

	// Auto-calculate price: (CR×10 + GS×5) × (Rarity+1)
	price := int64(g.CombatRating()*10+float64(g.Stats.Score())*5) * (int64(g.Rarity) + 1)
	if price < 10 {
		price = 10
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "tx"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "remove"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "already gone"})
		return
	}
	dataJSON, _ := json.Marshal(g)
	if _, err := tx.Exec(`
		INSERT INTO auction_house (seller_uid, item_type, item_id, item_name, item_data, price, durability, expires_at)
		VALUES ($1, 'gear', $2, $3, $4, $5, $6, NOW() + INTERVAL '7 days')`,
		uid, g.ID, g.Name, dataJSON, price, dur); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "list"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "commit"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "listed": g.Name, "price": price})
}
