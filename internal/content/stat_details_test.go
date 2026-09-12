package content

import "testing"

func TestStatDetailsCoverEveryAttributeInStableOrder(t *testing.T) {
	s := Stats{HP: 1, MNA: 2, STR: 3, DEF: 4, SPD: 5, CRT: 6, DGE: 7, LCK: 8, INT: 9, STA: 10, CHA: 11, STN: 12, SHN: 13, HGR: 14}
	values := s.Details()
	if len(values) != 14 {
		t.Fatalf("got %d stats", len(values))
	}
	for i, value := range values {
		if value.Value != i+1 || value.Description == "" || value.Code == "" {
			t.Fatalf("stat %d: %+v", i, value)
		}
		if value.Combat != (i < 10) {
			t.Fatalf("wrong combat classification: %+v", value)
		}
	}
}

func TestStatPowerIncludesManaWithoutRarityPremium(t *testing.T) {
	if got := (Stats{STR: 10, MNA: 1}).Power(); got != 12.1 {
		t.Fatalf("power = %v, want 12.1", got)
	}
	if got := (Stats{CHA: 1000}).Power(); got != 0 {
		t.Fatalf("flavour power = %v", got)
	}
}
