package bot

import (
	"fmt"
	"math"
	"strings"

	"ts3news/internal/content"
)

const abyssLowHealthSkillThreshold = 0.35

type abyssSkillModifierContext struct {
	Skill           content.Skill
	TreeBonus       content.TreeBonus
	Element         content.Element
	PreviousElement content.Element
	CurrentHP       int
	MaxHP           int
	PartySize       int
	Round           int
	RepeatCount     int
}

type abyssSkillModifiers struct {
	DamageMultiplier  float64
	HealingMultiplier float64
	IgnoreDefense     float64
	StunChance        float64
	EffectRounds      int
	CooldownRounds    int
	Active            []string
}

func calculateAbyssSkillModifiers(ctx abyssSkillModifierContext) abyssSkillModifiers {
	mod := abyssSkillModifiers{
		DamageMultiplier:  1,
		HealingMultiplier: 1,
		IgnoreDefense:     clampUnit(ctx.Skill.IgnoreDef),
		StunChance:        clampUnit(ctx.Skill.StunChance),
		EffectRounds:      max(0, ctx.Skill.EffectDurationRounds),
		CooldownRounds:    max(0, ctx.Skill.CooldownRounds),
	}
	pct := ctx.TreeBonus.Pct
	applyPower := func(key, label string, damage, healing bool) {
		value := pct[key]
		if value == 0 {
			return
		}
		if damage {
			mod.DamageMultiplier *= 1 + value
		}
		if healing {
			mod.HealingMultiplier *= 1 + value
		}
		mod.Active = append(mod.Active, modifierPercentLabel(label, value))
	}

	applyPower("skill_damage", "skill web", true, false)
	if ctx.Skill.Type == content.SkillPhysical {
		applyPower("physical_skill_power", "physical mastery", true, false)
	} else {
		applyPower("magic_skill_power", "magic mastery", true, false)
	}
	if ctx.Skill.HealPercent > 0 {
		applyPower("healing_skill_power", "healing mastery", false, true)
	}
	if ctx.Element != "" && ctx.Element != content.ElementPhysical {
		applyPower("elemental_skill_power", "elemental mastery", true, false)
	}
	if ctx.MaxHP > 0 && float64(ctx.CurrentHP)/float64(ctx.MaxHP) < abyssLowHealthSkillThreshold {
		applyPower("low_health_skill_power", "desperate focus", true, true)
	}
	if ctx.Round == 1 && ctx.MaxHP > 0 && ctx.CurrentHP >= ctx.MaxHP {
		applyPower("opener_skill_power", "opening focus", true, true)
	}
	if ctx.PreviousElement != "" && ctx.PreviousElement != content.ElementPhysical &&
		ctx.Element != "" && ctx.Element != content.ElementPhysical && ctx.PreviousElement != ctx.Element {
		applyPower("alternating_element_power", "elemental cadence", true, false)
	}
	if ctx.PartySize > 1 && (ctx.Skill.HealPercent > 0 || ctx.Skill.Role == "support") {
		applyPower("support_party_power", "party support", ctx.Skill.Power > 0, true)
	}

	if ctx.RepeatCount > 0 {
		retention := clampUnit(pct["repeated_skill_retention"])
		penalty := 0.10 * float64(min(ctx.RepeatCount, 3)) * (1 - retention)
		mod.DamageMultiplier *= 1 - penalty
		mod.Active = append(mod.Active, fmt.Sprintf("repeat retention -%.0f%%", penalty*100))
	}
	if value := pct["defense_penetration"]; value != 0 && ctx.Skill.Power > 0 {
		mod.IgnoreDefense = clampUnit(mod.IgnoreDefense + value)
		mod.Active = append(mod.Active, modifierPercentLabel("defense penetration", value))
	}
	if value := pct["stun_effectiveness"]; value != 0 && ctx.Skill.StunChance > 0 {
		mod.StunChance = clampUnit(mod.StunChance * (1 + value))
		mod.Active = append(mod.Active, modifierPercentLabel("stun effectiveness", value))
	}
	if value := skillDurationBonus(ctx.Skill, pct); value != 0 && mod.EffectRounds > 0 {
		mod.EffectRounds = max(1, int(math.Ceil(float64(mod.EffectRounds)*(1+value))))
		mod.Active = append(mod.Active, modifierPercentLabel("effect duration", value))
	}
	if value := clampRecovery(pct["skill_cooldown_recovery"]); value != 0 && mod.CooldownRounds > 0 {
		mod.CooldownRounds = max(1, int(math.Ceil(float64(mod.CooldownRounds)*(1-value))))
		mod.Active = append(mod.Active, modifierPercentLabel("cooldown recovery", value))
	}
	return mod
}

func skillDurationBonus(skill content.Skill, pct map[string]float64) float64 {
	if skill.Type == content.SkillDebuff || skill.StunChance > 0 {
		return pct["debuff_duration"]
	}
	if skill.Type == content.SkillBuff || skill.HealPercent > 0 {
		return pct["buff_duration"]
	}
	return 0
}

func abyssTreeActionMultiplier(bonus content.TreeBonus, key string) float64 {
	return max(0, 1+bonus.Pct[key])
}

func abyssModifierSummary(active []string) string {
	if len(active) == 0 {
		return ""
	}
	return "Tree modifiers: " + strings.Join(active, ", ")
}

func modifierPercentLabel(label string, value float64) string {
	return fmt.Sprintf("%s %+.0f%%", label, value*100)
}

func clampRecovery(value float64) float64 {
	if value < 0 {
		return 0
	}
	return min(value, 0.75)
}

func clampUnit(value float64) float64 {
	if value < 0 {
		return 0
	}
	return min(value, 1)
}
