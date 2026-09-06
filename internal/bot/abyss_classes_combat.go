package bot

import (
	"fmt"
	"strings"
	"ts3news/internal/content"
)

// abyssSkillBase is shared by combat and build previews. Only Abyss uses the
// declared stat; channel combat retains its existing scaling.
func abyssSkillBase(u *UserInCombat, s content.Skill) int {
	value := u.Stats.STR
	switch s.ScalingStat {
	case "INT":
		value = u.Stats.INT
	case "DEF":
		value = u.Stats.DEF
	case "HP":
		value = u.Stats.HP
	}
	mod := 1.0
	if s.ScalingStat == "STR" || s.ScalingStat == "" {
		mod = u.STRMod
	}
	if mod <= 0 {
		mod = 1
	}
	return max(1, int(float64(value)*mod))
}

func abyssClassSkillRole(s content.Skill) string {
	if s.Source != "class_signature" {
		return ""
	}
	if strings.HasSuffix(s.ID, "_build") {
		return "builder"
	}
	return "finisher"
}

// previewAbyssClassSkill never mutates encounter state. Target-specific bonuses
// are applied only to the marked enemy, with the same function at resolution.
func previewAbyssClassSkill(au *activeUser, s content.Skill, target *content.Mob) content.Skill {
	s = previewAbyssTalentSkill(au, s, target)
	if au == nil || au.u == nil || abyssClassSkillRole(s) != "finisher" {
		return s
	}
	charges := min(3, max(0, au.classResource))
	if charges == 0 {
		return s
	}
	s.Power *= 1 + float64(charges)*.20
	marked := target != nil && target == au.classMarkedTarget
	switch au.u.AbyssSubclass {
	case "vanguard":
		s.IgnoreDef = min(1, s.IgnoreDef+.30)
	case "berserker":
		if target != nil && target.MaxHP > 0 && target.Stats.HP <= target.MaxHP/2 {
			s.Power *= 1.25
		}
	case "marksman":
		if marked {
			s.IgnoreDef = min(1, s.IgnoreDef+.60)
		}
	case "beastmaster":
		living := 0
		for _, pet := range au.u.Pets {
			if pet != nil && pet.Stats.HP > 0 {
				living++
			}
		}
		s.Power *= 1 + float64(min(3, living))*.10
	case "elementalist":
		if marked {
			s.Power *= 1.20
		}
	case "oracle":
		s.Power *= 1.15
	case "geomancer":
		s.IgnoreDef = min(1, s.IgnoreDef+.45)
	case "bloodblade":
		s.HealPercent += .04 * float64(charges)
	case "voidwalker":
		if au.u.CurrentHP > max(1, au.u.Stats.HP/20) {
			s.Power *= 1.25
		}
		if marked {
			s.IgnoreDef = min(1, s.IgnoreDef+.25)
		}
	case "runesmith":
		if _, ok := au.u.Equipped[content.SlotRelic]; ok {
			s.Power *= 1.15
		}
	case "alchemist":
		if marked {
			s.IgnoreDef = min(1, s.IgnoreDef+.35)
		}
		s.HealPercent += .03 * float64(charges)
	}
	return s
}

func abyssClassBaseShield(au *activeUser, s content.Skill) int {
	if au == nil || au.u == nil {
		return 0
	}
	if s.ID == "S_AS" || strings.Contains(strings.ToLower(s.Name), "shield") && s.Source != "class_signature" {
		return max(1, au.u.Stats.INT/2)
	}
	if abyssClassSkillRole(s) == "builder" && (au.u.AbyssSubclass == "vanguard" || au.u.AbyssSubclass == "geomancer") {
		return max(1, au.u.Stats.DEF)
	}
	if abyssClassSkillRole(s) == "finisher" && au.u.AbyssSubclass == "runesmith" && au.classResource > 0 {
		return max(1, au.u.Stats.DEF/2)
	}
	return 0
}

func grantAbyssClassShield(au *activeUser, amount int) int {
	before := au.shield
	// Never shrink an existing opening Aegis. Repeated casts cannot bank infinite shields.
	au.shield = max(before, min(max(1, au.u.Stats.HP/2), before+max(0, amount)))
	au.maxShield = max(au.maxShield, au.shield)
	return au.shield - before
}

