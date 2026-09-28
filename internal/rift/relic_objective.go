package rift

import "math"

func (r *Run) beginRelicObjective() {
	relic := Actor{X: 430, Y: 330}
	destination := Actor{X: 1450, Y: 480}
	if e := r.Arena().Exit; e != nil {
		destination.X, destination.Y = e.X, e.Y
	}
	settle(&relic, r.Arena().solidObstacles())
	settle(&destination, r.Arena().solidObstacles())
	r.RoomObjective = &RoomObjective{Kind: "carry_relic", Name: "Carry the relic", Description: "Pick up the relic and carry it to the exit seal. Movement is 30% slower while carrying; attacks and jumps remain available. Defeat the patrol to secure the tier.", Target: 1, Relic: &ObjectivePickup{ID: 1, X: relic.X, Y: relic.Y}, Zone: &ObjectiveZone{X: destination.X, Y: destination.Y, RadiusX: 45, RadiusY: 28}}
}

func (r *Run) tickRelicObjective() {
	o := r.RoomObjective
	if o == nil || o.Kind != "carry_relic" || o.Complete || o.Relic == nil || o.Zone == nil || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 || r.Player.Jump > .1 {
		return
	}
	if !o.Carrying {
		target := Actor{X: o.Relic.X, Y: o.Relic.Y}
		if math.Hypot(r.Player.X-target.X, r.Player.Y-target.Y) > 28 || !r.clearMeleePath(&r.Player, &target) {
			return
		}
		o.Carrying = true
		o.Relic.Collected = true
		r.event("relic_pickup", target.X, target.Y, 0)
		return
	}
	if o.Zone.contains(r.Player) {
		o.Carrying = false
		o.Complete = true
		o.Collected = 1
		r.event("relic_delivered", o.Zone.X, o.Zone.Y, 0)
	}
}
