package bot

import "testing"

func TestAbyssBossNamesAtDepthUnlocksTwinTyrantsAfterSixty(t *testing.T) {
	tests := []struct {
		name  string
		depth int
		count int
	}{
		{name: "early boss", depth: 5, count: 1},
		{name: "boundary", depth: 60, count: 1},
		{name: "first twin floor", depth: 65, count: 2},
		{name: "final boss floor", depth: 100, count: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names := abyssBossNamesAtDepth(tt.depth)
			if len(names) != tt.count {
				t.Fatalf("boss count at depth %d = %d, want %d", tt.depth, len(names), tt.count)
			}
			if len(names) == 2 && names[0] == names[1] {
				t.Fatalf("depth %d repeated boss %q", tt.depth, names[0])
			}
		})
	}
}

func TestAbyssBossEncounterBalancesTwinStatBudget(t *testing.T) {
	solo := abyssBossEncounter(60, 100, 1)
	twins := abyssBossEncounter(65, 100, 1)
	if len(solo) != 1 || len(twins) != 2 {
		t.Fatalf("encounter sizes = solo %d, twins %d", len(solo), len(twins))
	}
	if twins[0].Stats.HP*2 <= solo[0].Stats.HP || twins[0].Stats.HP >= solo[0].Stats.HP {
		t.Fatalf("twin HP budget is not between one and two solo bosses: solo=%d twin=%d", solo[0].Stats.HP, twins[0].Stats.HP)
	}
	if twins[0].RewardXP+twins[1].RewardXP <= solo[0].RewardXP {
		t.Fatal("twin encounter did not increase total XP")
	}
}
