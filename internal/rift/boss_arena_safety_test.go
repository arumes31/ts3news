package rift

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"ts3news/internal/content"
)

// Exercise the actual final-tier spawn and approach in every campaign arena.
// Other defenders are omitted to attribute damage specifically to the boss.
func TestCampaignBossSpawnAndOpeningSafety(t *testing.T) {
	for _, level := range Campaign() {
		for phase := 1; phase <= 3; phase++ {
			for attack := 0; attack < 2; attack++ {
				t.Run(fmt.Sprintf("%d/phase-%d/attack-%d", level.ID, phase, attack), func(t *testing.T) {
					r := NewRunAtLevel("boss-arena-safety", Build{HP: 1000}, time.Unix(0, 0), content.AbyssMobCatalog(), level.ID)
					r.Room = 2
					r.spawnRoom()
					boss := r.Enemies[0]
					if boss.Kind != "boss" {
						t.Fatal("final tier has no opening boss")
					}
					boss.Phase, boss.Attacks = phase, attack
					r.Enemies = []Actor{boss}
					for _, actor := range []Actor{r.Player, boss} {
						for _, wall := range r.Arena().solidObstacles() {
							if contains(wall, actor.X, actor.Y, actorClearance(&actor)-.001) {
								t.Fatalf("%s spawned in cover", actor.Kind)
							}
						}
					}
					if math.Abs(boss.X-r.Player.X) < 190 {
						t.Fatal("boss spawned within attack reach")
					}
					hp := r.Player.HP
					for step := 0; step < 3000 && r.Enemies[0].Windup == 0; step++ {
						r.tick(Input{}, .02)
						if r.Player.HP != hp || r.Enemies[0].Attacks != attack {
							t.Fatal("spawn approach caused untelegraphed boss damage")
						}
					}
					if r.Enemies[0].Windup == 0 {
						t.Fatalf("boss cannot reach entry lane from %v,%v; stopped at %v,%v", boss.X, boss.Y, r.Enemies[0].X, r.Enemies[0].Y)
					}
					if r.Enemies[0].Windup != r.NextBossAttack(r.Enemies[0]).Windup {
						t.Fatal("opening warning was shortened")
					}
					saved, err := json.Marshal(r)
					if err != nil {
						t.Fatal(err)
					}
					var stationary Run
					if err := json.Unmarshal(saved, &stationary); err != nil {
						t.Fatal(err)
					}
					fan := r.NextBossAttack(r.Enemies[0]).Kind == "fan"
					impactSteps := 100
					if fan {
						impactSteps = 150
					}
					for step := 0; step < impactSteps; step++ {
						stationary.tick(Input{}, .02)
					}
					if stationary.Player.HP >= hp {
						t.Fatal("stationary control was never threatened by boss attack")
					}
					if fan {
						escaped := false
						for _, direction := range []Input{{Y: 1}, {Y: -1}, {X: 1}, {X: -1}, {X: 1, Y: 1}, {X: 1, Y: -1}, {X: -1, Y: 1}, {X: -1, Y: -1}} {
							var moving Run
							if err := json.Unmarshal(saved, &moving); err != nil {
								t.Fatal(err)
							}
							for step := 0; step < 15; step++ {
								moving.tick(Input{}, .02)
							}
							for step := 0; step < 150; step++ {
								moving.tick(direction, .02)
							}
							if moving.Player.HP == hp && moving.Enemies[0].Attacks == attack+1 {
								escaped = true
								break
							}
						}
						if !escaped {
							t.Fatal("fan has no tested movement escape after reaction delay")
						}
						for step := 0; step < 100 && r.Enemies[0].Attacks == attack; step++ {
							r.tick(Input{}, .02)
						}
						if len(r.Projectiles) != 5 {
							t.Fatal("fan did not release five shots")
						}
						return
					}
					if attack == 0 {
						for r.Enemies[0].Windup > .2 {
							r.tick(Input{}, .02)
						}
						r.tick(Input{Jump: true}, .02)
					}
					for step := 0; step < 100 && r.Enemies[0].Attacks == attack; step++ {
						r.tick(Input{}, .02)
					}
					if attack == 1 {
						if len(r.Projectiles) != 1 {
							t.Fatal("volley did not release from arena approach")
						}
						for step := 0; step < 40; step++ {
							r.tick(Input{Y: 1}, .02)
						}
					}
					if r.Enemies[0].Attacks != attack+1 || r.Player.HP != hp {
						t.Fatal("entry lane did not permit attack counterplay")
					}
				})
			}
		}
	}
}
