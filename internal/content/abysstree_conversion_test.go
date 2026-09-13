package content

import "testing"

func TestCombatConversionsPreserveAtLeastHalfOfEachSource(t *testing.T) {
	stats := Stats{HP: 100, STR: 100, SPD: 100, INT: 100}
	got := (TreeBonus{Pct: map[string]float64{"str_to_spd": 1.2, "hp_to_def": 3, "spd_to_dge": 1.2, "int_to_mna": 2}}).ApplyCombatPct(stats)
	if got.HP != 50 || got.STR != 50 || got.SPD != 75 || got.INT != 50 || got.DEF != 5 || got.DGE != 75 || got.MNA != 250 {
		t.Fatalf("uncapped conversion: %+v", got)
	}
}
func TestCombatConversionsKeepValidStats(t *testing.T) {
	got := (TreeBonus{Pct: map[string]float64{"spd_to_dge": -2, "hp_pct": -2, "limit_break": -2}}).ApplyCombatPct(Stats{HP: 10, SPD: 10, STR: 1, DEF: -4, INT: 1})
	if got.HP < 1 || got.SPD < 0 || got.STR < 0 || got.DEF < 0 || got.DGE < 0 || got.INT < 0 {
		t.Fatalf("invalid stats: %+v", got)
	}
}
