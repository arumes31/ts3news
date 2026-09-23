package rift

import (
	"encoding/json"
	"testing"
)

func TestOnlyExplicitlyJumpableHazardsAllowJumpEvasion(t *testing.T) {
	for _, jumpable := range []bool{false, true} {
		r := circleTestRun()
		r.RoomObjective = nil
		r.Clock = 1.3
		r.Player.X = 520
		r.Player.Y = 370
		r.Player.Jump = .4
		r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 100, 40}, Kind: "ice", Period: 7, Duration: 1}}}
		r.Level.Rooms[0].Hazards[0].Jumpable = jumpable
		before := r.Player.HP
		r.hazardTick()
		if jumpable && (r.Player.HP != before || r.Stats.HazardContacts != 0 || r.SkillTimers["slowed"] != 0) {
			t.Fatal("jumpable hazard hit airborne player")
		}
		if !jumpable && (r.Player.HP >= before || r.Stats.HazardContacts != 1 || r.SkillTimers["slowed"] != 1.4) {
			t.Fatal("unmarked hazard allowed jump evasion")
		}
	}
}
func TestLegacyHazardsMigrateJumpabilityWithoutChangingExplicitFlags(t *testing.T) {
	for _, kind := range []string{"fire", "ice", "poison", "thorns", "rune", "radiant", "void", "unknown"} {
		for _, flag := range []string{"", `,"jumpable":false`, `,"jumpable":true`} {
			raw := []byte(`{"kind":"` + kind + `","period":7,"duration":1` + flag + `}`)
			var h Hazard
			if err := json.Unmarshal(raw, &h); err != nil {
				t.Fatal(err)
			}
			saved, err := json.Marshal(h)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]any
			if err := json.Unmarshal(saved, &fields); err != nil {
				t.Fatal(err)
			}
			want := flag == `,"jumpable":true` || flag == "" && kind != "unknown"
			if fields["jumpable"] != want {
				t.Fatalf("%s %s: persisted jumpability=%v want=%v", kind, flag, fields["jumpable"], want)
			}
		}
	}
}

func TestCampaignFloorHazardsAreExplicitlyJumpable(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for _, room := range level.Rooms {
			for _, h := range room.Hazards {
				count++
				if !h.Jumpable {
					t.Fatalf("mission %d has unmarked floor hazard", level.ID)
				}
			}
		}
	}
	if count == 0 {
		t.Fatal("campaign has no hazards")
	}
}
