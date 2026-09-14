//go:build e2e

package bot

import (
	"encoding/json"
	"net/http"
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
			builds[cookie.Value] = selected
			if r.URL.Query().Has("subclass") {
				delete(runs, cookie.Value)
			}
		}
		selectedBuild := builds[cookie.Value]
		mu.Unlock()
		if r.URL.Query().Get("scenario") == "checkpoint" {
			mu.Lock()
			run := rift.NewRunAtLevel("checkpoint", selectedBuild, time.Now(), riftMobCatalog(time.Now()), 1)
			if r.URL.Query().Get("room") == "final" {
				run.Room = 2
				run.Level.Rooms[2].Hazards = nil
			}
			run.Status = "cleared"
			run.Epoch = "fixture"
			gear, lootErr := rollRiftGear(0, time.Now())
			if lootErr != nil {
				mu.Unlock()
				http.Error(w, lootErr.Error(), 500)
				return
			}
			run.Drops = []rift.Drop{{Mission: 1, Tier: run.Room + 1, ID: "fixture-drop", Gold: 30, Collected: true, Gear: &gear}}
			run.Gold = 30
			for i := range run.Enemies {
				run.Enemies[i].HP = 0
			}
			runs[cookie.Value] = run
			mu.Unlock()
		}
		server.render(w, "rift", map[string]any{"Title": "Rift Brawl Playtest", "Nav": "rift", "EnableAbyss": true, "AccountNav": true, "Fixture": true})
	})
	mux.HandleFunc("/api/abyss/rift", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		cookie, err := r.Cookie("rift_fixture")
		if err != nil {
			http.Error(w, "fixture session required", 401)
			return
		}
		run := runs[cookie.Value]
		build := builds[cookie.Value]
		if r.Method == http.MethodGet {
			writeJSON(w, map[string]any{"ok": true, "run": run, "build": build, "rooms": rift.Rooms, "levels": rift.Campaign(), "bestiary": riftBestiary(time.Now()), "rarities": riftRarities()})
			return
		}
		var req riftRequest
		if r.Method != http.MethodPost || json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req) != nil || !validRiftRequest(req) {
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
			var completed []int
			if run != nil {
				completed = run.CompletedLevels
			}
			run = rift.NewRunAtLevel(req.RequestID, selected, time.Now(), riftMobCatalog(time.Now()), req.LevelID)
			run.CompletedLevels = completed
			run.StartKey = req.RequestID
			run.Epoch = "fixture"
			runs[cookie.Value] = run
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
			case "step":
				run.Step(req.Input, time.Now())
			case "pause":
				run.Paused = true
			case "resume":
				run.Paused = false
				run.LastMS = time.Now().UnixMilli()
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
						}
						d.Banked = true
						d.Collected = true
					}
				}
				run.Gold = 0
				run.FinishCheckpoint(req.Kind, riftMobCatalog(time.Now()))
				run.LastMS = time.Now().UnixMilli()
			}
			run.Revision = req.Revision
		}
		for i := range run.Drops {
			drop := &run.Drops[i]
			if drop.NeedsGear && drop.Gear == nil {
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