func resolveAbyssClassCast(au *activeUser, s content.Skill, target *content.Mob, round int, logs *[]string) content.Skill {
	adjusted := previewAbyssClassSkill(au, s, target)
	if shield := abyssClassShield(au, s); shield > 0 {
		gained := grantAbyssClassShield(au, shield)
		*logs = append(*logs, fmt.Sprintf("%s grants %s a %d HP barrier.", s.Name, au.u.Nickname, gained))
		au.u.live.present(round, "skill", "ally:"+au.u.UID, s.ID, s.Name, s.Element, abyssLivePresentationOutcome{TargetID: "ally:" + au.u.UID, Status: "barrier"})
	}
	sub, ok := abyssUserStyle(au.u)
	if !ok || abyssClassSkillRole(s) == "" {
		return adjusted
	}
	if abyssClassSkillRole(s) == "builder" {
		au.classResource = min(3, au.classResource+1)
		au.classMarkedTarget = target
		*logs = append(*logs, fmt.Sprintf("%s: %s %d/3. %s prepares %s.", sub.Name, sub.Resource, au.classResource, s.Name, sub.Finisher))
	} else {
		spent := au.classResource
		if spent > 0 {
			switch sub.ID {
			case "chronomancer":
				for id, cooldown := range au.skillCooldowns {
					if id != s.ID {
						au.skillCooldowns[id] = max(0, cooldown-1)
					}
				}
				for _, ult := range au.u.Ultimates {
					if ult != nil {
						ult.CurrentCooldown = max(0, ult.CurrentCooldown-1)
					}
				}
				*logs = append(*logs, "Temporal Release recovers 1 round on other skills and ultimates.")
			case "voidwalker":
				cost := min(max(0, au.u.CurrentHP-1), max(0, au.u.Stats.HP/20))
				au.u.CurrentHP -= cost
				au.u.DamageTaken += cost
				au.u.live.present(round, "skill", "ally:"+au.u.UID, s.ID, s.Name, s.Element, abyssLivePresentationOutcome{TargetID: "ally:" + au.u.UID, Damage: cost, Status: "health cost"})
				*logs = append(*logs, fmt.Sprintf("Oblivion Burst spends %d HP; it cannot defeat its caster.", cost))
			}
			*logs = append(*logs, fmt.Sprintf("%s spends %d %s on %s.", au.u.Nickname, spent, sub.Resource, s.Name))
		}
		au.classResource = 0
		au.classMarkedTarget = nil
	}
	return adjusted
}

func abyssClassRecommendation(au *activeUser, skill content.Skill) (float64, string) {
	role := abyssClassSkillRole(skill)
	if role == "" {
		return 0, ""
	}
	if role == "finisher" && au.classResource > 0 {
		return 90, "Spend your stored class resource on the prepared finisher."
	}
	if role == "builder" && au.classResource == 0 {
		return 80, "Prepare your subclass payoff with its signature builder."
	}
	return 0, ""
}

func (b *Bot) recordAbyssClassSafeSkillUse(u *UserInCombat, id string) int {
	if u == nil || u.shadow || u.IsClone || b.DB == nil {
		return 0
	}
	return b.recordAbyssSkillUse(u.UID, id)
}
func abyssClassAutoSkill(au *activeUser, cost func(int) int) *content.Skill {
	var best *content.Skill
	bestScore := 0.0
	for i := range au.u.Skills {
		s := &au.u.Skills[i]
		if au.skillCooldowns[s.ID] > 0 || cost(s.ManaCost) > au.CurrentMana {
			continue
		}
		priority, _ := abyssClassRecommendation(au, *s)
		score := s.Power
		if priority > 0 {
			score = priority
		}
		if s.HealPercent > 0 && s.Power == 0 && au.u.CurrentHP < au.u.Stats.HP/2 {
			score = 100
		}
		if score > bestScore {
			best = s
			bestScore = score
		}
	}
	return best
}

func abyssSkillEquipmentMultiplier(u *UserInCombat, s content.Skill) float64 {
	multiplier := 1.0
	if abyssCombatant(u) && s.ScalingStat != "INT" {
		return multiplier
	}
	for _, slot := range []content.GearSlot{content.SlotOffHand, content.SlotTrinket2} {
		if gear, ok := u.Equipped[slot]; ok {
			lower := strings.ToLower(gear.Name)
			if strings.Contains(lower, "orb") || strings.Contains(lower, "battery") {
				multiplier *= 1.15
			}
		}
	}
	return multiplier
}
