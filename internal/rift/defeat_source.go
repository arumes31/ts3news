package rift

func (r *Run) hurtPlayerFromEnemy(damage, x, y float64, ownerID string) {
	guards := r.Stats.Guards
	alive := r.Player.HP > 0
	before := r.Stats.DamageTaken
	r.hurtPlayer(damage, x, y)
	r.rewardVanguardGuard(guards)
	r.Stats.EnemyDamageTaken += r.Stats.DamageTaken - before
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

func (r *Run) hurtPlayerFromHazard(damage, x, y float64) {
	before := r.Stats.DamageTaken
	r.hurtPlayer(damage, x, y)
	r.Stats.HazardDamageTaken += r.Stats.DamageTaken - before
}

// HazardDefeat records only the lethal contact, including its evasion rule.
type HazardDefeat struct {
	Kind     string `json:"kind"`
	Jumpable bool   `json:"jumpable"`
}

func (r *Run) hurtPlayerFromNamedHazard(damage, x, y float64, source HazardDefeat) {
	alive := r.Player.HP > 0
	r.hurtPlayerFromHazard(damage, x, y)
	if alive && r.Player.HP <= 0 {
		r.DefeatedByHazard = &source
	}
}
