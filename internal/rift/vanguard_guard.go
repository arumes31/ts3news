package rift

func (r *Run) rewardVanguardGuard(previousGuards int) {
	if r.Build.Class != "vanguard" || r.Player.HP <= 0 || r.Stats.Guards <= previousGuards || r.SkillTimers["perfect_guard"] <= 0 || r.SkillTimers["vanguard_guard_reward"] > 0 {
		return
	}
	r.SkillTimers["vanguard_guard_reward"] = 1
	if r.Resource >= 3 {
		return
	}
	r.Resource++
	r.eventAtHeight("resource", r.Player.X, r.Player.Y-35, 1, r.Player.Elevation)
	r.eventAtHeight("vanguard_guard", r.Player.X, r.Player.Y-35, 0, r.Player.Elevation)
}
