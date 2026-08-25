package bot

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCanonicalAbyssPactPreset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		slot      int
		label     string
		pacts     []string
		wantOK    bool
		wantName  string
		wantPacts []string
	}{
		{name: "canonicalizes known pacts", slot: 2, label: "  Boss push  ", pacts: []string{"famine", "unknown", "famine", "anemic"}, wantOK: true, wantName: "Boss push", wantPacts: []string{"anemic", "famine"}},
		{name: "supplies a default name", slot: 1, pacts: []string{"blind"}, wantOK: true, wantName: "Preset 1", wantPacts: []string{"blind"}},
		{name: "rejects an out of range slot", slot: 4, label: "Forged", pacts: []string{"blind"}, wantOK: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, ok := canonicalAbyssPactPreset(test.slot, test.label, test.pacts)
			if ok != test.wantOK {
				t.Fatalf("ok = %v, want %v", ok, test.wantOK)
			}
			if !ok {
				return
			}
			if got.Name != test.wantName || strings.Join(got.Pacts, ",") != strings.Join(test.wantPacts, ",") {
				t.Fatalf("preset = %#v, want name %q and pacts %v", got, test.wantName, test.wantPacts)
			}
		})
	}
}

func TestAbyssPactRewardMultWithMastery(t *testing.T) {
	t.Parallel()

	base := abyssPactRewardMultWithMastery([]string{"glass_cannon", "abstinence"}, nil)
	if base != 1.45 {
		t.Fatalf("base multiplier = %v, want 1.45", base)
	}
	mastered := abyssPactRewardMultWithMastery(
		[]string{"glass_cannon", "abstinence"},
		map[string]int{"glass_cannon": abyssPactMasteryRuns, "abstinence": abyssPactMasteryRuns - 1},
	)
	if mastered < 1.4649 || mastered > 1.4651 {
		t.Fatalf("mastered multiplier = %v, want 1.465", mastered)
	}
	state := abyssPactProgramStateFrom(nil, map[string]int{"glass_cannon": abyssPactMasteryRuns})
	if state.MasteredCount != 1 {
		t.Fatalf("mastered count = %d, want 1", state.MasteredCount)
	}
	for _, view := range state.Mastery {
		if view.Key == "glass_cannon" && (!view.Mastered || view.EffectiveBonusPct != 31.5) {
			t.Fatalf("glass cannon mastery view = %#v", view)
		}
	}
}

func TestAdvanceAbyssPactMasteryCountsEachPactOnce(t *testing.T) {
	t.Parallel()

	mastery := advanceAbyssPactMastery(map[string]int{"blind": 9}, []string{"blind", "blind", "unknown", "famine"})
	if mastery["blind"] != 10 || mastery["famine"] != 1 {
		t.Fatalf("mastery = %#v", mastery)
	}
}

func TestIncrementAbyssPactMasteryUsesCallerTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE")).
		WithArgs("abyss_pact_mastery_player").
		WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(`{"blind":9}`))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO app_meta (key, value) VALUES ($1, $2)")).
		WithArgs("abyss_pact_mastery_player", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO abyss_achievements (client_uid, code) VALUES ($1,$2) ON CONFLICT DO NOTHING")).
		WithArgs("player", "pact_blind").
		WillReturnResult(sqlmock.NewResult(1, 1))
	if err := incrementAbyssPactMastery(tx, "player", []string{"blind", "blind"}); err != nil {
		t.Fatal(err)
	}
	mock.ExpectCommit()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssPactAchievementsCoverCatalog(t *testing.T) {
	t.Parallel()

	views := abyssPactAchievementViews()
	if len(views) != len(abyssPactCatalog) {
		t.Fatalf("achievement views = %d, want %d", len(views), len(abyssPactCatalog))
	}
	for index, pact := range abyssPactCatalog {
		view := views[index]
		if view.Code != "pact_"+pact.Key || view.Name == "" || !strings.Contains(view.Condition, pact.Label) {
			t.Errorf("achievement for %q = %#v", pact.Key, view)
		}
		if got := abyssAchievementName(view.Code); got != view.Name {
			t.Errorf("achievement name %q = %q, want %q", view.Code, got, view.Name)
		}
	}
}

func TestAbyssPactProgramClientContract(t *testing.T) {
	t.Parallel()

	module, err := webAssets.ReadFile("webassets/abyss_pact_program.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_pact_program.css")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	server, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatal(err)
	}
	moduleText := string(module)
	for _, required := range []string{"abyssPactProgramPanel", "abyssPactProgramJS", "/api/abyss/pact/presets", "replaceChildren", "textContent"} {
		if !strings.Contains(moduleText, required) {
			t.Errorf("pact program module is missing %q", required)
		}
	}
	if strings.Contains(moduleText, "innerHTML") {
		t.Error("pact preset UI must not insert player-controlled names through innerHTML")
	}
	if !strings.Contains(string(styles), ".ab-pact-program") {
		t.Error("pact program stylesheet is missing its component root")
	}
	for _, required := range []string{`{{template "abyssPactProgramPanel" .}}`, `{{template "abyssPactProgramJS" .}}`, "/static/abyss_pact_program.css"} {
		if !strings.Contains(string(page), required) {
			t.Errorf("Abyss page is missing %q", required)
		}
	}
	for _, required := range []string{"/api/abyss/pact/presets", "/static/abyss_pact_program.css"} {
		if !strings.Contains(string(server), required) {
			t.Errorf("web routes are missing %q", required)
		}
	}
}

func TestAbyssPactMasteryCompletionHooksStayTransactional(t *testing.T) {
	t.Parallel()

	bankSource, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	forfeitSource, err := os.ReadFile("web_abyss_econ.go")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		source string
		start  string
	}{
		{name: "bank", source: string(bankSource), start: "func (s *WebServer) handleAbyssBank("},
		{name: "forfeit", source: string(forfeitSource), start: "func (b *Bot) forfeitAbyss("},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			start := strings.Index(test.source, test.start)
			if start < 0 {
				t.Fatalf("missing completion function %q", test.start)
			}
			body := test.source[start:]
			insertAt := strings.Index(body, "INSERT INTO abyss_runs")
			masteryAt := strings.Index(body, "incrementAbyssPactMastery(tx")
			commitAt := strings.Index(body, "tx.Commit()")
			if insertAt < 0 || masteryAt <= insertAt || commitAt <= masteryAt {
				t.Fatalf("completion order insert=%d mastery=%d commit=%d", insertAt, masteryAt, commitAt)
			}
		})
	}
}
