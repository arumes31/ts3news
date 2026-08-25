package bot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"ts3news/internal/content"
)

func TestAbyssRunInsightsUsePersistedAuthoritativeSources(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT end_reason, COUNT(*) FROM abyss_runs")).
		WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"end_reason", "count"}).
			AddRow("conceded", 2).AddRow("revive_failed", 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT tier, COUNT(*) FILTER (WHERE victory), COUNT(*)")).
		WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"tier", "wins", "runs"}).
			AddRow("normal", 3, 4).AddRow("hell", 1, 2))

	before, _ := json.Marshal(map[string]any{
		"item":      content.Gear{Stats: content.Stats{STR: 10}},
		"materials": map[string]int64{"dust": 20, "core": 5},
	})
	after, _ := json.Marshal(map[string]any{
		"item":      content.Gear{Stats: content.Stats{STR: 30}},
		"materials": map[string]int64{"dust": 12, "core": 3},
	})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT before_state, after_state FROM abyss_forge_mutation_audit")).
		WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"before_state", "after_state"}).AddRow(before, after))

	history := []abyssHistoryRow{{Depth: 12, Victory: true, Tier: "normal", EndReason: "banked"}}
	bestiary := []abyssBestiaryRow{{MobName: "Rat", Family: "Common"}, {MobName: "Bat", Family: "Common"}}
	run := abyssRun{Active: true, Escrow: 3600, StartedAt: time.Now().Add(-time.Hour)}
	view := (&Bot{DB: database}).abyssRunInsights("player", run, history, bestiary, 2)

	if len(view.Runs) != 1 || len(view.DeathCauses) != 2 || view.DeathCauses[0].Label != "Conceded" {
		t.Fatalf("run/death insights = %#v", view)
	}
	if len(view.TierRates) != 2 || view.TierRates[0].Percent != 75 {
		t.Fatalf("tier rates = %#v", view.TierRates)
	}
	if len(view.Bestiary) != 1 || view.Bestiary[0].Target != 12 || view.Bestiary[0].Seen != 2 {
		t.Fatalf("bestiary progress = %#v", view.Bestiary)
	}
	if view.ForgeROI.Actions != 1 || view.ForgeROI.MaterialSpent != 10 || view.ForgeROI.CRGained <= 0 {
		t.Fatalf("forge ROI = %#v", view.ForgeROI)
	}
	if view.SessionGoldPH < 3500 || view.SessionGoldPH > 3700 || view.Prestige.NextPct != 15 {
		t.Fatalf("session/prestige insight = %#v", view)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssUI171Through180Contracts(t *testing.T) {
	t.Parallel()

	pageBytes, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	moduleBytes, err := webAssets.ReadFile("webassets/abyss_insights.html")
	if err != nil {
		t.Fatal(err)
	}
	styleBytes, err := webAssets.ReadFile("webassets/abyss_insights.css")
	if err != nil {
		t.Fatal(err)
	}
	treeBytes, err := webAssets.ReadFile("webassets/abysstree.html")
	if err != nil {
		t.Fatal(err)
	}
	page, module, styles, tree := string(pageBytes), string(moduleBytes), string(styleBytes), string(treeBytes)
	for _, contract := range []string{
		`data-tier="{{.Tier}}"`, `data-end-reason="{{.EndReason}}"`,
		`template "abyss-insights"`, `treeAvailable`,
	} {
		if !strings.Contains(page+module+tree, contract) {
			t.Errorf("Abyss insight surface missing contract %q", contract)
		}
	}
	for _, contract := range []string{
		`slice(0,30)`, `death_causes`, `session_seconds`, `tier_rates`,
		`bestiary`, `forge_roi`, `prestige`, `Filter history by tier`,
		`Filtered history JSON copied`,
	} {
		if !strings.Contains(module, contract) {
			t.Errorf("Abyss insight module missing contract %q", contract)
		}
	}
	for _, forbidden := range []string{`ab_tier_wr`, `ab_lb_snap`, `ab_gph`} {
		if strings.Contains(page, forbidden) {
			t.Errorf("Abyss page retains browser-authoritative insight ledger %q", forbidden)
		}
	}
	for _, contract := range []string{`.ab-insight-grid`, `.ab-rank-delta.up`, `prefers-reduced-motion`} {
		if !strings.Contains(styles, contract) {
			t.Errorf("Abyss insight styles missing %q", contract)
		}
	}
}

func TestAbyssRunOutcomeMigrationSupportsCauseBreakdown(t *testing.T) {
	t.Parallel()
	root := abyssAAARepositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0077_abyss_run_outcomes.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sql := string(content)
	for _, contract := range []string{"end_reason", "revive_failed", "conceded", "timeout", "banked"} {
		if !strings.Contains(sql, contract) {
			t.Errorf("run outcome migration missing %q", contract)
		}
	}
}
