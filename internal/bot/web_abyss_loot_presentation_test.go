package bot

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssLootPresentationContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	var panel strings.Builder
	if err := server.tmpl.ExecuteTemplate(&panel, "abyssLootPresentationPin", nil); err != nil {
		t.Fatalf("render loot presentation pin: %v", err)
	}
	for _, required := range []string{`id="lootCustody"`, `id="bestLootPin"`, `aria-live="polite"`, "Escrowed until banked"} {
		if !strings.Contains(panel.String(), required) {
			t.Errorf("loot presentation panel is missing %q", required)
		}
	}
	script := server.tmpl.Lookup("abyssLootPresentationJS")
	if script == nil {
		t.Fatal("Abyss loot-presentation script partial is missing")
	}
	for _, required := range []string{
		"function abyssLootPresentationScore",
		"function updateLootRewardPresentation",
		"function releaseEscrowLoot",
		"function renderBankPreviewLoot",
		"BEST DROP THIS RUN",
		"row.classList.add('ab-escrowed')",
		"replaceChildren",
		"textContent=",
	} {
		if !strings.Contains(script.Tree.Root.String(), required) {
			t.Errorf("loot presentation script is missing %q", required)
		}
	}
}

func TestAbyssLootPresentationAssetsAndHooks(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	css, err := webAssets.ReadFile("webassets/abyss_loot_presentation.css")
	if err != nil {
		t.Fatalf("read loot-presentation stylesheet: %v", err)
	}
	for _, required := range []string{
		`{{asset "/static/abyss_loot_presentation.css"}}`,
		`{{template "abyssLootPresentationPin" .}}`,
		`{{template "abyssLootPresentationJS" .}}`,
		"function lootToast",
		"ab-loot-toast-icon",
		"id=\"lootFilters\"",
		"function updateLootSummary",
		"function abyssBank",
		"preview:true",
		"p.loot_preview",
		"function flashEdgeGlow",
		"function addUndoLink",
		"function showBankSummary",
		"releaseEscrowLoot",
		"document.addEventListener('focusin'",
		"e.key==='Escape'",
	} {
		if !strings.Contains(string(page), required) {
			t.Errorf("Abyss page is missing loot-presentation hook %q", required)
		}
	}
	for _, required := range []string{
		".ab-loot-custody",
		".ab-best-loot-card",
		".abyss-side-loot.ab-escrowed",
		".abyss-side-loot.ab-loot-released",
		".ab-bank-loot-preview",
		"@media (prefers-reduced-motion: reduce)",
		"@media (forced-colors: active)",
	} {
		if !strings.Contains(string(css), required) {
			t.Errorf("loot-presentation CSS is missing %q", required)
		}
	}
}

func TestCurrentAbyssBankPreviewLootIsBoundedAndPlainText(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	bot := &Bot{DB: database}
	mock.ExpectQuery("SELECT item_type, label FROM abyss_escrow_loot").
		WithArgs("user", abyssBankPreviewLootLimit).
		WillReturnRows(sqlmock.NewRows([]string{"item_type", "label"}).
			AddRow("gear", "[color=#ff9d3c]Crown[/color] [s:Head]").
			AddRow("cons", "[b]Lucky Draught[/b]").
			AddRow("unique", strings.Repeat("x", abyssBankPreviewLabelLimit+10))).
		RowsWillBeClosed()

	items, err := bot.currentAbyssBankPreviewLoot(context.Background(), "user", 999)
	if err != nil {
		t.Fatalf("currentAbyssBankPreviewLoot: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("items = %d, want 3", len(items))
	}
	if items[0].Label != "Crown" || items[0].Slot != "Head" {
		t.Fatalf("first preview item = %#v", items[0])
	}
	if strings.Contains(items[0].Label, "[") || strings.Contains(items[0].Label, "<") {
		t.Fatalf("preview label was not reduced to plain text: %q", items[0].Label)
	}
	if got := len([]rune(items[2].Label)); got != abyssBankPreviewLabelLimit+1 || !strings.HasSuffix(items[2].Label, "…") {
		t.Fatalf("bounded label length = %d, value suffix = %q", got, items[2].Label[len(items[2].Label)-3:])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCurrentAbyssBankPreviewLootReturnsQueryErrors(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectQuery("SELECT item_type, label FROM abyss_escrow_loot").
		WithArgs("user", 1).
		WillReturnError(errors.New("query failed"))

	_, err = (&Bot{DB: database}).currentAbyssBankPreviewLoot(context.Background(), "user", 0)
	if err == nil || !strings.Contains(err.Error(), "querying Abyss bank preview loot") {
		t.Fatalf("error = %v, want wrapped query error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCurrentAbyssBankPreviewLootReturnsRowErrors(t *testing.T) {
	t.Parallel()

	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	mock.ExpectQuery("SELECT item_type, label FROM abyss_escrow_loot").
		WithArgs("user", 1).
		WillReturnRows(sqlmock.NewRows([]string{"item_type", "label"}).
			AddRow("gear", "Crown").
			RowError(0, errors.New("read interrupted")))

	_, err = (&Bot{DB: database}).currentAbyssBankPreviewLoot(context.Background(), "user", 1)
	if err == nil || !strings.Contains(err.Error(), "iterating Abyss bank preview loot") {
		t.Fatalf("error = %v, want wrapped iteration error", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAbyssBankPreviewRemainsReadOnly(t *testing.T) {
	t.Parallel()

	source, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	serverBytes, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatalf("read bank handler: %v", err)
	}
	serverSource := string(serverBytes)
	handlerAt := strings.Index(serverSource, "func (s *WebServer) handleAbyssBank(")
	if handlerAt < 0 {
		t.Fatal("bank handler is missing")
	}
	serverSource = serverSource[handlerAt:]
	previewAt := strings.Index(serverSource, "if req.Preview {")
	beginAt := strings.Index(serverSource, "tx, err := s.bot.DB.Begin()")
	if previewAt < 0 || beginAt < 0 || previewAt >= beginAt {
		t.Fatal("bank preview must return before the mutation transaction begins")
	}
	previewBlock := serverSource[previewAt:beginAt]
	for _, required := range []string{`"preview": true`, `"depth_bonus"`, `"streak_bonus"`, `"loot_preview"`, "writeJSON", "return"} {
		if !strings.Contains(previewBlock, required) {
			t.Errorf("bank preview block is missing %q", required)
		}
	}
	if !strings.Contains(string(source), "abyssBankCommit(cursed)") {
		t.Error("bank preview UI does not separate confirmation from commit")
	}
}
