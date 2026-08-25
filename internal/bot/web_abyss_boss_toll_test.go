package bot

import (
	"database/sql"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssBossTollExpectedValueUsesForecastAndTwinRolls(t *testing.T) {
	soloCost, soloRolls := abyssBossTollExpectedValue(100, 60)
	twinCost, twinRolls := abyssBossTollExpectedValue(100, 65)
	if soloRolls != 1 || twinRolls != 2 {
		t.Fatalf("loot rolls = solo %d, twins %d", soloRolls, twinRolls)
	}
	if soloCost <= 0 || soloCost%100 != 0 || twinCost <= soloCost {
		t.Fatalf("toll costs = solo %d, twins %d", soloCost, twinCost)
	}
}

func TestAbyssBossTollAvailabilityRequiresResolvedPreBossFloor(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery("SELECT boss_contract_wager").WithArgs("hunter", 5).
		WillReturnRows(sqlmock.NewRows([]string{"boss_contract_wager"}).AddRow(int64(3)))
	view := (&Bot{DB: database}).abyssBossToll("hunter", abyssRun{Active: true, Depth: 4, FloorType: "combat"}, 100, abyssSecretBossChainView{})
	if !view.Available || view.TargetDepth != 5 || view.ContractForfeit != 3 || view.Cost <= 0 {
		t.Fatalf("available toll = %+v", view)
	}
	blocked := (&Bot{DB: database}).abyssBossToll("hunter", abyssRun{Active: true, Depth: 4, FloorType: "event", EventState: `{}`}, 100, abyssSecretBossChainView{})
	if blocked.Available {
		t.Fatalf("unresolved event exposed toll: %+v", blocked)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossTollQuotesTheSecretReplacement(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	mock.ExpectQuery("SELECT boss_contract_wager").WithArgs("hunter", 65).
		WillReturnRows(sqlmock.NewRows([]string{"boss_contract_wager"}).AddRow(int64(0)))
	chain := abyssSecretBossChainView{Unlocked: true, Stage: 1, NextDepth: 65}
	view := (&Bot{DB: database}).abyssBossToll("hunter", abyssRun{Active: true, Depth: 64, FloorType: "combat"}, 100, chain)
	wantCost, _ := abyssBossTollExpectedValueForRolls(100, 65, 1)
	if view.Bosses != "Mnemos, Keeper of Names" || view.Rolls != 1 || view.Cost != wantCost {
		t.Fatalf("secret toll = %+v", view)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossTollCommitIsAtomicAndRewardFree(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	cost, _ := abyssBossTollExpectedValue(100, 5)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT active.depth,active.floor_type").WithArgs("hunter").
		WillReturnRows(sqlmock.NewRows([]string{"depth", "floor_type", "event_state", "pending", "wager", "contract_depth", "current_hp", "level"}).
			AddRow(4, "combat", nil, nil, int64(3), 5, 100, 100))
	mock.ExpectQuery("UPDATE users SET gold=gold").WithArgs(cost, 5, "hunter").
		WillReturnRows(sqlmock.NewRows([]string{"gold"}).AddRow(int64(900_000)))
	mock.ExpectExec("UPDATE abyss_active SET depth").WithArgs(5, "hunter").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT value FROM app_meta").WithArgs(abyssRunFlagsKey("hunter")).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO app_meta").WithArgs(abyssRunFlagsKey("hunter"), `{"death_wish":0,"def_momentum":0,"perfect_run":0}`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest("POST", "/api/abyss/boss_toll", strings.NewReader(`{"target_depth":5,"quoted_cost":`+itoa64(cost)+`}`))
	response := httptest.NewRecorder()

	server.handleAbyssBossToll(response, request, "hunter")
	body := response.Body.String()
	for _, token := range []string{`"ok":true`, `"depth":5`, `"contract_forfeit":3`, `"loot_rolls_skipped":1`, "No boss rewards were granted"} {
		if !strings.Contains(body, token) {
			t.Errorf("response missing %q: %s", token, body)
		}
	}
	for _, forbidden := range []string{`"loot":`, `"reward_xp":`, `"boss_tokens":`, `"bonus":`} {
		if strings.Contains(body, forbidden) {
			t.Errorf("reward-free response contains %q: %s", forbidden, body)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBossTollUIContract(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss_boss_toll.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_boss_toll.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"THE CHANCELLOR'S GATE", "No combat loot, XP, trophy, kill credit, streak, or floor reward", "quoted_cost", "confirmModal", "/api/abyss/boss_toll"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("boss toll UI is missing %q", token)
		}
	}
	for _, token := range []string{".ab-boss-toll", ".ab-boss-toll-gate", "@media (max-width: 680px)"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("boss toll styles are missing %q", token)
		}
	}
}
