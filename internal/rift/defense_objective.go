package rift

import (
	"math"
	"slices"
)

// DefenseLane keeps its assigned patrol and ward damage stable across reloads.
type DefenseLane struct {
	Ward      Actor    `json:"ward"`
	EnemyIDs  []string `json:"enemy_ids"`
	Seconds   float64  `json:"seconds"`
	Contested bool     `json:"contested"`
}

func (r *Run) beginDefenseObjective() {
	o := &RoomObjective{Kind: "split_defense", Name: "Defend both lanes", Description: "Intercept the patrol in both lanes. Nearby fighters draw enemies away from their wards. Each attacker at a ward drains five light per second, up to two attackers. Both wards must survive until the patrol is defeated.", Target: 2}
	for i, y := range []float64{325, 485} {
		ward := Actor{ID: []string{"upper-ward", "lower-ward"}[i], Name: []string{"Upper ward", "Lower ward"}[i], Kind: "lantern", X: 350, Y: y, HP: 100, MaxHP: 100, Facing: 1, Pose: "idle"}
		settle(&ward, r.Arena().solidObstacles())
		o.Lanes = append(o.Lanes, DefenseLane{Ward: ward, EnemyIDs: []string{}})
	}
	assigned := 0
	for _, e := range r.Enemies {
		if e.HP <= 0 || e.Kind == "boss" || e.Kind == "treasure" || e.isObjectiveProp() {
			continue
		}
		lane := &o.Lanes[assigned%2]
		lane.EnemyIDs = append(lane.EnemyIDs, e.ID)
		assigned++
	}
	r.RoomObjective = o
}

func (r *Run) defenseLost() bool {
	o := r.RoomObjective
	if o == nil || o.Kind != "split_defense" {
		return false
	}
	for _, lane := range o.Lanes {
		if lane.Ward.HP <= 0 {
			return true
		}
	}
	return false
}

// Called after ordinary enemy recovery timers, before choosing a player attack.
func (r *Run) tickDefenseEnemy(e *Actor, dt float64) bool {
	o := r.RoomObjective
	if o == nil || o.Kind != "split_defense" || o.Complete || e.Windup > 0 || e.PoseTime > 0 || math.Hypot(r.Player.X-e.X, r.Player.Y-e.Y) < 180 {
		return false
	}
	for i := range o.Lanes {
		lane := &o.Lanes[i]
		if !slices.Contains(lane.EnemyIDs, e.ID) {
			continue
		}
		target := &lane.Ward
		if r.clearPursuitPath(e, target) {
			e.RouteX, e.RouteY = 0, 0
		}
		dx, dy := target.X-e.X, target.Y-e.Y
		distance := math.Hypot(dx, dy)
		if dx != 0 {
			e.Facing = math.Copysign(1, dx)
		}
		if distance <= 45 && r.clearMeleePath(e, target) {
			e.Pose = "attack"
			return true
		}
		speed := clamp(e.Speed, 75, 110)
		step := math.Min(distance, speed*dt)
		if distance > 0 {
			r.moveActor(e, dx/distance*step, dy/distance*step, true)
		}
		e.Pose = "run"
		return true
	}
	return false
}

func (r *Run) tickDefenseObjective(dt float64) {
	o := r.RoomObjective
	if o == nil || o.Kind != "split_defense" || o.Complete || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 || r.defenseLost() {
		return
	}
	alive := 0
	for _, e := range r.Enemies {
		if e.HP > 0 {
			alive++
		}
	}
	if alive == 0 {
		o.Complete = true
		o.Collected = 2
		for i := range o.Lanes {
			o.Lanes[i].Contested = false
		}
		r.event("lanes_protected", 350, 410, 0)
		return
	}
	for i := range o.Lanes {
		lane := &o.Lanes[i]
		threats := 0
		for _, e := range r.Enemies {
			if e.HP > 0 && e.Knockdown <= 0 && e.PoseTime <= 0 && slices.Contains(lane.EnemyIDs, e.ID) && math.Hypot(e.X-lane.Ward.X, e.Y-lane.Ward.Y) <= 45 && r.clearMeleePath(&e, &lane.Ward) {
				threats++
			}
		}
		if threats > 0 && !lane.Contested {
			r.event("lane_threat", lane.Ward.X, lane.Ward.Y, float64(i+1))
		}
		lane.Contested = threats > 0
		lane.Seconds += dt
		pulses := math.Floor(lane.Seconds)
		lane.Seconds -= pulses
		if pulses == 0 || threats == 0 {
			continue
		}
		damage := float64(min(2, threats)) * 5 * pulses
		lane.Ward.HP = math.Max(0, lane.Ward.HP-damage)
		r.event("lane_hurt", lane.Ward.X, lane.Ward.Y, damage)
		if lane.Ward.HP == 0 {
			r.event("lane_lost", lane.Ward.X, lane.Ward.Y, float64(i+1))
		}
	}
}
