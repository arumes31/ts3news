package bot

import (
	"testing"

	"ts3news/internal/rift"
)

func TestRiftDecodeRejectsInvalidCooldowns(t *testing.T) {
	changes := map[string]func(*rift.Run){
		"player":            func(r *rift.Run) { r.Player.Cooldown = -1 },
		"enemy":             func(r *rift.Run) { r.Enemies[0].Cooldown = -1 },
		"future encounter":  func(r *rift.Run) { r.EncounterPlan[0][0].Cooldown = -1 },
		"boss reset":        func(r *rift.Run) { r.Practice.BossStart.Cooldown = -1 },
		"lantern":           func(r *rift.Run) { r.RoomObjective.Lantern.Cooldown = -1 },
		"escort":            func(r *rift.Run) { r.RoomObjective.Escort.Cooldown = -1 },
		"ward":              func(r *rift.Run) { r.RoomObjective.Lanes[0].Ward.Cooldown = -1 },
		"wave":              func(r *rift.Run) { r.RoomObjective.Waves[0][0].Cooldown = -1 },
		"huge actor":        func(r *rift.Run) { r.Player.Cooldown = 1e100 },
		"negative skill":    func(r *rift.Run) { r.SkillTimers["fire"] = -.01 },
		"huge skill":        func(r *rift.Run) { r.SkillTimers["fire"] = 1e100 },
		"negative build":    func(r *rift.Run) { r.Build.Skills = []rift.Skill{{ID: "fire", Cooldown: -1}} },
		"huge signature":    func(r *rift.Run) { r.Build.Signatures = []rift.Skill{{ID: "fire", Cooldown: 1e100}} },
		"negative ultimate": func(r *rift.Run) { r.Build.Ultimate = &rift.Skill{ID: "ult", Cooldown: -1} },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			r := riftVitalsFixture()
			change(r)
			saved, err := encodeRift(r)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := decodeRift(saved); err == nil || got != nil {
				t.Fatal("invalid cooldown accepted")
			}
		})
	}
}

func TestRiftDecodePreservesCooldowns(t *testing.T) {
	r := riftVitalsFixture()
	r.Player.Cooldown = .35
	r.Enemies[0].Cooldown = 2.3
	r.Build.Ultimate = &rift.Skill{ID: "long-ultimate", Cooldown: 120}
	r.SkillTimers = map[string]float64{"long-ultimate": 119.5, "room_entry_grace": 1.2, "ready": 0}
	saved, err := encodeRift(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeRift(saved)
	if err != nil {
		t.Fatal(err)
	}
	if got.Player.Cooldown != .35 || got.Enemies[0].Cooldown != 2.3 || got.SkillTimers["long-ultimate"] != 119.5 || got.SkillTimers["room_entry_grace"] != 1.2 || got.SkillTimers["ready"] != 0 {
		t.Fatal("valid cooldown changed")
	}
}
