package bot

import (
	"fmt"
	"strings"

	"ts3news/internal/content"
)

const (
	abyssStormParty   = "party"
	abyssStormEnemies = "enemies"
)

var abyssCombatFloorModifiers = []string{
	"enraged",
	"no_healing",
	"artifact_corrupted",
	"treasure_goblin",
	"echo_encounter",
	"fragile_cache",
	"mirror_clone",
	"storm_floor",
	"darkness",
}

func hasAbyssFloorModifier(modifiers, want string) bool {
	for _, modifier := range strings.Fields(modifiers) {
		if modifier == want {
			return true
		}
	}
	return false
}

func abyssEncounterWarning(modifier string) string {
	switch {
	case hasAbyssFloorModifier(modifier, "mirror_clone"):
		return "MIRROR FLOOR — your current gear and active skills have been reflected into a hostile clone."
	case hasAbyssFloorModifier(modifier, "storm_floor"):
		return "STORM FLOOR — lightning announces which side it will strike one round before impact."
	case hasAbyssFloorModifier(modifier, "darkness"):
		return "DARKNESS FLOOR — hostile health is concealed; judge progress from wounds, effects, and defeats."
	default:
		return ""
	}
}

func abyssMirrorClone(user UserInCombat) content.Mob {
	equipped := make([]content.Gear, 0, len(user.Equipped))
	for _, gear := range user.Equipped {
		equipped = append(equipped, gear)
	}
	spells := append([]content.Skill(nil), user.Skills...)
	hp := max(1, user.Stats.HP)
	return content.Mob{
		Name:      "Mirror of " + user.Nickname,
		Type:      content.MobElite,
		Level:     user.Level,
		Stats:     user.Stats,
		CurrentHP: hp,
		MaxHP:     hp,
		RewardXP:  max(1, user.Level*15),
		Element:   content.ElementPhysical,
		Spells:    spells,
		Equipped:  equipped,
	}
}

func nextAbyssStormSide(random combatRandomSource) string {
	if random.IntN(2) == 0 {
		return abyssStormParty
	}
	return abyssStormEnemies
}

func abyssStormTelegraph(side string) string {
	if side == abyssStormParty {
		return "⚡ STORM WARNING — lightning will strike the PARTY at the start of next round!"
	}
	return "⚡ STORM WARNING — lightning will strike the ENEMIES at the start of next round!"
}

func strikeAbyssStorm(side string, users []activeUser, mobs []*content.Mob) (partyDamage, enemyDamage int) {
	if side == abyssStormParty {
		for i := range users {
			user := users[i].u
			if user == nil || user.CurrentHP <= 0 {
				continue
			}
			damage := min(user.CurrentHP, max(1, user.Stats.HP/20))
			user.CurrentHP -= damage
			user.DamageTaken += damage
			partyDamage += damage
		}
		return partyDamage, 0
	}
	for _, mob := range mobs {
		if mob == nil || mob.Stats.HP <= 0 {
			continue
		}
		damage := min(mob.Stats.HP, max(1, mob.MaxHP/20))
		mob.Stats.HP -= damage
		enemyDamage += damage
	}
	return 0, enemyDamage
}

func abyssStormImpactLog(side string, damage int) string {
	return fmt.Sprintf("⚡ The storm strikes the %s for %d total damage!", strings.ToUpper(side), damage)
}

func concealAbyssEnemyViews(enemies []abyssLiveCombatantView) {
	for i := range enemies {
		enemies[i].HP = -1
		enemies[i].MaxHP = -1
		enemies[i].HPHidden = true
	}
}

func concealAbyssTimeline(timeline []combatTimelineFrame) []combatTimelineFrame {
	concealed := append([]combatTimelineFrame(nil), timeline...)
	for i := range concealed {
		concealed[i].EnemyHP = -1
		concealed[i].EnemyMax = -1
	}
	return concealed
}
