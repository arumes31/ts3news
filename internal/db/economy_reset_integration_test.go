//go:build integration

package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
)

// The reset creates a fixed backup schema, so each test owns an entire disposable
// database. Applying every embedded migration catches constraints and columns
// that the small ledger fixture intentionally does not model.
func fullEconomyResetTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("GOLD_RESET_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set GOLD_RESET_TEST_DATABASE_URL to disposable PostgreSQL with CREATE DATABASE permission")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		t.Fatal("test database URL must be a PostgreSQL URL")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	name := fmt.Sprintf("economy_reset_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + pq.QuoteIdentifier(name) + " TEMPLATE template0"); err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	var database *sql.DB
	t.Cleanup(func() {
		if database != nil {
			_ = database.Close()
		}
		if _, err := admin.Exec("DROP DATABASE " + pq.QuoteIdentifier(name) + " WITH (FORCE)"); err != nil {
			t.Error(err)
		}
		_ = admin.Close()
	})
	parsed.Path = "/" + name
	query := parsed.Query()
	query.Del("search_path")
	parsed.RawQuery = query.Encode()
	database, err = sql.Open("postgres", parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(database); err != nil {
		t.Fatal(err)
	}
	if got := goldResetScalar(t, database, "SELECT version::text FROM schema_migrations WHERE dirty=FALSE"); got != "105" {
		t.Fatalf("full schema version=%s, want105", got)
	}
	return database
}

func seedFullEconomyReset(t *testing.T, database *sql.DB) {
	t.Helper()
	_, err := database.Exec(`
 INSERT INTO app_meta(key,value) VALUES ('gold_economy_version','1'),('economy_deployed_revision','integration-build'),
 ('abyss_talents_owner','{"phantom_footwork":3}'),('abyss_tree_owner','[1,2,3]'),('abyss_class_owner','rogue'),
 ('abyss_run_flags_owner','{"vault_keys":9,"run_bounty_reward":9007199254740993}'),
 ('abyss_deferred_event_owner','{"gold":999999}'),('abyss_echo_seed_owner','123456'),
 ('abyss_forge_undo2_owner_previous','{"gear":{"id":"old-sword"}}'),
 ('abyss_free_respec_owner','2026-W37'),('abyss_jewels_owner','{"power":999}'),
 ('abyss_sockets_owner','{"1":"power"}'),('abyss_shop_loyalty_owner','900'),
 ('abyss_mastery_shard_owner','999'),('abyss_goblin_tokens_owner','888'),
 ('abyss_temper_guard_owner','1'),('abyss_free_insurance_owner','1'),
 ('abyssXecho_seed_owner','boundary-must-survive'),('abyss_combat_archive_old','{"victory":true}');
 INSERT INTO users(client_uid,nickname,xp,level,prestige,gold,abyss_tokens,abyss_boss_tokens,scrap_stack,vip_points,
 abyss_best_depth,artifact_name,artifact_mult,artifact_durability,forge_undo,forge_undo_date,temper_fail_stacks,abyss_talent_credit)
 VALUES ('owner','Owner',987654,321,7,9007199254740993,111711935066,5,21,2141229522,
 400,'Old artifact',3.5,9,'{"gear_id":"old-sword"}',CURRENT_DATE,17,123),
 ('peer','Peer',12345,50,2,321,43,2,7,99999,30,NULL,1,0,NULL,NULL,0,0);
 INSERT INTO user_skills(client_uid,slot,skill_id) VALUES ('owner',1,'S_EQ');
 INSERT INTO user_gear(client_uid,slot,gear_id,durability) VALUES ('owner','MainHand','old-sword',50),('owner','Pet1','old-pet-sword',50);
 INSERT INTO user_inventory(client_uid,gear_id,durability) VALUES ('owner','old-spare',10);
 INSERT INTO user_materials(client_uid,mat_id,count) VALUES ('owner','dust',900),('peer','core',8);
 INSERT INTO user_consumables(client_uid,cons_id,remaining_fights) VALUES ('owner','potion',7),('peer','elixir',3);
 INSERT INTO user_pets(client_uid,name,mob_type,level,hp,max_hp,str,def,spd,autoskills)
 VALUES ('owner','Retained pet','beast',42,99999,100,30,20,10,'{"heal":true,"combat_health":{"hp":99999,"max_hp":99999}}');
 INSERT INTO abyss_pet_gifts(code,pet_id,sender_uid,recipient_uid) SELECT 'old-pet-gift',pet_id,'owner','peer' FROM user_pets;
 INSERT INTO abyss_pending_pet_captures(client_uid,name,mob_type,level,hp,max_hp,str,def,spd,loyalty)
 VALUES ('peer','Unclaimed old pet','beast',2,10,10,1,1,1,50);
 INSERT INTO abyss_active(client_uid,depth,escrow,economy_loan_fee,economy_loan_principal)
 VALUES ('owner',200,9007199254740993,50,1000);
 INSERT INTO abyss_escrow_loot(client_uid,item_type,label,item_data) VALUES
 ('owner','gold','Old gold','{"type":"gold","gold":999999}'),
 ('owner','gear','Old gear','{"type":"gear","gear":{"id":"old-escrow"}}');
 INSERT INTO abyss_combat_sessions(session_id,owner_uid,depth,phase,state) VALUES ('old-fight','owner',200,'paused','{"gold":999999}');
 INSERT INTO abyss_combat_members(session_id,client_uid) VALUES ('old-fight','owner');
 INSERT INTO auction_house(seller_uid,item_type,item_id,item_name,price,expires_at,current_bid,bidder_uid)
 VALUES ('owner','gear','old-listed','Old listed gear',500,NOW()+INTERVAL '1 day',400,'peer');
 INSERT INTO auction_house(seller_uid,item_type,item_id,item_name,price,expires_at,listed_at,sold_at,buyer_uid)
 VALUES ('owner','gear','historic-sale','Historic sale',100,NOW()+INTERVAL '1 day',NOW()-INTERVAL '1 day',NOW()-INTERVAL '1 hour','peer');
 INSERT INTO abyss_material_orders(buyer_uid,material,unit_price,remaining,escrow_gold) VALUES ('owner','dust',100,9,900);
 INSERT INTO abyss_consumable_trades(sender_uid,recipient_uid,offer_cons_id,offer_quantity,request_cons_id,request_quantity)
 VALUES ('owner','peer','reserved-potion',9,'elixir',2);
 INSERT INTO abyss_duels(challenger_uid,opponent_uid,wager_tokens) VALUES ('owner','peer',75);
 INSERT INTO abyss_wager_entries(week_key,bracket,client_uid,entry_fee) VALUES ('2026-W37',100,'owner',100);
 INSERT INTO abyss_deaths(client_uid,depth,killer_name,killer_family,lost_cache) VALUES ('owner',30,'Ogre','beast',9999);
 INSERT INTO abyss_rescue_missions(death_id,rescuer_uid,owner_uid,depth,lost_cache) SELECT id,'peer','owner',30,9999 FROM abyss_deaths;
 INSERT INTO arcade_jackpots(game_key,amount) VALUES ('reset-test',50000);
 INSERT INTO abyss_shop_gifts(code,sender_uid,recipient_uid,item_key) VALUES ('old-gift','owner','peer','potion');
 INSERT INTO abyss_vendor_buybacks(client_uid,gear_id,durability,acquired_at,sale_value,buyback_cost) VALUES ('owner','buyback-gear',30,NOW(),100,125);
 INSERT INTO abyss_economy_profiles(client_uid,potion_subscription,repair_until,auto_insure) VALUES ('owner',TRUE,NOW()+INTERVAL '1 day',TRUE);
 INSERT INTO abyss_weekly_rivals(week_key,client_uid,rival_uid,target_depth) VALUES ('2026-W37','owner','peer',100);
 INSERT INTO abyss_guilds(name,tag,owner_uid,invite_code) VALUES ('Retained Guild','TST','owner','test-guild');
 INSERT INTO abyss_guild_weekly_progress(guild_id,week_key,floors,goal) SELECT guild_id,'2026-W37',500,500 FROM abyss_guilds;
 INSERT INTO abyss_runs(client_uid,depth,gold_banked,victory) VALUES ('owner',300,987654,TRUE);
 INSERT INTO battle_history(client_uid,mob_name,victory,gold_won) VALUES ('owner','Historic monster',TRUE,777);
 `)
	if err != nil {
		t.Fatal(err)
	}
}

var economyResetSnapshotTables = []string{"users", "app_meta", "user_gear", "user_inventory", "user_materials", "user_consumables",
	"abyss_active", "abyss_escrow_loot", "abyss_combat_members", "abyss_combat_sessions", "abyss_pending_pet_captures",
	"auction_house", "abyss_material_orders", "abyss_consumable_trades", "abyss_duels", "abyss_wager_entries",
	"abyss_rescue_missions", "abyss_deaths", "arcade_jackpots", "abyss_shop_gifts", "abyss_vendor_buybacks",
	"abyss_pet_gifts", "abyss_economy_profiles", "abyss_guild_weekly_progress", "abyss_weekly_rivals", "user_pets"}

func economyResetSnapshot(t *testing.T, database *sql.DB, schema string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, table := range economyResetSnapshotTables {
		query := "SELECT md5(COALESCE(jsonb_agg(to_jsonb(row) ORDER BY to_jsonb(row)::text)::text,'[]')) FROM " + pq.QuoteIdentifier(schema) + "." + pq.QuoteIdentifier(table) + " row"
		out[table] = goldResetScalar(t, database, query)
	}
	return out
}

func TestEconomyResetIntegrationFullSchemaPreservesProgression(t *testing.T) {
	database := fullEconomyResetTestDatabase(t)
	seedFullEconomyReset(t, database)
	before := economyResetSnapshot(t, database, "public")
	applied, err := ResetEconomy(context.Background(), database, "economy")
	if err != nil || !applied {
		t.Fatalf("reset=%v,%v", applied, err)
	}
	backup := economyResetSnapshot(t, database, "economy_backup_2")
	for table, want := range before {
		if backup[table] != want {
			t.Errorf("backup %s differs from original", table)
		}
	}
	checks := map[string]string{
		`SELECT SUM(gold+abyss_tokens+abyss_boss_tokens+scrap_stack+vip_points)::text FROM users`:                                                                           "0",
		`SELECT xp||'/'||level||'/'||prestige||'/'||abyss_best_depth||'/'||abyss_talent_credit FROM users WHERE client_uid='owner'`:                                         "987654/321/7/400/123",
		`SELECT COUNT(*)::text FROM users WHERE artifact_name IS NOT NULL OR artifact_mult<>1 OR artifact_durability<>0 OR forge_undo IS NOT NULL OR temper_fail_stacks<>0`: "0",
		`SELECT skill_id FROM user_skills WHERE client_uid='owner'`:                                                                                                         "S_EQ",
		`SELECT value FROM app_meta WHERE key='abyss_talents_owner'`:                                                                                                        `{"phantom_footwork":3}`,
		`SELECT value FROM app_meta WHERE key='abyss_tree_owner'`:                                                                                                           "[1,2,3]",
		`SELECT value FROM app_meta WHERE key='abyss_class_owner'`:                                                                                                          "rogue",
		`SELECT value FROM app_meta WHERE key='abyssXecho_seed_owner'`:                                                                                                      "boundary-must-survive",
		`SELECT value FROM app_meta WHERE key='abyss_combat_archive_old'`:                                                                                                   `{"victory":true}`,
		`SELECT name||'/'||level||'/'||hp||'/'||max_hp||'/'||(autoskills->>'heal') FROM user_pets`:                                                                          "Retained pet/42/100/100/true",
		`SELECT COUNT(*)::text FROM user_pets WHERE autoskills?'combat_health'`:                                                                                             "0",
		`SELECT COUNT(*)::text FROM app_meta WHERE key IN ('abyss_run_flags_owner','abyss_deferred_event_owner','abyss_echo_seed_owner','abyss_forge_undo2_owner_previous','abyss_free_respec_owner','abyss_jewels_owner','abyss_sockets_owner','abyss_shop_loyalty_owner','abyss_mastery_shard_owner','abyss_goblin_tokens_owner','abyss_temper_guard_owner','abyss_free_insurance_owner')`: "0",
		`SELECT COUNT(*)::text FROM auction_house WHERE sold_at IS NULL`:                                                         "0",
		`SELECT item_id FROM auction_house`:                                                                                      "historic-sale",
		`SELECT remaining||'/'||escrow_gold||'/'||(closed_at IS NOT NULL) FROM abyss_material_orders`:                            "0/0/true",
		`SELECT status FROM abyss_consumable_trades`:                                                                             "cancelled",
		`SELECT status||'/'||(resolved_at IS NOT NULL) FROM abyss_duels`:                                                         "declined/true",
		`SELECT payout||'/'||(settled_at IS NOT NULL) FROM abyss_wager_entries`:                                                  "0/true",
		`SELECT SUM(lost_cache)::text FROM abyss_deaths`:                                                                         "0",
		`SELECT SUM(amount)::text FROM arcade_jackpots`:                                                                          "0",
		`SELECT COUNT(*)::text FROM abyss_economy_profiles WHERE potion_subscription OR auto_insure OR repair_until IS NOT NULL`: "0",
		`SELECT COUNT(*)::text FROM abyss_weekly_rivals WHERE claimed_at IS NULL`:                                                "0",
		`SELECT COUNT(*)::text FROM abyss_guild_weekly_progress WHERE reward_claimed_at IS NULL`:                                 "0",
		`SELECT gold_banked::text FROM abyss_runs`:                                                                               "987654",
		`SELECT gold_won::text FROM battle_history`:                                                                              "777",
		`SELECT COUNT(*)::text FROM abyss_guilds`:                                                                                "1",
		`SELECT COUNT(DISTINCT transaction_id)::text FROM economy_ledger WHERE source='economy_reset'`:                           "1",
		`SELECT COUNT(*)::text FROM economy_ledger WHERE source='economy_reset' AND (request_id<>'reset-2' OR economy_epoch<>'2' OR revision<>'integration-build')`:                     "0",
		`SELECT SUM(delta)::text FROM economy_ledger WHERE source='economy_reset' AND currency='gold'`:                                                                                  "-9007199254741314",
		`SELECT COUNT(*)::text FROM (SELECT currency FROM economy_ledger GROUP BY currency HAVING SUM(delta)<>CASE WHEN currency='abyss_talent_credit' THEN 123 ELSE 0 END) mismatches`: "0",
		`SELECT value FROM app_meta WHERE key='gold_economy_version'`:                                                                                                                   "2",
		`SELECT value::jsonb->>'scope' FROM app_meta WHERE key='economy_reset_applied_2'`:                                                                                               "economy",
	}
	for _, table := range []string{"user_gear", "user_inventory", "user_materials", "user_consumables", "abyss_active", "abyss_escrow_loot", "abyss_combat_sessions", "abyss_combat_members", "abyss_pending_pet_captures", "abyss_rescue_missions", "abyss_shop_gifts", "abyss_vendor_buybacks", "abyss_pet_gifts"} {
		checks["SELECT COUNT(*)::text FROM "+pq.QuoteIdentifier(table)] = "0"
	}
	for query, want := range checks {
		if got := goldResetScalar(t, database, query); got != want {
			t.Errorf("%s: got%q want%q", query, got, want)
		}
	}
	if _, err := database.Exec("UPDATE users SET gold=17 WHERE client_uid='owner'"); err != nil {
		t.Fatal(err)
	}
	if applied, err := ResetEconomy(context.Background(), database, "economy"); err != nil || applied {
		t.Fatalf("repeat=%v,%v", applied, err)
	}
	if got := goldResetScalar(t, database, "SELECT gold::text FROM users WHERE client_uid='owner'"); got != "17" {
		t.Fatal("repeat erased new gold")
	}
	repeatedBackup := economyResetSnapshot(t, database, "economy_backup_2")
	for table, want := range before {
		if got := repeatedBackup[table]; got != want {
			t.Errorf("repeat modified backup %s", table)
		}
	}
}

func TestEconomyResetIntegrationLateFailureRollsBackEverything(t *testing.T) {
	database := fullEconomyResetTestDatabase(t)
	seedFullEconomyReset(t, database)
	before := economyResetSnapshot(t, database, "public")
	_, err := database.Exec(`CREATE FUNCTION reject_test_reset() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected reset failure'; END $$;
 CREATE TRIGGER reject_test_reset BEFORE DELETE ON user_consumables FOR EACH ROW EXECUTE FUNCTION reject_test_reset();`)
	if err != nil {
		t.Fatal(err)
	}
	applied, err := ResetEconomy(context.Background(), database, "economy")
	if err == nil || applied || !strings.Contains(err.Error(), "injected reset failure") {
		t.Fatalf("injected failure=%v,%v", applied, err)
	}
	after := economyResetSnapshot(t, database, "public")
	for table, want := range before {
		if after[table] != want {
			t.Errorf("failure partially changed %s", table)
		}
	}
	for query, want := range map[string]string{
		`SELECT COUNT(*)::text FROM pg_namespace WHERE nspname='economy_backup_2'`: "0",
		`SELECT COUNT(*)::text FROM app_meta WHERE key='economy_reset_applied_2'`:  "0",
		`SELECT COUNT(*)::text FROM economy_ledger WHERE source='economy_reset'`:   "0",
	} {
		if got := goldResetScalar(t, database, query); got != want {
			t.Errorf("%s=%s want%s", query, got, want)
		}
	}
	if _, err := database.Exec("DROP TRIGGER reject_test_reset ON user_consumables"); err != nil {
		t.Fatal(err)
	}
	if applied, err := ResetEconomy(context.Background(), database, "economy"); err != nil || !applied {
		t.Fatalf("retry=%v,%v", applied, err)
	}
}
