package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"ts3news/internal/content"
)

type abyssEconomyNotice struct {
	ID      int64  `json:"id"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Amount  int64  `json:"amount"`
	When    string `json:"when"`
}

type abyssMaterialOrderView struct {
	ID        int64
	Material  string
	UnitPrice int64
	Remaining int
	Escrow    int64
	Buyer     string
	Mine      bool
}

type abyssAHEconomyView struct {
	CheapestLegendary int64
	UnseenNotices     int
	Orders            []abyssMaterialOrderView
	Expired           []ahListingView
	SoldTotal         int
	ListedTotal       int
	ExpiredTotal      int
	TaxLeaders        []abyssTaxLeaderView
}

func (b *Bot) abyssAHWatchlist(uid string) map[string]bool {
	rows, err := b.DB.Query("SELECT item_id FROM abyss_ah_watchlist WHERE client_uid=$1", uid)
	if err != nil {
		return map[string]bool{}
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var itemID string
		if rows.Scan(&itemID) == nil {
			out[itemID] = true
		}
	}
	return out
}

func (b *Bot) abyssAHEconomyPage(uid string) abyssAHEconomyView {
	view := abyssAHEconomyView{Expired: b.ahExpiredListings(uid), Orders: b.abyssMaterialOrders(uid), TaxLeaders: b.abyssTaxLeaders(5)}
	_ = b.DB.QueryRow(`SELECT COALESCE(MIN(price),0) FROM auction_house
		WHERE sold_at IS NULL AND expires_at>NOW() AND (item_data->>'Rarity')::int=$1`, int(content.RarityLegendary)).Scan(&view.CheapestLegendary)
	_ = b.DB.QueryRow("SELECT COUNT(*) FROM abyss_economy_events WHERE client_uid=$1 AND seen=FALSE", uid).Scan(&view.UnseenNotices)
	_ = b.DB.QueryRow("SELECT COUNT(*) FROM auction_house WHERE seller_uid=$1 AND sold_at IS NOT NULL", uid).Scan(&view.SoldTotal)
	_ = b.DB.QueryRow("SELECT COUNT(*) FROM auction_house WHERE seller_uid=$1 AND sold_at IS NULL AND expires_at>NOW()", uid).Scan(&view.ListedTotal)
	_ = b.DB.QueryRow("SELECT COUNT(*) FROM auction_house WHERE seller_uid=$1 AND sold_at IS NULL AND expires_at<=NOW()", uid).Scan(&view.ExpiredTotal)
	return view
}

func (b *Bot) ahExpiredListings(uid string) []ahListingView {
	rows, err := b.DB.Query(`SELECT id,item_type,item_id,item_name,item_data,price,listed_at,current_bid
		FROM auction_house WHERE seller_uid=$1 AND sold_at IS NULL AND expires_at<=NOW()
		ORDER BY expires_at DESC LIMIT 50`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []ahListingView
	for rows.Next() {
		var view ahListingView
		var data []byte
		var listed time.Time
		if rows.Scan(&view.ID, &view.ItemType, &view.ItemID, &view.Name, &data, &view.Price, &listed, &view.CurrentBid) != nil {
			continue
		}
		view.Icon, view.Listed, view.Mine = ahIcon(view.ItemType, view.ItemID), listed.Format("Jan 02"), true
		ahEnrichListing(&view, data)
		out = append(out, view)
	}
	return out
}

func (b *Bot) abyssMaterialOrders(uid string) []abyssMaterialOrderView {
	rows, err := b.DB.Query(`SELECT o.id,o.material,o.unit_price,o.remaining,o.escrow_gold,
		COALESCE(NULLIF(u.nickname,''),'Adventurer'),o.buyer_uid=$1
		FROM abyss_material_orders o JOIN users u ON u.client_uid=o.buyer_uid
		WHERE o.closed_at IS NULL AND o.remaining>0 ORDER BY o.unit_price DESC,o.created_at LIMIT 30`, uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []abyssMaterialOrderView
	for rows.Next() {
		var view abyssMaterialOrderView
		if rows.Scan(&view.ID, &view.Material, &view.UnitPrice, &view.Remaining, &view.Escrow, &view.Buyer, &view.Mine) == nil {
			out = append(out, view)
		}
	}
	return out
}

func (s *WebServer) handleAHWatch(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ItemID string `json:"item_id"`
	}
	if err := readJSON(r, &req); err != nil || req.ItemID == "" || len(req.ItemID) > 128 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid item"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec("DELETE FROM abyss_ah_watchlist WHERE client_uid=$1 AND item_id=$2", uid, req.ItemID)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	removed, _ := res.RowsAffected()
	watched := removed == 0
	if watched {
		if _, err := tx.Exec("INSERT INTO abyss_ah_watchlist (client_uid,item_id) VALUES ($1,$2)", uid, req.ItemID); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "watched": watched, "msg": map[bool]string{true: "Watchlist alert enabled.", false: "Watchlist alert removed."}[watched]})
}

func (s *WebServer) handleAHNotices(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	rows, err := tx.Query(`SELECT id,kind,message,amount,created_at FROM abyss_economy_events
		WHERE client_uid=$1 AND seen=FALSE ORDER BY created_at LIMIT 20 FOR UPDATE`, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var notices []abyssEconomyNotice
	for rows.Next() {
		var notice abyssEconomyNotice
		var created time.Time
		if rows.Scan(&notice.ID, &notice.Kind, &notice.Message, &notice.Amount, &created) == nil {
			notice.When = created.Format("Jan 02 15:04")
			notices = append(notices, notice)
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := rows.Close(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if len(notices) > 0 {
		if _, err := tx.Exec(`UPDATE abyss_economy_events SET seen=TRUE WHERE id IN
			(SELECT id FROM abyss_economy_events WHERE client_uid=$1 AND seen=FALSE ORDER BY created_at LIMIT 20)`, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "notices": notices})
}

func (s *WebServer) handleAHBulkRelist(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	res, err := s.bot.DB.Exec(`UPDATE auction_house SET price=GREATEST(1,(price*99)/100),
		listed_at=NOW(),expires_at=NOW()+INTERVAL '7 days'
		WHERE seller_uid=$1 AND sold_at IS NULL AND expires_at<=NOW() AND bidder_uid IS NULL`, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	count, _ := res.RowsAffected()
	writeJSON(w, map[string]any{"ok": true, "count": count, "msg": fmt.Sprintf("Relisted %d expired listings at 1%% lower buy-now prices.", count)})
}

func abyssMaterialOrderTotal(unitPrice int64, count int) (int64, bool) {
	if unitPrice < 1 || count < 1 || unitPrice > math.MaxInt64/int64(count) {
		return 0, false
	}
	return unitPrice * int64(count), true
}

func (s *WebServer) handleAHMaterialOrder(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Material  string `json:"material"`
		Count     int    `json:"count"`
		UnitPrice int64  `json:"unit_price"`
	}
	if err := readJSON(r, &req); err != nil || (req.Material != "dust" && req.Material != "shard" && req.Material != "core") || req.Count < 1 || req.Count > 10_000 || req.UnitPrice < 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid order (dust/shard/core, count 1–10000, positive price)"})
		return
	}
	total, validTotal := abyssMaterialOrderTotal(req.UnitPrice, req.Count)
	if !validTotal {
		writeJSON(w, map[string]any{"ok": false, "error": "order total is too large"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec("UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold >= $1", total, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	if _, err := tx.Exec(`INSERT INTO abyss_material_orders (buyer_uid,material,unit_price,remaining,escrow_gold)
		VALUES ($1,$2,$3,$4,$5)`, uid, req.Material, req.UnitPrice, req.Count, total); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "gold": s.bot.abyssGold(uid), "msg": fmt.Sprintf("Buy order posted: %d %s at %dg each (%dg escrowed).", req.Count, req.Material, req.UnitPrice, total)})
}

func (s *WebServer) handleAHMaterialFill(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID    int64 `json:"id"`
		Count int   `json:"count"`
	}
	if err := readJSON(r, &req); err != nil || req.ID <= 0 || req.Count < 1 || req.Count > 10_000 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid fill"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var buyer, material string
	var unitPrice, escrow int64
	var remaining int
	if err := tx.QueryRow(`SELECT buyer_uid,material,unit_price,remaining,escrow_gold
		FROM abyss_material_orders WHERE id=$1 AND closed_at IS NULL FOR UPDATE`, req.ID).Scan(&buyer, &material, &unitPrice, &remaining, &escrow); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "order unavailable"})
		return
	}
	if buyer == uid {
		writeJSON(w, map[string]any{"ok": false, "error": "cannot fill your own order"})
		return
	}
	count := min(req.Count, remaining)
	if !spendMaterials(tx, uid, map[string]int{material: count}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough materials"})
		return
	}
	if err := grantMaterialQ(tx, buyer, material, count); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	payout := unitPrice * int64(count)
	if payout > escrow {
		writeJSON(w, map[string]any{"ok": false, "error": "corrupt order escrow"})
		return
	}
	if _, err := tx.Exec("UPDATE users SET gold=gold+$1 WHERE client_uid=$2", payout, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec(`UPDATE abyss_material_orders SET remaining=remaining-$1,escrow_gold=escrow_gold-$2,
		closed_at=CASE WHEN remaining-$1=0 THEN NOW() ELSE NULL END WHERE id=$3`, count, payout, req.ID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec(`INSERT INTO abyss_economy_events (client_uid,kind,message,amount)
		VALUES ($1,'order_fill',$2,$3)`, buyer, fmt.Sprintf("Material order filled: %d %s.", count, material), -payout); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "count": count, "payout": payout, "gold": s.bot.abyssGold(uid), "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("Filled %d %s for %dg.", count, material, payout)})
}

func (s *WebServer) handleAHMaterialCancel(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID int64 `json:"id"`
	}
	if err := readJSON(r, &req); err != nil || req.ID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid order"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var escrow int64
	if err := tx.QueryRow(`SELECT escrow_gold FROM abyss_material_orders
		WHERE id=$1 AND buyer_uid=$2 AND closed_at IS NULL FOR UPDATE`, req.ID, uid).Scan(&escrow); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "order unavailable"})
		return
	}
	if _, err := tx.Exec(`UPDATE abyss_material_orders SET remaining=0,escrow_gold=0,closed_at=NOW()
		WHERE id=$1`, req.ID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("UPDATE users SET gold=gold+$1 WHERE client_uid=$2", escrow, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "refund": escrow, "gold": s.bot.abyssGold(uid),
		"msg": fmt.Sprintf("Buy order cancelled · %dg escrow refunded.", escrow)})
}

func abyssAntiSnipeExpiry(now, expiry time.Time) time.Time {
	if expiry.After(now) && !expiry.After(now.Add(time.Minute)) {
		return expiry.Add(time.Minute)
	}
	return expiry
}

func (s *WebServer) handleAHBid(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID     string `json:"id"`
		Amount int64  `json:"amount"`
	}
	if err := readJSON(r, &req); err != nil || req.Amount < 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid bid"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var seller, itemType, itemID string
	var itemData []byte
	var buyNow, current int64
	var previous sql.NullString
	var expiry time.Time
	if err := tx.QueryRow(`SELECT seller_uid,item_type,item_id,item_data,price,current_bid,bidder_uid,expires_at
		FROM auction_house WHERE id=$1 AND sold_at IS NULL AND expires_at>NOW() FOR UPDATE`, req.ID).
		Scan(&seller, &itemType, &itemID, &itemData, &buyNow, &current, &previous, &expiry); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "listing unavailable"})
		return
	}
	if seller == uid || itemType != "gear" {
		writeJSON(w, map[string]any{"ok": false, "error": "this listing cannot be bid on"})
		return
	}
	var gear content.Gear
	if err := json.Unmarshal(itemData, &gear); err != nil || gear.ID == "" || gear.ID != itemID {
		writeJSON(w, map[string]any{"ok": false, "error": "listing has invalid gear data"})
		return
	}
	minimum := (buyNow + 1) / 2
	if current > 0 {
		minimum = current + max(int64(1), current/20)
	}
	if req.Amount < minimum || req.Amount >= buyNow {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("bid must be %d–%d; use Buy Now at %d", minimum, buyNow-1, buyNow)})
		return
	}
	res, err := tx.Exec("UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold >= $1", req.Amount, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	if previous.Valid {
		if _, err := tx.Exec("UPDATE users SET gold=gold+$1 WHERE client_uid=$2", current, previous.String); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	newExpiry := abyssAntiSnipeExpiry(time.Now(), expiry)
	if _, err := tx.Exec("UPDATE auction_house SET current_bid=$1,bidder_uid=$2,expires_at=$3 WHERE id=$4", req.Amount, uid, newExpiry, req.ID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "bid": req.Amount, "extended": newExpiry.After(expiry), "gold": s.bot.abyssGold(uid),
		"msg": fmt.Sprintf("Bid reserved at %dg%s.", req.Amount, map[bool]string{true: " · anti-snipe +60s"}[newExpiry.After(expiry)])})
}

func (b *Bot) settleAbyssAuctionBids() {
	rows, err := b.DB.Query(`SELECT id::text FROM auction_house WHERE sold_at IS NULL AND expires_at<=NOW()
		AND bidder_uid IS NOT NULL AND current_bid>0 ORDER BY expires_at LIMIT 25`)
	if err != nil {
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	_ = rows.Close()
	for _, id := range ids {
		b.settleAbyssAuctionBid(id)
	}
}

func (b *Bot) settleAbyssAuctionBid(id string) {
	tx, err := b.DB.Begin()
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback() }()
	var seller, bidder, itemType, itemID, name string
	var data []byte
	var bid int64
	var durability sql.NullInt64
	if err := tx.QueryRow(`SELECT seller_uid,bidder_uid,item_type,item_id,item_name,item_data,current_bid,durability
		FROM auction_house WHERE id=$1 AND sold_at IS NULL AND expires_at<=NOW() AND bidder_uid IS NOT NULL FOR UPDATE`, id).
		Scan(&seller, &bidder, &itemType, &itemID, &name, &data, &bid, &durability); err != nil || itemType != "gear" || bid <= 0 {
		return
	}
	var gear content.Gear
	if err := json.Unmarshal(data, &gear); err != nil || gear.ID == "" || gear.ID != itemID {
		if _, err := tx.Exec("UPDATE users SET gold=gold+$1 WHERE client_uid=$2", bid, bidder); err != nil {
			return
		}
		if _, err := tx.Exec("UPDATE auction_house SET current_bid=0,bidder_uid=NULL WHERE id=$1", id); err != nil {
			return
		}
		if _, err := tx.Exec(`INSERT INTO abyss_economy_events (client_uid,kind,message,amount)
			VALUES ($1,'bid_refund',$2,$3)`, bidder, fmt.Sprintf("Bid refunded: %s could not be delivered safely.", name), bid); err != nil {
			return
		}
		_ = tx.Commit()
		return
	}
	dur := gear.MaxDurability
	if durability.Valid {
		dur = int(durability.Int64)
	}
	if _, err := tx.Exec("INSERT INTO user_inventory (client_uid,gear_id,durability,item_data) VALUES ($1,$2,$3,$4)", bidder, itemID, dur, data); err != nil {
		return
	}
	if _, err := tx.Exec("UPDATE users SET gold=gold+$1 WHERE client_uid=$2", bid, seller); err != nil {
		return
	}
	if _, err := tx.Exec("UPDATE auction_house SET buyer_uid=$1,sold_at=NOW() WHERE id=$2", bidder, id); err != nil {
		return
	}
	if _, err := tx.Exec(`INSERT INTO abyss_economy_events (client_uid,kind,message,amount) VALUES
		($1,'sale',$3,$4),($2,'bid_win',$5,$6)`, seller, bidder,
		fmt.Sprintf("Auction sold by bid: %s · %dg proceeds (0g fee).", name, bid), bid,
		fmt.Sprintf("Winning bid delivered: %s · %dg reserved.", name, bid), -bid); err != nil {
		return
	}
	_ = tx.Commit()
}

func (b *Bot) notifyAbyssAHListing(sellerUID, itemID, name string, price int64) {
	_, _ = b.DB.Exec(`INSERT INTO abyss_economy_events (client_uid,kind,message,amount)
		SELECT client_uid,'watchlist',$1,$2 FROM abyss_ah_watchlist
		WHERE item_id=$3 AND client_uid<>$4`, fmt.Sprintf("Watchlist match listed: %s at %dg.", name, price), price, itemID, sellerUID)
	var average int64
	_ = b.DB.QueryRow(`SELECT COALESCE(AVG(price)::bigint,0) FROM auction_house
		WHERE item_id=$1 AND sold_at IS NULL AND expires_at>NOW() AND seller_uid<>$2`, itemID, sellerUID).Scan(&average)
	if average > 0 && price*100 <= average*80 {
		_, _ = b.DB.Exec(`INSERT INTO abyss_economy_events (client_uid,kind,message,amount)
			SELECT DISTINCT seller_uid,'price_alert',$1,$2 FROM auction_house
			WHERE item_id=$3 AND sold_at IS NULL AND expires_at>NOW() AND seller_uid<>$4`,
			fmt.Sprintf("Price alert: %s listed at least 20%% below its active average (%dg).", name, average), price, itemID, sellerUID)
	}
}
