package bot

import "ts3news/internal/content"

// combatManaRegen is applied once per acting turn, never once per hit.
func combatManaRegen(user *UserInCombat) int {
	regen := 10 + user.Stats.MNA/20
	if abyssCombatant(user) {
		regen += int(user.classTalents["regen"])
	}
	return max(0, regen)
}

// combatSkillManaCost keeps action previews and resolved casts identical.
func combatSkillManaCost(user *activeUser, base, insight int) int {
	if base <= 0 {
		base = 20
	}
	if chest, ok := user.u.Equipped[content.SlotChest]; ok && chest.ID == "ABYSS_ARCHMAGE_ROBES" {
		base -= 5
	}
	base -= abyssTalentEffectiveInt(insight) * 2
	if reduction := user.treeBonus.Pct["skill_mana_cost"]; reduction > 0 {
		base = int(float64(base) * (1 - reduction))
	}
	return max(5, base)
}
