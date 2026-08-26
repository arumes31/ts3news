package bot

import (
	"fmt"

	"ts3news/internal/content"
)

type abyssWeaknessCriticalContext struct {
	target *content.Mob
	user   *activeUser
	track  *abyssFightTrack
	logs   *[]string
}

func armAbyssWeaknessWindow(target *content.Mob, abyss bool, logs *[]string) bool {
	if !abyss || target == nil || target.Stats.HP <= 0 || target.WeaknessWindow {
		return false
	}
	target.WeaknessWindow = true
	if logs != nil {
		*logs = append(*logs, fmt.Sprintf(
			"🎯 WEAKNESS WINDOW · %s is exposed — the next direct player hit is a guaranteed critical.",
			target.Name,
		))
	}
	return true
}

func consumeAbyssWeaknessCritical(target *content.Mob, damage int, abyss bool) (int, bool) {
	if !abyss || target == nil || !target.WeaknessWindow || damage <= 0 {
		return damage, false
	}
	target.WeaknessWindow = false
	maxInt := int(^uint(0) >> 1)
	if damage > maxInt/2 {
		return maxInt, true
	}
	return damage * 2, true
}

func abyssWeaknessCriticalLog(attacker, target string) string {
	return fmt.Sprintf(
		"💥 WEAKNESS CRITICAL! %s exploits %s's opening for 2× damage.",
		attacker,
		target,
	)
}

func resolveAbyssWeaknessCritical(
	ctx abyssWeaknessCriticalContext,
	damage int,
) (int, bool) {
	if ctx.user == nil || ctx.user.u == nil {
		return damage, false
	}
	resolved, critical := consumeAbyssWeaknessCritical(
		ctx.target,
		damage,
		abyssCombatant(ctx.user.u),
	)
	if !critical {
		return damage, false
	}
	if ctx.track != nil {
		ctx.track.weaknessCrits++
	}
	if ctx.logs != nil {
		*ctx.logs = append(*ctx.logs, abyssWeaknessCriticalLog(ctx.user.u.Nickname, ctx.target.Name))
	}
	return resolved, true
}
