package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestClearedRoomCelebrationRemainsSafeAcrossHazardPulsesAndReload(t *testing.T) {
	for _, kind := range []string{"fire", "ice", "poison", "thorns", "rune", "radiant", "void", "collapse"} {
		t.Run(kind, func(t *testing.T) {
			r := circleTestRun()
			r.RoomObjective = nil
			r.Enemies = nil
			r.Build.Armor = 0
			r.Player.X, r.Player.Y = 520, 370
			r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 100, 40}, Kind: kind, Period: 7, Duration: 1}}}
			if kind == "collapse" {
				r.Level.Rooms[0].Hazards = nil
				r.beginCollapseObjective()
				r.Player.X, r.Player.Y = r.RoomObjective.Zone.X, r.RoomObjective.Zone.Y
			}
			// Clear through the real transition while the floor pulse is still warning.
			r.Step(Input{}, time.UnixMilli(r.LastMS+20))
			if r.Status != "cleared" {
				t.Fatalf("room did not clear: %s", r.Status)
			}
			r.Player.HP = 1 // Even a single accidental hit would end this celebration.
			if kind == "collapse" {
				r.Player.X = 50
			}
			x, y := r.Player.X, r.Player.Y
			contacts, damage := r.Stats.HazardContacts, r.Stats.HazardDamageTaken
			for frame := 0; frame < 800; frame++ {
				if frame == 400 {
					raw, err := json.Marshal(r)
					if err != nil {
						t.Fatal(err)
					}
					var saved Run
					if err = json.Unmarshal(raw, &saved); err != nil {
						t.Fatal(err)
					}
					r = &saved
				}
				r.Step(Input{}, time.UnixMilli(r.LastMS+20))
				if r.Status != "cleared" || r.Player.HP != 1 || r.Player.X != x || r.Player.Y != y || r.SkillTimers["slowed"] != 0 || r.Stats.HazardContacts != contacts || r.Stats.HazardDamageTaken != damage {
					t.Fatalf("unsafe celebration at frame %d: status=%s hp=%v position=%v,%v", frame, r.Status, r.Player.HP, r.Player.X, r.Player.Y)
				}
			}
			// Positive control: the same zone still hurts during an unfinished encounter.
			r.Status = "fighting"
			r.Player.HP = r.Player.MaxHP
			before := r.Player.HP
			if kind == "collapse" {
				r.RoomObjective.Complete = false
				r.tickCollapseObjective(4)
			} else {
				r.Clock = 1.3
				r.hazardTick()
			}
			if r.Player.HP >= before {
				t.Fatal("positive control hazard did not damage player")
			}
		})
	}
}
