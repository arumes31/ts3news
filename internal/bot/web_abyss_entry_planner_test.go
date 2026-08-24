package bot

import (
	"strings"
	"testing"
)

func TestAbyssEntryPlannerAssetsAndContracts(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	script := server.tmpl.Lookup("abyssEntryPlannerJS")
	if script == nil {
		t.Fatal("Abyss entry planner script partial is missing")
	}
	for _, required := range []string{
		"function showAbyssEntryStep",
		"function renderAbyssEntrySummary",
		"function captureAbyssEntrySetup",
		"function restoreAbyssEntrySetup",
		"function saveAbyssEntrySetup",
		"abyssEntrySetupV1",
		"aria-current",
	} {
		if !strings.Contains(script.Tree.Root.String(), required) {
			t.Errorf("Abyss entry planner script is missing %q", required)
		}
	}

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	css, err := webAssets.ReadFile("webassets/abyss_entry_planner.css")
	if err != nil {
		t.Fatalf("read entry planner stylesheet: %v", err)
	}
	for _, required := range []string{
		`{{asset "/static/abyss_entry_planner.css"}}`,
		`{{template "abyssEntryPlannerPanel" .}}`,
		`{{template "abyssEntryPlannerJS" .}}`,
		`id="lockedCheckpointHint"`,
		`data-danger=`,
	} {
		if !strings.Contains(string(page), required) {
			t.Errorf("Abyss page is missing %q", required)
		}
	}
	for _, required := range []string{".ab-entry-planner", ".ab-entry-summary", "@media (max-width: 720px)"} {
		if !strings.Contains(string(css), required) {
			t.Errorf("Abyss entry planner CSS is missing %q", required)
		}
	}
}
