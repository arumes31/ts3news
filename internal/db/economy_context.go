package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// SetEconomyContext annotates balance changes made by tx. PostgreSQL transaction-
// local settings cannot leak into the next request on a pooled connection.
// Identifiers are opaque correlation IDs; never pass tokens, URLs or query text.
func SetEconomyContext(ctx context.Context, tx *sql.Tx, source, requestID, runID, itemID string) error {
	if tx == nil {
		return fmt.Errorf("economy context requires a transaction")
	}
	for name, value := range map[string]string{"source": source, "request_id": requestID, "run_id": runID, "item_id": itemID} {
		if len(value) > 128 || strings.ContainsAny(value, "\x00\r\n") || name == "source" && strings.TrimSpace(value) == "" {
			return fmt.Errorf("invalid economy %s", name)
		}
	}
	_, err := tx.ExecContext(ctx, `SELECT set_config('ts3news.economy_source',$1,true),
		set_config('ts3news.economy_request_id',$2,true),
		set_config('ts3news.economy_run_id',$3,true),
		set_config('ts3news.economy_item_id',$4,true)`, source, requestID, runID, itemID)
	if err != nil {
		return fmt.Errorf("setting economy context: %w", err)
	}
	return nil
}
