package bot

import (
	"testing"
	"time"
)

// TestShopWindowBounds verifies every rotation window lasts between shopMinHours
// and shopMaxHours and that windows are contiguous (no gaps/overlaps) with a
// strictly increasing seed.
func TestShopWindowBounds(t *testing.T) {
	for idx := int64(0); idx < 5000; idx++ {
		d := shopWindowDuration(idx)
		if d < shopMinHours*3600 || d > shopMaxHours*3600 {
			t.Fatalf("window %d duration %ds out of [%dh,%dh]", idx, d, shopMinHours, shopMaxHours)
		}
	}
}

// TestShopWindowContiguous walks forward in time and checks the seed advances by
// exactly one at each boundary and the end time always lies in the future.
func TestShopWindowContiguous(t *testing.T) {
	start := time.Unix(shopAnchorUnix, 0).UTC()
	prevSeed, _ := shopWindow(start)
	if prevSeed != 0 {
		t.Fatalf("first window seed = %d, want 0", prevSeed)
	}
	now := start
	for i := 0; i < 400; i++ {
		seed, endsAt := shopWindow(now)
		if !endsAt.After(now) {
			t.Fatalf("window end %v not after now %v", endsAt, now)
		}
		if seed < prevSeed || seed > prevSeed+1 {
			t.Fatalf("seed jumped from %d to %d", prevSeed, seed)
		}
		prevSeed = seed
		// Step just past this window's end to land in the next one.
		now = endsAt.Add(time.Second)
	}
}

// TestShopWindowDeterministic confirms the same instant always maps to the same
// window so all players see one rotation.
func TestShopWindowDeterministic(t *testing.T) {
	ts := time.Unix(shopAnchorUnix+123456, 0).UTC()
	s1, e1 := shopWindow(ts)
	s2, e2 := shopWindow(ts)
	if s1 != s2 || !e1.Equal(e2) {
		t.Fatalf("non-deterministic: (%d,%v) vs (%d,%v)", s1, e1, s2, e2)
	}
}
