package rift

var bossLaneReservationKeys = [3]string{"hazard-slam-lane-0", "hazard-slam-lane-1", "hazard-slam-lane-2"}

// reservedBossArea suppresses arena hazard contact only. Simultaneous ring,
// lane and channel danger zones take precedence over reserved escape areas.
func (r *Run) reservedBossArea(x, y float64) bool {
	lane := bossLane(y)
	reserved := false
	for danger, key := range bossLaneReservationKeys {
		if r.SkillTimers[key] > 0 {
			if lane == danger {
				return false
			}
			reserved = true
		}
	}
	for _, e := range r.Enemies {
		if e.Kind != "boss" {
			continue
		}
		warning := e.HP > 0 && e.Windup > 0
		if e.LaneSlams && (warning && e.AttackName == "Time Pulse" || r.SkillTimers["hazard-channel-"+e.ID] > 0) {
			if channelDanger(e, x, y) {
				return false
			}
			dx, dy := (x-e.TargetX)/200, (y-e.TargetY)/90
			if dx*dx+dy*dy <= 1 {
				reserved = true
			}
		}
		if e.RingAttack && (warning && e.AttackName == "Void Ring" || r.SkillTimers["hazard-ring-"+e.ID] > 0) {
			if ringDanger(e, x, y) {
				return false
			}
			dx, dy := (x-e.TargetX)/200, (y-e.TargetY)/90
			if dx*dx+dy*dy <= 1 {
				reserved = true
			}
		}
		if warning && e.LaneSlams && e.AttackName == "Lane Slam" && e.SlamLane >= 0 && e.SlamLane <= 2 {
			if lane == e.SlamLane {
				return false
			}
			reserved = true
		}
	}
	return reserved
}
