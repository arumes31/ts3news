package rift

import (
	"encoding/json"
	"testing"
)

func TestUnownedSkillCastCannotMutateSavedRun(t *testing.T) {
	for _, id := range []string{"unowned-catalog-skill", "CLASS_other_BUILDER", "unowned-ultimate", "fire\x00", ""} {
		t.Run(id, func(t *testing.T) {
			r := testRun()
			r.Build.Signatures = []Skill{{ID: "owned-builder", Role: "builder", Kind: "slash", Power: 1}}
			r.Build.Ultimate = &Skill{ID: "owned-ultimate", Kind: "ultimate", Power: 5}
			r.Player.Mana = 100
			r.Player.Cooldown = 0
			r.Resource = 3
			r.Gold = 50
			r.BankedGold = 100
			r.BankedItems = []string{"Saved loot"}
			r.Drops = []Drop{{ID: "pending", Gold: 50, Collected: true}}
			before, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			r.cast(id)
			after, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatalf("unowned skill %q changed run state", id)
			}
		})
	}
}
