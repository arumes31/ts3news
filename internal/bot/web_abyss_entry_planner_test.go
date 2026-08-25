package bot

import (
	"reflect"
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
		"last_setup",
		"entryFocus",
		"aria-current",
		"server risk forecast",
		"Floor 1 ",
	} {
		if !strings.Contains(script.Tree.Root.String(), required) {
			t.Errorf("Abyss entry planner script is missing %q", required)
		}
	}
	if strings.Contains(script.Tree.Root.String(), "localStorage") {
		t.Error("Abyss entry setup must not be persisted in browser-only localStorage")
	}

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	css, err := webAssets.ReadFile("webassets/abyss_entry_planner.css")
	if err != nil {
		t.Fatalf("read entry planner stylesheet: %v", err)
	}
	baseCSS, err := webAssets.ReadFile("webassets/style.css")
	if err != nil {
		t.Fatalf("read base stylesheet: %v", err)
	}
	for _, required := range []string{
		`{{asset "/static/abyss_entry_planner.css"}}`,
		`{{template "abyssEntryPlannerPanel" .}}`,
		`{{template "abyssEntryPlannerJS" .}}`,
		`id="lockedCheckpointHint"`,
		`data-danger=`,
		`data-risk=`,
		`id="entryFocus"`,
		`id="entryJackpot"`,
		`{{if not .Unlocked}}disabled{{end}}`,
		`🔒 depth {{.MinBest}}`,
		`Confirm descent costs`,
		`goldCost>1000||tokenCost>0`,
		`{{if .DailyMod}}`,
		`id="consPickOverlay"`,
		`id="consPickSel"`,
		`d.pick_consumables`,
		`id="gateOverlay"`,
		`g.classList.add('closing')`,
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
	for _, required := range []string{".ab-gate.closing", "@media (prefers-reduced-motion: reduce)", ".ab-gate-l"} {
		if !strings.Contains(string(baseCSS), required) {
			t.Errorf("Abyss base CSS is missing %q entry transition contract", required)
		}
	}
}

func TestAbyssConsumableEntryLoadoutValidation(t *testing.T) {
	t.Parallel()

	owned := []consumableOwned{
		{ID: "small_health_potion", Count: 3},
		{ID: "repair_kit", Count: 2},
	}
	tests := []struct {
		name      string
		picked    map[string]int
		maxCarry  int
		want      map[string]int
		wantError string
	}{
		{
			name: "valid selection",
			picked: map[string]int{
				"small_health_potion": 2,
				"repair_kit":           1,
			},
			maxCarry: 3,
			want: map[string]int{
				"small_health_potion": 2,
				"repair_kit":           1,
			},
		},
		{
			name:      "unknown consumable",
			picked:    map[string]int{"forged_client_item": 1},
			maxCarry:  3,
			wantError: "don't own",
		},
		{
			name:      "more than owned",
			picked:    map[string]int{"repair_kit": 3},
			maxCarry:  3,
			wantError: "more than you own",
		},
		{
			name: "over carry capacity",
			picked: map[string]int{
				"small_health_potion": 3,
				"repair_kit":           2,
			},
			maxCarry:  4,
			wantError: "at most 4",
		},
		{
			name:     "zero counts are discarded",
			picked:   map[string]int{"small_health_potion": 0, "repair_kit": -1},
			maxCarry: 3,
			want:     map[string]int{},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, message := abyssBuildConsumableLoadout(test.picked, owned, test.maxCarry)
			if test.wantError != "" {
				if !strings.Contains(strings.ToLower(message), test.wantError) {
					t.Fatalf("error = %q, want substring %q", message, test.wantError)
				}
				return
			}
			if message != "" {
				t.Fatalf("unexpected error: %s", message)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("loadout = %#v, want %#v", got, test.want)
			}
		})
	}
}
