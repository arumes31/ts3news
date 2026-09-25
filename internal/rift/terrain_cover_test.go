package rift

import (
	"encoding/json"
	"testing"
)

func terrainTestRun() *Run {
	r := circleTestRun()
	r.RoomObjective = nil
	r.Build.Damage = 30
	r.Level.Rooms[0] = Arena{Cover: []TerrainCover{{Obstacle: Obstacle{195, 380, 20, 60}, ID: "wood", Material: "wood", HP: 60, MaxHP: 60}}}
	r.Player.X, r.Player.Y, r.Player.Facing = 160, 410, 1
	return r
}

func TestWoodCoverBreaksWithoutKillLootOrSameSwingPenetration(t *testing.T) {
	r := terrainTestRun()
	r.Enemies[0].X, r.Enemies[0].Y = 230, 410
	for n := 0; n < 2; n++ {
		r.Player.Cooldown = 0
		r.tick(Input{Attack: true}, .02)
	}
	if r.Level.Rooms[0].Cover[0].HP != 0 {
		t.Fatal("wood did not break")
	}
	if r.Enemies[0].HP != 100 || r.Stats.Kills != 0 || len(r.Drops) != 0 {
		t.Fatal("breaking cover hit through it or granted monster rewards")
	}
	r.Player.Cooldown = 0
	r.tick(Input{Attack: true}, .02)
	if r.Enemies[0].HP >= 100 {
		t.Fatal("broken wood still blocks combat")
	}
	count := 0
	for _, e := range r.Events {
		if e.Kind == "cover_break" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("break events=%d", count)
	}
}

func TestCoverCollisionAndSavedDestruction(t *testing.T) {
	for _, jump := range []float64{0, .5} {
		r := terrainTestRun()
		r.Player.Jump = jump
		r.moveActor(&r.Player, 40, 0, false)
		if r.Player.X != 160 {
			t.Fatal("intact tall wood crossed")
		}
		r.damageTerrainCover(0, 60)
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		saved.moveActor(&saved.Player, 40, 0, false)
		if saved.Player.X != 200 || len(saved.Arena().solidObstacles()) != 0 || !saved.clearProjectilePath(&Actor{X: 160, Y: 410}, &Actor{X: 240, Y: 410}) {
			t.Fatal("saved broken wood still blocks route")
		}
	}
}

func TestCoverMeleeRespectsFacingAndInterveningStone(t *testing.T) {
	r := terrainTestRun()
	r.Player.Facing = -1
	r.attackTerrainCover(60)
	if r.Level.Rooms[0].Cover[0].HP != 60 {
		t.Fatal("rear attack broke wood")
	}
	r.Player.Facing = 1
	r.Level.Rooms[0].Cover = append(r.Level.Rooms[0].Cover, TerrainCover{Obstacle: Obstacle{180, 380, 5, 60}, ID: "stone", Material: "stone"})
	r.attackTerrainCover(60)
	if r.Level.Rooms[0].Cover[0].HP != 60 {
		t.Fatal("attack crossed stone")
	}
	r.damageTerrainCover(1, 10000)
	if !r.Level.Rooms[0].Cover[1].solid() {
		t.Fatal("stone destroyed")
	}
}

func TestProjectilesDamageOnlyFirstCoverAndDoNotPassThroughBreakingWood(t *testing.T) {
	for _, hostile := range []bool{false, true} {
		r := terrainTestRun()
		r.Level.Rooms[0].Cover[0].HP = 10
		r.Level.Rooms[0].Cover = append(r.Level.Rooms[0].Cover, TerrainCover{Obstacle: Obstacle{225, 380, 10, 60}, ID: "far", Material: "wood", HP: 60, MaxHP: 60})
		r.Projectiles = []Projectile{{X: 160, Y: 410, VX: 1000, Life: 2, Power: 20, Enemy: hostile}}
		r.tick(Input{}, .1)
		if r.Level.Rooms[0].Cover[0].HP != 0 || r.Level.Rooms[0].Cover[1].HP != 60 || len(r.Projectiles) != 0 {
			t.Fatal("projectile damaged farther cover")
		}
	}
}

func TestStaticCoverShieldsWoodFromProjectiles(t *testing.T) {
	r := terrainTestRun()
	r.Level.Rooms[0].HighCover = []Obstacle{{180, 380, 5, 60}}
	r.Projectiles = []Projectile{{X: 160, Y: 410, VX: 1000, Life: 2, Power: 100}}
	r.tick(Input{}, .1)
	if r.Level.Rooms[0].Cover[0].HP != 60 || len(r.Projectiles) != 0 {
		t.Fatal("projectile pierced earlier permanent cover")
	}
}

func TestTerrainCoverCampaignPlacement(t *testing.T) {
	wood, stone := 0, 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			want := room == 0 && (level.ID%10 == 2 || level.ID%10 == 4)
			if (len(arena.Cover) == 1) != want {
				t.Fatalf("wrong cover placement %d/%d", level.ID, room)
			}
			for _, c := range arena.Cover {
				switch c.Material {
				case "wood":
					wood++
					if c.HP != 60 || c.MaxHP != 60 {
						t.Fatal("invalid wooden durability")
					}
				case "stone":
					stone++
				default:
					t.Fatal("unknown material")
				}
			}
		}
	}
	if wood != 10 || stone != 10 {
		t.Fatalf("wood=%d stone=%d", wood, stone)
	}
}
