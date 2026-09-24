package rift

import (
	"encoding/json"
	"testing"
)

func TestBasicMissesCountOnlySwingsWithoutTargetOrBreakableCover(t *testing.T) {
	for _, mode := range []string{"range", "lane", "facing", "hit", "wood", "stone", "cooldown", "guard"} {
		t.Run(mode, func(t *testing.T) {
			r := terrainTestRun()
			r.Level.Rooms[0].Cover = nil
			r.Enemies = []Actor{{ID: "target", X: 200, Y: 410, HP: 1000, MaxHP: 1000, Cooldown: 10}}
			input := Input{Attack: true}
			switch mode {
			case "range":
				r.Enemies[0].X = 800
			case "lane":
				r.Enemies[0].Y = 300
			case "facing":
				r.Player.Facing = -1
			case "wood":
				r.Level.Rooms[0].Cover = terrainTestRun().Level.Rooms[0].Cover
				r.Enemies[0].X = 230
			case "stone":
				r.Level.Rooms[0].Cover = terrainTestRun().Level.Rooms[0].Cover
				r.Level.Rooms[0].Cover[0].Material = "stone"
				r.Enemies[0].X = 230
			case "cooldown":
				r.Player.Cooldown = 1
			case "guard":
				input.Guard = true
			}
			r.tick(input, .02)
			wantAttacks := 1
			if mode == "cooldown" || mode == "guard" {
				wantAttacks = 0
			}
			if r.Stats.Attacks != wantAttacks {
				t.Fatalf("attacks=%d want %d", r.Stats.Attacks, wantAttacks)
			}
			want := 0
			if mode == "range" || mode == "lane" || mode == "facing" || mode == "stone" {
				want = 1
			}
			if r.Stats.BasicMisses != want {
				t.Fatalf("misses=%d want %d", r.Stats.BasicMisses, want)
			}
			raw, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err = json.Unmarshal(raw, &saved); err != nil {
				t.Fatal(err)
			}
			if saved.Stats.BasicMisses != want {
				t.Fatal("miss count lost on save")
			}
		})
	}
}
