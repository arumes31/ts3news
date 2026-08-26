package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

const abyssForgeFloorType = "forge_floor"

func isAbyssForgeFloorOperation(operation string) bool {
	switch operation {
	case "temper", "punch_socket", "repair_all":
		return true
	default:
		return false
	}
}

func isAbyssForgeFloorState(raw string) bool {
	var state struct {
		Type string `json:"type"`
	}
	return json.Unmarshal([]byte(raw), &state) == nil && state.Type == abyssForgeFloorType
}

func (b *Bot) abyssForgeFloorAvailable(ctx context.Context, uid, operation string) bool {
	if !isAbyssForgeFloorOperation(operation) {
		return false
	}
	var raw string
	err := b.DB.QueryRowContext(
		ctx,
		"SELECT COALESCE(event_state::text, '') FROM abyss_active WHERE client_uid=$1 AND floor_type='event'",
		uid,
	).Scan(&raw)
	return err == nil && isAbyssForgeFloorState(raw)
}

func applyAbyssForgeFloorQuoteCost(
	active bool,
	cost abyssForgeQuoteCost,
	minimum abyssForgeQuoteCost,
	maximum abyssForgeQuoteCost,
) (abyssForgeQuoteCost, abyssForgeQuoteCost, abyssForgeQuoteCost) {
	if !active {
		return cost, minimum, maximum
	}
	return abyssForgeQuoteCost{Materials: map[string]int{}},
		abyssForgeQuoteCost{Materials: map[string]int{}},
		abyssForgeQuoteCost{Materials: map[string]int{}}
}

// claimAbyssForgeFloorInTx clears the Forge Floor only inside the forge
// mutation's transaction. Any later mutation or commit failure rolls the room
// back with the item and its costs, so the player never loses the free action.
func claimAbyssForgeFloorInTx(tx *sql.Tx, uid, operation string) (bool, error) {
	if !isAbyssForgeFloorOperation(operation) {
		return false, nil
	}
	var raw string
	err := tx.QueryRow(
		"SELECT COALESCE(event_state::text, '') FROM abyss_active WHERE client_uid=$1 AND floor_type='event' FOR UPDATE",
		uid,
	).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("lock forge floor: %w", err)
	}
	if !isAbyssForgeFloorState(raw) {
		return false, nil
	}
	result, err := tx.Exec(
		"UPDATE abyss_active SET event_state=NULL, last_action_at=NOW() WHERE client_uid=$1 AND floor_type='event'",
		uid,
	)
	if err != nil {
		return false, fmt.Errorf("consume forge floor: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count consumed forge floors: %w", err)
	}
	if updated != 1 {
		return false, fmt.Errorf("consume forge floor: updated %d rows", updated)
	}
	return true, nil
}
