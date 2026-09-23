package rift

import "math"

func (r *Run) beginCollapseObjective() {
	exit := Actor{X: 1450, Y: 320}
	settle(&exit, r.Arena().solidObstacles())
	r.RoomObjective = &RoomObjective{Kind: "escape_collapse", Name: "Escape the collapse", Description: "After a three-second warning, the collapse advances from the left. Stay ahead of it and reach the exit seal. Surviving enemies grant no loot or kill credit.", Target: 1, Zone: &ObjectiveZone{X: exit.X, Y: exit.Y, RadiusX: 45, RadiusY: 28}}
}

func (r *Run) tickCollapseObjective(dt float64) {
	o := r.RoomObjective
	if o == nil || o.Kind != "escape_collapse" || o.Complete || o.Zone == nil || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	old := o.Seconds
	o.Seconds += dt
	o.CollapseX = math.Min(Width, math.Max(0, o.Seconds-3)*90)
	o.CollapseHitCooldown = math.Max(0, o.CollapseHitCooldown-dt)
	if old < 3 && o.Seconds >= 3 {
		r.event("collapse_start", o.CollapseX, r.Player.Y, 0)
	}
	if o.CollapseX > 0 && r.Player.X <= o.CollapseX && o.CollapseHitCooldown == 0 {
		r.hurtPlayer(math.Max(15, r.Player.MaxHP*.12), r.Player.X, r.Player.Y)
		o.CollapseHitCooldown = 1
		r.event("collapse_hit", r.Player.X, r.Player.Y, 0)
	}
	if r.Player.HP <= 0 || r.Player.Jump > .1 || !o.Zone.contains(r.Player) {
		return
	}
	o.Complete = true
	o.Collected = 1
	for i := range r.Enemies {
		if r.Enemies[i].HP > 0 {
			r.escapeEnemy(i)
		}
	}
	r.Projectiles = []Projectile{}
	r.event("collapse_escaped", o.Zone.X, o.Zone.Y, 0)
}
