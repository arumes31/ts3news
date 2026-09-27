package bot

import (
	"testing"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/rift"
)

func riftVitalsFixture() *rift.Run {
	r := rift.NewRun("vitals", rift.Build{HP: 100}, time.Now())
	a := rift.Actor{HP: 100, MaxHP: 100, Mana: 100}
	r.Enemies = []rift.Actor{a}
	r.EncounterPlan = [][]rift.Actor{{a}}
	lantern, escort := a, a
	boss := a
	r.Practice = &rift.PracticeState{BossStart: &boss}
	r.RoomObjective = &rift.RoomObjective{Lantern: &lantern, Escort: &escort, Lanes: []rift.DefenseLane{{Ward: a}}, Waves: [][]rift.Actor{{a}}}
	return r
}
func TestRiftDecodeRejectsInvalidVitals(t *testing.T) {
	selectors := map[string]func(*rift.Run) *rift.Actor{
		"practice boss":    func(r *rift.Run) *rift.Actor { return r.Practice.BossStart },
		"player":           func(r *rift.Run) *rift.Actor { return &r.Player },
		"enemy":            func(r *rift.Run) *rift.Actor { return &r.Enemies[0] },
		"future encounter": func(r *rift.Run) *rift.Actor { return &r.EncounterPlan[0][0] },
		"lantern":          func(r *rift.Run) *rift.Actor { return r.RoomObjective.Lantern },
		"escort":           func(r *rift.Run) *rift.Actor { return r.RoomObjective.Escort },
		"ward":             func(r *rift.Run) *rift.Actor { return &r.RoomObjective.Lanes[0].Ward },
		"future wave":      func(r *rift.Run) *rift.Actor { return &r.RoomObjective.Waves[0][0] },
	}
	changes := map[string]func(*rift.Actor){
		"negative health":   func(a *rift.Actor) { a.HP = -1 },
		"excess health":     func(a *rift.Actor) { a.HP = a.MaxHP + 1 },
		"zero capacity":     func(a *rift.Actor) { a.MaxHP = 0 },
		"negative capacity": func(a *rift.Actor) { a.MaxHP = -1 },
		"negative mana":     func(a *rift.Actor) { a.Mana = -1 },
		"excess mana":       func(a *rift.Actor) { a.Mana = 101 },
	}
	for name, selectActor := range selectors {
		for invalid, change := range changes {
			t.Run(name+"/"+invalid, func(t *testing.T) {
				r := riftVitalsFixture()
				change(selectActor(r))
				saved, err := encodeRift(r)
				if err != nil {
					t.Fatal(err)
				}
				if got, err := decodeRift(saved); err == nil || got != nil {
					t.Fatal("invalid saved vitals accepted")
				}
			})
		}
	}
}
func TestRiftDecodePreservesValidVitals(t *testing.T) {
	for _, hp := range []float64{0, .5, 100, 1000000} {
		for _, mana := range []float64{0, .5, 100} {
			r := riftVitalsFixture()
			r.Player.HP, r.Player.MaxHP, r.Player.Mana = hp, 1000000, mana
			r.Enemies[0].HP = 0
			r.Gold, r.BankedGold = 17, 23
			saved, err := encodeRift(r)
			if err != nil {
				t.Fatal(err)
			}
			got, err := decodeRift(saved)
			if err != nil {
				t.Fatal(err)
			}
			if got.Player.HP != hp || got.Player.Mana != mana || got.Enemies[0].HP != 0 || got.Gold != 17 || got.BankedGold != 23 {
				t.Fatal("valid vitals or rewards changed")
			}
		}
	}
}

func TestRiftDecodeAcceptsCampaignVitals(t *testing.T) {
	catalog := content.AbyssMobCatalog()
	for level := 1; level <= rift.LevelCount; level++ {
		r := rift.NewRunAtLevel("campaign-vitals", rift.Build{HP: 100}, time.Now(), catalog, level)
		for room := 0; room < len(rift.Rooms); room++ {
			saved, err := encodeRift(r)
			if err != nil {
				t.Fatal(err)
			}
			r, err = decodeRift(saved)
			if err != nil {
				t.Fatalf("level %d room %d: %v", level, room, err)
			}
			r.Status = "cleared"
			if room < len(rift.Rooms)-1 && !r.NextRoom() {
				t.Fatal("failed to advance fixture")
			}
		}
	}
}
