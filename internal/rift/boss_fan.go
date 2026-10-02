package rift

import "math"

// fireBossFan uses the same saved aim and rotation shown during its warning.
func (r *Run) fireBossFan(e *Actor) {
	if !r.clearProjectilePath(e, e) {
		return
	}
	angle := math.Atan2(e.TargetY-e.Y, e.TargetX-e.X) + e.FanRotation
	shot := e.Shot
	if shot == "" {
		shot = "arrow"
	}
	for n := -2; n <= 2; n++ {
		direction := angle + float64(n)*.22
		r.Counter++
		r.Projectiles = append(r.Projectiles, Projectile{Elevation: e.Elevation, OwnerID: e.ID, ID: r.Counter, X: e.X, Y: e.Y, VX: math.Cos(direction) * 260, VY: math.Sin(direction) * 260, Power: math.Max(18, e.Damage) * .65, Enemy: true, Life: 4, Kind: shot})
	}
	r.eventAtHeight("boss_fan", e.X, e.Y-30, 0, e.Elevation)
}
