package bot

import (
	"math"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAbyssAffixCalendarUsesAuthoritativeDailySelector(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.August, 25, 18, 30, 0, 0, time.FixedZone("CEST", 2*60*60))
	days := abyssAffixCalendar(now)
	if len(days) != 7 {
		t.Fatalf("calendar length = %d, want 7", len(days))
	}
	if days[0].Date != "2026-08-24" || days[0].Weekday != "Mon" || days[6].Date != "2026-08-30" {
		t.Fatalf("calendar bounds = %#v .. %#v", days[0], days[6])
	}
	todayCount := 0
	for _, day := range days {
		parsed, err := time.Parse("2006-01-02", day.Date)
		if err != nil {
			t.Fatal(err)
		}
		_, wantKey := abyssDailyChallengeAt(parsed)
		if day.Key != wantKey || day.Label != abyssDailyAffixLabel(wantKey) {
			t.Errorf("day %s = %q/%q, want %q/%q", day.Date, day.Key, day.Label, wantKey, abyssDailyAffixLabel(wantKey))
		}
		if day.Today {
			todayCount++
		}
	}
	if todayCount != 1 || !days[1].Today {
		t.Fatalf("today markers = %d, Tuesday = %v", todayCount, days[1].Today)
	}
}

func TestAbyssFeaturedPactIsStableForISOWeek(t *testing.T) {
	t.Parallel()

	monday := time.Date(2026, time.August, 24, 0, 0, 0, 0, time.UTC)
	first := abyssFeaturedPactAt(monday)
	if first.Key == "" || first.Label == "" || first.Week != "2026-W35" {
		t.Fatalf("featured pact = %#v", first)
	}
	if later := abyssFeaturedPactAt(monday.AddDate(0, 0, 6)); later != first {
		t.Fatalf("featured pact changed within ISO week: %#v -> %#v", first, later)
	}
	if next := abyssFeaturedPactAt(monday.AddDate(0, 0, 7)); next.Key == first.Key || next.Week == first.Week {
		t.Fatalf("featured pact did not rotate next week: %#v -> %#v", first, next)
	}
}

func TestAbyssPactRewardBreakdownCombinesMasteryFeaturedAndSynergy(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, time.August, 25, 0, 0, 0, 0, time.UTC)
	featured := abyssFeaturedPactAt(at)
	pacts := []string{featured.Key, "anemic", "anemic", "unknown"}
	mastery := map[string]int{featured.Key: abyssPactMasteryRuns}
	breakdown := abyssPactRewardBreakdownAt(pacts, mastery, "bloodlust", at)
	if len(breakdown.Synergies) != 1 || breakdown.SynergyBonusPct != 5 {
		t.Fatalf("synergy breakdown = %#v", breakdown.Synergies)
	}
	featuredLines := 0
	for _, line := range breakdown.Lines {
		if line.Featured {
			featuredLines++
			if !line.Mastered || line.MasteryBonusPct <= 0 || line.FeaturedBonusPct <= 0 {
				t.Fatalf("featured line = %#v", line)
			}
		}
	}
	if featuredLines != 1 {
		t.Fatalf("featured lines = %d, want 1", featuredLines)
	}
	if math.Abs(breakdown.Multiplier-(1+breakdown.TotalBonusPct/100)) > 0.0001 {
		t.Fatalf("multiplier %v disagrees with total %v%%", breakdown.Multiplier, breakdown.TotalBonusPct)
	}
}

func TestAbyssPactCampaignClientAndRewardContract(t *testing.T) {
	t.Parallel()

	module, err := webAssets.ReadFile("webassets/abyss_pact_program.html")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	server, err := os.ReadFile("web_abyss.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"affixCalendar", "pactFeatured", "pactSynergyCallout", "renderAbyssPactSelectionBonus"} {
		if !strings.Contains(string(module), required) {
			t.Errorf("pact campaign module is missing %q", required)
		}
	}
	for _, required := range []string{"pactMath(p.pact_breakdown)", "Total pact multiplier", "mastery", "featured"} {
		if !strings.Contains(string(page), required) {
			t.Errorf("bank confirmation is missing %q", required)
		}
	}
	serverText := string(server)
	if strings.Count(serverText, "abyssPactRewardBreakdownAt(") < 3 {
		t.Error("combat, non-combat, and bank preview do not share the authoritative pact breakdown")
	}
	if !strings.Contains(serverText, `"pact_breakdown": pactBreakdown`) {
		t.Error("bank preview does not return its authoritative pact breakdown")
	}
}
