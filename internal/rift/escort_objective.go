package rift

import "math"

func (r *Run) beginEscortObjective() {
	r.RoomObjective = &RoomObjective{Kind: "escort_spirit", Name: "Escort the spirit", Description: "Stay near the spirit and clear nearby enemies so it can reach the exit. The spirit waits when threatened or left behind. Defeat the patrol to secure the tier.", Target: 1, Escort: &Actor{ID: "escort-spirit", Name: "Lost spirit", Kind: "spirit", X: 350, Y: 320, HP: 1, MaxHP: 1, Facing: 1, Speed: 75, Pose: "idle"}, Zone: &ObjectiveZone{X: 1450, Y: 320, RadiusX: 45, RadiusY: 28}}
}
func (r *Run) tickEscortObjective(dt float64) {
	o := r.RoomObjective
	if o == nil || o.Kind != "escort_spirit" || o.Complete || o.Escort == nil || o.Zone == nil || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	spirit := o.Escort
	wasMoving, wasContested := o.EscortMoving, o.Contested
	o.Contested = false
	for _, enemy := range r.Enemies {
		if enemy.HP > 0 && enemy.Kind != "treasure" && !enemy.isObjectiveProp() && math.Hypot((enemy.X-spirit.X)/100, (enemy.Y-spirit.Y)/55) <= 1 {
			o.Contested = true
			break
		}
	}
	nearby := math.Hypot((r.Player.X-spirit.X)/150, (r.Player.Y-spirit.Y)/90) <= 1
	o.EscortMoving = nearby && !o.Contested
	spirit.Pose = "idle"
	if o.Contested && !wasContested {
		r.event("spirit_threat", spirit.X, spirit.Y, 0)
	}
	if !o.EscortMoving {
		return
	}
	if !wasMoving {
		r.event("spirit_move", spirit.X, spirit.Y, 0)
	}
	spirit.Pose = "run"
	r.moveActor(spirit, math.Min(o.Zone.X-spirit.X, spirit.Speed*dt), 0, false)
	if spirit.X >= o.Zone.X-1e-9 {
		spirit.X = o.Zone.X
		spirit.Pose = "idle"
		o.EscortMoving = false
		o.Complete = true
		o.Collected = 1
		r.event("spirit_arrived", spirit.X, spirit.Y, 0)
	}
}
