package rift

func (r *Run) hurtPlayerFromEnemy(damage, x, y float64, ownerID string) {
	alive := r.Player.HP > 0
	r.hurtPlayer(damage, x, y)
	if !alive || r.Player.HP > 0 || ownerID == "" {
		return
	}
	for _, e := range r.Enemies {
		if e.ID == ownerID && e.Kind == "boss" {
			r.DefeatedByBoss = e.Name
			return
		}
	}
}
