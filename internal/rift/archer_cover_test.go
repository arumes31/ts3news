package rift

import "testing"

func archerBehindCover(high bool) *Run {
	r := testRun()
	r.Player.X = 160
	r.Player.Y = 410
	arena := Arena{}
	wall := Obstacle{195, 380, 10, 60}
	if high {
		arena.HighCover = []Obstacle{wall}
	} else {
		arena.Obstacles = []Obstacle{wall}
	}
	r.Level = &Level{Rooms: []Arena{arena}}
	r.Enemies = []Actor{{ID: "archer", Kind: "archer", X: 230, Y: 410, HP: 100, MaxHP: 100}}
	return r
}

func TestArcherSeeksClearShotAroundTallCover(t *testing.T) {
	r := archerBehindCover(true)
	r.enemyTick(0, .05)
	if r.Enemies[0].Windup > 0 || r.Enemies[0].X >= 230 {
		t.Fatal("archer aimed into tall cover instead of repositioning")
	}
	for i := 0; i < 200 && r.Enemies[0].Windup == 0; i++ {
		r.enemyTick(0, .05)
	}
	if r.Enemies[0].Windup == 0 {
		t.Fatal("archer never found a clear shot")
	}
	low := archerBehindCover(false)
	low.enemyTick(0, .05)
	if low.Enemies[0].Windup == 0 {
		t.Fatal("low cover incorrectly blocked archer aim")
	}
}

func TestArcherRechecksCoverBeforeRelease(t *testing.T) {
	r := archerBehindCover(true)
	r.Enemies[0].Windup = .01
	r.enemyTick(0, .02)
	if len(r.Projectiles) != 0 {
		t.Fatal("archer released into newly blocking cover")
	}
}

func TestMeleeEnemyRepositionsWhenTerrainBlocksRange(t *testing.T) {
	r := archerBehindCover(false)
	r.Enemies[0].Kind = "goblin"
	r.Enemies[0].X = 225
	r.enemyTick(0, .05)
	if r.Enemies[0].Windup > 0 || r.Enemies[0].X >= 225 {
		t.Fatal("melee enemy attacked through obstruction")
	}
}
