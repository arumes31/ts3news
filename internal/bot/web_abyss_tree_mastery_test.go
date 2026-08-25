package bot

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssMasteryRankMapsFailClosed(t *testing.T) {
	t.Parallel()

	ranks, err := decodeAbyssRankMap(`{"might":3}`, abyssParagonMaxRank)
	if err != nil || ranks["might"] != 3 {
		t.Fatalf("valid ranks = %v, %v", ranks, err)
	}
	for _, stored := range []string{`{"might":-1}`, `{"might":21}`, `{`} {
		if _, err := decodeAbyssRankMap(stored, abyssParagonMaxRank); err == nil {
			t.Errorf("invalid ranks accepted: %s", stored)
		}
	}
}

func TestAbyssBestiaryTalentCostsAreCumulative(t *testing.T) {
	t.Parallel()

	if abyssBestiaryKillsSpent(0) != 0 || abyssBestiaryKillsSpent(1) != 10 || abyssBestiaryKillsSpent(2) != 30 || abyssBestiaryKillsSpent(5) != 150 {
		t.Fatal("bestiary kill spending must accumulate 10, 20, 30, 40, 50")
	}
	if spent := abyssBestiaryRanksSpent(map[string]int{"Boss": 2, "Elite": 1}); spent != 40 {
		t.Fatalf("shared boss-kill spend = %d, want 40", spent)
	}
	if abyssFullSetCount(map[string]int{"predator": 6, "warden": 5, "harvester": 12}) != 2 {
		t.Fatal("set mastery must count complete six-piece sets, not partial tiers")
	}
}

func TestAbyssBestiaryTalentDamageUsesFightSnapshot(t *testing.T) {
	t.Parallel()

	user := &UserInCombat{
		killerExp: map[string]int{string(content.MobBoss): 20},
		treeBonus: content.TreeBonus{Pct: map[string]float64{"bestiary_damage_boss": 0.05}},
	}
	mob := &content.Mob{Type: content.MobBoss}
	if got := abyssKillerDamage(1_000, user, mob); got != 1_071 {
		t.Fatalf("family mastery damage = %d, want 1071", got)
	}
}

func TestAbyssBestiaryTalentViewsShareBossKillBudget(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectQuery("SELECT value FROM app_meta").
		WithArgs(abyssBestiaryTalentsKey("hunter")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"Boss":2}`))
	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(kills\\),0\\) FROM abyss_bestiary").
		WithArgs("hunter", string(content.MobBoss)).
		WillReturnRows(sqlmock.NewRows([]string{"kills"}).AddRow(44))

	views := (&Bot{DB: database}).abyssBestiaryTalentViews("hunter")
	if len(views) != len(abyssBestiaryTalentCatalog) {
		t.Fatalf("talent views = %d, want %d", len(views), len(abyssBestiaryTalentCatalog))
	}
	for _, view := range views {
		if view.Kills != 44 || view.Spent != 30 || view.Available != 14 {
			t.Fatalf("shared boss-kill view = %#v", view)
		}
		if view.Family == "Boss" && view.Rank != 2 {
			t.Fatalf("boss talent view = %#v", view)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssParagonAllocationPersistsOneRank(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT abyss_prestige FROM users").
		WithArgs("paragon-player").
		WillReturnRows(sqlmock.NewRows([]string{"abyss_prestige"}).AddRow(1))
	mock.ExpectQuery("SELECT value FROM app_meta").
		WithArgs(abyssParagonKey("paragon-player")).WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("INSERT INTO app_meta").
		WithArgs(abyssParagonKey("paragon-player"), `{"might":1}`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT abyss_prestige FROM users").
		WithArgs("paragon-player").
		WillReturnRows(sqlmock.NewRows([]string{"abyss_prestige"}).AddRow(1))
	mock.ExpectQuery("SELECT value FROM app_meta").
		WithArgs(abyssParagonKey("paragon-player")).
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"might":1}`))

	server := &WebServer{bot: &Bot{DB: database}}
	request := httptest.NewRequest(http.MethodPost, "/api/abyss/tree/paragon", strings.NewReader(`{"key":"might"}`))
	response := httptest.NewRecorder()
	server.handleAbyssTreeParagon(response, request, "paragon-player")
	if body := response.Body.String(); !strings.Contains(body, `"ok":true`) || !strings.Contains(body, `"rank":1`) {
		t.Fatalf("paragon response = %s", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssProgressionTrancheControls(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abysstree.html")
	if err != nil {
		t.Fatal(err)
	}
	mastery, err := webAssets.ReadFile("webassets/abysstree_mastery.html")
	if err != nil {
		t.Fatal(err)
	}
	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(page) + string(mastery) + string(routes)
	for _, required := range []string{
		"/api/abyss/tree/paragon", "/api/abyss/tree/bestiary_talent", "Paragon hex board",
		"Bestiary talents", "boss-kill counts", "Branch total", "Build delta", "treeMinimap", "treeCanvasBtn",
		"drawTreeCanvas()", "autoApplyAffordableQueue()", "abyssTreeCanvasMode",
		"loadoutSave(3)", "buildExport()", "schema:TREE_CATALOG.schema_version",
		"{code:code.trim()}", "toggleBeginner()", "tryUndoAlloc()",
	} {
		if !strings.Contains(combined, required) {
			t.Errorf("progression contract missing %q", required)
		}
	}
}
