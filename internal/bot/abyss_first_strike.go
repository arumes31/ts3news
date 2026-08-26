package bot

import "fmt"

const abyssFirstStrikeMaxBonusPct = 20

type abyssFirstStrikeBonus struct {
	AttackerSPD int
	TargetSPD   int
	BonusPct    int
}

func calculateAbyssFirstStrike(
	eligible bool,
	attackerSPD int,
	attackerModifier float64,
	targetSPD int,
	targetModifier float64,
) abyssFirstStrikeBonus {
	if !eligible {
		return abyssFirstStrikeBonus{}
	}
	attacker := abyssEffectiveSpeed(attackerSPD, attackerModifier)
	target := abyssEffectiveSpeed(targetSPD, targetModifier)
	if attacker <= target {
		return abyssFirstStrikeBonus{AttackerSPD: attacker, TargetSPD: target}
	}
	bonusPct := max(1, int(int64(attacker-target)*abyssFirstStrikeMaxBonusPct/int64(target)))
	return abyssFirstStrikeBonus{
		AttackerSPD: attacker,
		TargetSPD:   target,
		BonusPct:    min(abyssFirstStrikeMaxBonusPct, bonusPct),
	}
}

func abyssEffectiveSpeed(base int, modifier float64) int {
	if modifier <= 0 {
		modifier = 1
	}
	return max(1, int(float64(max(0, base))*modifier))
}

func applyAbyssFirstStrike(damage int, bonus abyssFirstStrikeBonus) int {
	if damage <= 0 || bonus.BonusPct <= 0 {
		return max(0, damage)
	}
	return damage * (100 + bonus.BonusPct) / 100
}

func abyssFirstStrikeLog(attacker, target string, bonus abyssFirstStrikeBonus) string {
	if bonus.BonusPct <= 0 {
		return ""
	}
	return fmt.Sprintf(
		"⚡ FIRST STRIKE · %s outruns %s (SPD %d vs %d) — opener +%d%%.",
		sanitizeBBCode(attacker),
		sanitizeBBCode(target),
		bonus.AttackerSPD,
		bonus.TargetSPD,
		bonus.BonusPct,
	)
}
