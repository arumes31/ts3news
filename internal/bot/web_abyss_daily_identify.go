package bot

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
)

const abyssDailyIdentifyDateSQL = "(CURRENT_TIMESTAMP AT TIME ZONE 'UTC')::date::text"

type abyssDailyIdentifyQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

var errAbyssDailyIdentifyQuoteStale = forgeContractError("daily identify benefit changed; refresh the forge quote")

func abyssDailyIdentifyKey(uid string) string {
	return "abyss_daily_identify_" + uid
}

func abyssDailyIdentifyAvailable(ctx context.Context, queryer abyssDailyIdentifyQuerier, uid string) (bool, error) {
	var available bool
	err := queryer.QueryRowContext(ctx,
		"SELECT NOT EXISTS(SELECT 1 FROM app_meta WHERE key=$1 AND value="+abyssDailyIdentifyDateSQL+")",
		abyssDailyIdentifyKey(uid),
	).Scan(&available)
	return available, err
}

// claimAbyssDailyIdentify atomically claims today's free identification. The
// caller owns the transaction, so a later item-write or commit failure also
// rolls the claim back.
func claimAbyssDailyIdentify(ctx context.Context, tx *sql.Tx, uid string) (bool, error) {
	var claimed bool
	err := tx.QueryRowContext(ctx, `INSERT INTO app_meta (key, value) VALUES ($1, `+abyssDailyIdentifyDateSQL+`)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value
		WHERE app_meta.value IS DISTINCT FROM EXCLUDED.value
		RETURNING TRUE`, abyssDailyIdentifyKey(uid)).Scan(&claimed)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return claimed, err
}

func (s *WebServer) dailyIdentifyCharge(
	w http.ResponseWriter,
	r *http.Request,
	tx *sql.Tx,
	uid string,
	normalCost int64,
	freeCredit int64,
) (int64, bool, bool) {
	free, err := claimAbyssDailyIdentify(r.Context(), tx, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return 0, false, false
	}
	cost := normalCost
	if free {
		cost = max(0, normalCost-freeCredit)
	}
	if quoted, present := s.quotedForgeGold(r, "identify", "identify_all"); present && quoted != cost {
		writeJSON(w, map[string]any{"ok": false, "error": errAbyssDailyIdentifyQuoteStale.Error()})
		return 0, false, false
	}
	return cost, free, true
}

func (s *WebServer) quotedForgeGold(r *http.Request, operations ...string) (int64, bool) {
	claims, err := s.verifyForgeClaims(r.Header.Get(abyssForgeQuoteHeader))
	if err != nil || claims.QuotedGold == nil {
		return 0, false
	}
	for _, operation := range operations {
		if claims.Operation == operation {
			return *claims.QuotedGold, true
		}
	}
	return 0, false
}
