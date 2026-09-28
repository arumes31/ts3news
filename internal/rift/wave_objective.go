package rift

import "math"

// Freeze the groups at room entry so later catalog changes cannot alter a saved fight.
func (r *Run) beginWaveObjective() {
	if len(r.Enemies) < 3 {
		return
	}
	o := &RoomObjective{Kind: "survive_waves", Name: "Survive the waves", Description: "Defeat all three waves. Reinforcements arrive after a short warning.", Target: 3, Wave: 1}
	for group := 0; group < o.Target; group++ {
		start, end := group*len(r.Enemies)/o.Target, (group+1)*len(r.Enemies)/o.Target
		o.Waves = append(o.Waves, append([]Actor(nil), r.Enemies[start:end]...))
	}
	if gate := r.Arena().WaveGate; gate != nil {
		o.Gate = &WaveGate{Obstacle: *gate}
		o.Description += " The central gate warns before closing; use the upper or lower bypass. It opens between waves."
	}
	r.RoomObjective = o
	r.Enemies = append([]Actor(nil), o.Waves[0]...)
	r.warnWaveGate()
}

func (r *Run) tickWaveObjective(dt float64) {
	o := r.RoomObjective
	if o == nil || o.Kind != "survive_waves" || o.Complete || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	for _, enemy := range r.Enemies {
		if enemy.HP > 0 {
			r.tickWaveGate(dt)
			return
		}
	}
	r.openWaveGate()
	if o.Wave == o.Target {
		o.Complete = true
		r.event("waves_complete", r.Player.X, r.Player.Y, float64(o.Wave))
		return
	}
	if o.NextWaveSeconds == 0 {
		o.NextWaveSeconds = 2.5
		r.event("wave_incoming", r.Player.X, r.Player.Y, float64(o.Wave+1))
		return
	}
	o.NextWaveSeconds = math.Max(0, o.NextWaveSeconds-dt)
	if o.NextWaveSeconds > 1e-9 {
		return
	}
	o.NextWaveSeconds = 0
	// Keep defeated actors for room summaries, boss records and treasure escape accounting.
	arrivals := append([]Actor{}, o.Waves[o.Wave]...)
	r.Arena().settleEnemySpawns(arrivals, bossSpawnReservations(r.Enemies)...)
	for _, enemy := range arrivals {
		enemy.Elevation = r.Arena().Elevation(enemy.X, enemy.Y)
		enemy.Cooldown = math.Max(enemy.Cooldown, 1.2)
		r.Enemies = append(r.Enemies, enemy)
	}
	o.Wave++
	r.warnWaveGate()
	r.observeRoomMonsters()
	r.event("wave_start", r.Player.X, r.Player.Y, float64(o.Wave))
}
