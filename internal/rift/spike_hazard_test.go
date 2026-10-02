package rift

import "testing"

func TestBarracksSpikeBedsAndJumpCounterplay(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		if level.Region != 6 {
			continue
		}
		h := level.Rooms[0].Hazards[0]
		if h.Kind != "spikes" || !h.Jumpable {
			t.Fatalf("mission %d missing spike bed", level.ID)
		}
		count++
		for _, jump := range []float64{0, .4} {
			r := circleTestRun()
			r.RoomObjective = nil
			r.Level.Rooms[0] = Arena{Hazards: []Hazard{h}}
			r.Player.X = h.X + h.W/2
			r.Player.Y = h.Y + h.H/2
			r.Player.Jump = jump
			r.Clock = h.Period - h.Offset + 1.3
			r.Build.Armor = 0
			hp := r.Player.HP
			r.hazardTick()
			if (r.Player.HP < hp) != (jump == 0) {
				t.Fatal("incorrect spike jump counterplay")
			}
			if r.SkillTimers["slowed"] != 0 {
				t.Fatal("spikes incorrectly slow")
			}
		}
	}
	if count != 10 {
		t.Fatal("missing barracks spike rooms")
	}
}
