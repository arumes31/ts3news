package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ts3news/internal/content"
)

// isAuctionUpgrade reports whether the gear item itemID is an upgrade over what the
// viewer currently has equipped in that gear's slot. An empty slot always counts as
// an upgrade; a non-gear or unknown item never does. Shared by the listing and count
// paths so the upgrades-only filter and its total stay in sync.
func isAuctionUpgrade(itemID string, equippedGear map[string]content.Gear) bool {
	cg, ok := content.GetGearByID(itemID)
	if !ok {
		return false
	}
	curr, ok := equippedGear[string(cg.Slot)]
	if !ok {
		return true
	}
	return cg.CombatRating() > curr.CombatRating() && cg.Stats.Score() > curr.Stats.Score()
}

type ahListingView struct {
	ID           string
	ItemType     string
	ItemID       string
	Icon         string
	Name         string
	Price        int64
	Seller       string
	Listed       string
	Mine         bool
	IsUpgrade    bool
	Insanity     bool
	Watched      bool
	SellerRep    int
	CurrentBid   int64
	FeeSummary   string
	PriceHistory ahPriceHistoryView

	// Display metadata resolved from the listed instance's item_data (falling
	// back to the catalog): gear score and rarity.
	GS          int
	Rarity      string
	RarityColor string
}

// ahEnrichListing fills GS / rarity from the listing's item_data JSON — the
// exact instance listed, including Abyss modifications — falling back to the
// catalog entry for older rows without stored data. All listable types (gear,
// ultimates, uniques, enchantments) marshal Rarity, and gear/enchantments
// marshal Stats, so one probe covers them all.
func ahEnrichListing(v *ahListingView, dataJSON []byte) {
	var probe struct {
		Rarity *content.Rarity
		Stats  *content.Stats
	}
	if len(dataJSON) > 0 {
		_ = json.Unmarshal(dataJSON, &probe)
	}
	if probe.Rarity == nil || probe.Stats == nil {
		switch v.ItemType {
		case "gear":
			if g, ok := content.GetGearByID(v.ItemID); ok {
				if probe.Rarity == nil {
					probe.Rarity = &g.Rarity
				}
				if probe.Stats == nil {
					probe.Stats = &g.Stats
				}
			}
		case "ultimate", "ultimate_skill":
			if us, ok := content.GetUltimateSkillByID(v.ItemID); ok && probe.Rarity == nil {
				probe.Rarity = &us.Rarity
			}
		}
	}
	if probe.Rarity != nil {
		v.Rarity = probe.Rarity.String()
		v.RarityColor = probe.Rarity.Color()
	}
	if probe.Stats != nil {
		v.GS = probe.Stats.Score()
	}
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
	s.bot.settleAbyssAuctionBids()
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}

	searchQuery := r.URL.Query().Get("q")
	upgradesOnly := r.URL.Query().Get("upgrades") == "1"
	insanityOnly := r.URL.Query().Get("insanity") == "1"
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

	activeListings := s.bot.ahActiveListings(uid, equippedGear, searchQuery, upgradesOnly, insanityOnly, limit, offset)
	priceHistories := s.bot.ahPriceHistories(activeListings)
	totalCount := s.bot.ahActiveListingsCount(searchQuery, equippedGear, upgradesOnly, insanityOnly)
	watched := s.bot.abyssAHWatchlist(uid)
	for i := range activeListings {
		activeListings[i].PriceHistory = priceHistories[activeListings[i].ItemID]
		activeListings[i].Watched = watched[activeListings[i].ItemID]
		activeListings[i].FeeSummary = "Buyer fee 0g · seller receives the exact buy-now price"
	}
	totalPages := (totalCount + limit - 1) / limit
	if totalPages < 1 {
		totalPages = 1
	}

	s.render(w, "ah", map[string]any{
		"Title":        "Auction House",
		"Nav":          "ah",
		"U":            u,
		"Active":       activeListings,
		"Mine":         s.bot.ahMyListings(uid),
		"History":      s.bot.ahHistory(uid, 20),
		"Sellable":     s.bot.inventoryItems(uid),
		"SearchQuery":  searchQuery,
		"UpgradesOnly": upgradesOnly,
		"InsanityOnly": insanityOnly,
		"CurrentPage":  page,
		"TotalPages":   totalPages,
		"TotalCount":   totalCount,
		"PrevPage":     page - 1,
		"NextPage":     page + 1,
		"Economy":      s.bot.abyssAHEconomyPage(uid),
	})
}

