package rift

import "testing"

func TestGeomancerTerrainCueTracksPlayerImpactsOnly(t *testing.T) {
	for _, mode := range []string{"melee", "projectile", "enemy", "other", "stone"} {
		t.Run(mode, func(t *testing.T) {
			r := terrainTestRun()
			r.Build.Class = "geomancer"
			if mode == "other" {
				r.Build.Class = "vanguard"
			}
			if mode == "stone" {
				r.Level.Rooms[0].Cover[0].Material = "stone"
				r.Level.Rooms[0].Cover[0].HP = 0
			}
			if mode == "melee" || mode == "other" {
				r.attackTerrainCover(20)
			} else {
				r.Projectiles = []Projectile{{X: 160, Y: 410, VX: 1000, Life: 2, Power: 20, Enemy: mode == "enemy"}}
				r.tick(Input{}, .1)
			}
			count := 0
			for _, event := range r.Events {
				if event.Kind == "geomancer_terrain" {
					count++
				}
			}
			want := 1
			if mode == "enemy" || mode == "other" {
				want = 0
			}
			if count != want {
				t.Fatalf("cue count=%d want %d", count, want)
			}
			if mode == "stone" {
				if r.Level.Rooms[0].Cover[0].HP != 0 {
					t.Fatal("stone damaged")
				}
			} else if r.Level.Rooms[0].Cover[0].HP != 40 {
				t.Fatal("terrain cue changed damage")
			}
			if r.Stats.Kills != 0 || len(r.Drops) != 0 {
				t.Fatal("terrain cue granted rewards")
			}
		})
	}
}
