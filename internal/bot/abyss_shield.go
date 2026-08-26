package bot

import "fmt"

const (
	abyssShieldDEFMultiplier = 10
	abyssShieldMaxHPPct      = 15
)

func abyssShieldCapacity(maxHP, defense int) int {
	if maxHP <= 0 || defense <= 0 {
		return 0
	}
	hpCap := maxHP/100*abyssShieldMaxHPPct + maxHP%100*abyssShieldMaxHPPct/100
	if hpCap < 1 {
		hpCap = 1
	}
	if defense > hpCap/abyssShieldDEFMultiplier {
		return hpCap
	}
	return defense * abyssShieldDEFMultiplier
}

func initializeAbyssShield(user *activeUser) string {
	if user == nil || user.u == nil || !abyssCombatant(user.u) {
		return ""
	}
	capacity := abyssShieldCapacity(user.u.Stats.HP, user.u.Stats.DEF)
	user.shield = capacity
	user.maxShield = capacity
	if capacity == 0 {
		return ""
	}
	return fmt.Sprintf(
		"🛡️ AEGIS · %s raises a %d-point barrier from %d DEF (15%% HP cap).",
		user.u.Nickname,
		capacity,
		user.u.Stats.DEF,
	)
}

func absorbAbyssShield(user *activeUser, damage int) (remaining, absorbed int) {
	if damage <= 0 || user == nil || user.u == nil || !abyssCombatant(user.u) || user.shield <= 0 {
		return max(0, damage), 0
	}
	absorbed = min(damage, user.shield)
	user.shield -= absorbed
	return damage - absorbed, absorbed
}

func abyssShieldAbsorbLog(name string, absorbed, remaining int) string {
	state := fmt.Sprintf("%d shield remains", remaining)
	if remaining == 0 {
		state = "barrier broken"
	}
	return fmt.Sprintf("🛡️ AEGIS · %s absorbs %d damage — %s.", name, absorbed, state)
}
