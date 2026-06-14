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
	for _, name := range []string{"armory", "inventory", "battle", "arcade", "shop", "ah", "denied", "head", "foot"} {
		if ws.tmpl.Lookup(name) == nil {
			t.Errorf("template %q not defined", name)
		}
	}
}

func TestCombineUnits(t *testing.T) {
	st := &tftState{Units: []tftUnit{
		{ID: "a", Key: "brute", Star: 1, Pos: -1},
		{ID: "b", Key: "brute", Star: 1, Pos: -1},
		{ID: "c", Key: "brute", Star: 1, Pos: -1},
		{ID: "d", Key: "archer", Star: 1, Pos: -1},
	}}
	combineUnits(st)
	if len(st.Units) != 2 {
		t.Fatalf("expected 2 units after combine, got %d", len(st.Units))
	}
	var brute *tftUnit
	for i := range st.Units {
		if st.Units[i].Key == "brute" {
			brute = &st.Units[i]
		}
	}
	if brute == nil || brute.Star != 2 {
		t.Errorf("expected a 2-star brute, got %+v", brute)
	}
}

func TestStarStatsScale(t *testing.T) {
	d, _ := tftDefByKey("brute")
	hp1, atk1 := starStats(d, 1)
	hp2, atk2 := starStats(d, 2)
	if hp2 <= hp1 || atk2 <= atk1 {
		t.Errorf("star-2 should be stronger: hp %d->%d atk %d->%d", hp1, hp2, atk1, atk2)
	}
}

func TestRollShopSize(t *testing.T) {
	if got := len(rollShop()); got != tftShopSize {
		t.Errorf("rollShop len = %d, want %d", got, tftShopSize)
	}
}

func TestSimulateTFT_PlayerWins(t *testing.T) {
	units := []*simUnit{
		{id: "p1", icon: "🪓", side: "you", star: 3, pos: 21, hp: 5000, maxhp: 5000, atk: 800, rng: 1},
		{id: "e1", icon: "👹", side: "enemy", star: 1, pos: 3, hp: 100, maxhp: 100, atk: 5, rng: 1},
	}
	frames, victory := simulateTFT(units)
	if !victory {
		t.Errorf("expected player victory")
	}
	if len(frames) < 2 {
		t.Errorf("expected multiple frames, got %d", len(frames))
	}
}

func TestSimulateTFT_PlayerLoses(t *testing.T) {
	units := []*simUnit{
		{id: "p1", icon: "🪓", side: "you", star: 1, pos: 21, hp: 50, maxhp: 50, atk: 5, rng: 1},
		{id: "e1", icon: "👹", side: "enemy", star: 1, pos: 3, hp: 5000, maxhp: 5000, atk: 800, rng: 1},
	}
	_, victory := simulateTFT(units)
	if victory {
		t.Errorf("expected player defeat")
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
	allowed := map[int64]bool{0: true, 200: true, 500: true, 2500: true} // bet=100
	for i := 0; i < 2000; i++ {
		_, pay, _ := playSlots(rng, 100)
		if !allowed[pay] {
			t.Fatalf("unexpected slots payout %d", pay)
		}
	}
}
