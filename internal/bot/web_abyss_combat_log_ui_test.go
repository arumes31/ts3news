package bot

import (
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssCombatLogUIContracts(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_ui200.css")
	if err != nil {
		t.Fatalf("read Abyss UI styles: %v", err)
	}
	xpSource, err := os.ReadFile("xp.go")
	if err != nil {
		t.Fatalf("read combat engine: %v", err)
	}
	source := string(page) + string(styles) + string(xpSource)
	for _, required := range []string{
		"function highlightLogCombatant",
		"—— Floor '+curDepth+' ——",
		"ab-log-crit",
		"ab-log-dodge",
		"lifesteal: +%d HP",
		"data-round",
		"complete middle rounds",
		"rar-legendary",
		"function pinFatalBlow",
		"ab-log-capture",
		"'🔧 '+l",
		"Momentum ×'+curMomentum+': broken by consumable use",
		"d.legendary_pity!==undefined",
		"Daily affix: ",
		"% reward / +'+danger+'% danger",
		`id="logSearch"`,
		"navigator.clipboard.writeText(lastFightText)",
		"font-variant-ligatures: none",
		"function srSummary",
		"d.querySelector('.ab-dot-signal')",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Abyss combat log is missing %q", required)
		}
	}
	if strings.Contains(source, "pity counter approximation — gear drops move it") {
		t.Error("legendary pity must not be inferred from rendered loot")
	}
}

func TestAbyssDoTMarkerBecomesLocaleIndependentPresentationSignal(t *testing.T) {
	t.Parallel()

	html := abyssCombatLogHTML(markAbyssDoTLog("localized poison line", true))
	if !strings.Contains(html, `class="ab-dot-signal" hidden`) {
		t.Fatalf("DoT log lacks presentation signal: %q", html)
	}
	if strings.Contains(html, abyssDoTMarker) {
		t.Fatalf("internal DoT marker leaked into rendered combat log: %q", html)
	}
	if plain := abyssCombatLogHTML("ordinary hit"); strings.Contains(plain, "ab-dot-signal") {
		t.Fatalf("ordinary hit received DoT signal: %q", plain)
	}
}

func TestAddAbyssLegendaryPityUsesPersistedValue(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`SELECT legendary_pity FROM users WHERE client_uid=\$1`).
		WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"legendary_pity"}).AddRow(31))
	out := map[string]any{}
	(&Bot{DB: db}).addAbyssLegendaryPity(out, "player")
	if got := out["legendary_pity"]; got != 31 {
		t.Fatalf("legendary_pity = %v, want persisted value 31", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
