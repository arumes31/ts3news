package bot

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

var abyssGiftableShopItems = map[string]string{
	"great_potions": "great_health_potion", "repair_kits": "master_repair_kit",
	"phoenix": "phoenix_feather", "elixir_of_life": "elixir_of_life",
}

func abyssGiftCode() (string, error) {
	var data [6]byte
	if _, err := cryptorand.Read(data[:]); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(data[:])), nil
}

func abyssGiftQuantity(itemKey string) int {
	switch itemKey {
	case "great_potions":
		return 3
	case "repair_kits":
		return 2
	default:
		return 1
	}
}

func (s *WebServer) handleAbyssGiftCreate(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		Item      string `json:"item"`
		Recipient string `json:"recipient"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if _, ok := abyssGiftableShopItems[req.Item]; !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "that shop item cannot be gifted"})
		return
	}
	item, _ := abyssShopByKey(req.Item)
	var recipientUID string
	if err := s.bot.DB.QueryRow(`SELECT client_uid FROM users WHERE client_uid=$1 OR LOWER(nickname)=LOWER($1) LIMIT 1`, strings.TrimSpace(req.Recipient)).Scan(&recipientUID); err != nil || recipientUID == uid {
		writeJSON(w, map[string]any{"ok": false, "error": "recipient not found or invalid"})
		return
	}
	code, err := abyssGiftCode()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "code generation failed"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens-$1 WHERE client_uid=$2 AND abyss_tokens >= $1", item.Cost, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens"})
		return
	}
	if _, err := tx.Exec(`INSERT INTO abyss_shop_gifts (code,sender_uid,recipient_uid,item_key) VALUES ($1,$2,$3,$4)`, code, uid, recipientUID, req.Item); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "code": code, "tokens": s.bot.abyssTokens(uid), "msg": "Gift code created: " + code})
}

func (s *WebServer) handleAbyssGiftRedeem(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		Code string `json:"code"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var itemKey string
	if err := tx.QueryRow(`UPDATE abyss_shop_gifts SET claimed_at=NOW()
		WHERE code=$1 AND recipient_uid=$2 AND claimed_at IS NULL RETURNING item_key`, strings.ToUpper(strings.TrimSpace(req.Code)), uid).Scan(&itemKey); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "gift code invalid, claimed, or meant for another player"})
		return
	}
	consID := abyssGiftableShopItems[itemKey]
	quantity := abyssGiftQuantity(itemKey)
	if _, err := tx.Exec(`INSERT INTO user_consumables (client_uid,cons_id,remaining_fights) VALUES ($1,$2,$3)
		ON CONFLICT (client_uid,cons_id) DO UPDATE SET remaining_fights=user_consumables.remaining_fights+$3`, uid, consID, quantity); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "consumables": s.bot.getConsumables(uid),
		"msg": fmt.Sprintf("Gift redeemed: %d × %s.", quantity, consID)})
}
