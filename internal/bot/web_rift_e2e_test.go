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
	var mu sync.Mutex
	runs := map[string]*rift.Run{}
	builds := map[string]rift.Build{}
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
			if _, ok := content.AbyssSubclassByID(style); !ok {
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
			selected.Ultimate = &rift.Skill{ID: "fixture_ultimate", Name: "Rift Nova", Kind: "ultimate", Power: 5, Cost: 70, Cooldown: 18}
			owned := []string{"Rift Nova"}
			selected.OwnedUltimates = &owned
			builds[cookie.Value] = selected
			if r.URL.Query().Has("subclass") {
				delete(runs, cookie.Value)
			}
		}
		selectedBuild := builds[cookie.Value]
		mu.Unlock()
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
			run.Status = "cleared"
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
			run.Level.Rooms[0].Hazards = []rift.Hazard{{Obstacle: rift.Obstacle{X: 100, Y: 380, W: 180, H: 60}, Kind: "fire", Period: 6, Duration: 2}}
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
		if r.URL.Query().Get("scenario") == "objective-results" {
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
		if r.Method == http.MethodGet {
			ch := riftChallenge(time.Now())
			if custom := r.URL.Query().Get("challenge"); custom != "" {
				ch = riftChallengeFor(custom)
			}
			writeJSON(w, map[string]any{"ok": true, "run": run, "build": build, "rooms": rift.Rooms, "levels": rift.Campaign(), "objective_options": rift.ObjectiveOptions(build), "bestiary": riftBestiary(time.Now()), "rarities": riftRarities(), "challenge": ch})
			return
		}
		var req riftRequest
		if r.Method != http.MethodPost || json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil || !validRiftRequest(req) || !validRiftModeAction(mode, req.Kind) {
			http.Error(w, "invalid controls", 400)
			return
		}
		if req.Kind == "start" {
			if run != nil && run.StartKey == req.RequestID {
				writeJSON(w, map[string]any{"ok": true, "run": run})
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
				writeJSON(w, map[string]any{"ok": true, "run": run})
				return
			}
			switch req.Kind {
			case "practice_health", "practice_mana", "practice_cooldowns":
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
							run.BankedLoot = append(run.BankedLoot, rift.BankedLoot{Name: d.Gear.Name, Rarity: int(d.Gear.Rarity)})
						}
						d.Banked = true
						d.Collected = true
					}
				}
				run.Gold = 0
				run.LastMS = time.Now().UnixMilli()
				run.BankedAtMS = run.LastMS
				run.FinishCheckpoint(req.Kind, riftMobCatalog(time.Now()))
			}
			run.Revision = req.Revision
		}
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
		writeJSON(w, map[string]any{"ok": true, "run": run})
	})
}
