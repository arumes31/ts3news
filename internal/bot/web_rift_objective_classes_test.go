package bot

import (
	"encoding/json"
	"math"
	"sort"
	"testing"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

// These fixtures isolate objective/class compatibility from encounter difficulty:
// terrain is clear and enemies are held still, but casts, hits, guards, room
// completion, checkpoint advancement and objective banking use the real engine.
func TestRiftObjectivesCompatibleWithEverySelectableSubclass(t *testing.T) {
	ultimate, ok := content.GetUltimateSkillByID("ULT_1")
	if !ok {
		t.Fatal("canonical ultimate missing")
	}
	catalog := []content.Mob{{Name: "Priority mage", Type: content.MobElite}, {Name: "Treasure target", Type: content.MobTreasureGoblin}, {Name: "Boss target", Type: content.MobBoss}}
	passive := map[string]bool{"timed": true, "no_damage": true, "basic_only": true, "hazard_avoidance": true, "treasure_capture": true, "melee_only": true, "ranged_priority": true, "elite_priority": true, "limited_dodge": true, "save_ultimate": true}
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500, STR: 80, INT: 90, DEF: 60}, Skills: content.AbyssClassSkills(sub.ID), Ultimates: []*content.UltimateSkill{&ultimate}}, "Compatibility", 24)
			if len(build.Signatures) != 2 || build.Signatures[0].Role != "builder" || build.Signatures[1].Role != "finisher" {
				t.Fatalf("%s lacks its canonical actions", sub.ID)
			}
			for _, option := range rift.ObjectiveOptions(build) {
				t.Run(sub.ID+"/"+option.ID, func(t *testing.T) {
					if !passive[option.ID] && option.ID != "guard" && option.ID != "aerial_finish" && option.ID != "finisher" {
						t.Fatalf("new objective needs a compatibility scenario: %s", option.ID)
					}
					now := time.Unix(100, 0)
					run := rift.NewRunAtLevel("class-objectives", build, now, catalog, 1)
					offered := false
					for _, entry := range run.Objectives.Entries {
						offered = offered || entry.ID == option.ID
					}
					if !offered {
						t.Fatal("eligible objective missing")
					}
					step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
					ready := func() {
						recovering := func() bool {
							return run.Player.Cooldown > 0 || run.Player.Pose == "recovery" || run.Player.Pose == "ultimate_anticipation"
						}
						for n := 0; recovering() && n < 100; n++ {
							step(rift.Input{})
						}
						if recovering() {
							t.Fatal("action failed to recover")
						}
					}
					for room := 0; room < 3; room++ {
						arena := &run.Level.Rooms[room]
						arena.Obstacles = nil
						arena.HighCover = nil
						arena.Hazards = nil
						for i := range run.Enemies {
							e := &run.Enemies[i]
							e.X = 1200
							e.Y = 500
							e.HP = 1
							e.MaxHP = 1
							e.Knockdown = 100
						}
						if room == 0 && option.ID == "guard" {
							for n := 0; n < 5; n++ {
								e := &run.Enemies[0]
								e.Kind = "goblin"
								e.X = run.Player.X + 40
								e.Y = run.Player.Y
								e.Knockdown = 0
								e.Windup = .01
								e.Cooldown = 0
								step(rift.Input{Guard: true})
							}
							run.Enemies[0].Knockdown = 100
							step(rift.Input{})
						}
						if room == 0 && option.ID == "finisher" {
							step(rift.Input{Skill: build.Signatures[0].ID})
							ready()
							step(rift.Input{Skill: build.Signatures[1].ID})
							ready()
						}
						order := make([]int, len(run.Enemies))
						for i := range order {
							order[i] = i
						}
						sort.SliceStable(order, func(i, j int) bool {
							return run.Enemies[order[i]].Kind == "archer" && run.Enemies[order[j]].Kind != "archer"
						})
						for _, i := range order {
							if run.Enemies[i].HP <= 0 {
								continue
							}
							ready()
							run.Enemies[i].X = run.Player.X + 40
							run.Enemies[i].Y = run.Player.Y
							step(rift.Input{Attack: true, Jump: option.ID == "aerial_finish"})
							if run.Enemies[i].HP > 0 {
								t.Fatal("basic attack could not finish target")
							}
						}
						if run.Status != "cleared" {
							t.Fatalf("tier %d did not clear: %s", room, run.Status)
						}
						if room < 2 {
							run.FinishCheckpoint("advance", catalog)
						} else {
							run.FinishCheckpoint("bank", catalog)
						}
					}
					if run.Stats.RoomsCleared != 3 || run.ObjectiveHistory["Wayfarer"][option.ID] != 1 {
						t.Fatalf("objective not earned: %+v", run.Objectives)
					}
					data, err := json.Marshal(run)
					if err != nil {
						t.Fatal(err)
					}
					var saved rift.Run
					if err = json.Unmarshal(data, &saved); err != nil {
						t.Fatal(err)
					}
					if saved.ObjectiveHistory["Wayfarer"][option.ID] != 1 || !saved.Objectives.Banked {
						t.Fatal("class objective result did not survive save")
					}
				})
			}
		}
	}
}

func TestRiftSigilRoomsSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Walker", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-sigils", build, now, catalog, 3)
				step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				step(rift.Input{})
				run.FinishCheckpoint("advance", catalog)
				if run.RoomObjective == nil {
					t.Fatal("campaign sigil room missing")
				}
				run.Level.Rooms[1].Obstacles = nil
				run.Level.Rooms[1].HighCover = nil
				run.Level.Rooms[1].Hazards = nil
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				for _, pickup := range run.RoomObjective.Pickups {
					for n := 0; n < 1000 && !run.RoomObjective.Pickups[pickup.ID-1].Collected; n++ {
						dx, dy := pickup.X-run.Player.X, pickup.Y-run.Player.Y
						input := rift.Input{}
						if math.Abs(dx) > 5 {
							input.X = math.Copysign(1, dx)
						}
						if math.Abs(dy) > 5 {
							input.Y = math.Copysign(1, dy)
						}
						step(input)
					}
				}
				if run.Status != "cleared" || !run.RoomObjective.Complete {
					t.Fatal("subclass could not collect sigils through movement")
				}
			})
		}
	}
}

func TestRiftCircleRoomsSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Holder", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-circle", build, now, catalog, 4)
				step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				step(rift.Input{})
				run.FinishCheckpoint("advance", catalog)
				if run.RoomObjective == nil {
					t.Fatal("campaign circle missing")
				}
				run.Level.Rooms[1].Hazards = nil
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				for n := 0; n < 1500 && !run.RoomObjective.Complete; n++ {
					input := rift.Input{}
					if run.Player.X < run.RoomObjective.Zone.X-5 {
						input.X = 1
					}
					step(input)
				}
				if run.Status != "cleared" || run.RoomObjective.Seconds != 15 {
					t.Fatal("subclass could not charge circle through movement")
				}
			})
		}
	}
}

func TestRiftWaveRoomsSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Survivor", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-waves", build, now, catalog, 5)
				step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				step(rift.Input{})
				run.FinishCheckpoint("advance", catalog)
				run.Level.Rooms[1].Hazards = nil
				for group := range run.RoomObjective.Waves {
					for i := range run.RoomObjective.Waves[group] {
						enemy := &run.RoomObjective.Waves[group][i]
						enemy.X, enemy.Y, enemy.HP, enemy.Knockdown = 205, 410, 1, 100
					}
				}
				run.Enemies = append([]rift.Actor(nil), run.RoomObjective.Waves[0]...)
				for n := 0; n < 1500 && run.Status == "fighting"; n++ {
					step(rift.Input{Attack: true})
				}
				if run.Status != "cleared" || !run.RoomObjective.Complete || run.RoomObjective.Wave != 3 {
					t.Fatal("subclass could not defeat all waves")
				}
				if run.Stats.Kills != len(run.EncounterPlan[1]) {
					t.Fatal("wave kills not recorded once per enemy")
				}
			})
		}
	}
}

func TestRiftTotemRoomsSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Breaker", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-totems", build, now, catalog, 6)
				step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				step(rift.Input{})
				run.FinishCheckpoint("advance", catalog)
				run.Level.Rooms[1].Hazards = nil
				run.Level.Rooms[1].Obstacles = nil
				run.Level.Rooms[1].HighCover = nil
				for i := range run.Enemies {
					if run.Enemies[i].Kind != "totem" {
						run.Enemies[i].HP = 0
					}
				}
				for i := range run.Enemies {
					target := &run.Enemies[i]
					if target.Kind != "totem" {
						continue
					}
					for n := 0; n < 1000 && target.HP > 0; n++ {
						dx, dy := target.X-30-run.Player.X, target.Y-run.Player.Y
						input := rift.Input{Attack: true}
						if math.Abs(dx) > 5 {
							input.X = math.Copysign(1, dx)
						}
						if math.Abs(dy) > 5 {
							input.Y = math.Copysign(1, dy)
						}
						step(input)
					}
				}
				if run.Status != "cleared" || !run.RoomObjective.Complete || run.Stats.Kills != 0 || len(run.Drops) != 0 {
					t.Fatal("subclass prop completion or reward isolation failed")
				}
			})
		}
	}
}
