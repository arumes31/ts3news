package bot

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func takeAbyssConsumableTx(tx *sql.Tx, uid, consID string, quantity int) error {
	if quantity <= 0 {
		return fmt.Errorf("invalid consumable quantity")
	}
	var owned int
	if err := tx.QueryRow(`SELECT remaining_fights FROM user_consumables
		WHERE client_uid=$1 AND cons_id=$2 FOR UPDATE`, uid, consID).Scan(&owned); err != nil || owned < quantity {
		return fmt.Errorf("not enough %s", consID)
	}
	if owned == quantity {
		_, err := tx.Exec("DELETE FROM user_consumables WHERE client_uid=$1 AND cons_id=$2", uid, consID)
		return err
	}
	_, err := tx.Exec(`UPDATE user_consumables SET remaining_fights=remaining_fights-$1
		WHERE client_uid=$2 AND cons_id=$3`, quantity, uid, consID)
	return err
}

func giveAbyssConsumableTx(tx *sql.Tx, uid, consID string, quantity int) error {
	if quantity <= 0 {
		return fmt.Errorf("invalid consumable quantity")
	}
	_, err := tx.Exec(`INSERT INTO user_consumables (client_uid,cons_id,remaining_fights) VALUES ($1,$2,$3)
		ON CONFLICT (client_uid,cons_id) DO UPDATE
		SET remaining_fights=user_consumables.remaining_fights+EXCLUDED.remaining_fights`, uid, consID, quantity)
	return err
}

func (s *WebServer) handleAbyssConsumableTrade(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Action          string `json:"action"`
		TradeID         int64  `json:"trade_id"`
		RecipientUID    string `json:"recipient_uid"`
		OfferConsID     string `json:"offer_cons_id"`
		OfferQuantity   int    `json:"offer_quantity"`
		RequestConsID   string `json:"request_cons_id"`
		RequestQuantity int    `json:"request_quantity"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	req.Action = strings.TrimSpace(req.Action)
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	message := "Trade updated."
	switch req.Action {
	case "create":
		req.RecipientUID = strings.TrimSpace(req.RecipientUID)
		req.OfferConsID = strings.TrimSpace(req.OfferConsID)
		req.RequestConsID = strings.TrimSpace(req.RequestConsID)
		if req.RecipientUID == "" || req.RecipientUID == uid || req.OfferConsID == "" || req.RequestConsID == "" ||
			req.OfferQuantity < 1 || req.OfferQuantity > 99 || req.RequestQuantity < 1 || req.RequestQuantity > 99 {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid consumable trade"})
			return
		}
		var recipient string
		if err := tx.QueryRow("SELECT client_uid FROM users WHERE client_uid=$1 FOR UPDATE", req.RecipientUID).Scan(&recipient); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "recipient not found"})
			return
		}
		if err := takeAbyssConsumableTx(tx, uid, req.OfferConsID, req.OfferQuantity); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if _, err := tx.Exec(`INSERT INTO abyss_consumable_trades
			(sender_uid,recipient_uid,offer_cons_id,offer_quantity,request_cons_id,request_quantity)
			VALUES ($1,$2,$3,$4,$5,$6)`, uid, recipient, req.OfferConsID, req.OfferQuantity,
			req.RequestConsID, req.RequestQuantity); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "finish or cancel your existing open trade"})
			return
		}
		message = "Consumables reserved in a 24-hour trade offer."
	case "accept", "cancel":
		if req.TradeID <= 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "invalid trade"})
			return
		}
		var sender, recipient, offerID, requestID, status string
		var offerQty, requestQty int
		var expired bool
		if err := tx.QueryRow(`SELECT sender_uid,recipient_uid,offer_cons_id,offer_quantity,
			request_cons_id,request_quantity,status,expires_at<=NOW() FROM abyss_consumable_trades
			WHERE trade_id=$1 FOR UPDATE`, req.TradeID).Scan(&sender, &recipient, &offerID, &offerQty,
			&requestID, &requestQty, &status, &expired); err != nil || status != "open" {
			writeJSON(w, map[string]any{"ok": false, "error": "open trade not found"})
			return
		}
		if expired {
			if err := giveAbyssConsumableTx(tx, sender, offerID, offerQty); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			_, err = tx.Exec("UPDATE abyss_consumable_trades SET status='expired' WHERE trade_id=$1", req.TradeID)
			message = "Trade expired; reserved consumables returned."
		} else if req.Action == "cancel" {
			if sender != uid {
				writeJSON(w, map[string]any{"ok": false, "error": "only the sender can cancel this trade"})
				return
			}
			if err := giveAbyssConsumableTx(tx, sender, offerID, offerQty); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			_, err = tx.Exec("UPDATE abyss_consumable_trades SET status='cancelled' WHERE trade_id=$1", req.TradeID)
			message = "Trade cancelled; reserved consumables returned."
		} else {
			if recipient != uid {
				writeJSON(w, map[string]any{"ok": false, "error": "this trade belongs to another recipient"})
				return
			}
			if err := takeAbyssConsumableTx(tx, recipient, requestID, requestQty); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			if err := giveAbyssConsumableTx(tx, recipient, offerID, offerQty); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			if err := giveAbyssConsumableTx(tx, sender, requestID, requestQty); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
			_, err = tx.Exec("UPDATE abyss_consumable_trades SET status='accepted' WHERE trade_id=$1", req.TradeID)
			message = "Consumable trade settled atomically."
		}
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "invalid trade action"})
		return
	}
	if err != nil || tx.Commit() != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": message})
}

func (s *WebServer) handleAbyssShoutbox(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Message string `json:"message"`
	}
	if readJSON(r, &req) != nil || !abyssSocialTextValid(req.Message, abyssSocialMessageMaxRunes) {
		writeJSON(w, map[string]any{"ok": false, "error": "use 1–240 printable characters"})
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	var recent int
	if err := s.bot.DB.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM abyss_shoutbox_messages
		WHERE sender_uid=$1 AND created_at>NOW()-INTERVAL '30 seconds'`, uid).Scan(&recent); err != nil || recent > 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "wait 30 seconds before another shout"})
		return
	}
	if _, err := s.bot.DB.ExecContext(r.Context(), `INSERT INTO abyss_shoutbox_messages (sender_uid,message)
		VALUES ($1,$2)`, uid, req.Message); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "Shout posted and queued for the sender's TeamSpeak channel."})
}

