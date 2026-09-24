package content

// SkillIcon returns the shared display symbol for a canonical skill category.
func SkillIcon(kind SkillType) string {
	switch kind {
	case SkillPhysical:
		return "⚔"
	case SkillBuff:
		return "⬆"
	case SkillDebuff:
		return "⬇"
	case SkillUltimate:
		return "★"
	default:
		return "✦"
	}
}
