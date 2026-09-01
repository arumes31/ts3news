package bot

import (
	"math"
	"strings"
	"testing"
)

func TestAbyssCombatSidesAndCrowdedTargetingContract(t *testing.T) {
	live, err := webAssets.ReadFile("webassets/abyss_live.html")
	if err != nil {
		t.Fatal(err)
	}
	pixel, err := webAssets.ReadFile("webassets/abyss_pixel.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_pixel.css")
	if err != nil {
		t.Fatal(err)
	}
	source := string(live)
	if strings.Index(source, `id="livePixelAllies"`) > strings.Index(source, `id="livePixelEnemies"`) {
		t.Fatal("battlefield must render the party left of hostiles")
	}
	if strings.Index(source, `id="liveAllies"`) > strings.Index(source, `id="liveEnemies"`) {
		t.Fatal("tactical targets must render allies before enemies")
	}
	for _, token := range []string{"host.classList.toggle('crowded'", "selectLiveTarget(unit.id)", ".ab-pixel-party.crowded", "flex-wrap: wrap", "#livePixelAllies { grid-column: 1", "#livePixelEnemies { grid-column: 2", "align-content: start"} {
		if !strings.Contains(string(pixel)+string(styles), token) {
			t.Errorf("crowded targeting contract is missing %q", token)
		}
	}
	for _, token := range []string{"window.AB_EXACT_ICON_MANIFEST", "exact.atlas='catalog'", "ab-semantic-action-icon", "liveActionIconCell(option)", "exactArt?exactArt.family:'actions'"} {
		if !strings.Contains(string(pixel)+source+string(styles), token) {
			t.Errorf("semantic combat art contract is missing %q", token)
		}
	}
}

func TestAbyssEventStageAndBossPanelsStayScoped(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	events, err := webAssets.ReadFile("webassets/abyss_events.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_command.css")
	if err != nil {
		t.Fatal(err)
	}
	record, err := webAssets.ReadFile("webassets/abyss_best_kill.html")
	if err != nil {
		t.Fatal(err)
	}
	cosmetics, err := webAssets.ReadFile("webassets/abyss_boss_cosmetics.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"setAbyssEventStage(true)", "setAbyssEventStage(false)", ".abyss-stage.ab-event-stage", ".ab-event-stage .ab-elevator { display: none", "Number.isFinite(multiplier)"} {
		if !strings.Contains(string(page)+string(events)+string(styles), token) {
			t.Errorf("compact event-stage contract is missing %q", token)
		}
	}
	if !strings.Contains(string(record), `data-abyss-section="leaderboards"`) {
		t.Error("personal boss record must be scoped to Leaderboards")
	}
	if !strings.Contains(string(cosmetics), `data-abyss-section="progression"`) {
		t.Error("boss cosmetics must be scoped to Progression")
	}
}

func TestAbyssPlannedDescentControlsContract(t *testing.T) {
	if abyssDescendPlanMin != 3 || abyssDescendPlanMax != 20 {
		t.Fatalf("planned descent bounds = %d..%d, want 3..20", abyssDescendPlanMin, abyssDescendPlanMax)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_command.css")
	if err != nil {
		t.Fatal(err)
	}
	autoStyles, err := webAssets.ReadFile("webassets/abyss_auto_descend.css")
	if err != nil {
		t.Fatal(err)
	}
	source := string(page)
	for _, token := range []string{"3-20 paths", "count >= 20", "paths.length > 20", "floor_results", "abyss:batch-floor", "abyssFloatingDescend", "focusAbyssDescend", "autoDescendRules", "stop_rules", "auto_stopped", "Legendary+ secured"} {
		if !strings.Contains(source, token) {
			t.Errorf("planned descent UI is missing %q", token)
		}
	}
	for _, token := range []string{".ab-path-queue-editor", "#btnQueueMore", ".ab-floating-descend"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("planned descent CSS is missing %q", token)
		}
	}
	for _, token := range []string{".ab-auto-stop", "@media (max-width: 520px)"} {
		if !strings.Contains(string(autoStyles), token) {
			t.Errorf("auto-descend CSS is missing %q", token)
		}
	}
}

func TestAbyssCombatCockpitContract(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	live, err := webAssets.ReadFile("webassets/abyss_live.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_command.css")
	if err != nil {
		t.Fatal(err)
	}

	for _, token := range []string{
		`id="abyssCombatCockpit"`,
		"function focusAbyssCockpit",
		"function scheduleAbyssCockpitFocus",
		"ab_focus_cockpit",
		"cockpit.classList.add('is-playing')",
		"scheduleAbyssCockpitFocus('btnDescend')",
	} {
		if !strings.Contains(string(page), token) {
			t.Errorf("combat cockpit page contract is missing %q", token)
		}
	}
	for _, token := range []string{
		"updateAbyssCombatCockpit(true)",
		"scheduleAbyssCockpitFocus('liveAction',true)",
		"updateAbyssCombatCockpit(false)",
	} {
		if !strings.Contains(string(live), token) {
			t.Errorf("live combat cockpit contract is missing %q", token)
		}
	}
	for _, token := range []string{
		".ab-combat-cockpit.is-active",
		"height: calc(100dvh",
		"grid-template-rows:",
		".ab-combat-cockpit.has-live-combat",
		"@media (max-width: 900px), (max-height: 899px)",
	} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("combat cockpit CSS contract is missing %q", token)
		}
	}
}

func TestAbyssCursedElevatorContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		roll float64
		want bool
	}{
		{name: "first roll", roll: 0, want: true},
		{name: "inside five percent", roll: 0.049999, want: true},
		{name: "boundary excluded", roll: abyssCursedElevatorChance, want: false},
		{name: "ordinary roll", roll: 0.5, want: false},
		{name: "invalid negative", roll: -0.01, want: false},
		{name: "invalid NaN", roll: math.NaN(), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := abyssCursedElevatorTriggered(tt.roll); got != tt.want {
				t.Fatalf("abyssCursedElevatorTriggered(%v) = %t, want %t", tt.roll, got, tt.want)
			}
		})
	}

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"d.cursed_elevator", "Cursed elevator · 0/2", "playAbyssMultiFloorResults(d)"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("cursed elevator playback contract is missing %q", token)
		}
	}
}
