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
		r.Resource = min(3, r.Resource+1)
		return 0, ""
	}
	if skill.Role != "finisher" {
		return 0, ""
	}
	charges, marked := r.Resource, r.Marked
	r.Resource = 0
	r.Marked = ""
	if charges == 0 {
		return 0, marked
	}
	p := &r.Player
	switch r.Build.Class {
	case "chronomancer":
		for id, remaining := range r.SkillTimers {
			if id != skill.ID {
				r.SkillTimers[id] = math.Max(0, remaining-1.5)
			}
		}
	case "bloodblade":
		p.HP = math.Min(p.MaxHP, p.HP+p.MaxHP*.04*float64(charges))
	case "alchemist":
		p.HP = math.Min(p.MaxHP, p.HP+p.MaxHP*.03*float64(charges))
	case "voidwalker":
		cost := math.Min(math.Max(0, p.HP-1), p.MaxHP*.05)
		p.HP -= cost
		r.event("void_cost", p.X, p.Y-35, cost)
	case "runesmith":
		r.Barrier = math.Min(p.MaxHP*.5, r.Barrier+15+r.Build.Armor*2)
	}
	return charges, marked
}

func (r *Run) skillHit(index int, damage float64, skill Skill, charges int, marked string) {
	e := &r.Enemies[index]
	if skill.Role == "builder" {
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
	if skill.Kind == "ice" && e.Kind != "boss" {
		e.Windup = 0
		e.Cooldown = math.Max(e.Cooldown, 1)
	}
	effect := skill.Kind
	if effect == "" {
		effect = "hit"
	}
	r.hurtEnemyPiercing(index, damage, effect, pierce)
}
