package bot

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssNextNaturalBossDepth(t *testing.T) {
	tests := []struct {
		name  string
		depth int
		want  int
	}{
		{name: "threshold", depth: 0, want: 5},
		{name: "before boss", depth: 4, want: 5},
		{name: "boss cleared", depth: 5, want: 10},
		{name: "deep run", depth: 62, want: 65},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := abyssNextNaturalBossDepth(tt.depth); got != tt.want {
				t.Fatalf("abyssNextNaturalBossDepth(%d) = %d, want %d", tt.depth, got, tt.want)
			}
		})
	}
}

func TestRecordAbyssBossKillSettlesContractAtomically(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO abyss_boss_kills").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT boss_contract_wager,boss_contract_depth").WithArgs("hunter").
		WillReturnRows(sqlmock.NewRows([]string{"boss_contract_wager", "boss_contract_depth"}).AddRow(int64(3), 50))
	mock.ExpectExec("UPDATE abyss_active SET boss_contract_wager=0").WithArgs("hunter").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE users SET abyss_boss_tokens").WithArgs(int64(7), "hunter").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	awarded, payout, _ := (&Bot{DB: database}).recordAbyssBossKillWithTokenRolls("hunter", "Abyssus", 50, time.Second, "hell", 1, 0)
	if !awarded || payout != 6 {
		t.Fatalf("boss rewards = awarded %v, payout %d", awarded, payout)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestForfeitAbyssBossContractReturnsCommittedStake(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery("WITH contract AS").WithArgs("hunter", 25).
		WillReturnRows(sqlmock.NewRows([]string{"boss_contract_wager"}).AddRow(int64(5)))

	if got := (&Bot{DB: database}).forfeitAbyssBossContract("hunter", 25); got != 5 {
		t.Fatalf("forfeited wager = %d, want 5", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossContractDeclarationIsAtomic(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT active.depth,active.boss_contract_wager,active.boss_contract_depth,users.current_hp").WithArgs("hunter").
		WillReturnRows(sqlmock.NewRows([]string{"depth", "wager", "target", "current_hp"}).AddRow(7, int64(0), 0, 100))
	mock.ExpectQuery("UPDATE users SET abyss_boss_tokens").WithArgs(int64(3), "hunter").
		WillReturnRows(sqlmock.NewRows([]string{"abyss_boss_tokens"}).AddRow(int64(8)))
	mock.ExpectExec("UPDATE abyss_active SET boss_contract_wager").WithArgs(int64(3), 10, "hunter").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest("POST", "/api/abyss/boss_contract", strings.NewReader(`{"wager":3}`))
	response := httptest.NewRecorder()

	server.handleAbyssBossContract(response, request, "hunter")
	body := response.Body.String()
	for _, token := range []string{`"ok":true`, `"target_depth":10`, `"payout":6`, `"boss_tokens":8`} {
		if !strings.Contains(body, token) {
			t.Errorf("response missing %s: %s", token, body)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossContractRejectsDownedDelver(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT active.depth,active.boss_contract_wager,active.boss_contract_depth,users.current_hp").WithArgs("hunter").
		WillReturnRows(sqlmock.NewRows([]string{"depth", "wager", "target", "current_hp"}).AddRow(7, int64(0), 0, 0))
	mock.ExpectRollback()
	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest("POST", "/api/abyss/boss_contract", strings.NewReader(`{"wager":1}`))
	response := httptest.NewRecorder()

	server.handleAbyssBossContract(response, request, "hunter")
	if body := response.Body.String(); !strings.Contains(body, "revive or concede") {
		t.Fatalf("response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossContractEvidence(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss_boss_contracts.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_boss_contracts.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"Boss Contract", "Stake 🏆 1", "Stake 🏆 3", "Stake 🏆 5", "/api/abyss/boss_contract", "forfeits the stake"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("boss contract UI is missing %q", token)
		}
	}
	if !strings.Contains(string(styles), ".ab-boss-contract-wagers") {
		t.Fatal("boss contract wager styles are missing")
	}
	migration, err := os.ReadFile(filepath.Join(abyssAAARepositoryRoot(t), "internal", "db", "migrations", "0089_abyss_boss_contracts.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"boss_contract_wager", "boss_contract_depth", "IN (1, 3, 5)"} {
		if !strings.Contains(string(migration), token) {
			t.Errorf("boss contract migration is missing %q", token)
		}
	}
}
