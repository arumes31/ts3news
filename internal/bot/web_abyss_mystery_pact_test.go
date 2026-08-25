package bot

import (
	"bytes"
	"errors"
	"math"
	"testing"
	"time"
)

func TestResolveAbyssMysteryPactExcludesExplicitChoices(t *testing.T) {
	t.Parallel()

	pact, flag, err := resolveAbyssMysteryPact(bytes.NewReader(make([]byte, 32)), []string{"double_hazards"})
	if err != nil {
		t.Fatal(err)
	}
	if pact.Key == "double_hazards" || flag <= 0 {
		t.Fatalf("resolved pact = %#v, flag = %d", pact, flag)
	}
	fromFlag, ok := abyssMysteryPactFromFlags(map[string]int64{abyssRunFlagMysteryPact: int64(flag)})
	if !ok || fromFlag.Key != pact.Key {
		t.Fatalf("flag resolved %#v, %v; want %q", fromFlag, ok, pact.Key)
	}
}

func TestResolveAbyssMysteryPactRejectsUnavailableAndFailedRandomness(t *testing.T) {
	t.Parallel()

	all := make([]string, 0, len(abyssPactCatalog))
	for _, pact := range abyssPactCatalog {
		all = append(all, pact.Key)
	}
	if _, _, err := resolveAbyssMysteryPact(bytes.NewReader(nil), all); err == nil {
		t.Fatal("expected an unavailable-pact error")
	}
	if _, _, err := resolveAbyssMysteryPact(errorReader{}, nil); err == nil {
		t.Fatal("expected a randomness error")
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, errors.New("entropy unavailable") }

func TestAbyssMysteryPactConcealsIdentityAndAddsReward(t *testing.T) {
	t.Parallel()

	hidden := abyssPactCatalog[0]
	flags := map[string]int64{abyssRunFlagMysteryPact: 1}
	visible := abyssVisiblePacts([]string{hidden.Key, "anemic"}, flags)
	if len(visible) != 2 || visible[0].Key != "anemic" || visible[1].Key != abyssMysteryPactKey {
		t.Fatalf("visible pacts = %#v", visible)
	}
	breakdown := abyssPactRewardBreakdownForRunAt(
		[]string{hidden.Key, "anemic"}, nil, "bloodlust", time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC), true,
	)
	if breakdown.MysteryBonusPct != 40 || math.Abs(breakdown.Multiplier-(1+breakdown.TotalBonusPct/100)) > 0.0001 {
		t.Fatalf("mystery breakdown = %#v", breakdown)
	}
	redacted := redactAbyssMysteryPactBreakdown(breakdown, flags)
	for _, line := range redacted.Lines {
		if line.Key == hidden.Key {
			t.Fatalf("hidden pact leaked in %#v", redacted.Lines)
		}
	}
	if reveal := abyssMysteryRevealFromFlags(flags); reveal == nil || reveal.Key != hidden.Key {
		t.Fatalf("reveal = %#v", reveal)
	}
}

func TestAbyssPactBankTokenGrantIsBounded(t *testing.T) {
	t.Parallel()

	cases := []struct {
		floors int
		bonus  float64
		want   int
	}{
		{0, 100, 0},
		{10, 0, 0},
		{10, 30, 1},
		{10, 100, 2},
		{3, 1000, 3},
	}
	for _, tc := range cases {
		if got := abyssPactBankTokenGrant(tc.floors, tc.bonus); got != tc.want {
			t.Errorf("grant(%d, %.1f) = %d, want %d", tc.floors, tc.bonus, got, tc.want)
		}
	}
}

func TestAbyssPactTokenRiskDoesNotLeakHiddenPact(t *testing.T) {
	t.Parallel()

	flags := map[string]int64{abyssRunFlagMysteryPact: 1}
	withLowRewardHidden := abyssPactTokenRiskPct([]string{"double_hazards", "anemic"}, flags)
	flags[abyssRunFlagMysteryPact] = 4
	withHighRewardHidden := abyssPactTokenRiskPct([]string{"glass_cannon", "anemic"}, flags)
	if withLowRewardHidden != 65 || withHighRewardHidden != withLowRewardHidden {
		t.Fatalf("hidden risk quotes = %.1f and %.1f, want identical 65", withLowRewardHidden, withHighRewardHidden)
	}
}

func TestCanonicalAbyssPactRequestKeepsOneMystery(t *testing.T) {
	t.Parallel()

	got := canonicalAbyssPactRequest([]string{"mystery", "anemic", "mystery", "unknown"})
	if len(got) != 2 || got[0] != "anemic" || got[1] != "mystery" {
		t.Fatalf("canonical request = %#v", got)
	}
}
