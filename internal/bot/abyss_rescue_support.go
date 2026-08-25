package bot

import (
	"fmt"

	"ts3news/internal/content"
)

const (
	abyssRunFlagExplorerGuardFloors = "explorer_guard_floors"
	abyssRunFlagExplorerSupportID   = "explorer_support_id"
	abyssExplorerSupportFloors      = 3
)

var abyssExplorerNames = [...]string{"Mara Flint", "Orin Vale", "Tessa Quill", "Bram Hollow"}

type abyssRescueSupport struct {
	Name      string
	Remaining int
	Power     int
	Speed     int
}

type abyssRescueSupportView struct {
	Active    bool   `json:"active"`
	Name      string `json:"name,omitempty"`
	Remaining int    `json:"remaining,omitempty"`
}

func abyssExplorerName(id int) string {
	if id < 0 || id >= len(abyssExplorerNames) {
		return "Unknown Delver"
	}
	return abyssExplorerNames[id]
}

func abyssRescueSupportFromFlags(flags map[string]int64, strength, speed int) *abyssRescueSupport {
	remaining := int(flags[abyssRunFlagExplorerGuardFloors])
	id := int(flags[abyssRunFlagExplorerSupportID]) - 1
	if remaining <= 0 || id < 0 || id >= len(abyssExplorerNames) {
		return nil
	}
	return &abyssRescueSupport{
		Name:      abyssExplorerName(id),
		Remaining: min(remaining, abyssExplorerSupportFloors),
		Power:     max(1, strength/4),
		Speed:     max(1, speed-1),
	}
}

func abyssRescueSupportViewFromFlags(flags map[string]int64) abyssRescueSupportView {
	support := abyssRescueSupportFromFlags(flags, 1, 1)
	if support == nil {
		return abyssRescueSupportView{}
	}
	return abyssRescueSupportView{Active: true, Name: support.Name, Remaining: support.Remaining}
}

func tickAbyssRescueSupport(flags map[string]int64) bool {
	if flags[abyssRunFlagExplorerGuardFloors] <= 0 {
		if flags[abyssRunFlagExplorerSupportID] == 0 {
			return false
		}
		delete(flags, abyssRunFlagExplorerSupportID)
		return true
	}
	flags[abyssRunFlagExplorerGuardFloors]--
	if flags[abyssRunFlagExplorerGuardFloors] == 0 {
		delete(flags, abyssRunFlagExplorerSupportID)
	}
	return true
}

func (b *Bot) abyssRescueSupportView(uid string) abyssRescueSupportView {
	return abyssRescueSupportViewFromFlags(b.loadRunFlags(uid))
}

func abyssRescueSupportForUsers(users []activeUser) (*UserInCombat, *abyssRescueSupport) {
	for i := range users {
		if users[i].u != nil && users[i].u.CurrentHP > 0 && users[i].u.abyssSupport != nil {
			return users[i].u, users[i].u.abyssSupport
		}
	}
	return nil, nil
}

func abyssRescueSupportDamage(power, defense int, intensify float64) int {
	if intensify <= 0 {
		intensify = 1
	}
	damage := int(float64(max(1, power-defense/4)) * intensify)
	return max(1, damage)
}

func (b *Bot) applyAbyssRescueSupportTurn(
	activeUsers []activeUser,
	mobs *[]*content.Mob,
	zone content.Zone,
	intensify float64,
	logs *[]string,
	totalUserDamage *int,
	avgLvl int,
	diffFactor float64,
	originalUsers []UserInCombat,
	loots *[]LootResult,
	rand combatRandomSource,
) {
	owner, support := abyssRescueSupportForUsers(activeUsers)
	if owner == nil || support == nil {
		return
	}
	target := lowestHealthMobExcept(*mobs, nil)
	if target == nil {
		return
	}
	damage := abyssRescueSupportDamage(support.Power, target.Stats.DEF, intensify)
	damage = abyssKillerDamage(damage, owner, target)
	target.Stats.HP -= damage
	applyAbyssBreakDamage(target, damage, logs)
	*totalUserDamage += damage
	*logs = append(*logs, fmt.Sprintf(
		"🤝 %s fights beside %s, striking %s for %d damage.",
		support.Name, owner.Nickname, target.Name, damage,
	))
	if target.Stats.HP > 0 {
		return
	}
	*logs = append(*logs, fmt.Sprintf("☠️ %s was defeated by rescued delver %s.", target.Name, support.Name))
	if winner := randomLootEligibleUser(originalUsers, rand); winner != nil {
		b.awardCombatLoot(winner, *target, zone, logs, loots)
	}
	b.handleDeathEffects(target, mobs, logs, avgLvl, diffFactor, activeUsers, rand)
}
