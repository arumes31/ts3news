package bot

import (
	"database/sql"
	"fmt"
)

// commitAbyssVictoryRunState keeps the visible escrow receipt and every
// server-owned run flag in one commit. A failed commit cannot pay a chest while
// leaving its quest active, or consume a quest without paying its reward.
func commitAbyssVictoryRunState(
	database *sql.DB,
	uid string,
	escrow int64,
	flags map[string]int64,
	rewards ...func(*sql.Tx) error,
) error {
	tx, err := database.Begin()
	if err != nil {
		return fmt.Errorf("begin victory state: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(
		"UPDATE abyss_active SET escrow=$1, floor_type='combat', modifier='', "+
			"event_state=NULL, last_action_at=NOW() WHERE client_uid=$2",
		escrow,
		uid,
	); err != nil {
		return fmt.Errorf("update victory escrow: %w", err)
	}
	if err := saveAbyssRunFlagsInTx(tx, uid, flags); err != nil {
		return fmt.Errorf("save victory flags: %w", err)
	}
	for _, reward := range rewards {
		if err := reward(tx); err != nil {
			return fmt.Errorf("save victory reward: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit victory state: %w", err)
	}
	return nil
}
