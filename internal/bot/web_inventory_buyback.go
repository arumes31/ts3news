package bot

import (
	"database/sql"
	"errors"
	"math"
	"net/http"
	"time"
)

const abyssVendorBuybackLimit = 10

var errVendorBuybackFunds = errors.New("not enough gold")

type vendorBuybackView struct {
	gearView
	BuybackID   int64
	BuybackCost int64
	SaleValue   int64
	SoldAt      string
}

func abyssVendorBuybackCost(saleValue int64) int64 {
	if saleValue <= 0 {
		return 0
	}
	fee := saleValue / 10
	if saleValue%10 != 0 {
		fee++
	}
	fee = max(fee, int64(1))
	if saleValue > math.MaxInt64-fee {
		return math.MaxInt64
	}
	return saleValue + fee
}

func recordVendorBuyback(
	tx *sql.Tx,
	uid string,
	gearID string,
	durability int,
	itemData sql.NullString,
	acquiredAt time.Time,
	saleValue int64,
) error {
	buybackCost := abyssVendorBuybackCost(saleValue)
	if buybackCost <= 0 {
		return errors.New("invalid vendor buyback cost")
	}
	if _, err := tx.Exec(`INSERT INTO abyss_vendor_buybacks
		(client_uid,gear_id,durability,item_data,acquired_at,sale_value,buyback_cost)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		uid, gearID, durability, itemData, acquiredAt, saleValue, buybackCost); err != nil {
		return err
	}
	_, err := tx.Exec(`DELETE FROM abyss_vendor_buybacks WHERE id IN (
		SELECT id FROM abyss_vendor_buybacks WHERE client_uid=$1
		ORDER BY sold_at DESC,id DESC OFFSET $2
	)`, uid, abyssVendorBuybackLimit)
	return err
}

func (b *Bot) vendorBuybacks(uid string) []vendorBuybackView {
	rows, err := b.DB.Query(`SELECT id,gear_id,durability,item_data,sale_value,buyback_cost,sold_at
		FROM abyss_vendor_buybacks WHERE client_uid=$1
		ORDER BY sold_at DESC,id DESC LIMIT $2`, uid, abyssVendorBuybackLimit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()

	buybacks := make([]vendorBuybackView, 0, abyssVendorBuybackLimit)
	for rows.Next() {
		var id, saleValue, buybackCost int64
		var gearID string
		var durability int
		var itemData sql.NullString
		var soldAt time.Time
		if err := rows.Scan(&id, &gearID, &durability, &itemData, &saleValue, &buybackCost, &soldAt); err != nil {
			continue
		}
		gear, ok := b.makeGear(gearID, itemData)
		if !ok {
			continue
		}
		view := toGearView(gear.Slot, gear)
		view.Durability = durability
		buybacks = append(buybacks, vendorBuybackView{
			gearView: view, BuybackID: id, BuybackCost: buybackCost, SaleValue: saleValue,
			SoldAt: soldAt.UTC().Format("02 Jan · 15:04 UTC"),
		})
	}
	return buybacks
}

func buyBackVendorItem(tx *sql.Tx, uid string, buybackID int64) (int64, error) {
	var gearID string
	var durability int
	var itemData sql.NullString
	var acquiredAt time.Time
	var cost int64
	if err := tx.QueryRow(`SELECT gear_id,durability,item_data,acquired_at,buyback_cost
		FROM abyss_vendor_buybacks WHERE id=$1 AND client_uid=$2 FOR UPDATE`,
		buybackID, uid).Scan(&gearID, &durability, &itemData, &acquiredAt, &cost); err != nil {
		return 0, err
	}
	var gold int64
	if err := tx.QueryRow(`UPDATE users SET gold=gold-$1
		WHERE client_uid=$2 AND gold >= $1 RETURNING gold`, cost, uid).Scan(&gold); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errVendorBuybackFunds
		}
		return 0, err
	}
	if _, err := tx.Exec(`INSERT INTO user_inventory (client_uid,gear_id,durability,item_data,acquired_at)
		VALUES ($1,$2,$3,$4,$5)`, uid, gearID, durability, itemData, acquiredAt); err != nil {
		return 0, err
	}
	result, err := tx.Exec("DELETE FROM abyss_vendor_buybacks WHERE id=$1 AND client_uid=$2", buybackID, uid)
	if err != nil {
		return 0, err
	}
	if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		if err != nil {
			return 0, err
		}
		return 0, sql.ErrNoRows
	}
	return gold, nil
}

func (s *WebServer) handleInventoryBuyback(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var request struct {
		ID int64 `json:"id"`
	}
	if err := readJSON(r, &request); err != nil || request.ID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	unlock := s.lockAbyss(uid)
	defer unlock()
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	gold, err := buyBackVendorItem(tx, uid, request.ID)
	if errors.Is(err, errVendorBuybackFunds) {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, map[string]any{"ok": false, "error": "buyback item not found"})
		return
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "gold": gold, "msg": "Item returned to your inventory."})
}
