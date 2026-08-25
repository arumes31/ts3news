package bot

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssForgeExperienceContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	partial := server.tmpl.Lookup("abyss-forge-experience")
	if partial == nil {
		t.Fatal("forge experience template is missing")
	}
	for _, required := range []string{
		"forgeSelectionCard",
		"forgeTemperChance",
		"ab-action-price",
		"Confirm dismantle manifest",
		"inv_ids:ids",
		"revision:preview.revision",
		"function fusionCatalog",
		"/api/abyss/fuse_preview",
		"Loading authoritative result preview",
		"function updateMaterialRates",
		"function updateHistory",
	} {
		if !strings.Contains(partial.Tree.Root.String(), required) {
			t.Errorf("forge experience is missing %q", required)
		}
	}
}

func TestAbyssForgeExperienceAssetsAndHooks(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	css, err := webAssets.ReadFile("webassets/abyss_forge_experience.css")
	if err != nil {
		t.Fatalf("read forge experience CSS: %v", err)
	}
	for _, required := range []string{
		`{{asset "/static/abyss_forge_experience.css"}}`,
		`{{template "abyss-forge-experience" .}}`,
		"can make ×{{.Craftable}}",
		"{{.Missing}}",
		"data-created=\"{{$entry.AtUnix}}\"",
		"ab-forge-history-undo",
	} {
		if !strings.Contains(string(page), required) {
			t.Errorf("Abyss page is missing forge hook %q", required)
		}
	}
	for _, required := range []string{
		".ab-forge-selection",
		".ab-forge-chance-track",
		"#forgeActions button.ab-forge-applicable",
		".ab-forge-review-list",
		".ab-fusion-grid",
		"@media (prefers-reduced-motion: reduce)",
		"@media (forced-colors: active)",
	} {
		if !strings.Contains(string(css), required) {
			t.Errorf("forge experience CSS is missing %q", required)
		}
	}
	serverSource, err := os.ReadFile("web.go")
	if err != nil {
		t.Fatalf("read web routes: %v", err)
	}
	if !strings.Contains(string(serverSource), `/static/abyss_forge_experience.css`) {
		t.Error("forge experience stylesheet is not served")
	}
}

func TestAbyssDismantleSelectionIsExactAndBounded(t *testing.T) {
	t.Parallel()

	set, ok := abyssDismantleIDSet([]int64{7, 11, 19})
	if !ok || len(set) != 3 || !set[11] {
		t.Fatalf("valid dismantle set = %#v, %v", set, ok)
	}
	for _, ids := range [][]int64{{0}, {4, 4}, make([]int64, abyssDismantleBatchLimit+1)} {
		if _, valid := abyssDismantleIDSet(ids); valid {
			t.Fatalf("invalid dismantle selection accepted: %#v", ids)
		}
	}
	items := make([]abyssDismantleSpare, abyssDismantleBatchLimit+3)
	bounded, remaining := boundAbyssDismantleBatch(items)
	if len(bounded) != abyssDismantleBatchLimit || remaining != 3 {
		t.Fatalf("bounded batch = %d + %d, want %d + 3", len(bounded), remaining, abyssDismantleBatchLimit)
	}
}

func TestAbyssDismantleCommitRequiresReviewedManifest(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/api/abyss/dismantle", strings.NewReader(`{"preview":false}`))
	server := &WebServer{bot: &Bot{DB: database}}
	server.handleAbyssDismantle(recorder, request, "user")
	if body := recorder.Body.String(); !strings.Contains(body, "preview dismantle before confirming") {
		t.Fatalf("empty dismantle commit response = %q", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssDismantleManifestRevisionBindsItemState(t *testing.T) {
	t.Parallel()

	items := []abyssDismantleSpare{{id: 7, state: "before"}, {id: 9, state: "stable"}}
	before := abyssDismantleManifestRevision(items)
	items[0].state = "after"
	if after := abyssDismantleManifestRevision(items); before == after {
		t.Fatal("manifest revision did not change with item state")
	}
	query, args := abyssDismantleInventoryQuery("user", []int64{7, 9}, false)
	if !strings.Contains(query, "id IN ($2,$3)") || !strings.Contains(query, "FOR UPDATE") || len(args) != 3 {
		t.Fatalf("exact commit query = %q %#v", query, args)
	}
	previewQuery, previewArgs := abyssDismantleInventoryQuery("user", nil, true)
	if !strings.Contains(previewQuery, "LIMIT $2") || strings.Contains(previewQuery, "FOR UPDATE") || len(previewArgs) != 2 {
		t.Fatalf("bounded preview query = %q %#v", previewQuery, previewArgs)
	}
}

func TestAbyssDismantleManifestRevisionBindsReviewedState(t *testing.T) {
	t.Parallel()

	items := []abyssDismantleSpare{{id: 7, state: "first"}, {id: 11, state: "second"}}
	revision := abyssDismantleManifestRevision(items)
	if revision == "" {
		t.Fatal("dismantle revision is empty")
	}
	items[1].state = "changed"
	if changed := abyssDismantleManifestRevision(items); changed == revision {
		t.Fatal("dismantle revision did not bind the reviewed item state")
	}
	previewQuery, previewArgs := abyssDismantleInventoryQuery("user", nil, true)
	if !strings.Contains(previewQuery, "LIMIT $2") || len(previewArgs) != 2 || previewArgs[1] != abyssDismantleScanLimit+1 {
		t.Fatalf("preview query is not bounded: %q %#v", previewQuery, previewArgs)
	}
	commitQuery, commitArgs := abyssDismantleInventoryQuery("user", []int64{7, 11}, false)
	if !strings.Contains(commitQuery, "id IN ($2,$3)") || !strings.Contains(commitQuery, "FOR UPDATE") || len(commitArgs) != 3 {
		t.Fatalf("commit query is not selection-scoped and locked: %q %#v", commitQuery, commitArgs)
	}
}

func TestAbyssRecipeViewsExposeAffordability(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectQuery("SELECT recipe_id FROM user_recipes").WithArgs("user").
		WillReturnRows(sqlmock.NewRows([]string{"recipe_id"}))
	views := abyssRecipeViews(&Bot{DB: database}, "user", map[string]int64{"dust": 3})
	if len(views) != len(craftRecipes) {
		t.Fatalf("recipe views = %d, want %d", len(views), len(craftRecipes))
	}
	if got := views[0]["Craftable"]; got != 0 {
		t.Fatalf("first recipe craftable = %#v, want 0", got)
	}
	if got := views[0]["Affordable"]; got != false {
		t.Fatalf("first recipe affordable = %#v, want false", got)
	}
	if missing := views[0]["Missing"].(string); !strings.Contains(missing, "1") {
		t.Fatalf("first recipe missing = %q, want one missing dust", missing)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestForgeHistoryCarriesClientRelativeTimestamp(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	created := time.Date(2026, time.August, 25, 8, 30, 0, 0, time.UTC)
	mock.ExpectQuery("SELECT action, detail, cost, created_at FROM forge_history").
		WithArgs("user", 1).
		WillReturnRows(sqlmock.NewRows([]string{"action", "detail", "cost", "created_at"}).
			AddRow("temper", "Blade → +4", "100g", created))

	rows := (&Bot{DB: database}).loadForgeHistory("user", 1)
	if len(rows) != 1 || rows[0].AtUnix != created.Unix() {
		t.Fatalf("forge history timestamp = %#v, want %d", rows, created.Unix())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
