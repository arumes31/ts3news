package bot

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssBossCosmeticRatesAndPoolsScaleByTier(t *testing.T) {
	previous := 0.0
	for _, tier := range abyssTierOrder {
		chance := abyssBossCosmeticDropChance(tier)
		if chance <= previous {
			t.Fatalf("%s chance %.2f does not exceed %.2f", tier, chance, previous)
		}
		previous = chance
	}
	normal, ok := abyssRollBossCosmetic("normal", 0, .99)
	if !ok || abyssBossCosmeticTierRank(normal.MinTier) != 0 {
		t.Fatalf("normal pool leaked a higher-tier cosmetic: %+v", normal)
	}
	insanity, ok := abyssRollBossCosmetic("insanity", 0, .99)
	if !ok || insanity.MinTier != "insanity" {
		t.Fatalf("insanity pool cannot reach its exclusive: %+v", insanity)
	}
	if _, ok := abyssRollBossCosmetic("insanity", .12, 0); ok {
		t.Fatal("roll at the exclusive upper bound produced a drop")
	}
}

func TestAbyssBossCosmeticGrantSharesBossRewardTransaction(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO abyss_boss_kills").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT boss_contract_wager,boss_contract_depth").WithArgs("hunter").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("UPDATE users SET abyss_boss_tokens").WithArgs(int64(1), "hunter").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO abyss_shop_cosmetics").WithArgs("hunter", "boss_banner_crownless").WillReturnError(sql.ErrConnDone)
	mock.ExpectRollback()
	awarded, _, cosmetic := (&Bot{DB: database}).recordAbyssBossKillWithTokenRolls("hunter", "Abyssus", 100, time.Second, "insanity", 0, .99)
	if awarded || cosmetic != "" {
		t.Fatal("cosmetic escaped a failed boss reward transaction")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossCosmeticDropCommitsWithBossRecord(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO abyss_boss_kills").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT boss_contract_wager,boss_contract_depth").WithArgs("hunter").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("UPDATE users SET abyss_boss_tokens").WithArgs(int64(1), "hunter").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO abyss_shop_cosmetics").WithArgs("hunter", "boss_banner_crownless").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	awarded, payout, cosmetic := (&Bot{DB: database}).recordAbyssBossKillWithTokenRolls("hunter", "Abyssus", 100, time.Second, "insanity", 0, .99)
	if !awarded || payout != 0 || cosmetic != "Crownless Banner" {
		t.Fatalf("boss cosmetic reward = %v, %d, %q", awarded, payout, cosmetic)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestGrantAbyssBossCosmeticIsDuplicateSafe(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO abyss_shop_cosmetics").WithArgs("hunter", "boss_banner_crownless").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	_, granted, err := grantAbyssBossCosmeticTx(tx, "hunter", "insanity", 0, .99)
	if err != nil || granted {
		t.Fatalf("duplicate grant = %v, %v", granted, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossCosmeticEquipRequiresOwnership(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectExec("INSERT INTO abyss_boss_cosmetic_loadouts").WithArgs("hunter", "boss_mount_bone_runner").WillReturnResult(sqlmock.NewResult(0, 0))
	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/boss_cosmetic/equip", strings.NewReader(`{"key":"boss_mount_bone_runner"}`))
	response := httptest.NewRecorder()
	server.handleAbyssBossCosmeticEquip(response, request, "hunter")
	if !strings.Contains(response.Body.String(), "cosmetic not owned") {
		t.Fatalf("equip response = %s", response.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossCosmeticAssetsAndMigration(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss_boss_cosmetics.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_boss_cosmetics.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"SOVEREIGN SPOILS", "Cosmetic only", "boss_cosmetic/equip", "No item changes stats"} {
		if !strings.Contains(strings.ToLower(string(page)), strings.ToLower(token)) {
			t.Errorf("boss cosmetic UI is missing %q", token)
		}
	}
	for _, token := range []string{".ab-boss-cosmetics", ".ab-cosmetic-grid", "@media (max-width: 560px)"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("boss cosmetic styles are missing %q", token)
		}
	}
	root := abyssAAARepositoryRoot(t)
	up, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0092_abyss_boss_cosmetic_loadouts.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0092_abyss_boss_cosmetic_loadouts.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"PRIMARY KEY", "mount_key", "banner_key"} {
		if !strings.Contains(string(up), token) {
			t.Errorf("cosmetic loadout migration is missing %q", token)
		}
	}
	if !strings.Contains(string(down), "DROP TABLE IF EXISTS abyss_boss_cosmetic_loadouts") {
		t.Fatal("cosmetic loadout migration is not reversible")
	}
}
