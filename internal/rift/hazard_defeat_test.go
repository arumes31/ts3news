package rift

import (
	"encoding/json"
	"testing"
)

func TestLethalHazardSourceSurvivesEncounterAndSave(t *testing.T) {
	for _, kind := range []string{"fire", "ice", "poison", "thorns", "rune", "radiant", "void", "collapse"} {
		r := circleTestRun()
		r.RoomObjective = nil
		r.Player.HP = 1
		r.Build.Armor = 0
		r.Player.X = 520
		r.Player.Y = 370
		r.Clock = 1.3
		if kind == "collapse" {
			r.beginCollapseObjective()
			r.Player.X = 50
			r.tickCollapseObjective(4)
		} else {
			r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 100, 40}, Kind: kind, Period: 7, Duration: 1, Jumpable: true}}}
			r.hazardTick()
		}
		r.RecordEncounterSummary("defeated")
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved struct {
			Source *struct {
				Kind     string `json:"kind"`
				Jumpable bool   `json:"jumpable"`
			} `json:"defeated_by_hazard"`
			Encounter struct {
				Source *struct {
					Kind     string `json:"kind"`
					Jumpable bool   `json:"jumpable"`
				} `json:"defeated_by_hazard"`
			} `json:"last_encounter"`
		}
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		if saved.Source == nil || saved.Source.Kind != kind || saved.Source.Jumpable != (kind != "collapse") || saved.Encounter.Source == nil || *saved.Source != *saved.Encounter.Source {
			t.Fatalf("%s lethal source lost", kind)
		}
	}
}

func TestHazardDefeatSourceRequiresLethalContactAndResetsAtEntry(t *testing.T) {
	r := circleTestRun()
	r.Build.Armor = 0
	r.hurtPlayerFromNamedHazard(10, r.Player.X, r.Player.Y, HazardDefeat{Kind: "ice", Jumpable: true})
	if r.DefeatedByHazard != nil {
		t.Fatal("nonlethal contact recorded as defeat")
	}
	r.Player.HP = 1
	r.Barrier = 100
	r.hurtPlayerFromNamedHazard(10, r.Player.X, r.Player.Y, HazardDefeat{Kind: "poison"})
	if r.DefeatedByHazard != nil {
		t.Fatal("barrier contact recorded as defeat")
	}
	r.Barrier = 0
	r.hurtPlayerFromNamedHazard(10, r.Player.X, r.Player.Y, HazardDefeat{Kind: "void"})
	r.hurtPlayerFromNamedHazard(10, r.Player.X, r.Player.Y, HazardDefeat{Kind: "fire", Jumpable: true})
	if r.DefeatedByHazard == nil || r.DefeatedByHazard.Kind != "void" || r.DefeatedByHazard.Jumpable {
		t.Fatal("later contact replaced lethal source")
	}
	r.RecordEncounterSummary("defeated")
	summary := r.LastEncounter
	r.spawnRoom()
	if r.DefeatedByHazard != nil || summary.DefeatedByHazard == nil || summary.DefeatedByHazard.Kind != "void" {
		t.Fatal("room reset retained cause or erased saved summary")
	}
}
