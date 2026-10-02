package rift

// Refuge safety is derived from the saved room and current combat state. It
// never protects against a living wave, including after loading an older run.
func (r *Run) inWaveRestAlcove() bool {
	o := r.RoomObjective
	if r.Practice != nil || r.Status != "fighting" || r.Player.HP <= 0 || o == nil || o.Kind != "survive_waves" || o.Complete || o.NextWaveSeconds <= 0 || o.Wave >= o.Target {
		return false
	}
	rest := r.Arena().RestAlcove
	if rest == nil || !contains(*rest, r.Player.X, r.Player.Y, 0) {
		return false
	}
	for _, enemy := range r.Enemies {
		if enemy.HP > 0 {
			return false
		}
	}
	return true
}
