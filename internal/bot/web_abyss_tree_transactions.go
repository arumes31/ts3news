package bot

import (
	"context"
	"database/sql"
	"fmt"
)

type abyssTreeTransaction interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Commit() error
	Rollback() error
}

func commitAbyssTreeReplacement(ctx context.Context, tx abyssTreeTransaction, uid string, ids []int) error {
	rollback := func(err error) error {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM user_abyss_tree WHERE client_uid=$1", uid); err != nil {
		return rollback(fmt.Errorf("clear tree allocations: %w", err))
	}
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO user_abyss_tree (client_uid, node_id) VALUES ($1,$2) ON CONFLICT DO NOTHING",
			uid,
			id,
		); err != nil {
			return rollback(fmt.Errorf("allocate tree node %d: %w", id, err))
		}
	}
	if err := tx.Commit(); err != nil {
		return rollback(fmt.Errorf("commit tree replacement: %w", err))
	}
	return nil
}
