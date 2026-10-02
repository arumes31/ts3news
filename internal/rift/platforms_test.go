package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
	"ts3news/internal/content"
)

func platformTestRun() *Run {
	r := dropEdgeTestRun()
	r.Level.Rooms[0] = Arena{Floor: "grass", Platforms: []RaisedPlatform{{Obstacle: Obstacle{400, 350, 220, 120}, ID: "terrace", Rise: 20, Ramp: 30, Floor: "stone"}}}
	return r
}

func TestRaisedPlatformHeightRampsContinuouslyFromEveryEdge(t *testing.T) {
	r := platformTestRun()
	arena := r.Arena()
	for _, tc := range []struct{ x, y, want float64 }{{399, 410, 0}, {400, 410, 0}, {415, 410, 10}, {430, 410, 20}, {510, 410, 20}, {605, 410, 10}, {620, 410, 0}, {510, 350, 0}, {510, 365, 10}, {510, 380, 20}, {510, 455, 10}, {510, 470, 0}, {415, 365, 10}} {
		if got := arena.Elevation(tc.x, tc.y); math.Abs(got-tc.want) > 1e-9 {
			t.Fatalf("height %.0f/%.0f: got %.3f want %.3f", tc.x, tc.y, got, tc.want)
		}
	}
}

func TestRaisedPlatformWalkingUpdatesSurfaceAndNeverCountsAsJump(t *testing.T) {
	r := platformTestRun()
	r.Player.X, r.Player.Y = 395, 410
	hp := r.Player.HP
	for n := 0; n < 46; n++ {
		before := r.Player.X
		r.moveActor(&r.Player, 5, 0, false)
		if r.Player.X != before+5 {
			t.Fatal("walkable platform blocked movement")
		}
		if r.Player.Elevation != r.Arena().Elevation(r.Player.X, r.Player.Y) {
			t.Fatal("actor height diverged from platform")
		}
		want := "grass"
		if r.Player.Elevation > 0 {
			want = "stone"
		}
		if r.FloorMaterial() != want || r.Floor != want {
			t.Fatal("platform footstep surface did not follow movement")
		}
	}
	if r.Player.Elevation != 0 || r.Player.HP != hp || r.Stats.Jumps != 0 {
		t.Fatal("crossing created a jump, damage or stuck elevation")
	}
}