func (b *Bot) ahActiveListings(uid string, equippedGear map[string]content.Gear, search string, upgradesOnly, insanityOnly bool, limit, offset int) []ahListingView {
	query := `
		SELECT a.id, a.item_type, a.item_id, a.item_name, a.item_data, a.price, a.listed_at, COALESCE(u.nickname,'?'), a.seller_uid,
		       a.current_bid, (SELECT COUNT(*) FROM auction_house sold WHERE sold.seller_uid=a.seller_uid AND sold.sold_at IS NOT NULL)
		FROM auction_house a LEFT JOIN users u ON u.client_uid = a.seller_uid
		WHERE a.sold_at IS NULL AND a.expires_at > NOW()`
	var args []any
	if search != "" {
		query += ` AND a.item_name ILIKE $1`
		args = append(args, "%"+search+"%")
	}
	if insanityOnly {
		query += ` AND a.item_id LIKE 'INSANITY_%'`
	}
	query += ` ORDER BY a.price ASC`
	// upgradesOnly filters in Go after fetching, so it must scan the full result set
	// (like ahActiveListingsCount) and paginate the filtered slice — a SQL LIMIT here
	// would make deep pages come up empty even though the count reports more results.
	if !upgradesOnly {
		// The Sprintf only interpolates placeholder *positions* ($N), never user
		// data — search/limit/offset all flow through args as bound parameters.
		query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2) // #nosec G202 -- placeholder numbers only, values are parameterized below
		args = append(args, limit, offset)
	}
	rows, err := b.DB.Query(query, args...) // #nosec G701 -- query is fully parameterized; args are never concatenated into it
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var all []ahListingView
	for rows.Next() {
		var v ahListingView
		var t time.Time
		var seller string
		var dataJSON []byte
		if err := rows.Scan(&v.ID, &v.ItemType, &v.ItemID, &v.Name, &dataJSON, &v.Price, &t, &v.Seller, &seller, &v.CurrentBid, &v.SellerRep); err != nil {
			continue
		}
		v.Icon = ahIcon(v.ItemType, v.ItemID)
		v.Listed = t.Format("Jan 02")
		v.Mine = seller == uid
		v.Insanity = content.IsInsanityGearID(v.ItemID)
		ahEnrichListing(&v, dataJSON)

		if v.ItemType == "gear" {
			v.IsUpgrade = isAuctionUpgrade(v.ItemID, equippedGear)
		}

		if upgradesOnly && !v.IsUpgrade {
			continue
		}
		all = append(all, v)
	}
	if upgradesOnly {
		// Apply manual pagination over the filtered results
		start := offset
		if start > len(all) {
			return nil
		}
		end := start + limit
		if end > len(all) {
			end = len(all)
		}
		return all[start:end]
	}
	return all
}

func (b *Bot) ahActiveListingsCount(search string, equippedGear map[string]content.Gear, upgradesOnly, insanityOnly bool) int {
	if upgradesOnly {
		// For upgrades-only count we must enumerate and filter
		var rows *sql.Rows
		var err error
		if search != "" {
			rows, err = b.DB.Query(`
				SELECT a.item_type, a.item_id
				FROM auction_house a
				WHERE a.sold_at IS NULL AND a.expires_at > NOW() AND a.item_name ILIKE $1
				  AND ($2=FALSE OR a.item_id LIKE 'INSANITY_%')`, "%"+search+"%", insanityOnly)
		} else {
			rows, err = b.DB.Query(`
				SELECT item_type, item_id
				FROM auction_house
				WHERE sold_at IS NULL AND expires_at > NOW()
				  AND ($1=FALSE OR item_id LIKE 'INSANITY_%')`, insanityOnly)
		}
		if err != nil {
			return 0
		}
		defer func() { _ = rows.Close() }()
		count := 0
		for rows.Next() {
			var itemType, itemID string
			if err := rows.Scan(&itemType, &itemID); err != nil {
				continue
			}
			if itemType == "gear" && isAuctionUpgrade(itemID, equippedGear) {
				count++
			}
		}
		return count
	}
	// Normal count
	var count int
	var err error
	if search != "" {
		err = b.DB.QueryRow(`
			SELECT COUNT(*)
			FROM auction_house
			WHERE sold_at IS NULL AND expires_at > NOW() AND item_name ILIKE $1
			  AND ($2=FALSE OR item_id LIKE 'INSANITY_%')`, "%"+search+"%", insanityOnly).Scan(&count)
	} else {
		err = b.DB.QueryRow(`
			SELECT COUNT(*)
			FROM auction_house
			WHERE sold_at IS NULL AND expires_at > NOW()
			  AND ($1=FALSE OR item_id LIKE 'INSANITY_%')`, insanityOnly).Scan(&count)
	}
	if err != nil {
		return 0
	}
	return count
}

