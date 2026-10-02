package rift

import "strings"

// WeaponFamily categorizes a weapon name and optional class into a semantic family
// used for physical sound synthesis and impact feedback.
// Supported families: "blade", "blunt", "pierce", "arcane", "fist", "ranged".
func WeaponFamily(weapon, class string) string {
	w := strings.ToLower(strings.TrimSpace(weapon))
	switch {
	case strings.Contains(w, "hammer") || strings.Contains(w, "mace") || strings.Contains(w, "club") ||
		strings.Contains(w, "spade") || strings.Contains(w, "maul") || strings.Contains(w, "flail") ||
		strings.Contains(w, "scepter") || strings.Contains(w, "earthshaker"):
		return "blunt"

	case strings.Contains(w, "dagger") || strings.Contains(w, "knife") || strings.Contains(w, "stiletto") ||
		strings.Contains(w, "rapier") || strings.Contains(w, "spear") || strings.Contains(w, "pike") ||
		strings.Contains(w, "hook") || strings.Contains(w, "harvester") || strings.Contains(w, "scythe"):
		return "pierce"

	case strings.Contains(w, "staff") || strings.Contains(w, "wand") || strings.Contains(w, "rod") ||
		strings.Contains(w, "orb") || strings.Contains(w, "relic") || strings.Contains(w, "grimoire") ||
		strings.Contains(w, "tome"):
		return "arcane"

	case strings.Contains(w, "bow") || strings.Contains(w, "crossbow") || strings.Contains(w, "sling") ||
		strings.Contains(w, "gun"):
		return "ranged"

	case strings.Contains(w, "fist") || strings.Contains(w, "knuckle") || strings.Contains(w, "claw") ||
		strings.Contains(w, "gauntlet") || strings.Contains(w, "brawl") || w == "unarmed":
		return "fist"

	case strings.Contains(w, "sword") || strings.Contains(w, "blade") || strings.Contains(w, "claymore") ||
		strings.Contains(w, "cleaver") || strings.Contains(w, "longsword") || strings.Contains(w, "saber") ||
		strings.Contains(w, "katana") || strings.Contains(w, "axe") || strings.Contains(w, "reaver") ||
		strings.Contains(w, "edge") || strings.Contains(w, "firebrand") || strings.Contains(w, "sunder") ||
		strings.Contains(w, "verdict"):
		return "blade"
	}

	// Fallback based on class archetype
	c := strings.ToLower(strings.TrimSpace(class))
	switch c {
	case "geomancer", "runesmith":
		return "blunt"
	case "marksman":
		return "ranged"
	case "elementalist", "chronomancer", "oracle", "alchemist", "voidwalker":
		return "arcane"
	case "beastmaster":
		return "pierce"
	case "vanguard", "berserker", "bloodblade", "warrior", "reaver":
		return "blade"
	default:
		if w == "unarmed" || w == "" {
			return "fist"
		}
		return "blade"
	}
}

// WeaponFamily returns the semantic weapon family for this build.
func (b Build) WeaponFamily() string {
	return WeaponFamily(b.Weapon, b.Class)
}

// WeaponFamily returns the semantic weapon family for the active run's player build.
func (r *Run) WeaponFamily() string {
	return r.Build.WeaponFamily()
}
