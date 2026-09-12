package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"
)

type shopBuffPurchaseRequest struct {
	Kind          string `json:"kind"`
	Amount        *int64 `json:"amount"`
	ExpectedOwned *int64 `json:"expected_owned"`
	ExpectedGold  *int64 `json:"expected_gold"`
}

// handleShopBuffAPI charges gold and activates a batch of permanent tokens.
// The reviewed count is a replay guard even after prices stop increasing.
// An uncertain response must be verified, never automatically retried.
func (s *WebServer) handleShopBuffAPI(w http.ResponseWriter, r *http.Request, uid string) {
	fail := func(message string) { writeJSON(w, map[string]any{"ok": false, "error": message}) }
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fail("Use the shop's token purchase button.")
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		w.WriteHeader(http.StatusUnsupportedMediaType)
		fail("The token request must use JSON.")
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || !strings.EqualFold(parsed.Host, r.Host) {
			w.WriteHeader(http.StatusForbidden)
			fail("Open the shop on this site before buying a token.")
			return
		}
	}
	if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
		w.WriteHeader(http.StatusForbidden)
		fail("Open the shop on this site before buying a token.")
		return
	}
	var req shopBuffPurchaseRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2048))
	if err := decoder.Decode(&req); err != nil || (req.Kind != "rarity" && req.Kind != "quantity") || req.ExpectedOwned == nil || req.ExpectedGold == nil || *req.ExpectedOwned < 0 || *req.ExpectedGold < 0 {
		fail("Review a rarity or quantity token before buying.")
		return
	}
	amount := int64(1) // Older clients omit the amount for a single token.
	if req.Amount != nil {
		amount = *req.Amount
	}
	if amount < 1 {
		fail("Choose a positive whole number of tokens.")
		return
	}
	internalFailure := func(stage string, err error) {
		slog.ErrorContext(r.Context(), "shop buff purchase failed", "stage", stage, "error", err)
		fail("The token purchase could not be saved. Your gold and bonuses are unchanged; try again.")
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		internalFailure("begin", err)
		return
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			slog.ErrorContext(r.Context(), "shop buff rollback failed", "error", err)
		}
	}()
	var gold int64
	if err := tx.QueryRowContext(r.Context(), "SELECT gold FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&gold); err != nil {
		internalFailure("load wallet", err)
		return
	}
	state, err := loadShopBuffs(r.Context(), tx, uid)
	if err != nil {
		internalFailure("load buffs", err)
		return
	}
	owned := state.Rarity
	if req.Kind == "quantity" {
		owned = state.Quantity
	}
	if *req.ExpectedGold != gold || *req.ExpectedOwned != owned {
		writeJSON(w, map[string]any{"ok": false, "review_required": true, "error": "Your balance or token count changed. Refresh the shop and review the new price."})
		return
	}
	price, err := shopBuffTotalPrice(owned, amount)
	if err != nil {
		fail(err.Error())
		return
	}
	if gold < price {
		fail("You do not have enough gold for these tokens. Your permanent bonuses are unchanged.")
		return
	}
	if req.Kind == "rarity" {
		state.Rarity += amount
	} else {
		state.Quantity += amount
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		internalFailure("encode buffs", err)
		return
	}
	result, err := tx.ExecContext(r.Context(), "UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", price, uid)
	if err != nil {
		internalFailure("debit wallet", err)
		return
	}
	affected, err := result.RowsAffected()
	if err != nil {
		internalFailure("verify debit", err)
		return
	}
	if affected != 1 {
		fail("Your gold balance changed. Refresh the shop before buying.")
		return
	}
	if _, err := tx.ExecContext(r.Context(), "INSERT INTO app_meta (key,value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value", shopBuffKey(uid), string(encoded)); err != nil {
		internalFailure("save buffs", err)
		return
	}
	if err := tx.Commit(); err != nil {
		slog.ErrorContext(r.Context(), "shop buff commit unconfirmed", "error", err)
		writeJSON(w, map[string]any{"ok": false, "unconfirmed": true, "error": "The token purchase result is unconfirmed. Refresh to verify your gold and permanent bonus before trying again."})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "kind": req.Kind, "amount": amount, "gold": gold - price, "buffs": state, "price": price, "next_price": shopBuffPrice(owned + amount)})
}
