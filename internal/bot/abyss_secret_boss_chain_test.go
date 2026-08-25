package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssSecretBossChainRequiresEveryLoreFragment(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery("SELECT COUNT").WithArgs("hunter", 10).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(9))
	view := (&Bot{DB: database}).abyssSecretBossChain("hunter", abyssRun{Depth: 12})
	if view.Unlocked || view.LoreFound != 9 || view.LoreTotal != 10 {
		t.Fatalf("locked chain = %+v", view)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssSecretBossChainRestoresPersistentStage(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery("SELECT COUNT").WithArgs("hunter", 10).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectExec("INSERT INTO abyss_secret_boss_chains").WithArgs("hunter").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT stage FROM abyss_secret_boss_chains").WithArgs("hunter").WillReturnRows(sqlmock.NewRows([]string{"stage"}).AddRow(1))
	view := (&Bot{DB: database}).abyssSecretBossChain("hunter", abyssRun{Depth: 12})
	if !view.Unlocked || view.Completed || view.Stage != 1 || view.NextStage != 2 || view.NextDepth != 15 || view.Boss != "Mnemos, Keeper of Names" {
		t.Fatalf("restored chain = %+v", view)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssSecretBossChainFinalAdvanceIsAtomic(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE abyss_secret_boss_chains SET stage").WithArgs(3, 3, "hunter", 2).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO abyss_achievements").WithArgs("hunter", abyssSecretBossAchievement).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	next, completed, achievement := (&Bot{DB: database}).advanceAbyssSecretBossChain("hunter", 2)
	if next != 3 || !completed || achievement != "Abyss Unmasked (Secret Sovereigns)" {
		t.Fatalf("final advance = %d, %v, %q", next, completed, achievement)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssSecretBossReplacesOnlyNaturalBossFloors(t *testing.T) {
	if _, _, active := (&Bot{}).abyssSecretBossForFloor("hunter", 11, false); active {
		t.Fatal("ordinary floor activated the secret chain")
	}
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery("SELECT COUNT").WithArgs("hunter", 10).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	mock.ExpectExec("INSERT INTO abyss_secret_boss_chains").WithArgs("hunter").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT stage FROM abyss_secret_boss_chains").WithArgs("hunter").WillReturnRows(sqlmock.NewRows([]string{"stage"}).AddRow(2))
	def, stage, active := (&Bot{DB: database}).abyssSecretBossForFloor("hunter", 20, true)
	if !active || stage != 2 || def.Name != "The Abyss That Remembers" {
		t.Fatalf("secret encounter = %+v, stage %d, active %v", def, stage, active)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssSecretBossesEscalateAcrossTheChain(t *testing.T) {
	first := abyssSecretBossEncounter(abyssSecretBosses[0], 100, 2)[0]
	final := abyssSecretBossEncounter(abyssSecretBosses[2], 100, 2)[0]
	if first.Name == final.Name || final.Stats.HP <= first.Stats.HP || final.Stats.STR <= first.Stats.STR || final.RewardXP <= first.RewardXP {
		t.Fatalf("secret bosses do not escalate: first=%+v final=%+v", first, final)
	}
}

func TestAbyssSecretBossForecastUsesItsRealAffinity(t *testing.T) {
	run := abyssRun{Depth: 4}
	first := abyssBossAffinityForecastForSecret(run, time.Time{}, abyssSecretBossChainView{Unlocked: true, Stage: 0, NextDepth: 5})
	if first.Element != "Air" || first.WeakTo != "Fire" || first.StrongAgainst != "Earth" || first.TwinBosses || first.Neutral {
		t.Fatalf("first secret affinity = %+v", first)
	}
	final := abyssBossAffinityForecastForSecret(run, time.Time{}, abyssSecretBossChainView{Unlocked: true, Stage: 2, NextDepth: 5})
	if final.Element != "Physical" || !final.Neutral || final.WeakTo != "" || final.StrongAgainst != "" {
		t.Fatalf("final secret affinity = %+v", final)
	}
}

func TestAbyssSecretBossTollUsesOneLootRoll(t *testing.T) {
	single, rolls := abyssBossTollExpectedValueForRolls(100, 65, 1)
	twins, twinRolls := abyssBossTollExpectedValueForRolls(100, 65, 2)
	if rolls != 1 || twinRolls != 2 || single <= 0 || twins <= single {
		t.Fatalf("secret toll %d/%d rolls, normal toll %d/%d rolls", single, rolls, twins, twinRolls)
	}
}

func TestAbyssSecretBossChainAssetsAndMigration(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss_secret_boss_chain.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_secret_boss_chain.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"FORBIDDEN FOLIO", "Hidden sovereign", "next natural boss", "defeat or toll leaves this stage waiting", "Abyss Unmasked"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("secret-chain UI is missing %q", token)
		}
	}
	for _, token := range []string{".ab-secret-chain", ".ab-secret-chain.active", "@media (max-width: 680px)"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("secret-chain styles are missing %q", token)
		}
	}
	root := abyssAAARepositoryRoot(t)
	combat, err := os.ReadFile(filepath.Join(root, "internal", "bot", "web_abyss.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"abyssSecretBossForFloor", "abyssSecretBossEncounter", "advanceAbyssSecretBossChain", "SecretAchievement"} {
		if !strings.Contains(string(combat), token) {
			t.Errorf("combat pipeline is missing secret-chain hook %q", token)
		}
	}
	up, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0091_abyss_secret_boss_chain.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0091_abyss_secret_boss_chain.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"PRIMARY KEY", "CHECK (stage BETWEEN 0 AND 3)", "completed_at"} {
		if !strings.Contains(string(up), token) {
			t.Errorf("secret-chain migration is missing %q", token)
		}
	}
	if !strings.Contains(string(down), "DROP TABLE IF EXISTS abyss_secret_boss_chains") {
		t.Fatal("secret-chain migration is not reversible")
	}
}
