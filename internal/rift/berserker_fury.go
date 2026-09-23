package rift

// BerserkerFury is evaluated at impact, including for traveling projectiles.
func (r *Run) BerserkerFury() bool {
	return r.Build.Class == "berserker" && r.Player.MaxHP > 0 && r.Player.HP > 0 && r.Player.HP <= r.Player.MaxHP*.3
}
