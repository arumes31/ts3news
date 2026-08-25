package bot

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssForgeFloorOperationAndStateValidation(t *testing.T) {
	t.Parallel()

	for _, operation := range []string{"temper", "punch_socket", "repair_all"} {
		if !isAbyssForgeFloorOperation(operation) {
			t.Errorf("eligible operation %q was rejected", operation)
		}
	}
	for _, operation := range []string{"", "temper_surge", "socket_gem", "polish"} {
		if isAbyssForgeFloorOperation(operation) {
			t.Errorf("ineligible operation %q was accepted", operation)
		}
	}
	if !isAbyssForgeFloorState(`{"type":"forge_floor"}`) {
		t.Fatal("canonical forge floor state was rejected")
	}
	for _, raw := range []string{"", `{"type":"merchant"}`, `{"type":`, `null`} {
		if isAbyssForgeFloorState(raw) {
			t.Errorf("invalid forge floor state %q was accepted", raw)
		}
	}
}

func TestAbyssForgeFloorAvailabilityUsesAuthoritativeRun(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectQuery("SELECT COALESCE\\(event_state::text, ''\\) FROM abyss_active").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"event_state"}).AddRow(`{"type":"forge_floor"}`))
	bot := &Bot{DB: database}
	if !bot.abyssForgeFloorAvailable(context.Background(), "user", "temper") {
		t.Fatal("active forge floor was not available")
	}
	if bot.abyssForgeFloorAvailable(context.Background(), "user", "awaken") {
		t.Fatal("forge floor covered an ineligible operation")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestApplyAbyssForgeFloorQuoteCost(t *testing.T) {
	t.Parallel()

	cost := abyssForgeQuoteCost{Gold: 400, Tokens: 2, Materials: map[string]int{"shard": 10}}
	minimum := abyssForgeQuoteCost{Gold: 200, Materials: map[string]int{"dust": 1}}
	maximum := abyssForgeQuoteCost{Gold: 800, Materials: map[string]int{"core": 2}}
	got, gotMinimum, gotMaximum := applyAbyssForgeFloorQuoteCost(true, cost, minimum, maximum)
	for label, quoteCost := range map[string]abyssForgeQuoteCost{
		"cost": got, "minimum": gotMinimum, "maximum": gotMaximum,
	} {
		if quoteCost.Gold != 0 || quoteCost.Tokens != 0 || len(quoteCost.Materials) != 0 || quoteCost.Materials == nil {
			t.Errorf("free %s = %#v", label, quoteCost)
		}
	}
	unchanged, _, _ := applyAbyssForgeFloorQuoteCost(false, cost, minimum, maximum)
	if unchanged.Gold != cost.Gold || unchanged.Tokens != cost.Tokens || unchanged.Materials["shard"] != 10 {
		t.Fatalf("inactive forge floor changed quote cost: %#v", unchanged)
	}
}

func TestClaimAbyssForgeFloorIsTransactional(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT COALESCE\\(event_state::text, ''\\) FROM abyss_active").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"event_state"}).AddRow(`{"type":"forge_floor"}`))
	mock.ExpectExec("UPDATE abyss_active SET event_state=NULL").
		WithArgs("user").
		WillReturnResult(sqlmock.NewResult(0, 1))
	used, err := claimAbyssForgeFloorInTx(tx, "user", "punch_socket")
	if err != nil || !used {
		t.Fatalf("claim = %v, %v", used, err)
	}
	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestClaimAbyssForgeFloorPreservesRoomOnMutationError(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	tx, err := database.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT COALESCE\\(event_state::text, ''\\) FROM abyss_active").
		WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"event_state"}).AddRow(`{"type":"forge_floor"}`))
	mock.ExpectExec("UPDATE abyss_active SET event_state=NULL").
		WithArgs("user").
		WillReturnError(errors.New("write failed"))
	used, err := claimAbyssForgeFloorInTx(tx, "user", "repair_all")
	if err == nil || used {
		t.Fatalf("failed claim = %v, %v", used, err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssForgeFloorHandlerAndGUIContracts(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_forge_experience.css")
	if err != nil {
		t.Fatal(err)
	}
	features, err := os.ReadFile("web_abyss_features.go")
	if err != nil {
		t.Fatal(err)
	}
	forge, err := os.ReadFile("web_abyss_forge2.go")
	if err != nil {
		t.Fatal(err)
	}
	quotes, err := os.ReadFile("web_abyss_forge_quotes.go")
	if err != nil {
		t.Fatal(err)
	}
	joined := string(page) + string(styles) + string(features) + string(forge) + string(quotes)
	for _, token := range []string{
		"Silent Anvil charge ready", "openAbyssForgeFloor", "repairable", "forge_floor_used",
		`claimAbyssForgeFloorInTx(tx, uid, "temper")`,
		`claimAbyssForgeFloorInTx(tx, uid, "punch_socket")`,
		`claimAbyssForgeFloorInTx(tx, uid, "repair_all")`,
		"The active Silent Anvil makes one temper, socket punch, or full repair free",
		"forgeQuoteRequiresFloor",
	} {
		if !strings.Contains(joined, token) {
			t.Errorf("forge floor contract is missing %q", token)
		}
	}
}
