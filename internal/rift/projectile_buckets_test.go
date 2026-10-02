package rift

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
)

func linearProjectileTarget(enemies []Actor, shot Projectile) int {
	ref := shot.Skill.Reference()
	for i, e := range enemies {
		if e.HP > 0 && math.Abs(shot.X-e.X) < ref.Horizontal && math.Abs(shot.Y-e.Y) < ref.Depth {
			return i
		}
	}
	return -1
}

func TestProjectileBucketsMatchOrderedScan(t *testing.T) {
	rng := rand.New(rand.NewSource(510))
	var buckets projectileBuckets
	for pass := 0; pass < 20; pass++ {
		enemies := make([]Actor, 200-pass*7)
		for i := range enemies {
			enemies[i] = Actor{X: rng.Float64()*1800 - 100, Y: rng.Float64() * 500, HP: float64(rng.Intn(3))}
		}
		buckets.reset(enemies)
		for j := 0; j < 1000; j++ {
			shot := Projectile{X: rng.Float64()*1800 - 100, Y: rng.Float64() * 500, Skill: Skill{Kind: []string{"fire", "quake", "ultimate", "shield"}[j%4]}}
			if got, want := buckets.target(enemies, shot), linearProjectileTarget(enemies, shot); got != want {
				t.Fatalf("pass %d shot %d: target %d, want %d", pass, j, got, want)
			}
		}
	}
	// Crossing a bucket edge must not change list-order priority or retain dead targets.
	enemies := []Actor{{X: 101, Y: 400, HP: 1}, {X: 99, Y: 400, HP: 1}}
	shot := Projectile{X: 100, Y: 400, Skill: Skill{Kind: "fire"}}
	buckets.reset(enemies)
	if got := buckets.target(enemies, shot); got != 0 {
		t.Fatalf("overlap selected %d", got)
	}
	enemies[0].HP = 0
	if got := buckets.target(enemies, shot); got != 1 {
		t.Fatalf("dead target selected %d", got)
	}
	if n := testing.AllocsPerRun(100, func() { buckets.reset(enemies); buckets.target(enemies, shot) }); n != 0 {
		t.Fatalf("warmed buckets allocate %v", n)
	}
}

func BenchmarkProjectileCandidates(b *testing.B) {
	for _, n := range []int{12, 120, 400} {
		enemies := make([]Actor, n)
		for i := range enemies {
			enemies[i] = Actor{X: float64(i%20)*80 + 30, Y: float64(i/20)*20 + 300, HP: 100}
		}
		for _, indexed := range []bool{false, true} {
			name := "linear"
			if indexed {
				name = "buckets"
			}
			b.Run(fmt.Sprint(n)+"/"+name, func(b *testing.B) {
				var buckets projectileBuckets
				buckets.reset(enemies)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if indexed {
						buckets.reset(enemies)
					}
					for j := 0; j < 40; j++ {
						shot := Projectile{X: float64(j%20)*80 + 40, Y: float64(n/20)*20 + 300 + float64(j/20)*100, Skill: Skill{Kind: "fire"}}
						if indexed {
							buckets.target(enemies, shot)
						} else {
							linearProjectileTarget(enemies, shot)
						}
					}
				}
			})
		}
	}
}

func TestProjectileBucketsRefreshAfterEnemyMovement(t *testing.T) {
	r := testRun()
	r.Status = "cleared"
	r.Enemies = []Actor{{ID: "first", X: 101, Y: 400, HP: 100, MaxHP: 100}, {ID: "second", X: 99, Y: 400, HP: 100, MaxHP: 100}}
	shot := Projectile{X: 100, Y: 400, Life: 2, Power: 10, Skill: Skill{Kind: "fire"}}
	r.Projectiles = []Projectile{shot}
	r.tick(Input{}, 0)
	if r.Enemies[0].HP != 90 || r.Enemies[1].HP != 100 {
		t.Fatalf("first collision lost list order: %v %v", r.Enemies[0].HP, r.Enemies[1].HP)
	}
	r.Enemies[0].X = 800
	r.Projectiles = []Projectile{shot}
	r.tick(Input{}, 0)
	if r.Enemies[0].HP != 90 || r.Enemies[1].HP != 90 {
		t.Fatalf("next tick retained stale buckets: %v %v", r.Enemies[0].HP, r.Enemies[1].HP)
	}
}
