package rift

import "math"

func (r *Run) beginLanternObjective() {
	lantern := Actor{ID: "ward-lantern", Name: "Ward lantern", Kind: "lantern", X: 800, Y: 480, HP: 100, MaxHP: 100, Facing: 1, Pose: "idle"}
	settle(&lantern, r.Arena().solidObstacles())
	r.RoomObjective = &RoomObjective{Kind: "protect_lantern", Name: "Protect the lantern", Description: "Keep enemies away from the lantern until the patrol is defeated. Each nearby enemy drains five light per second, up to three enemies. If its light reaches zero, the expedition fails.", Target: 1, Lantern: &lantern, Zone: &ObjectiveZone{X: lantern.X, Y: lantern.Y, RadiusX: 150, RadiusY: 90}}
}

func (r *Run) lanternExtinguished() bool {
	o := r.RoomObjective
	return o != nil && o.Kind == "protect_lantern" && o.Lantern != nil && o.Lantern.HP <= 0
}

func (r *Run) tickLanternObjective(dt float64) {
	o := r.RoomObjective
	if o == nil || o.Kind != "protect_lantern" || o.Complete || o.Lantern == nil || o.Zone == nil || o.Lantern.HP <= 0 || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	living, threats := 0, 0
	for _, e := range r.Enemies {
		if e.HP > 0 {
			living++
			if e.Kind != "treasure" && !e.isObjectiveProp() && o.Zone.contains(e) {
				threats++
			}
		}
	}
	contested := threats > 0
	if contested && !o.Contested {
		r.event("lantern_threat", o.Lantern.X, o.Lantern.Y, 0)
	}
	o.Contested = contested
	if living == 0 {
		o.Complete = true
		o.Collected = 1
		o.Contested = false
		r.event("lantern_protected", o.Lantern.X, o.Lantern.Y, 0)
		return
	}
	o.Seconds += dt
	if o.Seconds < 1 {
		return
	}
	pulses := math.Floor(o.Seconds)
	o.Seconds -= pulses
	if threats == 0 {
		return
	}
	damage := float64(min(3, threats)) * 5 * pulses
	o.Lantern.HP = math.Max(0, o.Lantern.HP-damage)
	r.event("lantern_hurt", o.Lantern.X, o.Lantern.Y, damage)
	if o.Lantern.HP == 0 {
		r.event("lantern_extinguished", o.Lantern.X, o.Lantern.Y, 0)
	}
}
