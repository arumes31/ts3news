package bot

import (
	"math/rand/v2"
	"testing"
)

func TestNewWebServerTemplatesParse(t *testing.T) {
	ws, err := NewWebServer(&Bot{})
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	for _, name := range []string{"armory", "inventory", "arcade", "shop", "ah", "denied", "head", "foot"} {
		if ws.tmpl.Lookup(name) == nil {
			t.Errorf("template %q not defined", name)
		}
	}
}


func TestPlayArcadeUnknownGame(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	out := playArcade(rng, "nope", 100, "")
	if out.OK {
		t.Errorf("unknown game should not be OK")
	}
}

func TestPlayWheelPayoutMatchesMult(t *testing.T) {
	rng := rand.New(rand.NewPCG(7, 9))
	seg, mult, pay, _ := playWheel(rng, 1000)
	if seg < 0 || seg >= len(wheelSegments) {
		t.Fatalf("segment out of range: %d", seg)
	}
	if want := int64(float64(1000) * mult); pay != want {
		t.Errorf("payout %d != bet*mult %d", pay, want)
	}
}

func TestPlaySlotsPayoutTiers(t *testing.T) {
	// Run many spins; verify payouts are only ever from the allowed set.
	rng := rand.New(rand.NewPCG(42, 42))
	// bet=100 → 3-kind ×3, 4-kind ×16, 5-kind ×88 (see playSlots), or no match.
	allowed := map[int64]bool{0: true, 300: true, 1600: true, 8800: true}
	for i := 0; i < 2000; i++ {
		_, pay, _ := playSlots(rng, 100)
		if !allowed[pay] {
			t.Fatalf("unexpected slots payout %d", pay)
		}
	}
}
