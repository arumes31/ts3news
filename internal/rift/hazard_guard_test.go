package rift

import (
	"math"
	"testing"
)

func TestDocumentedGuardInteractionsForEveryFloorHazard(t *testing.T) {
	for _, kind := range []string{"fire", "ice", "poison", "thorns", "rune", "radiant", "void"} {
		for _, facing := range []float64{-1, 1} {
			r := circleTestRun()
			r.RoomObjective = nil
			r.Build.Armor = 5
			r.Clock = 1.3
			r.Player.X = 520
			r.Player.Y = 370
			r.Player.Facing = facing
			r.Player.Guard = true
			r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 100, 40}, Kind: kind, Period: 7, Duration: 1}}}
			before := r.Player.HP
			r.hazardTick()
			if math.Abs(before-r.Player.HP-1.8) > 1e-8 || math.Abs(r.Stats.GuardBlocked-8.2) > 1e-8 {
				t.Fatalf("%s facing %v does not reduce damage 82%% after armor", kind, facing)
			}
			slow := kind == "ice" || kind == "poison" || kind == "thorns"
			if (r.SkillTimers["slowed"] == 1.4) != slow {
				t.Fatalf("%s guard changed slowing behavior", kind)
			}
			if kind == "void" && r.Player.X <= 520 {
				t.Fatal("guard incorrectly prevented void pull")
			}
			if r.Stats.HazardContacts != 1 {
				t.Fatal("guard prevented hazard contact")
			}
		}
	}
}
func TestDocumentedJumpDistinguishesFloorHazardsFromCollapse(t *testing.T) {
	for _, kind := range []string{"fire", "ice", "poison", "thorns", "rune", "radiant", "void"} {
		r := circleTestRun()
		r.Clock = 1.3
		r.Player.X = 520
		r.Player.Y = 370
		r.Player.Jump = .4
		r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 100, 40}, Kind: kind, Period: 7, Duration: 1}}}
		before := r.Player.HP
		r.hazardTick()
		if r.Player.HP != before || r.Stats.HazardContacts != 0 || r.SkillTimers["slowed"] != 0 || r.Player.X != 520 {
			t.Fatalf("%s jump did not evade hazard", kind)
		}
	}
	r := collapseTestRun()
	r.Player.X = 50
	r.Player.Jump = .4
	r.Player.Guard = true
	r.Build.Armor = 0
	before := r.Player.HP
	incoming := math.Max(15, r.Player.MaxHP*.12)
	r.tickCollapseObjective(4)
	if math.Abs(before-r.Player.HP-incoming*.18) > 1e-8 {
		t.Fatal("collapse guard/jump guidance no longer matches damage")
	}
}
