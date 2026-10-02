//go:build e2e

package bot

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/rift"
)

// registerRiftFixture uses the production simulation with explicitly isolated
// character/reward storage. No fixture route is compiled into production.
func registerRiftFixture(mux *http.ServeMux, server *WebServer) {
	mux.HandleFunc("/api/abyss/rift/metadata", handleRiftMetadata)
	var mu sync.Mutex
	runs := map[string]*rift.Run{}
	builds := map[string]rift.Build{}
	potions := map[string]int{}
	build := rift.Build{Name: "Rowan", Class: "vanguard", Level: 24, HP: 340, Damage: 32, Armor: 12, Weapon: "Mossbound Longsword", Gear: []string{"Mossbound Longsword", "Warden's Aegis", "Lanternkeeper's Cloak"}, Skills: []rift.Skill{{ID: "guard", Name: "Iron Guard", Kind: "shield", Cost: 18, Cooldown: 5, Power: 1.5}, {ID: "bash", Name: "Resolute Bash", Kind: "slash", Cost: 22, Cooldown: 3, Power: 2.4}, {ID: "spark", Name: "Cinder Bolt", Kind: "fire", Cost: 20, Cooldown: 2, Power: 2}}}
	mux.HandleFunc("/abyss/rift", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("rift_fixture")
		if err != nil {
			id, tokenErr := newAccountToken()
			if tokenErr != nil {
				http.Error(w, "fixture session", 500)
				return
			}
			cookie = &http.Cookie{Name: "rift_fixture", Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode}
			http.SetCookie(w, cookie)
		}
		mu.Lock()
		if _, exists := builds[cookie.Value]; !exists || r.URL.Query().Has("subclass") {
			style := r.URL.Query().Get("subclass")
			if _, ok := content.AbyssCombatStyle(style); !ok {
				style = "vanguard"
			}
			u := UserInCombat{AbyssSubclass: style, Stats: content.Stats{HP: 500, STR: 80, INT: 90, DEF: 60}, Skills: content.AbyssClassSkills(style)}
			selected := riftBuildFromUser(u, "Rowan", 24)
			selected.HP = 340
			selected.Damage = 32
			selected.Armor = 12
			selected.Weapon = build.Weapon
			selected.Gear = build.Gear
			selected.Skills = build.Skills
			selected.Pets = 2
			selected.Ultimate = &rift.Skill{ID: "fixture_ultimate", Name: "Rift Nova", Kind: "ultimate", Power: 5, Cost: 40, Cooldown: 18}
			owned := []string{"Rift Nova"}
			selected.OwnedUltimates = &owned
			builds[cookie.Value] = selected
			if r.URL.Query().Has("subclass") {
				delete(runs, cookie.Value)
			}
		}
		selectedBuild := builds[cookie.Value]
		mu.Unlock()
		if r.URL.Query().Get("scenario") == "circular-court" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("circular-court", selectedBuild, now, riftMobCatalog(now), 2)
			run.Epoch = "fixture"
			for index := range run.Enemies { run.Enemies[index].Knockdown = 1000 }
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "potions" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("potions", selectedBuild, now, riftMobCatalog(now), 1)
			run.Epoch = "fixture"
			run.Player.HP = 100
			run.Level.Rooms[0] = rift.Arena{Name: "Potion workshop"}
			run.Enemies = []rift.Actor{{ID: "watcher", Name: "Watcher", Kind: "goblin", X: 1450, Y: 485, HP: 100, MaxHP: 100, Knockdown: 1000}}
			run.Drops = []rift.Drop{}
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			potions[cookie.Value] = 2
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "visual" {
			run, err := riftVisualFixture(r.URL.Query(), selectedBuild)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "long-receipt" {
			run, err := buildLargeRiftReceipt(selectedBuild, time.Date(2026, 9, 13, 12, 0, 0, 0, time.UTC))
			if err != nil {
				http.Error(w, "receipt fixture unavailable", 500)
				return
			}
			run.Epoch = "fixture"
			run.BankedAtMS = run.SavedAtMS
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			mu.Lock()
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if run := riftHazardFixture(r.URL.Query().Get("scenario"), selectedBuild); run != nil {
			mu.Lock()
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if run := riftTerminalFixture(r.URL.Query().Get("scenario"), selectedBuild); run != nil {
			mu.Lock()
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "practice-enrage" && r.URL.Query().Get("practice") == "boss" {
			run, _ := rift.NewPracticeRun("practice-enrage", selectedBuild, "boss", time.Now())
			run.Epoch, run.Paused, run.Clock = "fixture", true, 28.5
			run.Practice.EnrageSeconds = rift.BossPracticeEnrageSeconds
			run.Enemies[0].Cooldown = 3
			mu.Lock()
			runs[cookie.Value+":boss"] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "practice-tools" && rift.ValidPracticeMode(r.URL.Query().Get("practice")) {
			mode := r.URL.Query().Get("practice")
			run, _ := rift.NewPracticeRun("practice-tools", selectedBuild, mode, time.Now())
			run.Epoch = "fixture"
			run.Paused = true
			run.Player.HP = run.Player.MaxHP / 2
			run.Player.Mana = 12
			run.Practice.Hits = 2
			run.Stats.DamageTaken = 30
			for _, skill := range run.Build.Skills {
				run.SkillTimers[skill.ID] = 8
			}
			for _, skill := range run.Build.Signatures {
				run.SkillTimers[skill.ID] = 8
			}
			if run.Build.Ultimate != nil {
				run.SkillTimers[run.Build.Ultimate.ID] = 8
			}
			run.SkillTimers["slowed"] = .6
			run.SkillTimers["hazard-0"] = .7
			mu.Lock()
			runs[cookie.Value+":"+mode] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "history" {
			mu.Lock()
			run := rift.NewRunAtLevel("history", selectedBuild, time.Now(), riftMobCatalog(time.Now()), 2)
			run.Status = "banked"
			run.HistoryActive = false
			run.Epoch = "fixture"
			run.CompletedLevels = []int{1, 4, 5, 6}
			run.History = map[int]rift.MissionHistory{
				1: {Attempts: 3, Completions: 2, LastStartedMS: 1000, LastOutcome: "completed", BestSeconds: 80},
				2: {Attempts: 2, LastStartedMS: 5000, LastOutcome: "defeated"},
				3: {Attempts: 1, LastStartedMS: 4000, LastOutcome: "exited"},
				4: {Attempts: 1, Completions: 1, LastStartedMS: 3000, LastOutcome: "completed", BestSeconds: 40},
				5: {Attempts: 2, Completions: 1, LastStartedMS: 2000, LastOutcome: "completed", BestSeconds: 40},
			}
			for id, history := range run.History {
				history.Definition = rift.NewRunAtLevel("history-definition", selectedBuild, time.Now(), nil, id).MissionDefinition
				run.History[id] = history
			}
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "enemy-rock" {
			mu.Lock()
			run := rift.NewRunAtLevel("enemy-rock", selectedBuild, time.Now(), riftMobCatalog(time.Now()), 91)
			run.Paused, run.Clock = true, 1.19
			if r.URL.Query().Get("condition") == "lethal" {
				run.Player.HP = 1
			}
			run.RoomObjective = nil
			run.Level.Rooms[0] = rift.Arena{Name: "Falling Rock Trial", Hazards: []rift.Hazard{{Kind: "falling_rock", Obstacle: rift.Obstacle{X: 450, Y: 350, W: 100, H: 100}, Period: 5, Duration: .18}}}
			run.Enemies = run.Enemies[:2]
			run.Enemies[0].X, run.Enemies[0].Y, run.Enemies[0].HP = 500, 410, 1
			run.Enemies[0].Cooldown = 10
			run.Enemies[1].X, run.Enemies[1].Y, run.Enemies[1].Cooldown = 1450, 480, 10
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "checkpoint" {
			mu.Lock()
			levelID := 1
			if l, err := strconv.Atoi(r.URL.Query().Get("level")); err == nil && l > 0 {
				levelID = l
			}
			run := rift.NewRunAtLevel("checkpoint", selectedBuild, time.Now(), riftMobCatalog(time.Now()), levelID)
			if r.URL.Query().Get("room") == "final" {
				run.Room = 2
				run.Level.Rooms[2].Hazards = nil
			} else if r.URL.Query().Get("room") == "court" || r.URL.Query().Get("room") == "preboss" {
				run.Room = 1
				run.Level.Rooms[1].Hazards = nil
			}
			if charges, err := strconv.Atoi(r.URL.Query().Get("charges")); err == nil && charges >= 0 && charges <= 3 {
				run.Resource = charges
			}
			run.Status = "cleared"
			run.Paused = true
			if r.URL.Query().Get("condition") == "wounded" {
				run.Player.HP = run.Player.MaxHP / 2
				run.Player.Mana = 30
			}
			run.Epoch = "fixture"
			gear, lootErr := rollRiftGear(0, time.Now())
			if lootErr != nil {
				mu.Unlock()
				http.Error(w, lootErr.Error(), 500)
				return
			}
			run.Drops = []rift.Drop{{Mission: run.Level.ID, Tier: run.Room + 1, ID: "fixture-drop", Gold: 30, Collected: true, Gear: &gear}}
			run.Gold = 30
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.RecordEncounterSummary("cleared")
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "hazard-objective" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("hazard-objective", selectedBuild, now, riftMobCatalog(now), 1)
			run.Level.Rooms[0].Hazards = []rift.Hazard{{Obstacle: rift.Obstacle{X: 100, Y: 380, W: 180, H: 60}, Kind: "fire", Jumpable: true, Period: 6, Duration: 2}}
			run.Clock = 1.3
			run.Epoch = "fixture"
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "treasure-objective" {
			mu.Lock()
			now := time.Now()
			var catalog []content.Mob
			for _, mob := range riftMobCatalog(now) {
				if rift.AdaptMonster(mob).Kind == "treasure" {
					catalog = append(catalog, mob)
				}
			}
			run := rift.NewRunAtLevel("treasure-objective", selectedBuild, now, catalog, 1)
			run.Enemies = run.Enemies[:1]
			run.Enemies[0].X = run.Player.X + 45
			run.Enemies[0].Y = run.Player.Y
			run.Enemies[0].HP = 1
			run.Enemies[0].Knockdown = 100
			run.Epoch = "fixture"
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if scenario := r.URL.Query().Get("scenario"); scenario == "ranged-priority" || scenario == "elite-priority" {
			mu.Lock()
			now := time.Now()
			var catalog []content.Mob
			for _, mob := range riftMobCatalog(now) {
				if (scenario == "ranged-priority" && rift.AdaptMonster(mob).Kind == "archer") || (scenario == "elite-priority" && mob.Type == content.MobElite) {
					catalog = append(catalog, mob)
					break
				}
			}
			run := rift.NewRunAtLevel("ranged-priority", selectedBuild, now, catalog, 1)
			run.Enemies = []rift.Actor{{ID: "priority-melee", Name: "Melee target", Kind: "goblin", X: 205, Y: 410, HP: 1, MaxHP: 50, Knockdown: 100}, {ID: "priority-ranged", Name: "Ranged target", Kind: "archer", X: 1300, Y: 410, HP: 100, MaxHP: 100, Knockdown: 100}}
			if scenario == "elite-priority" {
				run.Enemies[1].Tier = string(content.MobElite)
			}
			run.Epoch = "fixture"
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "dodge-objective" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("dodge-objective", selectedBuild, now, riftMobCatalog(now), 1)
			run.Enemies = []rift.Actor{{ID: "distant", Kind: "goblin", X: 1400, Y: 500, HP: 100, MaxHP: 100, Knockdown: 100}}
			run.Level.Rooms[0].HighCover = nil
			run.Level.Rooms[0].Hazards = nil
			for i := 0; i < 4; i++ {
				run.Projectiles = append(run.Projectiles, rift.Projectile{ID: 1000 + i, Enemy: true, X: 40, Y: run.Player.Y, VX: 300, Life: 2, Power: 10})
			}
			run.Epoch = "fixture"
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "sigils" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("sigil-room", selectedBuild, now, riftMobCatalog(now), 3)
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Step(rift.Input{}, now.Add(20*time.Millisecond))
			run.FinishCheckpoint("advance", riftMobCatalog(now))
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Level.Rooms[1].Hazards = nil
			run.Player.X = 400
			run.Player.Y = 330
			run.Epoch = "fixture"
			run.SetPaused(true, now.Add(20*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}

		if r.URL.Query().Get("scenario") == "rune_gate" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("rune-gate-room", selectedBuild, now, riftMobCatalog(now), 9)
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			e := &run.Enemies[0]
			e.HP = 1
			e.X, e.Y = run.Player.X+30, run.Player.Y
			e.Knockdown = 100
			run.Epoch = "fixture"
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if scenario := r.URL.Query().Get("scenario"); scenario == "bridge" || scenario == "terrain-cover" || scenario == "volatile-cover" || scenario == "terrain-projectile" || scenario == "terrain-stone-projectile" || scenario == "vault-cover" || scenario == "drop-edge" || scenario == "drop-edge-pursuit" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("terrain-cover", selectedBuild, now, riftMobCatalog(now), 1)
			run.Level.Rooms[0] = rift.Arena{Name: "Cover workshop", Cover: []rift.TerrainCover{
				{Obstacle: rift.Obstacle{X: 300, Y: 380, W: 80, H: 40}, ID: "wood-workshop", Material: "wood", HP: 60, MaxHP: 60},
				{Obstacle: rift.Obstacle{X: 650, Y: 380, W: 80, H: 40}, ID: "stone-workshop", Material: "stone"},
			}}
			run.Enemies = []rift.Actor{{ID: "watcher", Name: "Watcher", Kind: "goblin", X: 1450, Y: 485, HP: 100, MaxHP: 100, Knockdown: 1000}}
			run.Drops = []rift.Drop{{ID: "behind-wood", X: 410, Y: 410, Gold: 15}}
            if scenario == "drop-edge" || scenario == "drop-edge-pursuit" {
                run.Level.Rooms[0] = rift.Arena{Name:"Ledge approach",DropEdges:[]rift.DropEdge{{ID:"ledge-workshop",X:260,Y:365,W:80,LandingY:425}}}
                run.Player.X, run.Player.Y = 300,345
                run.Drops=[]rift.Drop{}
                if scenario == "drop-edge-pursuit" {
                    run.Player.Y=330
                    run.Enemies=[]rift.Actor{{ID:"pursuer",Name:"Ledge pursuer",Kind:"goblin",X:300,Y:425,HP:100,MaxHP:100,Speed:90,Damage:1,Facing:-1}}
                }
            }
			if scenario == "bridge" {
				run.Level.Rooms[0] = rift.Arena{Name: "Bridge crossing", Bridges: []rift.NarrowBridge{{ID: "workshop-bridge", Obstacle: rift.Obstacle{X: 400, Y: 370, W: 200, H: 90}}}}
				run.Player.X, run.Player.Y = 300, 340
				run.Drops = []rift.Drop{}
			}
			if scenario == "vault-cover" {
				run.Level.Rooms[0].Cover = nil
				run.Level.Rooms[0].Obstacles = []rift.Obstacle{{X: 300, Y: 380, W: 80, H: 40}}
				run.Level.Rooms[0].HighCover = []rift.Obstacle{{X: 650, Y: 380, W: 80, H: 40}}
				run.Drops = []rift.Drop{}
			}
			if scenario == "volatile-cover" {
				run.Level.Rooms[0].Cover = nil
				for i, x := range []float64{300, 390, 480} {
					run.Level.Rooms[0].Cover = append(run.Level.Rooms[0].Cover, rift.TerrainCover{Obstacle: rift.Obstacle{X: x, Y: 395, W: 32, H: 30}, ID: "volatile-" + strconv.Itoa(i), Material: "wood", HP: 60, MaxHP: 60, Volatile: true})
				}
				run.Player.X, run.Player.Y = 260, 410
				run.Drops = []rift.Drop{}
				run.Enemies = append(run.Enemies,
					rift.Actor{ID: "blast-target", Name: "Blast target", Kind: "goblin", X: 410, Y: 450, HP: 100, MaxHP: 100, Knockdown: 1000},
					rift.Actor{ID: "blast-loot", Name: "Blast loot", Kind: "goblin", X: 500, Y: 450, HP: 1, MaxHP: 100, Knockdown: 1000})
			}
			if scenario == "terrain-projectile" {
				run.Projectiles = []rift.Projectile{{X: 160, Y: 410, VX: 350, Life: 4, Power: 80, Kind: "fire"}}
			}
			if scenario == "terrain-stone-projectile" {
				run.Projectiles = []rift.Projectile{{X: 900, Y: 410, VX: -350, Life: 4, Power: 80, Kind: "fire", Enemy: true}}
			}
			run.Epoch = "fixture"
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if scenario := r.URL.Query().Get("scenario"); scenario == "split-defense" || scenario == "split-defense-failure" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("split-defense-room", selectedBuild, now, riftMobCatalog(now), 10)
			run.Level.Rooms[0].Hazards = nil
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			for _, lane := range run.RoomObjective.Lanes {
				for i := range run.Enemies {
					e := &run.Enemies[i]
					if e.ID == lane.EnemyIDs[0] {
						e.HP, e.MaxHP = selectedBuild.Damage*2, selectedBuild.Damage*2
						e.Armor = 0
						e.X, e.Y = lane.Ward.X, lane.Ward.Y
					}
				}
			}
			if scenario == "split-defense-failure" {
				run.RoomObjective.Lanes[1].Ward.HP = 5
				run.Player.X = 1500
			}
			run.Epoch = "fixture"
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if scenario := r.URL.Query().Get("scenario"); scenario == "lantern" || scenario == "lantern_failure" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("lantern-room", selectedBuild, now, riftMobCatalog(now), 8)
			run.Level.Rooms[0].Hazards = nil
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			lamp := run.RoomObjective.Lantern
			e := &run.Enemies[0]
			e.HP, e.MaxHP = selectedBuild.Damage*3, selectedBuild.Damage*3
			e.Armor = 0
			e.X, e.Y = lamp.X, lamp.Y
			e.Knockdown = 100
			run.Player.X, run.Player.Y = lamp.X-50, lamp.Y
			if scenario == "lantern_failure" {
				lamp.HP = 10
			}
			run.Epoch = "fixture"
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "rescue" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("rescue-room", selectedBuild, now, riftMobCatalog(now), 7)
			run.Level.Rooms[0].Hazards = nil
			for i := range run.Enemies {
				if run.Enemies[i].Kind != "cage" {
					run.Enemies[i].HP = 0
				}
			}
			run.Player.X, run.Player.Y = 430, 330
			run.Epoch = "fixture"
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "guardians" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("guardian-room", selectedBuild, now, riftMobCatalog(now), 6)
			run.Level.Rooms[0].Hazards = nil
			for i := range run.Enemies {
				e := &run.Enemies[i]
				n := -1
				for j, id := range run.RoomObjective.Targets {
					if e.ID == id {
						n = j
					}
				}
				if n >= 0 {
					e.X, e.Y = 400+float64(n)*180, 330
					e.HP, e.MaxHP = selectedBuild.Damage*3, selectedBuild.Damage*3
					e.Armor = 0
					e.Speed = 0.0001
				} else {
					e.HP = 0
				}
			}
			run.Player.X, run.Player.Y = 350, 330
			run.Step(rift.Input{}, now.Add(20*time.Millisecond))
			run.Epoch = "fixture"
			run.SetPaused(true, now.Add(20*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "collapse" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("collapse-room", selectedBuild, now, riftMobCatalog(now), 5)
			run.Level.Rooms[0].Hazards = nil
			for i := range run.Enemies {
				run.Enemies[i].Knockdown = 100
			}
			run.Epoch = "fixture"
			run.SetPaused(true, now)
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "ritual" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("ritual-room", selectedBuild, now, riftMobCatalog(now), 11)
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Step(rift.Input{}, now.Add(20*time.Millisecond))
			run.FinishCheckpoint("advance", riftMobCatalog(now))
			run.Level.Rooms[1].Hazards = nil
			channelIDs := map[string]int{}
			for i, c := range run.RoomObjective.Channels {
				channelIDs[c.EnemyID] = i
			}
			for i := range run.Enemies {
				e := &run.Enemies[i]
				if n, ok := channelIDs[e.ID]; ok {
					e.X, e.Y = 400+float64(n)*400, 330
					e.HP, e.MaxHP = selectedBuild.Damage*2, selectedBuild.Damage*2
					e.Armor = 0
					e.Damage = 12
				} else {
					e.HP = 0
				}
			}
			run.Player.X, run.Player.Y = 350, 330
			run.Epoch = "fixture"
			run.SetPaused(true, now.Add(20*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "spirit" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("spirit-room", selectedBuild, now, riftMobCatalog(now), 10)
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Step(rift.Input{}, now.Add(20*time.Millisecond))
			run.FinishCheckpoint("advance", riftMobCatalog(now))
			run.Level.Rooms[1].Hazards = nil
			run.Enemies = []rift.Actor{{ID: "spirit-threat", Name: "Spirit stalker", Kind: "goblin", HP: 1, MaxHP: 50, X: 355, Y: 330, Knockdown: 100}}
			run.Player.X, run.Player.Y = 310, 330
			run.Epoch = "fixture"
			run.SetPaused(true, now.Add(20*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "beacons" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("beacon-room", selectedBuild, now, riftMobCatalog(now), 2)
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Step(rift.Input{}, now.Add(20*time.Millisecond))
			run.FinishCheckpoint("advance", riftMobCatalog(now))
			run.Level.Rooms[1].Hazards = nil
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Player.X, run.Player.Y = 330, 330
			run.Epoch = "fixture"
			run.SetPaused(true, now.Add(20*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "hunt" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("hunt-room", selectedBuild, now, riftMobCatalog(now), 9)
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			now = now.Add(20 * time.Millisecond)
			run.Step(rift.Input{}, now)
			for _, id := range run.RoomObjective.Sequence {
				for _, seal := range run.RoomObjective.Pickups {
					if seal.ID == id {
						run.Player.X, run.Player.Y = seal.X, seal.Y
						now = now.Add(20 * time.Millisecond)
						run.Step(rift.Input{}, now)
					}
				}
			}
			run.FinishCheckpoint("advance", riftMobCatalog(now))
			run.Level.Rooms[1].Hazards = nil
			run.Level.Rooms[1].Obstacles = nil
			run.Level.Rooms[1].HighCover = nil
			for i := range run.Enemies {
				run.Enemies[i].X = 1400
				run.Enemies[i].Y = 490
				run.Enemies[i].Knockdown = 100
			}
			for index, id := range run.RoomObjective.Targets {
				for i := range run.Enemies {
					e := &run.Enemies[i]
					if e.ID == id {
						e.X, e.Y, e.HP = 225+float64(index)*250, 410, 1
						if index == 2 {
							e.Kind = "wolf"
							e.ArtKey = ""
						}
					}
				}
			}
			run.Player.X, run.Player.Y = 180, 410
			run.Epoch = "fixture"
			run.SetPaused(true, now.Add(20*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "generators" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("generator-room", selectedBuild, now, riftMobCatalog(now), 8)
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Step(rift.Input{}, now.Add(20*time.Millisecond))
			run.FinishCheckpoint("advance", riftMobCatalog(now))
			run.Level.Rooms[1].Obstacles = nil
			run.Level.Rooms[1].HighCover = nil
			index := 0
			for i := range run.Enemies {
				e := &run.Enemies[i]
				if e.Kind != "generator" {
					e.HP = 0
					continue
				}
				e.X, e.Y = 225+float64(index)*250, 350
				h := &run.Level.Rooms[1].Hazards[index]
				h.X, h.Y, h.W, h.H = 200+float64(index)*250, 390, 90, 50
				index++
			}
			run.Player.X, run.Player.Y = 180, 350
			run.Epoch = "fixture"
			run.SetPaused(true, now.Add(20*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "relic" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("relic-room", selectedBuild, now, riftMobCatalog(now), 7)
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Step(rift.Input{}, now.Add(20*time.Millisecond))
			run.FinishCheckpoint("advance", riftMobCatalog(now))
			run.Level.Rooms[1].Hazards = nil
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Player.X, run.Player.Y = 390, 330
			run.Epoch = "fixture"
			run.SetPaused(true, now.Add(20*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "totems" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("totem-room", selectedBuild, now, riftMobCatalog(now), 6)
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Step(rift.Input{}, now.Add(20*time.Millisecond))
			run.FinishCheckpoint("advance", riftMobCatalog(now))
			run.Level.Rooms[1].Hazards = nil
			run.Level.Rooms[1].Obstacles = nil
			run.Level.Rooms[1].HighCover = nil
			index := 0
			for i := range run.Enemies {
				e := &run.Enemies[i]
				if e.Kind != "totem" {
					e.HP = 0
					continue
				}
				e.X, e.Y = 225+float64(index)*250, 410
				index++
			}
			run.Player.X, run.Player.Y = 180, 410
			run.Epoch = "fixture"
			run.SetPaused(true, now.Add(20*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}
        if r.URL.Query().Get("scenario") == "raised-platform" {
            mu.Lock()
            now:=time.Now()
            run:=rift.NewRunAtLevel("raised-platform",selectedBuild,now,riftMobCatalog(now),2)
            run.Enemies=[]rift.Actor{{ID:"watcher",Name:"Watcher",Kind:"goblin",X:1450,Y:485,HP:100,MaxHP:100,Knockdown:1000}}
            run.Epoch="fixture"
            run.SetPaused(true,now)
            runs[cookie.Value]=run
            mu.Unlock()
        }
        if r.URL.Query().Get("scenario") == "spawn-hazards" {
            mu.Lock()
            now:=time.Now()
            run:=rift.NewRunAtLevel("spawn-hazards",selectedBuild,now,riftMobCatalog(now),1)
            for run.Room<2 {run.Status="cleared";run.NextRoom()}
            h:=run.Level.Rooms[2].Hazards[0]
            run.Clock=1.3-h.Offset
            if run.Clock<0 {run.Clock+=h.Period}
            run.Epoch="fixture"
            run.SetPaused(true,now)
            runs[cookie.Value]=run
            mu.Unlock()
        }
		if r.URL.Query().Get("scenario") == "waves" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("wave-room", selectedBuild, now, riftMobCatalog(now), 5)
			run.Player.X, run.Player.Y = run.RoomObjective.Zone.X, run.RoomObjective.Zone.Y
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Step(rift.Input{}, now.Add(20*time.Millisecond))
			run.FinishCheckpoint("advance", riftMobCatalog(now))
			run.Level.Rooms[1].Hazards = nil
			for group := range run.RoomObjective.Waves {
				for i := range run.RoomObjective.Waves[group] {
					enemy := &run.RoomObjective.Waves[group][i]
					enemy.X, enemy.Y, enemy.HP, enemy.Knockdown = 345, 410, 1, 100
				}
			}
			run.Enemies = append([]rift.Actor(nil), run.RoomObjective.Waves[0]...)
			run.Player.X, run.Player.Y = 300, 410
			if r.URL.Query().Get("condition") == "rest" {
				run.Player.X = 160
				for group := range run.RoomObjective.Waves {
					for i := range run.RoomObjective.Waves[group] {
						run.RoomObjective.Waves[group][i].X = 205
					}
				}
				run.Enemies = append([]rift.Actor(nil), run.RoomObjective.Waves[0]...)
			}
			if r.URL.Query().Get("condition") == "floor" {
                run.Player.X,run.Player.Y=410,405
                for group:=range run.RoomObjective.Waves {for i:=range run.RoomObjective.Waves[group] {run.RoomObjective.Waves[group][i].X=780;run.RoomObjective.Waves[group][i].Y=405}}
                run.Enemies=append([]rift.Actor(nil),run.RoomObjective.Waves[0]...)
            }
			if r.URL.Query().Get("condition") == "gate" {
				run.Player.X, run.Player.Y = 800, 395
				for group := range run.RoomObjective.Waves {
					for i := range run.RoomObjective.Waves[group] {
						run.RoomObjective.Waves[group][i].X = 1200
						run.RoomObjective.Waves[group][i].Y = 395
					}
				}
				run.Enemies = append([]rift.Actor(nil), run.RoomObjective.Waves[0]...)
			}
			run.Epoch = "fixture"
			run.SetPaused(true, now.Add(20*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "circle" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("circle-room", selectedBuild, now, riftMobCatalog(now), 4)
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Step(rift.Input{}, now.Add(20*time.Millisecond))
			run.FinishCheckpoint("advance", riftMobCatalog(now))
			run.Level.Rooms[1].Hazards = nil
			run.Enemies = []rift.Actor{{ID: "contester", Name: "Circle defender", Kind: "goblin", X: 480, Y: 410, HP: 1, MaxHP: 50, Knockdown: 100}}
			run.Player.X = 430
			run.Player.Y = 410
			run.Epoch = "fixture"
			run.SetPaused(true, now.Add(20*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if scenario := r.URL.Query().Get("scenario"); scenario == "objective-results" || scenario == "objective-rewards" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("objective-results", selectedBuild, now, riftMobCatalog(now), 1)
			run.Room = 2
			run.Enemies = append([]rift.Actor{}, run.EncounterPlan[2]...)
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			run.Stats.Guards = 5
			run.Stats.Seconds = 42
			run.Epoch = "fixture"
			run.Step(rift.Input{}, now.Add(20*time.Millisecond))
			run.SetPaused(true, now.Add(20*time.Millisecond))
			if scenario == "objective-rewards" {
				run.Gold = 30
				run.Drops = []rift.Drop{{ID: "fight-gold", Gold: 30, Collected: true}}
			}
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "boss-retry" {
			mu.Lock()
			now := time.Now()
			run := rift.NewRunAtLevel("boss-retry", selectedBuild, now, riftMobCatalog(now), 1)
			run.Room = 2
			run.Enemies = append([]rift.Actor{}, run.EncounterPlan[2]...)
			run.BankedGold = 50
			run.Epoch = "fixture"
			run.Player.HP = 0
			run.Step(rift.Input{}, now.Add(100*time.Millisecond))
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "boss-windup" {
			mu.Lock()
			run := rift.NewRunAtLevel("boss-windup", selectedBuild, time.Now(), riftMobCatalog(time.Now()), 1)
			run.Room = 2
			run.Status = "fighting"
			run.Paused = true
			run.Epoch = "fixture"
			if len(run.EncounterPlan) > 2 {
				run.Enemies = append([]rift.Actor{}, run.EncounterPlan[2]...)
			}
			for i := range run.Enemies {
				if run.Enemies[i].Kind == "boss" {
					run.Enemies[i].Windup = 20.0
					run.Enemies[i].AttackName = "Mossbound Slam"
					run.Enemies[i].TargetX = run.Player.X
					run.Enemies[i].TargetY = run.Player.Y
				}
			}
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "long-boss-name" {
			mu.Lock()
			run := rift.NewRunAtLevel("long-boss-name", selectedBuild, time.Now(), riftMobCatalog(time.Now()), 1)
			run.Room = 2
			run.Status = "fighting"
			run.Paused = true
			run.Epoch = "fixture"
			if len(run.EncounterPlan) > 2 {
				run.Enemies = append([]rift.Actor{}, run.EncounterPlan[2]...)
			}
			for i := range run.Enemies {
				if run.Enemies[i].Kind == "boss" {
					run.Enemies[i].Name = "Ancient Mossbound Colossus of the Verdant Abyss and Eternal Primordial Shadows"
					run.Enemies[i].Windup = 15.0
					run.Enemies[i].AttackName = "Verdant Overgrowth Obliteration Slam"
					run.Enemies[i].TargetX = run.Player.X
					run.Enemies[i].TargetY = run.Player.Y
				}
			}
			runs[cookie.Value] = run
			mu.Unlock()
		}
		if r.URL.Query().Get("scenario") == "server-catchup" {
			mu.Lock()
			run := rift.NewRunAtLevel("server-catchup", selectedBuild, time.Now(), riftMobCatalog(time.Now()), 1)
			run.Status = "fighting"
			run.Paused = true
			run.Epoch = "fixture"
			run.Catchup = true
			runs[cookie.Value] = run
			mu.Unlock()
		}
		server.render(w, "rift", map[string]any{"Title": "Rift Brawl Playtest", "Nav": "rift", "EnableAbyss": true, "AccountNav": true, "Fixture": true, "Practice": r.URL.Query().Get("practice")})
	})
	// Isolated reset trigger: uses the same expiration projection as production reads.
	mux.HandleFunc("/api/e2e/rift-economy-reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost { http.Error(w, "POST only", 405); return }
		cookie, err := r.Cookie("rift_fixture")
		if err != nil { http.Error(w, "fixture session required", 401); return }
		mu.Lock()
		defer mu.Unlock()
		run := runs[cookie.Value]
		if run == nil { http.Error(w, "fixture run required", 404); return }
		run.Epoch = "fixture-old"
		expireRiftEconomy(run)
		run.UpdateObjectives()
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/abyss/rift", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		cookie, err := r.Cookie("rift_fixture")
		if err != nil {
			http.Error(w, "fixture session required", 401)
			return
		}
		mode := r.URL.Query().Get("practice")
		key := cookie.Value
		if mode != "" {
			if !rift.ValidPracticeMode(mode) {
				http.Error(w, "invalid practice", 400)
				return
			}
			key += ":" + mode
		}
		run := runs[key]
		build := builds[cookie.Value]
		if r.Method == http.MethodGet && r.URL.Query().Get("inventory") == "potions" {
			items := []riftPotionOption{}
			if count := potions[cookie.Value]; mode == "" && count > 0 {
				items = append(items, riftPotionOption{ID: "small_health_potion", Name: "Small Health Potion", Count: count, HealHP: 50})
			}
			writeJSON(w, map[string]any{"ok": true, "potions": items})
			return
		}
		if r.Method == http.MethodGet {
			ch := riftChallenge(time.Now())
			if custom := r.URL.Query().Get("challenge"); custom != "" {
				ch = riftChallengeFor(custom)
			}
			response, wireErr := riftHTTPResponse(r, run, "load")
			if wireErr != nil {
				riftFailure(w, r, wireErr)
				return
			}
			response["build"], response["objective_options"], response["challenge"] = build, rift.ObjectiveOptions(build), ch
			if r.Header.Get("X-Rift-Metadata") != "separate" {
				for key, value := range riftPublicMetadata(time.Now()) {
					response[key] = value
				}
			}
			writeJSON(w, response)
			return
		}
		var req riftRequest
		if r.Method != http.MethodPost || json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil || !validRiftRequest(req) || !validRiftModeAction(mode, req.Kind) {
			http.Error(w, "invalid controls", 400)
			return
		}
		if req.Kind != "start" && run != nil && run.Status == "expired" {
			riftFailure(w, r, errRiftEconomyReset)
			return
		}
		if req.Kind == "start" {
			if run != nil && run.StartKey == req.RequestID {
				writeRiftSnapshot(w, r, req.Kind, run)
				return
			}
			if run != nil && (run.Status == "fighting" || run.Status == "cleared") {
				http.Error(w, "active run", 409)
				return
			}
			selected := build
			selected.Skills = []rift.Skill{}
			for _, id := range req.Skills {
				for _, sk := range build.Skills {
					if sk.ID == id {
						selected.Skills = append(selected.Skills, sk)
					}
				}
			}
			if req.Skills == nil {
				selected.Skills = build.Skills
			}
			previous := run
			if mode != "" {
				var err error
				run, err = newRiftPractice(req, req.RequestID, selected, mode, time.Now())
				if err != nil {
					http.Error(w, err.Error(), 400)
					return
				}
			} else {
				run = rift.NewRunAtLevel(req.RequestID, selected, time.Now(), riftMobCatalog(time.Now()), req.LevelID)
				run.InheritCampaignHistory(previous)
			}
			run.StartKey = req.RequestID
			run.Epoch = "fixture"
			runs[key] = run
		} else {
			if run == nil || run.ID != req.RunID || req.Revision > run.Revision+1 {
				http.Error(w, "stale run", 409)
				return
			}
			if req.Revision <= run.Revision {
				if req.Revision == run.Revision && req.RequestID != run.LastRequestID {
					writeJSONStatus(w, http.StatusConflict, map[string]any{"ok": false, "error": errRiftConflict.Error()})
					return
				}
				writeRiftSnapshot(w, r, req.Kind, run)
				return
			}
			switch req.Kind {
			case "potion":
				amount, err := riftPotionAmount(req.ConsumableID, run.Player.MaxHP)
				if err != nil || req.ConsumableID != "small_health_potion" || potions[cookie.Value] <= 0 {
					http.Error(w, "potion unavailable", 409)
					return
				}
				if err := run.UseHealingPotion(amount); err != nil {
					http.Error(w, err.Error(), 409)
					return
				}
				potions[cookie.Value]--
			case "practice_spawn", "practice_clear":
				if err := applyRiftPracticeEnemy(run, req, time.Now()); err != nil { http.Error(w, err.Error(), 409); return }
			case "practice_health", "practice_mana", "practice_cooldowns", "practice_freeze", "practice_bank":
				if err := run.PracticeTool(req.Kind); err != nil {
					http.Error(w, err.Error(), 409)
					return
				}
			case "practice_reset":
				if err := resetRiftPractice(run, req, time.Now()); err != nil {
					http.Error(w, err.Error(), 400)
					return
				}
			case "retry_boss":
				if err := run.RetryBossEncounter(time.Now()); err != nil {
					http.Error(w, err.Error(), 409)
					return
				}
			case "step":
				run.Step(req.Input, time.Now())
			case "pause":
				run.SetPaused(true, time.Now())
			case "resume":
				run.SetPaused(false, time.Now())
			case "bank", "exit", "next", "advance":
				if run.Status != "cleared" {
					http.Error(w, "room not clear", 409)
					return
				}
				for i := range run.Drops {
					d := &run.Drops[i]
					if !d.Banked {
						run.BankedGold += d.Gold
						if d.Gear != nil {
							run.BankedItems = append(run.BankedItems, d.Gear.Name)
							run.BankedLoot = append(run.BankedLoot, d.LootReceipt())
						}
						d.Banked = true
						d.Collected = true
					}
				}
				run.UpdateObjectives()
				if bonus := run.PendingObjectiveGold(); bonus > 0 {
					run.Objectives.RewardGold = bonus
					run.BankedObjectiveGold += bonus
					run.BankedGold += bonus
				}
				run.Gold = 0
				run.LastMS = time.Now().UnixMilli()
				run.BankedAtMS = run.LastMS
				run.FinishCheckpoint(req.Kind, riftMobCatalog(time.Now()))
			}
			run.Revision = req.Revision
		}
		run.LastRequestID = req.RequestID
		for i := range run.Drops {
			drop := &run.Drops[i]
			if run.Practice == nil && drop.NeedsGear && drop.Gear == nil {
				gear, err := rollRiftGear(run.Room, time.Now())
				if err != nil {
					http.Error(w, "loot unavailable", 500)
					return
				}
				drop.Gear = &gear
				drop.Gear.FoundBoss = riftGearOrigin(run)
			}
		}
		writeRiftSnapshot(w, r, req.Kind, run)
	})
}