func TestRaisedPlatformElevationSurvivesSaveAndJump(t *testing.T) {
	r := platformTestRun()
	r.Player.X, r.Player.Y = 510, 410
	r.Player.Jump = .5
	r.moveActor(&r.Player, 0, 0, false)
	if r.Player.Elevation != 20 || r.Player.Jump != .5 {
		t.Fatal("surface height replaced airborne jump")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Player.Elevation != 20 || saved.Arena().Elevation(510, 410) != 20 {
		t.Fatal("platform not saved")
	}
	saved.moveActor(&saved.Player, 0, 65, false)
	if saved.Player.Elevation != 0 {
		t.Fatal("leaving platform retained elevated floor")
	}
}

func TestRaisedPlatformDoesNotGrantJumpHazardImmunity(t *testing.T) {
	for _, jump := range []float64{0, .5} {
		r := platformTestRun()
		r.Player.X, r.Player.Y = 510, 410
		r.Player.Jump = jump
		r.moveActor(&r.Player, 0, 0, false)
		r.Level.Rooms[0].Hazards = []Hazard{{Obstacle: Obstacle{490, 390, 40, 40}, Kind: "fire", Jumpable: true, Period: 7, Duration: 1}}
		r.Clock = 1.3
		hp := r.Player.HP
		r.hazardTick()
		if jump == 0 && r.Player.HP >= hp {
			t.Fatal("ground elevation granted airborne immunity")
		}
		if jump > 0 && r.Player.HP != hp {
			t.Fatal("actual jump lost hazard immunity")
		}
	}
}

func TestRaisedPlatformRoomAndWaveArrivalsRecomputeElevation(t *testing.T) {
	r := platformTestRun()
	r.EncounterPlan[0] = []Actor{{ID: "terrace-enemy", Kind: "goblin", X: 510, Y: 410, HP: 100, MaxHP: 100}}
	r.spawnRoom()
	if len(r.Enemies) != 1 || r.Enemies[0].Elevation != 20 || r.Player.Elevation != 0 {
		t.Fatal("spawn elevation incorrect")
	}
	r.Enemies[0].HP = 0
	r.RoomObjective = &RoomObjective{Kind: "survive_waves", Target: 3, Wave: 1, NextWaveSeconds: .01, Waves: [][]Actor{nil, {{ID: "wave-enemy", Kind: "goblin", X: 510, Y: 410, HP: 100, MaxHP: 100}}, nil}}
	r.tickWaveObjective(.02)
	if len(r.Enemies) != 2 || r.Enemies[1].Elevation != 20 {
		t.Fatal("reinforcement did not stand on platform")
	}
}

func TestPlatformCastAndHitEffectsKeepActorSurfaceHeight(t *testing.T) {
	r := platformTestRun()
	r.Player.X, r.Player.Y = 510, 365
	r.moveActor(&r.Player, 0, 0, false)
	r.Build.Skills = []Skill{{ID: "terrace-fire", Kind: "fire", Power: 1}}
	r.cast("terrace-fire")
	if len(r.Projectiles) != 1 || r.Projectiles[0].Elevation != 10 {
		t.Fatal("projectile launch lost ramp height")
	}
	found := false
	for _, event := range r.Events {
		if event.Kind == "fire" {
			found = true
			if event.Y != 330 || event.Elevation != 10 {
				t.Fatal("cast sampled offset effect point instead of caster height")
			}
		}
	}
	if !found {
		t.Fatal("missing cast effect")
	}
	r.Enemies = []Actor{{ID: "ramp-enemy", Kind: "goblin", X: 510, Y: 365, HP: 10, MaxHP: 10}}
	r.moveActor(&r.Enemies[0], 0, 0, false)
	r.hurtEnemy(0, 100, "hit")
	if len(r.Drops) != 1 || r.Drops[0].Elevation != 10 {
		t.Fatal("loot lost defeated enemy's floor height")
	}
	for _, event := range r.Events {
		if event.Kind == "hit" && event.Elevation != 10 {
			t.Fatal("hit effect lost target height")
		}
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Drops[0].Elevation != 10 || saved.Projectiles[0].Elevation != 10 {
		t.Fatal("saved visual heights changed")
	}
}

func TestPlatformEnemyProjectilesKeepHeightThroughExpiration(t *testing.T) {
	r := platformTestRun()
	r.Player.X, r.Player.Y = 1000, 365
	r.Enemies = []Actor{{ID: "archer", Kind: "archer", X: 510, Y: 365, HP: 100, MaxHP: 100, Windup: .01, Damage: 10}}
	r.moveActor(&r.Enemies[0], 0, 0, false)
	r.enemyTick(0, .02)
	if len(r.Projectiles) != 1 || r.Projectiles[0].Elevation != 10 {
		t.Fatal("enemy projectile lost launch height")
	}
	r.Projectiles[0].X = 800
	r.Projectiles[0].Life = .01
	r.Enemies[0].Knockdown = 100
	r.tick(Input{}, .02)
	found := false
	for _, event := range r.Events {
		if event.Kind == "projectile_expire" {
			found = true
			if event.Elevation != 10 {
				t.Fatal("expiration sampled ground after projectile left platform")
			}
		}
	}
	if !found {
		t.Fatal("missing projectile expiration")
	}
}

func TestCampaignPlatformsHaveClearApproachesForEverySubclass(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			if (len(arena.Platforms) == 1) != (room == 0 && level.ID%10 == 2) {
				t.Fatal("incorrect platform placement")
			}
			for _, platform := range arena.Platforms {
				count++
				for x := platform.X - 12; x <= platform.X+platform.W+12; x += 4 {
					for y := platform.Y - 12; y <= platform.Y+platform.H+12; y += 4 {
						for _, o := range arena.solidObstacles() {
							if contains(o, x, y, 0) {
								t.Fatal("platform approach intersects cover")
							}
						}
						for _, h := range arena.Hazards {
							if !h.Disabled && contains(h.Obstacle, x, y, 0) {
								t.Fatal("platform approach intersects hazard")
							}
						}
					}
				}
			}
		}
	}
	if count != 10 {
		t.Fatalf("platform count %d", count)
	}
	for _, style := range []string{"vanguard", "berserker", "marksman", "beastmaster", "elementalist", "chronomancer", "oracle", "geomancer", "bloodblade", "voidwalker", "runesmith", "alchemist"} {
		r := NewRunAtLevel("platform-class", Build{Class: style, HP: 200}, time.Unix(100, 0), content.AbyssMobCatalog(), 2)
		r.Enemies = []Actor{{ID: "watcher", Kind: "goblin", X: 1450, Y: 485, HP: 100, MaxHP: 100, Knockdown: 1000}}
		for n := 0; n < 100 && r.Player.X < 250; n++ {
			r.tick(Input{X: 1}, .02)
		}
		if r.Player.Elevation != 16 || r.Floor != "stone" {
			t.Fatalf("%s did not walk onto platform", style)
		}
		for n := 0; n < 100 && r.Player.X < 375; n++ {
			r.tick(Input{X: 1}, .02)
		}
		if r.Player.Elevation != 0 || r.Floor != "grass" || r.Stats.Jumps != 0 || r.Player.HP != 200 {
			t.Fatalf("%s did not leave platform safely", style)
		}
	}
}
