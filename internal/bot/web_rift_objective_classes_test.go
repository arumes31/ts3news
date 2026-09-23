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
				run.Player.X, run.Player.Y = run.RoomObjective.Zone.X, run.RoomObjective.Zone.Y
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

func TestRiftRelicRoomsSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Carrier", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-relic", build, now, catalog, 7)
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
					run.Enemies[i].HP = 0
				}
				for n := 0; n < 1500 && !run.RoomObjective.Complete; n++ {
					x, y := run.RoomObjective.Relic.X, run.RoomObjective.Relic.Y
					if run.RoomObjective.Carrying {
						x, y = run.RoomObjective.Zone.X, run.RoomObjective.Zone.Y
					}
					dx, dy := x-run.Player.X, y-run.Player.Y
					input := rift.Input{}
					if math.Abs(dx) > 5 {
						input.X = math.Copysign(1, dx)
					}
					if math.Abs(dy) > 5 {
						input.Y = math.Copysign(1, dy)
					}
					step(input)
				}
				if run.Status != "cleared" || !run.RoomObjective.Complete || run.RoomObjective.Carrying {
					t.Fatal("subclass could not deliver relic")
				}
			})
		}
	}
}

func TestRiftGeneratorRoomsSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Disabler", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-generator", build, now, catalog, 8)
				step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				step(rift.Input{})
				run.FinishCheckpoint("advance", catalog)
				run.Level.Rooms[1].Obstacles = nil
				run.Level.Rooms[1].HighCover = nil
				for i := range run.Enemies {
					if run.Enemies[i].Kind != "generator" {
						run.Enemies[i].HP = 0
					}
				}
				for i := range run.Enemies {
					target := &run.Enemies[i]
					if target.Kind != "generator" {
						continue
					}
					for n := 0; n < 1000; n++ {
						dx, dy := target.X-30-run.Player.X, target.Y-run.Player.Y
						if math.Abs(dx) <= 5 && math.Abs(dy) <= 5 {
							break
						}
						input := rift.Input{}
						if math.Abs(dx) > 5 {
							input.X = math.Copysign(1, dx)
						}
						if math.Abs(dy) > 5 {
							input.Y = math.Copysign(1, dy)
						}
						step(input)
					}
					step(rift.Input{X: 1})
					for n := 0; n < 300 && target.HP > 0; n++ {
						step(rift.Input{Attack: true})
					}
				}
				if run.Status != "cleared" || !run.RoomObjective.Complete || run.Stats.Kills != 0 || len(run.Drops) != 0 {
					t.Fatal("subclass shutdown or prop reward isolation failed")
				}
				for _, h := range run.Level.Rooms[1].Hazards {
					if !h.Disabled {
						t.Fatal("hazard still powered")
					}
				}
			})
		}
	}
}

func TestRiftHuntRoomsSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Hunter", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-hunt", build, now, catalog, 9)
				step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				step(rift.Input{})
				for _, id := range run.RoomObjective.Sequence {
					for _, seal := range run.RoomObjective.Pickups {
						if seal.ID == id {
							run.Player.X, run.Player.Y = seal.X, seal.Y
							step(rift.Input{})
						}
					}
				}
				run.FinishCheckpoint("advance", catalog)
				run.Level.Rooms[1].Hazards = nil
				run.Level.Rooms[1].Obstacles = nil
				run.Level.Rooms[1].HighCover = nil
				for i := range run.Enemies {
					run.Enemies[i].X = 1400
					run.Enemies[i].Y = 490
					run.Enemies[i].Knockdown = 100
				}
				for _, id := range run.RoomObjective.Targets {
					for i := range run.Enemies {
						e := &run.Enemies[i]
						if e.ID != id {
							continue
						}
						e.X, e.Y, e.HP = 205, 410, 1
						for n := 0; n < 100 && e.HP > 0; n++ {
							step(rift.Input{Attack: true})
						}
					}
				}
				if run.Status != "cleared" || !run.RoomObjective.Complete || run.Stats.Kills != run.RoomObjective.Target || len(run.Drops) != run.RoomObjective.Target {
					t.Fatal("subclass hunt/reward isolation failed")
				}
				escaped := 0
				for _, e := range run.Enemies {
					if e.Pose == "escape" {
						escaped++
					}
				}
				if escaped != len(run.Enemies)-run.RoomObjective.Target {
					t.Fatal("survivors did not retreat")
				}
			})
		}
	}
}

func TestRiftBeaconRoomsSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Capturer", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-beacon", build, now, catalog, 2)
				step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				step(rift.Input{})
				run.FinishCheckpoint("advance", catalog)
				run.Level.Rooms[1].Hazards = nil
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				for n := 0; n < 3000 && !run.RoomObjective.Complete; n++ {
					o := run.RoomObjective
					x, y := o.Zone.X, o.Zone.Y
					if o.Collected == 1 && run.Player.Y < 460 {
						x = 600
						y = run.Player.Y
						if math.Abs(run.Player.X-600) < 10 {
							y = 480
						}
					}
					if o.Collected == 2 && run.Player.X < 1100 {
						x = 1120
						y = 480
					}
					dx, dy := x-run.Player.X, y-run.Player.Y
					input := rift.Input{}
					if math.Abs(dx) > 5 {
						input.X = math.Copysign(1, dx)
					}
					if math.Abs(dy) > 5 {
						input.Y = math.Copysign(1, dy)
					}
					step(input)
				}
				if run.Status != "cleared" || !run.RoomObjective.Complete {
					t.Fatal("subclass could not capture moving beacons through arena geometry")
				}
			})
		}
	}
}

func TestRiftSpiritRoomsSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Escort", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-spirit", build, now, catalog, 10)
				step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				step(rift.Input{})
				run.FinishCheckpoint("advance", catalog)
				run.Level.Rooms[1].Hazards = nil
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				for n := 0; n < 1800 && !run.RoomObjective.Complete; n++ {
					spirit := run.RoomObjective.Escort
					dx, dy := spirit.X-50-run.Player.X, spirit.Y-run.Player.Y
					input := rift.Input{}
					if math.Abs(dx) > 5 {
						input.X = math.Copysign(1, dx)
					}
					if math.Abs(dy) > 5 {
						input.Y = math.Copysign(1, dy)
					}
					step(input)
				}
				if run.Status != "cleared" || !run.RoomObjective.Complete {
					t.Fatalf("subclass failed escort: status=%s player=(%.2f,%.2f) spirit=(%.2f,%.2f) moving=%v contested=%v", run.Status, run.Player.X, run.Player.Y, run.RoomObjective.Escort.X, run.RoomObjective.Escort.Y, run.RoomObjective.EscortMoving, run.RoomObjective.Contested)
				}
			})
		}
	}
}

func TestRiftRitualRoomsSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Breaker", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-ritual", build, now, catalog, 11)
				step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				step(rift.Input{})
				run.FinishCheckpoint("advance", catalog)
				run.Level.Rooms[1].Hazards = nil
				run.Level.Rooms[1].Obstacles = nil
				run.Level.Rooms[1].HighCover = nil
				channelIDs := map[string]bool{}
				for _, c := range run.RoomObjective.Channels {
					channelIDs[c.EnemyID] = true
				}
				for i := range run.Enemies {
					if !channelIDs[run.Enemies[i].ID] {
						run.Enemies[i].HP = 0
					}
				}
				for i := range run.Enemies {
					target := &run.Enemies[i]
					if !channelIDs[target.ID] {
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
				if run.Status != "cleared" || !run.RoomObjective.Complete || run.Stats.Kills != len(channelIDs) || len(run.Drops) != len(channelIDs) {
					t.Fatal("subclass ritual completion or reward accounting failed")
				}
			})
		}
	}
}

func TestRiftCollapseRoomsSupportEverySubclassAndRegion(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				for level := 5; level <= 95; level += 10 {
					build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Escape", 24)
					now := time.Unix(100, 0)
					run := rift.NewRunAtLevel("class-collapse", build, now, content.AbyssMobCatalog(), level)
					if run.RoomObjective == nil || run.RoomObjective.Kind != "escape_collapse" {
						t.Fatal("collapse missing")
					}
					run.Level.Rooms[0].Hazards = nil
					for i := range run.Enemies {
						run.Enemies[i].Knockdown = 100
					}
					for n := 0; n < 1400 && run.Status == "fighting"; n++ {
						input := rift.Input{}
						dy := run.RoomObjective.Zone.Y - run.Player.Y
						if math.Abs(dy) > 4 {
							input.Y = math.Copysign(1, dy)
						} else {
							input.X = 1
						}
						now = now.Add(20 * time.Millisecond)
						run.Step(input, now)
					}
					if run.Status != "cleared" || !run.RoomObjective.Complete || run.Stats.Kills != 0 || len(run.Drops) != 0 {
						t.Fatalf("mission %d escape failed: status=%s player=(%.1f,%.1f) edge=%.1f", level, run.Status, run.Player.X, run.Player.Y, run.RoomObjective.CollapseX)
					}
				}
			})
		}
	}
}

func TestRiftLinkedGuardiansSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Bondbreaker", 24)
				now := time.Unix(100, 0)
				run := rift.NewRunAtLevel("class-guardians", build, now, content.AbyssMobCatalog(), 6)
				if run.RoomObjective == nil || run.RoomObjective.Kind != "linked_guardians" {
					t.Fatal("guardians missing")
				}
				run.Level.Rooms[0].Hazards = nil
				run.Level.Rooms[0].Obstacles = nil
				run.Level.Rooms[0].HighCover = nil
				ids := run.RoomObjective.Targets
				for i := range run.Enemies {
					e := &run.Enemies[i]
					if e.ID == ids[0] {
						e.X, e.Y = 400, 330
						e.Knockdown = 100
					} else if e.ID == ids[1] {
						e.X, e.Y = 550, 330
						e.Knockdown = 100
					} else {
						e.HP = 0
					}
				}
				run.Player.X, run.Player.Y = 350, 330
				for _, id := range ids {
					for n := 0; n < 1000; n++ {
						var target *rift.Actor
						for i := range run.Enemies {
							if run.Enemies[i].ID == id {
								target = &run.Enemies[i]
							}
						}
						if target.HP <= 0 {
							break
						}
						input := rift.Input{Attack: true}
						if target.X-run.Player.X > 35 {
							input.X = 1
						}
						now = now.Add(20 * time.Millisecond)
						run.Step(input, now)
					}
				}
				if run.Status != "cleared" || !run.RoomObjective.Complete || run.Stats.Kills != 2 || len(run.Drops) != 2 {
					t.Fatal("subclass guardian completion or rewards failed")
				}
			})
		}
	}
}

func TestRiftRescueRoomsSupportEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Breaker", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-rescue", build, now, catalog, 7)
				step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
				run.Level.Rooms[0].Hazards = nil
				run.Level.Rooms[0].Obstacles = nil
				run.Level.Rooms[0].HighCover = nil
				for i := range run.Enemies {
					if run.Enemies[i].Kind != "cage" {
						run.Enemies[i].HP = 0
					}
				}
				for i := range run.Enemies {
					target := &run.Enemies[i]
					if target.Kind != "cage" {
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

func TestRiftLanternDefenseSupportsEverySubclassAndRegion(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				for level := 8; level <= 98; level += 10 {
					build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Defender", 24)
					now := time.Unix(100, 0)
					run := rift.NewRunAtLevel("class-lantern", build, now, content.AbyssMobCatalog(), level)
					if run.RoomObjective == nil || run.RoomObjective.Kind != "protect_lantern" {
						t.Fatal("lantern missing")
					}
					run.Level.Rooms[0].Hazards = nil
					for i := range run.Enemies {
						run.Enemies[i].HP = 0
					}
					enemy := &run.Enemies[0]
					enemy.HP, enemy.MaxHP = build.Damage*2, build.Damage*2
					enemy.X, enemy.Y = run.RoomObjective.Lantern.X, run.RoomObjective.Lantern.Y
					enemy.Knockdown = 100
					for n := 0; n < 1200 && run.Status == "fighting"; n++ {
						input := rift.Input{Attack: true}
						dy := enemy.Y - run.Player.Y
						if math.Abs(dy) > 4 {
							input.Y = math.Copysign(1, dy)
						} else if enemy.X-run.Player.X > 35 {
							input.X = 1
						}
						now = now.Add(20 * time.Millisecond)
						run.Step(input, now)
					}
					if run.Status != "cleared" || !run.RoomObjective.Complete || run.RoomObjective.Lantern.HP <= 0 || run.Stats.Kills != 1 || len(run.Drops) != 1 {
						t.Fatalf("mission %d lantern defense failed", level)
					}
				}
			})
		}
	}
}

func TestRiftRuneGateSupportsEverySubclass(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Stats: content.Stats{HP: 500}, Skills: content.AbyssClassSkills(sub.ID)}, "Walker", 24)
				now := time.Unix(100, 0)
				catalog := content.AbyssMobCatalog()
				run := rift.NewRunAtLevel("class-gate", build, now, catalog, 9)
				step := func(input rift.Input) { now = now.Add(20 * time.Millisecond); run.Step(input, now) }
				if run.RoomObjective == nil {
					t.Fatal("campaign rune gate missing")
				}
				run.Level.Rooms[0].Obstacles = nil
				run.Level.Rooms[0].HighCover = nil
				run.Level.Rooms[0].Hazards = nil
				for i := range run.Enemies {
					run.Enemies[i].HP = 0
				}
				for _, id := range run.RoomObjective.Sequence {
					pickup := run.RoomObjective.Pickups[id-1]
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
					t.Fatal("subclass could not open rune gate through movement")
				}
			})
		}
	}
}
