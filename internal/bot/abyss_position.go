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

func abyssCombatTargetIndices(activeUsers []activeUser) []int {
	alive := make([]int, 0, len(activeUsers))
	frontline := make([]int, 0, len(activeUsers))
	for i := range activeUsers {
		u := activeUsers[i].u
		if u == nil || u.CurrentHP <= 0 {
			continue
		}
		alive = append(alive, i)
		if u.Position == content.PositionFrontline {
			frontline = append(frontline, i)
		}
	}
	if len(frontline) > 0 {
		return frontline
	}
	return alive
}
