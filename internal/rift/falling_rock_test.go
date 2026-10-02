package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func rockTestRun() *Run {
	r := circleTestRun()
	r.RoomObjective = nil
	r.Build.Armor = 0
	r.SkillTimers = map[string]float64{}
	r.Player.X, r.Player.Y = 540, 380
	r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 100, 60}, Kind: "falling_rock", Period: 7, Duration: .18}}}
	return r
}

func TestFallingRockWarningImpactAndRecovery(t *testing.T) {
	r := rockTestRun()
	hp := r.Player.HP
	seen := map[int]bool{}
	counts := map[string]int{}
	for i := 0; i < 100; i++ {
		r.Step(Input{}, time.UnixMilli(r.LastMS+20))
		if r.Clock < 1.2 && r.Player.HP != hp {
			t.Fatal("damage before impact")
		}
		for _, e := range r.Events {
			if !seen[e.ID] {
				seen[e.ID] = true
				counts[e.Kind]++
			}
		}
	}
	if r.Stats.HazardContacts != 1 || counts["rock_warning"] != 1 || counts["rock_impact"] != 1 {
		t.Fatalf("contacts=%v cues=%v", r.Stats.HazardContacts, counts)
	}
	hp = r.Player.HP
	r.Clock = 3
	r.hazardTick()
	if r.Player.HP != hp {
		t.Fatal("rubble dealt damage")
	}
}

func TestFallingRockEvasionAndSave(t *testing.T) {
	for _, mode := range []string{"jump", "outside", "dodge", "guard"} {
		t.Run(mode, func(t *testing.T) {
			r := rockTestRun()
			r.Clock = 1.21
			hp := r.Player.HP
			switch mode {
			case "jump":
				r.Player.Jump = .4
			case "outside":
				r.Player.X = 620
			case "dodge":
				r.SkillTimers["dodge_invulnerability"] = .3
			case "guard":
				r.Player.Guard = true
			}
			r.hazardTick()
			if (r.Player.HP < hp) != (mode == "jump" || mode == "guard") {
				t.Fatalf("wrong %s counterplay", mode)
			}
		})
	}
	r := rockTestRun()
	r.Clock = 1.1
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		now := time.UnixMilli(r.LastMS + 20)
		r.Step(Input{}, now)
		saved.Step(Input{}, now)
		if r.Player.HP != saved.Player.HP || r.Clock != saved.Clock {
			t.Fatal("saved rock timing diverged")
		}
	}
}

func TestCampaignFallingRocksHaveExplicitOverheadCounterplay(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for _, arena := range level.Rooms {
			for _, h := range arena.Hazards {
				if h.Kind == "falling_rock" {
					count++
					if h.Jumpable || h.Duration != .18 {
						t.Fatal("rock must have brief overhead impact")
					}
				}
			}
		}
	}
	if count != 10 {
		t.Fatalf("falling rock arenas=%d", count)
	}
}
