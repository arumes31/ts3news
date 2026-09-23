package rift

import "math"

func (r *Run) abilities() []Skill {
	skills := append([]Skill{}, r.Build.Signatures...)
	skills = append(skills, r.Build.Skills...)
	if r.Build.Ultimate != nil {
		skills = append(skills, *r.Build.Ultimate)
	}
	return skills
}

// classCast preserves Abyss's three-charge builder/finisher model, with round
// recovery expressed as 1.5 seconds in this action mode.
func (r *Run) classCast(skill Skill) (int, string) {
	if skill.Role == "builder" {
		before := r.Resource
		r.Resource = min(3, r.Resource+1)
		if r.Resource > before {
			r.eventAtHeight("resource", r.Player.X, r.Player.Y-35, float64(r.Resource-before), r.Player.Elevation)
		}
		return 0, ""
	}
	if skill.Role != "finisher" {
		return 0, ""
	}
	charges, marked := r.Resource, r.Marked
	r.Resource = 0
	r.Marked = ""
	if charges == 0 {
		r.Stats.EmptyFinishers++
		return 0, marked
	}
	r.Stats.ChargedFinishers++
	r.Stats.ChargesSpent += charges
	p := &r.Player
	switch r.Build.Class {
	case "chronomancer":
		// Rewind actions, never environment/status timers such as hazard-hit
		// protection. Reducing those can make a helpful skill harm its caster.
		for _, ability := range r.abilities() {
			if ability.ID != skill.ID {
				r.SkillTimers[ability.ID] = math.Max(0, r.SkillTimers[ability.ID]-1.5)
			}
		}
		r.SkillTimers["jump"] = math.Max(0, r.SkillTimers["jump"]-1.5)
	case "bloodblade":
		r.healPlayerBySkill(p.MaxHP*.04*float64(charges), skill.ID)
	case "alchemist":
		r.healPlayerBySkill(p.MaxHP*.03*float64(charges), skill.ID)
	case "voidwalker":
		cost := math.Min(math.Max(0, p.HP-1), p.MaxHP*.05)
		p.HP -= cost
		r.eventAtHeight("void_cost", p.X, p.Y-35, cost, p.Elevation)
	case "runesmith":
		r.addBarrier(15+r.Build.Armor*2, skill.ID)
	}
	return charges, marked
}

func (r *Run) skillHit(index int, damage float64, skill Skill, charges int, marked string) {
	e := &r.Enemies[index]
	if e.HP <= 0 {
		return
	}
	if skill.Role == "builder" {
		if r.Marked != e.ID {
			r.eventAtHeight("mark_target", e.X, e.Y-30, 0, e.Elevation)
		}
		r.Marked = e.ID
	}
	pierce := skill.Pierce
	if charges > 0 && skill.Role == "finisher" {
		switch r.Build.Class {
		case "vanguard":
			pierce += .3
		case "berserker":
			if e.HP <= e.MaxHP*.5 {
				damage *= 1.25
			}
		case "marksman":
			if marked == e.ID {
				pierce += .6
			}
		case "beastmaster":
			damage *= 1 + float64(min(3, r.Build.Pets))*.1
			r.event("pack", e.X, e.Y, 0)
		case "elementalist":
			if marked == e.ID {
				damage *= 1.2
			}
		case "oracle":
			damage *= 1.15
		case "geomancer":
			pierce += .45
		case "voidwalker":
			if r.Player.HP > 1 {
				damage *= 1.25
			}
			if marked == e.ID {
				pierce += .25
			}
		case "runesmith":
			if r.Build.Relic {
				damage *= 1.15
			}
		case "alchemist":
			if marked == e.ID {
				pierce += .35
			}
		}
	}
	if e.Kind == "boss" && e.HP > 0 && ((skill.Role == "finisher" && charges > 0) || skill.Kind == "quake" || skill.Kind == "ultimate") {
		e.Pose = "stagger"
		e.PoseTime = .45
		e.Windup = 0
		e.Cooldown = math.Max(e.Cooldown, 0.8)
		r.eventAtHeight("boss_stagger", e.X, e.Y-30, 0, e.Elevation)
	}
	if skill.Kind == "ice" && EnemyTraining(e.Kind).Interruptible {
		e.Windup = 0
		e.Cooldown = math.Max(e.Cooldown, 1)
	}
	effect := skill.Kind
	if effect == "" {
		effect = "hit"
	}
	before := e.HP
	r.hurtEnemyPiercing(index, damage, effect, pierce)
	if e.HP < before && skill.ID != "" {
		if r.Stats.SkillHits == nil {
			r.Stats.SkillHits = map[string]int{}
		}
		r.Stats.SkillHits[skill.ID]++
	}
}
