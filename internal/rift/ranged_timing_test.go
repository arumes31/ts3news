package rift

import (
	"encoding/json"
	"testing"
)

func TestArchersHaveIndividualInitialAndRepeatCooldowns(t *testing.T) {
	r := testRun()
	r.EncounterPlan = make([][]Actor, len(Rooms))
	for _, id := range []string{"archer-a", "archer-b", "archer-c"} {
		r.EncounterPlan[0] = append(r.EncounterPlan[0], Actor{ID: id, Kind: "archer", X: 500, Y: 410, HP: 100})
	}
	r.spawnRoom()
	seen := map[float64]bool{}
	for i := range r.Enemies {
		initial := r.Enemies[i].Cooldown
		if initial <= 1.5 || initial > 2.15 {
			t.Fatal("initial ranged delay must include arrival grace and a bounded individual offset")
		}
		seen[initial] = true
		r.Enemies[i].Windup = .01
		r.enemyTick(i, .02)
		if r.Enemies[i].Cooldown <= 1.6 || r.Enemies[i].Cooldown > 2.25 {
			t.Fatal("repeat shot has no bounded individual delay")
		}
	}
	if len(seen) != len(r.Enemies) {
		t.Fatal("archers started with synchronized cooldowns")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	restored.Enemies[0], restored.Enemies[2] = restored.Enemies[2], restored.Enemies[0]
	cooldowns := map[string]float64{}
	for _, e := range r.Enemies {
		cooldowns[e.ID] = e.Cooldown
	}
	for i := range restored.Enemies {
		restored.Enemies[i].Windup = .01
		restored.enemyTick(i, .02)
		if restored.Enemies[i].Cooldown != cooldowns[restored.Enemies[i].ID] {
			t.Fatal("cooldown timing changed after save or actor reordering")
		}
	}
}

func TestArcherVolleyReleaseTimesAreStaggered(t *testing.T) {
	r := testRun()
	r.EncounterPlan = make([][]Actor, len(Rooms))
	for _, id := range []string{"archer-a", "archer-b", "archer-c"} {
		r.EncounterPlan[0] = append(r.EncounterPlan[0], Actor{ID: id, Kind: "archer", X: 500, Y: 410, HP: 100})
	}
	r.spawnRoom()
	releases := map[string][]int{}
	for step := 0; step < 400; step++ {
		for i := range r.Enemies {
			before := r.Enemies[i].Attacks
			r.enemyTick(i, .02)
			if r.Enemies[i].Attacks > before {
				releases[r.Enemies[i].ID] = append(releases[r.Enemies[i].ID], step)
			}
		}
	}
	for round := 0; round < 2; round++ {
		times := map[int]bool{}
		for _, e := range r.Enemies {
			if len(releases[e.ID]) <= round {
				t.Fatal("archer failed to release repeated shots")
			}
			times[releases[e.ID][round]] = true
		}
		if len(times) != len(r.Enemies) {
			t.Fatal("actual initial or repeated volley was synchronized")
		}
	}
}
