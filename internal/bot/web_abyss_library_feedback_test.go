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

func TestAbyssHistoryLoadsBoundedAuthoritativeLootSummary(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = database.Close() }()

	when := time.Date(2026, time.August, 25, 4, 30, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT depth, gold_banked, victory").
		WithArgs("player", 50).
		WillReturnRows(sqlmock.NewRows([]string{
			"depth", "gold_banked", "victory", "tier", "hardcore",
			"loot_count", "loot_summary", "created_at",
		}).AddRow(17, int64(4200), true, "hell", true, 3, []byte(`["Crown","Ward"]`), when))

	rows := (&Bot{DB: database}).abyssHistory("player", 500)
	if len(rows) != 1 {
		t.Fatalf("history rows = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.Depth != 17 || row.Tier != "hell" || !row.Hardcore || row.LootCount != 3 {
		t.Fatalf("history row = %#v", row)
	}
	if strings.Join(row.Loot, ",") != "Crown,Ward" || row.LootTruncated != 1 {
		t.Fatalf("loot summary = %#v, truncated = %d", row.Loot, row.LootTruncated)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBestiaryUsesPersistedFamilyAndLegacyFallback(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectQuery("SELECT mob_name, mob_family, kills FROM abyss_bestiary").
		WithArgs("player").
		WillReturnRows(sqlmock.NewRows([]string{"mob_name", "mob_family", "kills"}).
			AddRow("Frost Lich", "Elite", 9).
			AddRow("Void Watcher", "", 4))

	rows := (&Bot{DB: database}).loadAbyssBestiary("player")
	if len(rows) != 2 || rows[0].Family != "Elite" || rows[1].Family != abyssLegacyBestiaryFamily {
		t.Fatalf("bestiary rows = %#v", rows)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBestiaryKillPersistsEncounterFamily(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = database.Close() }()

	mock.ExpectExec("INSERT INTO abyss_bestiary").
		WithArgs("player", "Ancient Dragon", "Boss").
		WillReturnResult(sqlmock.NewResult(1, 1))
	(&Bot{DB: database}).recordAbyssKills("player", []abyssBestiaryKill{{
		MobName: "Ancient Dragon",
		Family:  "Boss",
	}})
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssLeaderboardsMarkOnlyAuthenticatedUID(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer func() { _ = database.Close() }()

	descents := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"client_uid", "nick", "depth", "gold"}).
			AddRow("other", "Twin", 20, int64(900)).
			AddRow("player", "Twin", 18, int64(800))
	}
	for range 3 {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT a.client_uid")).
			WithArgs("normal", sqlmock.AnyArg(), 10).
			WillReturnRows(descents())
	}
	mock.ExpectQuery(regexp.QuoteMeta("SELECT a.client_uid")).
		WithArgs("normal", 10).
		WillReturnRows(descents())
	mock.ExpectQuery(regexp.QuoteMeta("SELECT k.client_uid")).
		WithArgs(10, "normal").
		WillReturnRows(sqlmock.NewRows([]string{
			"client_uid", "nick", "boss_name", "depth", "kill_time_ms", "killed_at",
		}).AddRow("player", "Twin", "The Maw", 25, int64(1234), time.Now()))

	boards := (&Bot{DB: database}).abyssLeaderboardsForUID("normal", "player")
	for name, rows := range map[string][]abyssRow{
		"day": boards.Day, "season": boards.Season, "all_time": boards.AllTime, "hardcore": boards.Hardcore,
	} {
		if len(rows) != 2 || rows[0].IsCurrent || !rows[1].IsCurrent {
			t.Fatalf("%s ownership markers = %#v", name, rows)
		}
	}
	if len(boards.BossKills) != 1 || !boards.BossKills[0].IsCurrent {
		t.Fatalf("boss ownership markers = %#v", boards.BossKills)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssUI121Through140Contracts(t *testing.T) {
	t.Parallel()

	pageBytes, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	partialBytes, err := webAssets.ReadFile("webassets/partials.html")
	if err != nil {
		t.Fatal(err)
	}
	cssBytes, err := webAssets.ReadFile("webassets/abyss_ui200.css")
	if err != nil {
		t.Fatal(err)
	}
	codexBytes, err := webAssets.ReadFile("webassets/abyss_codex_navigation.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(pageBytes) + string(codexBytes)
	partials, css := string(partialBytes), string(cssBytes)

	pageContracts := []string{
		`data-family="{{.Family}}"`, `data-total="{{.LoreTotal}}"`,
		`tr.ab-me, #abyssBossLeaderboards tr.ab-me`, `class="ab-hist-detail" hidden`,
		`title="{{if .Earned}}Earned ✓{{else}}Unlock: {{.Condition}}{{end}}"`,
		`id="badgePreview"`, `ab_tab_scroll`, `ab-skel ab-skel-h`,
		`data-ts`, `ab-toast-copy`, `okLabel:'Pay '+total+' & Enter`,
		`e.stopImmediatePropagation()`, `id="bankSkipBtn"`, `initLevelFlash()`,
		`achievementBanner(d.achievement)`,
		`{{template "abyssCodexNavigationJS" .}}`,
	}
	for _, contract := range pageContracts {
		if !strings.Contains(page, contract) {
			t.Errorf("Abyss page missing UI contract %q", contract)
		}
	}
	partialContracts := []string{
		`{{if .IsCurrent}} class="ab-me" aria-current="true"{{end}}`,
		`var modalArmTimer = null`, `armSeconds`,
		`<button class="ghost" id="modalCancelBtn">`,
	}
	for _, contract := range partialContracts {
		if !strings.Contains(partials, contract) {
			t.Errorf("shared modal/leaderboard partial missing UI contract %q", contract)
		}
	}
	cssContracts := []string{
		`.ab-best-fam td`, `.ab-lore-ring`, `position: sticky`, `.ab-history-loot`,
		`.ab-ach.locked`, `@media print`, `.toast2:hover::after`, `.ab-ach-banner`,
	}
	for _, contract := range cssContracts {
		if !strings.Contains(css, contract) {
			t.Errorf("Abyss UI stylesheet missing contract %q", contract)
		}
	}
}

func TestAbyssLibraryMigrationsBoundHistoryAndPersistFamilies(t *testing.T) {
	t.Parallel()

	root := abyssAAARepositoryRoot(t)
	for path, contracts := range map[string][]string{
		"0075_abyss_run_loot_history.up.sql": {"loot_summary JSONB", "loot_count INTEGER", "CHECK (loot_count >= 0)"},
		"0076_abyss_bestiary_family.up.sql":  {"mob_family TEXT", "NOT NULL DEFAULT ''"},
	} {
		content, err := os.ReadFile(filepath.Join(root, "internal", "db", "migrations", path))
		if err != nil {
			t.Fatal(err)
		}
		for _, contract := range contracts {
			if !strings.Contains(string(content), contract) {
				t.Errorf("%s missing %q", path, contract)
			}
		}
	}
}
