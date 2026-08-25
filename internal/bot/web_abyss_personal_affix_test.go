package bot

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestChooseAbyssAffixRerollAlwaysChangesAffix(t *testing.T) {
	t.Parallel()

	for _, current := range abyssDailyMods {
		selected, err := chooseAbyssAffixReroll(bytes.NewReader(make([]byte, 32)), current)
		if err != nil {
			t.Fatal(err)
		}
		if selected == current || abyssDailyAffixIndex(selected) == 0 {
			t.Fatalf("reroll from %q selected %q", current, selected)
		}
	}
}

func TestCurrentPersonalAbyssAffixUsesValidatedDailyOverride(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	at := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT value FROM app_meta").
		WithArgs("abyss_personal_affix_player_2026-08-25").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("bloodlust"))
	_, affix, rerolled := (&Bot{DB: db}).currentPersonalAbyssAffixAt("player", at)
	if affix != "bloodlust" || !rerolled {
		t.Fatalf("personal affix = %q, rerolled = %v", affix, rerolled)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssPersonalAffixClientAndRunContract(t *testing.T) {
	t.Parallel()

	server, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	routes, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	module, err := webAssets.ReadFile("webassets/abyss_pact_program.html")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(server), "abyssRunDailyChallenge(uid)") < 4 {
		t.Fatal("combat and every reward path do not share the run-snapshotted personal affix")
	}
	if !strings.Contains(string(routes), `"/api/abyss/affix/reroll"`) {
		t.Fatal("personal affix reroll route is missing")
	}
	for _, required := range []string{"personalAffixControl", "rerollAbyssPersonalAffix", "personal_affix"} {
		if !strings.Contains(string(module), required) {
			t.Errorf("personal affix UI is missing %q", required)
		}
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"suppressDailyAffix", "suppress_affix"} {
		if !strings.Contains(string(page), required) {
			t.Errorf("affix suppressor entry UI is missing %q", required)
		}
	}
	if item, ok := content.GetConsumableByID("abyss_affix_suppressor"); !ok || item.Name != "Affix Suppressor" {
		t.Fatalf("affix suppressor consumable = %#v, %v", item, ok)
	}
	if shop, ok := abyssShopByKey("affix_suppressor"); !ok || shop.Cost <= 0 {
		t.Fatalf("affix suppressor shop entry = %#v, %v", shop, ok)
	}
}

func TestAbyssRunDailyChallengeHonorsSuppressorFlag(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	mock.ExpectQuery("SELECT value FROM app_meta").
		WithArgs("abyss_run_flags_player").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"daily_affix":-1}`))
	_, affix := (&Bot{DB: db}).abyssRunDailyChallenge("player")
	if affix != "" {
		t.Fatalf("suppressed run affix = %q, want empty", affix)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
