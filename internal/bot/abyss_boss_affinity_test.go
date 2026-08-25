package bot

import (
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestAbyssDailyBossAffinityRotatesOnUTCDate(t *testing.T) {
	start := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
	seen := make(map[content.Element]bool)
	for day := 0; day < 4; day++ {
		affinity := abyssDailyBossAffinity(start.AddDate(0, 0, day))
		seen[affinity.Element] = true
		if affinity.WeakTo == "" || affinity.StrongAgainst == "" || affinity.WeakTo == affinity.StrongAgainst {
			t.Fatalf("incomplete affinity: %+v", affinity)
		}
	}
	if len(seen) != 4 {
		t.Fatalf("four UTC days produced %d affinities", len(seen))
	}
	beforeMidnight := abyssDailyBossAffinity(start.Add(23*time.Hour + 59*time.Minute))
	if afterMidnight := abyssDailyBossAffinity(start.Add(24 * time.Hour)); afterMidnight.Element == beforeMidnight.Element {
		t.Fatal("affinity did not rotate at UTC midnight")
	}
}

func TestAbyssBossAffinityForecastUIContract(t *testing.T) {
	page, err := webAssets.ReadFile("webassets/abyss_boss_affinity.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_boss_affinity.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"TODAY'S BOSS AFFINITY · UTC", "for 2× damage", "at ½ damage", "Twin Tyrants", "TargetDepth"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("affinity forecast is missing %q", token)
		}
	}
	for _, token := range []string{".is-fire", ".is-water", ".is-earth", ".is-air", "@media (max-width: 620px)"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("affinity styles are missing %q", token)
		}
	}
}

func TestAbyssBossAffinityForecastShowsNextEncounter(t *testing.T) {
	view := abyssBossAffinityForecast(abyssRun{Active: true, Depth: 62}, time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC))
	if view.TargetDepth != 65 || !view.TwinBosses || view.Element == "" || view.WeakTo == "" {
		t.Fatalf("forecast = %+v", view)
	}
}

func TestAbyssBossEncounterAppliesDailyAffinity(t *testing.T) {
	now := time.Date(2026, time.August, 25, 12, 0, 0, 0, time.UTC)
	want := abyssDailyBossAffinity(now).Element
	mobs := abyssBossEncounterAt(65, 100, 1, now)
	if len(mobs) != 2 {
		t.Fatalf("boss count = %d", len(mobs))
	}
	for _, mob := range mobs {
		if mob.Element != want {
			t.Errorf("%s element = %s, want %s", mob.Name, mob.Element, want)
		}
	}
}
