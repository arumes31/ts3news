package rift

// BossPracticeEnrageSeconds is the fixed combat-time limit for the optional drill.
const BossPracticeEnrageSeconds = 30

func (r *Run) bossPracticeEnraged() bool {
	return r.Practice != nil && r.Practice.Mode == "boss" && r.Practice.EnrageSeconds > 0 && r.Clock >= r.Practice.EnrageSeconds
}

func (r *Run) tickBossPracticeEnrage() {
	if !r.bossPracticeEnraged() || r.Practice.EnrageTriggered || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	for _, boss := range r.Enemies {
		if boss.HP > 0 && boss.Kind == "boss" {
			r.Practice.EnrageTriggered = true
			r.eventAtHeight("practice_enrage", boss.X, boss.Y-30, 0, boss.Elevation)
			return
		}
	}
}

// Apply enrage at impact, before the usual armor and guard reduction. This also
// covers projectiles already in flight without changing their saved power.
func (r *Run) bossPracticeDamage(damage float64, ownerID string) float64 {
	if r.bossPracticeEnraged() && r.Practice.BossStart != nil && ownerID == r.Practice.BossStart.ID {
		return damage * 1.5
	}
	return damage
}
