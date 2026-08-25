package bot

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssLongTermStatusUsesPersistedPaceBountiesAndMaterialFlow(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT bounty_day FROM abyss_bounty_claims")).
		WithArgs("player", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"bounty_day"}).AddRow(now.Truncate(24 * time.Hour)))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT mat_id, direction, amount, created_at FROM abyss_forge_material_flow")).
		WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"mat_id", "direction", "amount", "created_at"}).
			AddRow("dust", "source", int64(9), now).
			AddRow("dust", "sink", int64(4), now))
	mock.ExpectQuery("SELECT depth,floors_cleared,duration_ms").WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"depth", "floors_cleared", "duration_ms"}).AddRow(44, 20, int64(100_000)))

	run := abyssRun{Active: true, Depth: 15, CheckpointStart: 10, FloorType: "combat", StartedAt: now.Add(-time.Minute)}
	history := []abyssHistoryRow{
		{Depth: 22, FloorsCleared: 12, DurationMS: 120_000},
		{Depth: 31, FloorsCleared: 18, DurationMS: 90_000},
	}
	view := (&Bot{DB: database}).abyssLongTermStatus(t.Context(), "player", run, history, 37, 31)

	if view.Pace.CurrentFloors != 5 || view.Pace.CurrentFloorsPerMinute < 4.8 || view.Pace.CurrentFloorsPerMinute > 5.2 {
		t.Fatalf("current pace = %#v", view.Pace)
	}
	if view.Pace.LastFloorsPerMinute != 6 || view.Pace.LastDepth != 22 {
		t.Fatalf("last-run ghost = %#v", view.Pace)
	}
	if view.Pace.BestFloorsPerMinute != 12 || view.Pace.BestDepth != 44 {
		t.Fatalf("best-run ghost = %#v", view.Pace)
	}
	if len(view.BountyDays) != 7 || !view.BountyDays[6].Claimed {
		t.Fatalf("bounty history = %#v", view.BountyDays)
	}
	if got := view.MaterialFlow["dust"][6]; got != 5 {
		t.Fatalf("today dust flow = %d, want 5", got)
	}
	if view.LegendaryPity != 31 || view.LegendaryCap != abyssLegendaryPityCap || view.MapDepth != 50 {
		t.Fatalf("long-term counters = %#v", view)
	}
	if len(view.Milestones) < 12 {
		t.Fatalf("milestones = %d, want boss/checkpoint cadence plus unlocks", len(view.Milestones))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBestPaceFromHistoryUsesFastestRunAndDepthTieBreak(t *testing.T) {
	t.Parallel()
	pace, depth := bestPaceFromHistory([]abyssHistoryRow{
		{Depth: 18, FloorsCleared: 10, DurationMS: 60_000},
		{Depth: 24, FloorsCleared: 20, DurationMS: 120_000},
		{Depth: 99, FloorsCleared: 0, DurationMS: 1},
	})
	if pace != 10 || depth != 24 {
		t.Fatalf("best history pace = %.1f at depth %d, want 10.0 at depth 24", pace, depth)
	}
}

func TestAbyssUI181Through190AuthoritativeModule(t *testing.T) {
	t.Parallel()

	pageBytes, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	moduleBytes, err := webAssets.ReadFile("webassets/abyss_longterm.html")
	if err != nil {
		t.Fatal(err)
	}
	styleBytes, err := webAssets.ReadFile("webassets/abyss_longterm.css")
	if err != nil {
		t.Fatal(err)
	}
	baseStyleBytes, err := webAssets.ReadFile("webassets/abyss_ui200.css")
	if err != nil {
		t.Fatal(err)
	}
	page, module, styles := string(pageBytes), string(moduleBytes), string(styleBytes)+string(baseStyleBytes)
	for _, contract := range []string{
		`/static/abyss_longterm.css`, `template "abyss-longterm"`,
		`last_floors_per_minute`, `best_floors_per_minute`, `best_depth`, `current_floors`, `started_unix`,
		`bounty_days`, `material_flow_7d`, `legendary_pity`, `legendary_cap`,
		`Authoritative milestone map`, `window.trackPaceFloor`, `markBountyClaimedToday`,
		`Search settings`, `prefers-reduced-motion`, `.ab-milestone`,
	} {
		if !strings.Contains(page+module+styles, contract) {
			t.Errorf("long-term module missing contract %q", contract)
		}
	}
	for _, forbidden := range []string{"ab_pace_best", "ab_bounty_days", "ab_mat_income"} {
		if strings.Contains(page+module, forbidden) {
			t.Errorf("long-term UI retains browser-authoritative ledger %q", forbidden)
		}
	}
}

func TestAbyssRunDurationMigrationPersistsComparablePace(t *testing.T) {
	t.Parallel()
	root := abyssAAARepositoryRoot(t)
	content, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", "0078_abyss_run_duration.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, contract := range []string{"duration_ms BIGINT", "floors_cleared INTEGER", "duration_ms >= 0", "floors_cleared >= 0"} {
		if !strings.Contains(string(content), contract) {
			t.Errorf("run duration migration missing %q", contract)
		}
	}
}
