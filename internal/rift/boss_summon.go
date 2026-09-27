package rift

import (
	"fmt"
	"math"
)

// tickBossSummon may append actors. Call before taking a pointer into Enemies.
func (r *Run) tickBossSummon(i int, dt float64) bool {
	e := &r.Enemies[i]
	if e.HP <= 0 || e.Kind != "boss" || e.SummonTimer <= 0 {
		return false
	}
	e.SummonTimer = math.Max(0, math.Min(4, e.SummonTimer)-dt)
	e.Windup = 0
	e.AttackName = ""
	e.Cooldown = math.Max(e.Cooldown, 1)
	e.PoseTime = math.Max(0, e.PoseTime-dt)
	if e.PoseTime == 0 {
		e.Pose = "idle"
	}
	if e.SummonReleased || e.SummonTimer > 3 {
		return true
	}
	e.SummonReleased = true
	boss := *e
	alive, owned := 0, 0
	reserved := []Obstacle{{r.Player.X - 65, r.Player.Y - 35, 130, 70}}
	for _, other := range r.Enemies {
		if other.HP <= 0 {
			continue
		}
		alive++
		if other.SummonOwner == boss.ID {
			owned++
		}
		reserved = append(reserved, Obstacle{other.X - 25, other.Y - 15, 50, 30})
	}
	var templates []Actor
	for _, room := range r.EncounterPlan {
		for _, a := range room {
			if a.Kind != "boss" && a.Kind != "treasure" && !a.isObjectiveProp() && a.ArtKey != "" {
				templates = append(templates, a)
			}
		}
	}
	if len(templates) == 0 {
		return true
	}
	for n := 0; n < 2 && owned < 2 && alive < 8; n++ {
		a := templates[(boss.SummonPhase+n)%len(templates)]
		a.X = boss.X + float64(n*2-1)*110
		a.Y = boss.Y
		if !r.Arena().settleEnemySpawn(&a, reserved...) {
			continue
		}
		a.Elevation = r.Arena().Elevation(a.X, a.Y)
		r.Counter++
		a.ID = fmt.Sprintf("boss-summon-%d", r.Counter)
		a.Summoned = true
		a.SummonOwner = boss.ID
		a.ArrivalVulnerability = .85
		a.Pose = "spawn"
		a.PoseTime = .85
		a.Cooldown = .85
		a.Alerted = true
		a.Patrol = false
		a.HP = a.MaxHP
		r.Enemies = append(r.Enemies, a)
		r.eventAtHeight("summon_spawn", a.X, a.Y-25, 0, a.Elevation)
		reserved = append(reserved, Obstacle{a.X - 25, a.Y - 15, 50, 30})
		owned++
		alive++
	}
	return true
}

// Dismissal removes owned attacks without going through defeat or reward logic.
func (r *Run) dismissBossSummons(owner string) {
	for i := range r.Enemies {
		a := &r.Enemies[i]
		if a.SummonOwner != owner {
			continue
		}
		if a.HP <= 0 {
			continue
		}
		a.HP = 0
		a.Pose = "escape"
		a.PoseTime = 0
		a.Windup = 0
		a.AttackName = ""
		r.endTargetMark(*a, "dismissed")
		r.eventAtHeight("summon_dismiss", a.X, a.Y, 0, a.Elevation)
	}
}
