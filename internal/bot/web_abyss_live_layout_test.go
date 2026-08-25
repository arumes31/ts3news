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
	if strings.Index(source, `id="livePixelEnemies"`) > strings.Index(source, `id="livePixelAllies"`) {
		t.Fatal("battlefield must render hostiles left of allies")
	}
	if strings.Index(source, `id="liveEnemies"`) > strings.Index(source, `id="liveAllies"`) {
		t.Fatal("tactical targets must render enemies before allies")
	}
	for _, token := range []string{"host.classList.toggle('crowded'", "selectLiveTarget(unit.id)", ".ab-pixel-party.crowded", "flex-wrap: wrap", "#livePixelEnemies { grid-column: 1", "#livePixelAllies { grid-column: 2"} {
		if !strings.Contains(string(pixel)+string(styles), token) {
			t.Errorf("crowded targeting contract is missing %q", token)
		}
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
	source := string(page)
	for _, token := range []string{"3-20 paths", "count >= 20", "paths.length > 20", "floor_results", "abyss:batch-floor", "abyssFloatingDescend", "focusAbyssDescend"} {
		if !strings.Contains(source, token) {
			t.Errorf("planned descent UI is missing %q", token)
		}
	}
	for _, token := range []string{".ab-path-queue-editor", "#btnQueueMore", ".ab-floating-descend"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("planned descent CSS is missing %q", token)
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
