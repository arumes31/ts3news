package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ts3news/internal/content"
)

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
