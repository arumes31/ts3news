package rift

import "math"

// The target opens for two seconds after a three-second preparation, every six seconds.
func (r *Run) ultimatePracticeWindow() bool {
	phase := math.Mod(r.Clock, 6)
	return phase >= 3 && phase < 5
}

func (r *Run) recordPracticeUltimate(skill Skill) {
	if r.Practice == nil || r.Practice.Mode != "ultimate" || r.Build.Ultimate == nil || skill.ID != r.Build.Ultimate.ID || !r.ultimatePracticeWindow() {
		return
	}
	r.Practice.UltimateTimed = true
}
