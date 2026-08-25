package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ts3news/internal/content"
)

type abyssSetTradeOffer struct {
	SetID      string `json:"set_id"`
	ItemID     string `json:"item_id,omitempty"`
	Name       string `json:"name,omitempty"`
	Slot       string `json:"slot,omitempty"`
	SpareCount int    `json:"spare_count"`
	CanTrade   bool   `json:"can_trade"`
}

type abyssSetTradeItem struct {
	invID int64
	gear  content.Gear
}

func abyssRotatingMissingSetItem(catalog []content.Gear, owned map[string]bool, year, week int) (content.Gear, bool) {
	if len(catalog) == 0 {
		return content.Gear{}, false
	}
	start := (year*53 + week) % len(catalog)
	for offset := range len(catalog) {
		gear := catalog[(start+offset)%len(catalog)]
		if !owned[gear.ID] {
			return gear, true
		}
	}
	return content.Gear{}, false
}

func abyssSetTradeSpareIDs(items []abyssSetTradeItem, setID string, equipped ...map[string]bool) []int64 {
	var spareIDs []int64
	kept := make(map[string]bool)
	if len(equipped) > 0 {
		for gearID, owned := range equipped[0] {
			kept[gearID] = owned
		}
	}
	for _, item := range items {
		if item.gear.SetID != setID {
			continue
		}
		if !kept[item.gear.ID] {
			kept[item.gear.ID] = true
			continue
		}
		spareIDs = append(spareIDs, item.invID)
	}
	return spareIDs
}

func (s *WebServer) abyssSetTradeState(queryer interface {
	Query(string, ...any) (*sql.Rows, error)
}, uid string, now time.Time) ([]abyssSetTradeOffer, map[string][]int64, error) {
	rows, err := queryer.Query("SELECT id, gear_id, item_data FROM user_inventory WHERE client_uid=$1 AND locked=FALSE ORDER BY id FOR UPDATE", uid)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = rows.Close() }()
	items := []abyssSetTradeItem{}
	owned := make(map[string]bool)
	for rows.Next() {
		var item abyssSetTradeItem
		var gearID string
		var itemData sql.NullString
		if err := rows.Scan(&item.invID, &gearID, &itemData); err != nil {
			return nil, nil, err
		}
		gear, ok := s.bot.makeGear(gearID, itemData)
		if !ok {
			continue
		}
		item.gear = gear
		items = append(items, item)
		owned[gear.ID] = true
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, nil, err
	}
	equippedRows, err := queryer.Query("SELECT gear_id, item_data FROM user_gear WHERE client_uid=$1 FOR UPDATE", uid)
	if err != nil {
		return nil, nil, err
	}
	equipped := make(map[string]bool)
	for equippedRows.Next() {
		var gearID string
		var itemData sql.NullString
		if err := equippedRows.Scan(&gearID, &itemData); err != nil {
			_ = equippedRows.Close()
			return nil, nil, err
		}
		if gear, ok := s.bot.makeGear(gearID, itemData); ok {
			owned[gear.ID] = true
			equipped[gear.ID] = true
		}
	}
	if err := equippedRows.Err(); err != nil {
		_ = equippedRows.Close()
		return nil, nil, err
	}
	if err := equippedRows.Close(); err != nil {
		return nil, nil, err
	}
	year, week := now.ISOWeek()
	offers := make([]abyssSetTradeOffer, 0, 3)
	sparesBySet := make(map[string][]int64, 3)
	for _, setID := range []string{"predator", "warden", "harvester"} {
		spares := abyssSetTradeSpareIDs(items, setID, equipped)
		sparesBySet[setID] = spares
		offer := abyssSetTradeOffer{SetID: setID, SpareCount: len(spares)}
		if gear, ok := abyssRotatingMissingSetItem(content.AbyssSetCatalog(setID), owned, year, week); ok {
			offer.ItemID = gear.ID
			offer.Name = gear.Name
			offer.Slot = string(gear.Slot)
			offer.CanTrade = len(spares) >= 2
		}
		offers = append(offers, offer)
	}
	return offers, sparesBySet, nil
}

func (s *WebServer) handleAbyssSetTrade(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	offers, sparesBySet, err := s.abyssSetTradeState(tx, uid, time.Now().UTC())
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if r.Method == http.MethodGet {
		_ = tx.Rollback()
		writeJSON(w, map[string]any{"ok": true, "offers": offers})
		return
	}
	var req struct {
		SetID string `json:"set_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	var selected abyssSetTradeOffer
	for _, offer := range offers {
		if offer.SetID == req.SetID {
			selected = offer
			break
		}
	}
	if selected.ItemID == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "that set is already complete or unavailable"})
		return
	}
	spares := sparesBySet[selected.SetID]
	if len(spares) < 2 {
		writeJSON(w, map[string]any{"ok": false, "error": "two duplicate set pieces are required"})
		return
	}
	for _, invID := range spares[:2] {
		res, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2 AND locked=FALSE", invID, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if affected, _ := res.RowsAffected(); affected != 1 {
			writeJSON(w, map[string]any{"ok": false, "error": "a duplicate vanished during the trade"})
			return
		}
	}
	gear, ok := content.GetGearByID(selected.ItemID)
	if !ok || gear.SetID != selected.SetID {
		writeJSON(w, map[string]any{"ok": false, "error": "offer expired"})
		return
	}
	data, err := json.Marshal(gear)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability, item_data) VALUES ($1,$2,$3,$4)", uid, gear.ID, gear.MaxDurability, string(data)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("🧩 Traded two duplicates for %s.", gear.Name), "item": gear.Name})
}
