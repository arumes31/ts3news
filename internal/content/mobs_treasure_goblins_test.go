package content

import "testing"

func TestTreasureGoblinVariantCatalog(t *testing.T) {
	if len(TreasureGoblinNames) != 3 {
		t.Fatalf("treasure goblin variants = %d, want 3", len(TreasureGoblinNames))
	}
	want := map[string]bool{"Gem Goblin": true, "Token Goblin": true, "Key Goblin": true}
	for _, name := range TreasureGoblinNames {
		if !want[name] {
			t.Errorf("unexpected treasure goblin variant %q", name)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("missing treasure goblin variants: %#v", want)
	}
}
