package rift

// deadBossProjectile checks ownership before damage and after projectile hits,
// since a friendly shot can defeat the boss during the same projectile pass.
func (r *Run) deadBossProjectile(shot Projectile) bool {
	if !shot.Enemy || shot.OwnerID == "" {
		return false
	}
	for _, e := range r.Enemies {
		if e.ID == shot.OwnerID {
			return (e.Kind == "boss" || e.SummonOwner != "") && e.HP <= 0
		}
	}
	return false
}
