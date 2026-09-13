package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// EconomyResetVersion is applied by the explicit maintenance command, never by
// ordinary bot startup. All application writers must be stopped first.
const EconomyResetVersion = 2

// ResetEconomy starts a new economy while preserving character progression.
// A full external pg_dump and a verified restore are prerequisites. A second,
// transactional snapshot preserves every affected row alongside the reset.
func ResetEconomy(ctx context.Context, database *sql.DB, scope string) (bool, error) {
	if scope != "economy" {
		return false, fmt.Errorf("unsupported reset scope %q (economy)", scope)
	}
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(2026091302)"); err != nil {
		return false, err
	}
	var done bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM app_meta WHERE key='economy_reset_applied_2')").Scan(&done); err != nil {
		return false, err
	}
	if done {
		return false, nil
	}
	// No runtime combat snapshot or deferred claim may outlive the economy.
	tables := []string{"users", "app_meta", "user_gear", "user_inventory", "user_materials", "user_consumables",
		"abyss_active", "abyss_escrow_loot", "abyss_combat_members", "abyss_combat_sessions", "abyss_pending_pet_captures",
		"auction_house", "abyss_material_orders", "abyss_consumable_trades", "abyss_duels", "abyss_wager_entries",
		"abyss_rescue_missions", "abyss_deaths", "arcade_jackpots", "abyss_shop_gifts", "abyss_vendor_buybacks",
		"abyss_pet_gifts", "abyss_economy_profiles", "abyss_guild_weekly_progress", "abyss_weekly_rivals", "user_pets"}
	quoted := make([]string, 0, len(tables))
	for _, table := range tables {
		quoted = append(quoted, pq.QuoteIdentifier(table))
	}
	if _, err := tx.ExecContext(ctx, "LOCK TABLE "+strings.Join(quoted, ",")+" IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, "CREATE SCHEMA economy_backup_2"); err != nil {
		return false, err
	}
	for _, table := range tables {
		q := pq.QuoteIdentifier(table)
		if _, err := tx.ExecContext(ctx, "CREATE TABLE economy_backup_2."+q+" AS TABLE "+q); err != nil {
			return false, fmt.Errorf("backup %s: %w", table, err)
		}
	}
	if err := SetEconomyContext(ctx, tx, "economy_reset", "reset-2", "", ""); err != nil {
		return false, err
	}
	steps := []string{
		`INSERT INTO app_meta(key,value) VALUES ('gold_economy_version','2'),('economy_started_at',CURRENT_TIMESTAMP::text)
		 ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`,
		`/* economy:db.ResetEconomy */ UPDATE users SET gold=0, abyss_tokens=0, abyss_boss_tokens=0, scrap_stack=0, vip_points=0,
		 abyss_day_gold=0, forge_undo=NULL, forge_undo_date=NULL, temper_fail_stacks=0`,
		`DELETE FROM abyss_combat_members`, `DELETE FROM abyss_combat_sessions`, `DELETE FROM abyss_escrow_loot`,
		`DELETE FROM abyss_active`, `DELETE FROM abyss_pending_pet_captures`,
		`DELETE FROM auction_house WHERE sold_at IS NULL`,
		`UPDATE abyss_material_orders SET remaining=0,escrow_gold=0,closed_at=NOW() WHERE closed_at IS NULL`,
		`UPDATE abyss_consumable_trades SET status='cancelled' WHERE status='open'`,
		`UPDATE abyss_duels SET status='declined',resolved_at=NOW() WHERE status='pending'`,
		`UPDATE abyss_wager_entries SET settled_at=NOW(),payout=0 WHERE settled_at IS NULL`,
		`DELETE FROM abyss_rescue_missions WHERE completed_at IS NULL`, `UPDATE abyss_deaths SET lost_cache=0`,
		`UPDATE arcade_jackpots SET amount=0,updated_at=NOW()`,
		`DELETE FROM abyss_shop_gifts WHERE claimed_at IS NULL`, `DELETE FROM abyss_vendor_buybacks`,
		`DELETE FROM abyss_pet_gifts`,
		`UPDATE abyss_economy_profiles SET potion_subscription=FALSE,repair_until=NULL,auto_insure=FALSE`,
		`UPDATE abyss_guild_weekly_progress SET reward_claimed_at=NOW() WHERE reward_claimed_at IS NULL`,
		`UPDATE abyss_weekly_rivals SET claimed_at=COALESCE(claimed_at,NOW())`,
		`UPDATE user_pets SET hp=GREATEST(0,LEAST(hp,GREATEST(1,max_hp))),max_hp=GREATEST(1,max_hp),
		 autoskills=autoskills-'combat_health'`,
	}
	if scope == "economy" {
		steps = append(steps, `DELETE FROM user_gear`, `DELETE FROM user_inventory`, `/* economy:db.ResetEconomy */ DELETE FROM user_materials`,
			`/* economy:db.ResetEconomy */ DELETE FROM user_consumables`, `/* economy:db.ResetEconomy */ UPDATE users SET artifact_name=NULL,artifact_mult=1,artifact_durability=0,
			title=NULL,title_mult=NULL,title_expires=NULL`)
	}
	for _, query := range steps {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return false, fmt.Errorf("reset statement %q: %w", query, err)
		}
	}
	prefixes := []string{"abyss_run_flags_", "abyss_deferred_event_", "abyss_echo_seed_", "abyss_cartographer_route_",
		"abyss_event_preview_", "abyss_next_event_depth_", "abyss_entry_setup_", "abyss_run_provenance_", "abyss_class_pending:",
		"abyss_raffle_pot_", "abyss_raffle_entry_", "abyss_free_insurance_", "abyss_temper_guard_"}
	if scope == "economy" {
		prefixes = append(prefixes, "abyss_jewels_", "abyss_sockets_", "abyss_mastery_shard_", "abyss_shop_loyalty_", "abyss_goblin_tokens_")
	}
	for _, prefix := range prefixes {
		if _, err := tx.ExecContext(ctx, "DELETE FROM app_meta WHERE LEFT(key,LENGTH($1))=$1", prefix); err != nil {
			return false, err
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM app_meta WHERE LEFT(key,LENGTH('abyss_forge_undo2_'))='abyss_forge_undo2_' AND RIGHT(key,LENGTH('_previous'))='_previous'"); err != nil {
		return false, err
	}
	// Free weekly skill-web respec becomes available again; talent respecs return
	// bound build credit, so retained allocations cannot restore old token wealth.
	if _, err := tx.ExecContext(ctx, "DELETE FROM app_meta WHERE LEFT(key,LENGTH('abyss_free_respec_'))='abyss_free_respec_'"); err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO app_meta(key,value) VALUES ('economy_reset_applied_2',
		jsonb_build_object('version',2,'scope',$1::text,'applied_at',CURRENT_TIMESTAMP,'backup_schema','economy_backup_2')::text)`, scope); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
