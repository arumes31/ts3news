package rift

import (
	"encoding/json"
	"testing"
)

func TestSlowSourceFollowsAppliedHazardAndPersists(t *testing.T) {
	r := circleTestRun()
	r.RoomObjective = nil
	r.Player.X = 520
	r.Player.Y = 370
	r.Clock = 1.3
	for _, kind := range []string{"ice", "poison", "thorns"} {
		r.SkillTimers = map[string]float64{}
		r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 100, 40}, Kind: kind, Period: 7, Duration: 1}}}
		r.hazardTick()
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved struct {
			Source string `json:"slow_source"`
		}
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		if saved.Source != kind {
			t.Fatalf("slow source=%q want %q", saved.Source, kind)
		}
	}
}

func TestSlowSourceClearsOnExpiryAndRoomEntry(t *testing.T) {
	for _, mode := range []string{"expire", "room"} {
		r := circleTestRun()
		r.SlowSource = "poison"
		r.SkillTimers["slowed"] = .01
		if mode == "expire" {
			r.tick(Input{}, .02)
		} else {
			r.spawnRoom()
		}
		if r.SlowSource != "" {
			t.Fatalf("%s retained source", mode)
		}
	}
}
func TestAvoidedAndCooldownHazardsDoNotReplaceSlowSource(t *testing.T) {
	for _, mode := range []string{"jump", "cooldown", "disabled"} {
		r := circleTestRun()
		r.SlowSource = "ice"
		r.SkillTimers["slowed"] = 1
		r.Clock = 1.3
		r.Player.X = 520
		r.Player.Y = 370
		r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 100, 40}, Kind: "poison", Period: 7, Duration: 1, Jumpable: true, Disabled: mode == "disabled"}}}
		if mode == "jump" {
			r.Player.Jump = .5
		}
		if mode == "cooldown" {
			r.SkillTimers["hazard-hit"] = 1
		}
		r.hazardTick()
		if r.SlowSource != "ice" || r.SkillTimers["slowed"] != 1 {
			t.Fatalf("%s changed slow source", mode)
		}
	}
}