func (b *Bot) ahMyListings(uid string) []ahListingView {
	rows, err := b.DB.Query(`
		SELECT id, item_type, item_id, item_name, item_data, price, listed_at
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
		var dataJSON []byte
		if err := rows.Scan(&v.ID, &v.ItemType, &v.ItemID, &v.Name, &dataJSON, &v.Price, &t); err != nil {
			continue
		}
		v.Icon = ahIcon(v.ItemType, v.ItemID)
		v.Listed = t.Format("Jan 02")
		v.Mine = true
		ahEnrichListing(&v, dataJSON)
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
	var price, currentBid int64
	var bidderUID sql.NullString
	var durability sql.NullInt64
	err = tx.QueryRow(`
		SELECT item_type, item_id, item_name, item_data, price, seller_uid, durability, current_bid, bidder_uid
		FROM auction_house
		WHERE id=$1 AND sold_at IS NULL AND expires_at > NOW() FOR UPDATE`, req.ID).
		Scan(&itemType, &itemID, &name, &dataJSON, &price, &sellerUID, &durability, &currentBid, &bidderUID)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "listing unavailable"})
		return
	}
	if sellerUID == uid {
		writeJSON(w, map[string]any{"ok": false, "error": "cannot buy your own listing"})
		return
	}

	// Validate the complete delivery before moving any gold. Legacy listings can
	// contain several collectible types; a corrupt or unknown payload must never
	// become a successful purchase with no item delivered.
	var gear *content.Gear
	var ultimate *content.UltimateSkill
	var unique *content.UniqueItem
	switch itemType {
	case "gear":
		var value content.Gear
		if err := json.Unmarshal(dataJSON, &value); err != nil || value.ID == "" || value.ID != itemID {
			writeJSON(w, map[string]any{"ok": false, "error": "listing has invalid gear data"})
			return
		}
		gear = &value
	case "ultimate":
		var value content.UltimateSkill
		if err := json.Unmarshal(dataJSON, &value); err != nil || value.ID == "" || value.ID != itemID {
			writeJSON(w, map[string]any{"ok": false, "error": "listing has invalid ultimate data"})
			return
		}
		var owned bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM user_ultimate_skills
			WHERE client_uid=$1 AND ultimate_id=$2)`, uid, itemID).Scan(&owned); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if owned {
			writeJSON(w, map[string]any{"ok": false, "error": "ultimate already owned"})
			return
		}
		ultimate = &value
	case "unique":
		var value content.UniqueItem
		if err := json.Unmarshal(dataJSON, &value); err != nil || value.Name == "" || value.Name != itemID {
			writeJSON(w, map[string]any{"ok": false, "error": "listing has invalid unique-item data"})
			return
		}
		var owned bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM user_unique_items
			WHERE client_uid=$1 AND item_name=$2)`, uid, itemID).Scan(&owned); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if owned {
			writeJSON(w, map[string]any{"ok": false, "error": "unique item already owned"})
			return
		}
		unique = &value
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "this legacy listing type cannot be delivered safely"})
		return
	}

	// Buy Now cancels any reserved bid. Refund it before charging so the leading
	// bidder can use their own reservation toward the fixed-price purchase.
	if bidderUID.Valid && currentBid > 0 {
		if _, err := tx.Exec("UPDATE users SET gold=gold+$1 WHERE client_uid=$2", currentBid, bidderUID.String); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "refund bid"})
			return
		}
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
	// Mark sold, pay the seller net of the disclosed 5% community tax, and
	// route that tax to the shared Abyss jackpot in the same transaction.
	salesTax := abyssAuctionSalesTax(price)
	sellerNet := price - salesTax
	if _, err := tx.Exec("UPDATE auction_house SET buyer_uid=$1, sold_at=NOW(), current_bid=0, bidder_uid=NULL WHERE id=$2", uid, req.ID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "sold"})
		return
	}
	if _, err := tx.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", sellerNet, sellerUID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "pay"})
		return
	}
	if _, err := tx.Exec("UPDATE arcade_jackpots SET amount=amount+$1,updated_at=NOW() WHERE game_key='abyss'", salesTax); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "tax"})
		return
	}
	if _, err := tx.Exec(`INSERT INTO abyss_economy_events (client_uid,kind,message,amount)
		VALUES ($1,'sale',$2,$3)`, sellerUID, fmt.Sprintf("Sale proceeds: %s sold for %dg · community tax %dg · net %dg.", name, price, salesTax, sellerNet), sellerNet); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "notice"})
		return
	}
	var equippedMsg = ""
	// Deliver gear into the buyer's inventory, preserving the listing's durability.
	// Auto-equips if the gear is an upgrade, displacing the old item back into inventory.
	switch {
	case gear != nil:
		dur := gear.MaxDurability
		if durability.Valid {
			dur = max(0, int(durability.Int64))
		}
		if s.bot.shouldEquip(uid, *gear) {
			if err := s.bot.equipGear(tx, uid, *gear, dur, dataJSON); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "equip"})
				return
			}
			equippedMsg = " and equipped!"
		} else if _, err := tx.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability, item_data) VALUES ($1, $2, $3, $4)", uid, gear.ID, dur, dataJSON); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "deliver"})
			return
		}
	case ultimate != nil:
		if _, err := tx.Exec("INSERT INTO user_ultimate_skills (client_uid,ultimate_id) VALUES ($1,$2)", uid, ultimate.ID); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "deliver"})
			return
		}
		if _, err := tx.Exec("UPDATE users SET ultimate_skills_count=ultimate_skills_count+1 WHERE client_uid=$1", uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "deliver"})
			return
		}
	case unique != nil:
		if _, err := tx.Exec(`INSERT INTO user_unique_items (client_uid,item_name,rarity,power)
			VALUES ($1,$2,$3,$4)`, uid, unique.Name, unique.Rarity, unique.Power); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "deliver"})
			return
		}
		if _, err := tx.Exec("UPDATE users SET unique_items_count=unique_items_count+1 WHERE client_uid=$1", uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "deliver"})
			return
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

	writeJSON(w, map[string]any{"ok": true, "bought": name + equippedMsg, "gold": gold})
}

func abyssAuctionSalesTax(price int64) int64 {
	if price <= 0 {
		return 0
	}
	return max(int64(1), (price*5+99)/100)
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

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "tx"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Read the row through the transaction with a row lock so the item can't be
	// sold, fused or transmuted between our validation and the delete below.
	var gid string
	var dur int
	var itemData sql.NullString
	if err := tx.QueryRow("SELECT gear_id, durability, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2 AND locked=FALSE FOR UPDATE", req.InvID, uid).Scan(&gid, &dur, &itemData); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	// Price and payload are built from the reconstructed instance (makeGear), so
	// gems, runes, temper and other forge data survive the listing — and an
	// attuned item stays bound.
	g, ok := s.bot.makeGear(gid, itemData)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown gear"})
		return
	}
	if g.Attuned {
		writeJSON(w, map[string]any{"ok": false, "error": g.Name + " is attuned to you and cannot be auctioned"})
		return
	}

	// Auto-calculate price: (CR×10 + GS×5) × (Rarity+1)
	price := int64(g.CombatRating()*10+float64(g.Stats.Score())*5) * (int64(g.Rarity) + 1)
	if price < 10 {
		price = 10
	}
	res, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2 AND locked=FALSE", req.InvID, uid)
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
	s.bot.notifyAbyssAHListing(uid, g.ID, g.Name, price)
	writeJSON(w, map[string]any{"ok": true, "listed": g.Name, "price": price,
		"fee": 0, "msg": fmt.Sprintf("Listed %s at %dg · 0g listing fee.", g.Name, price)})
}
