package bot

import (
	"fmt"
	"strings"

	"ts3news/internal/content"
)

const (
	abyssSkillVarietyTarget   = 3
	abyssSkillVarietyBonusPct = 5
)

type abyssSkillVarietyView struct {
	Distinct int  `json:"distinct"`
	Target   int  `json:"target"`
	BonusPct int  `json:"bonus_pct"`
	Unlocked bool `json:"unlocked"`
}

func recordAbyssSkillVariety(user *UserInCombat, skill content.Skill, party []activeUser, logs *[]string) {
	if !abyssCombatant(user) || strings.TrimSpace(skill.ID) == "" {
		return
	}
	before := abyssSkillVarietyForActiveUsers(party)
	if user.abyssSkillsUsed == nil {
		user.abyssSkillsUsed = make(map[string]struct{}, abyssSkillVarietyTarget)
	}
	if _, exists := user.abyssSkillsUsed[skill.ID]; exists {
		return
	}
	user.abyssSkillsUsed[skill.ID] = struct{}{}
	view := abyssSkillVarietyForActiveUsers(party)
	if user.live != nil {
		view = user.live.recordSkillVariety(skill.ID)
	}
	if logs == nil || view.Distinct == before.Distinct || view.Distinct > abyssSkillVarietyTarget {
		return
	}
	if view.Unlocked {
		*logs = append(*logs, fmt.Sprintf(
			"🎨 VARIETY COMPLETE · %d/%d distinct skills — +%d%% floor XP secured.",
			view.Distinct,
			view.Target,
			view.BonusPct,
		))
		return
	}
	*logs = append(*logs, fmt.Sprintf(
		"🎨 VARIETY %d/%d · %s registered.",
		view.Distinct,
		view.Target,
		skill.Name,
	))
}

func abyssSkillVarietyForActiveUsers(users []activeUser) abyssSkillVarietyView {
	combatants := make([]UserInCombat, 0, len(users))
	for i := range users {
		if users[i].u != nil {
			combatants = append(combatants, *users[i].u)
		}
	}
	return abyssSkillVarietyForCombatants(combatants)
}

func abyssSkillVarietyForCombatants(users []UserInCombat) abyssSkillVarietyView {
	unique := make(map[string]struct{}, abyssSkillVarietyTarget)
	for i := range users {
		for skillID := range users[i].abyssSkillsUsed {
			unique[skillID] = struct{}{}
		}
	}
	return newAbyssSkillVarietyView(len(unique))
}

func newAbyssSkillVarietyView(distinct int) abyssSkillVarietyView {
	distinct = max(0, distinct)
	return abyssSkillVarietyView{
		Distinct: distinct,
		Target:   abyssSkillVarietyTarget,
		BonusPct: abyssSkillVarietyBonusPct,
		Unlocked: distinct >= abyssSkillVarietyTarget,
	}
}

func applyAbyssSkillVarietyXP(rewardXP int, variety abyssSkillVarietyView) (int, int) {
	if rewardXP <= 0 || !variety.Unlocked {
		return max(0, rewardXP), 0
	}
	bonus := max(1, (rewardXP*abyssSkillVarietyBonusPct+99)/100)
	return rewardXP + bonus, bonus
}

func (c *abyssLiveCombat) recordSkillVariety(skillID string) abyssSkillVarietyView {
	if c == nil || skillID == "" {
		return newAbyssSkillVarietyView(0)
	}
	c.mu.Lock()
	if c.varietySkills == nil {
		c.varietySkills = make(map[string]struct{}, abyssSkillVarietyTarget)
	}
	c.varietySkills[skillID] = struct{}{}
	c.skillVariety = newAbyssSkillVarietyView(len(c.varietySkills))
	view := c.skillVariety
	c.mu.Unlock()
	return view
}