func (s *WebServer) handleAbyssFloorMessage(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Kind    string `json:"kind"`
		Message string `json:"message"`
	}
	if readJSON(r, &req) != nil || !abyssSocialTextValid(req.Message, 160) || (req.Kind != "hint" && req.Kind != "taunt") {
		writeJSON(w, map[string]any{"ok": false, "error": "choose a hint or taunt using 1–160 printable characters"})
		return
	}
	var depth int
	var startedAt time.Time
	if err := s.bot.DB.QueryRowContext(r.Context(), `SELECT depth,started_at FROM abyss_active
		WHERE client_uid=$1 AND hp>0`, uid).Scan(&depth, &startedAt); err != nil || depth <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "leave a floor message during a living run"})
		return
	}
	result, err := s.bot.DB.ExecContext(r.Context(), `INSERT INTO abyss_floor_messages
		(sender_uid,run_started_at,depth,kind,message) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (sender_uid,run_started_at) DO NOTHING`, uid, startedAt, depth, req.Kind, strings.TrimSpace(req.Message))
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "this run already left a floor message"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "Your floor message now echoes for other delvers."})
}

func (s *WebServer) handleAbyssKudos(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		UID string `json:"uid"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	req.UID = strings.TrimSpace(req.UID)
	_, _, ok := abyssPair(uid, req.UID)
	if !ok || s.bot.abyssDuoAssists(uid, req.UID) <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "kudos require a completed shared floor"})
		return
	}
	result, err := s.bot.DB.ExecContext(r.Context(), `INSERT INTO abyss_kudos (week_key,sender_uid,recipient_uid)
		VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`, abyssCurrentWeek(timeNowUTC()), uid, req.UID)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "you already thanked this ally this week"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "Kudos recorded for the weekly appreciation board."})
}
