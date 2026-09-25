package rift

func (r *Run) hurtPlayerFromEnemy(damage, x, y float64, ownerID string) {
	guards := r.Stats.Guards
	alive := r.Player.HP > 0
	before := r.Stats.DamageTaken
	r.hurtPlayer(damage, x, y)
	r.rewardVanguardGuard(guards)
	loss := r.Stats.DamageTaken - before
	r.Stats.EnemyDamageTaken += loss
	r.recordMonsterDamage(ownerID, loss)
	if !alive || r.Player.HP > 0 || ownerID == "" {
		return
	}
	for _, e := range r.Enemies {
		if e.ID == ownerID {
			if e.Kind == "boss" {
				r.DefeatedByBoss = e.Name
			}
			r.DefeatedByEnemy = e.Name
			r.DefeatCause = "Fallen to enemy: " + e.Name
			return
		}
	}
}

func (r *Run) hurtPlayerFromHazard(damage, x, y float64) {
	alive := r.Player.HP > 0
	before := r.Stats.DamageTaken
	r.hurtPlayer(damage, x, y)
	r.Stats.HazardDamageTaken += r.Stats.DamageTaken - before
	if alive && r.Player.HP <= 0 && r.DefeatCause == "" {
		r.DefeatCause = "Fallen to hazard damage"
	}
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
		if source.Jumpable {
			r.DefeatCause = "Fallen to " + source.Kind + " hazard (jump to evade)"
		} else {
			r.DefeatCause = "Fallen to " + source.Kind + " hazard (move clear)"
		}
	}
}
