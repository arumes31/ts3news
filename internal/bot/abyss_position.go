package bot

import (
	"strings"

	"ts3news/internal/content"
)

const abyssRunFlagPosition = "combat_position"

var abyssCombatPositions = map[string]int64{
	"frontline": 1,
	"backline":  2,
}

func normalizeAbyssCombatPosition(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if _, ok := abyssCombatPositions[value]; ok {
		return value
	}
	return "frontline"
}

func applyAbyssCombatPosition(u *UserInCombat, flags map[string]int64) {
	if u == nil {
		return
	}
	if flags[abyssRunFlagPosition] == abyssCombatPositions["backline"] {
		u.Position = content.PositionBackline
		return
	}
	u.Position = content.PositionFrontline
}
