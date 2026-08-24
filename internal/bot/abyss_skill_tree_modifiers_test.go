package bot

import (
	"math"
	"testing"

	"ts3news/internal/content"
)

func TestCalculateAbyssSkillModifiersConditionalPower(t *testing.T) {
	skill := content.Skill{
		ID: "test", Type: content.SkillMagic, Role: "healing", Power: 2,
		HealPercent: 0.25, Element: content.ElementFire,
	}
	mod := calculateAbyssSkillModifiers(abyssSkillModifierContext{
		Skill: skill,
		TreeBonus: content.TreeBonus{Pct: map[string]float64{
			"skill_damage": 0.10, "magic_skill_power": 0.20,
			"elemental_skill_power": 0.10, "alternating_element_power": 0.15,
			"healing_skill_power": 0.25, "support_party_power": 0.20,
		}},
		Element: content.ElementFire, PreviousElement: content.ElementWater,
		CurrentHP: 80, MaxHP: 100, PartySize: 2, Round: 2,
	})
	wantDamage := 1.10 * 1.20 * 1.10 * 1.15 * 1.20
	if math.Abs(mod.DamageMultiplier-wantDamage) > 1e-9 {
		t.Fatalf("damage multiplier = %.6f, want %.6f", mod.DamageMultiplier, wantDamage)
	}
	if wantHeal := 1.25 * 1.20; math.Abs(mod.HealingMultiplier-wantHeal) > 1e-9 {
		t.Fatalf("healing multiplier = %.6f, want %.6f", mod.HealingMultiplier, wantHeal)
	}
	if len(mod.Active) != 6 {
		t.Fatalf("active modifiers = %v, want six", mod.Active)
	}
}

func TestCalculateAbyssSkillModifiersControlAndRecovery(t *testing.T) {
	skill := content.Skill{
		ID: "control", Type: content.SkillDebuff, Power: 1,
		IgnoreDef: 0.25, StunChance: 0.40, EffectDurationRounds: 2,
		CooldownRounds: 4,
	}
	mod := calculateAbyssSkillModifiers(abyssSkillModifierContext{
		Skill: skill,
		TreeBonus: content.TreeBonus{Pct: map[string]float64{
			"defense_penetration": 0.15, "stun_effectiveness": 0.50,
			"debuff_duration": 0.50, "skill_cooldown_recovery": 0.25,
		}},
	})
	if math.Abs(mod.IgnoreDefense-0.40) > 1e-9 || math.Abs(mod.StunChance-0.60) > 1e-9 {
		t.Fatalf("penetration/stun = %.2f/%.2f, want .40/.60", mod.IgnoreDefense, mod.StunChance)
	}
	if mod.EffectRounds != 3 || mod.CooldownRounds != 3 {
		t.Fatalf("duration/cooldown = %d/%d, want 3/3", mod.EffectRounds, mod.CooldownRounds)
	}
}

func TestCalculateAbyssSkillModifiersRepeatRetention(t *testing.T) {
	skill := content.Skill{ID: "repeat", Type: content.SkillPhysical, Power: 1}
	without := calculateAbyssSkillModifiers(abyssSkillModifierContext{Skill: skill, RepeatCount: 3})
	with := calculateAbyssSkillModifiers(abyssSkillModifierContext{
		Skill: skill, RepeatCount: 3,
		TreeBonus: content.TreeBonus{Pct: map[string]float64{"repeated_skill_retention": 0.60}},
	})
	if math.Abs(without.DamageMultiplier-0.70) > 1e-9 || math.Abs(with.DamageMultiplier-0.88) > 1e-9 {
		t.Fatalf("repeat multipliers = %.2f/%.2f, want .70/.88", without.DamageMultiplier, with.DamageMultiplier)
	}
}

func TestAbyssTreeActionMultiplier(t *testing.T) {
	bonus := content.TreeBonus{Pct: map[string]float64{"item_skill_power": 0.20, "relic_skill_power": -2}}
	if got := abyssTreeActionMultiplier(bonus, "item_skill_power"); got != 1.20 {
		t.Fatalf("item multiplier = %.2f, want 1.20", got)
	}
	if got := abyssTreeActionMultiplier(bonus, "relic_skill_power"); got != 0 {
		t.Fatalf("negative action multiplier = %.2f, want 0", got)
	}
}
