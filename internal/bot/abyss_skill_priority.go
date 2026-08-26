package bot

import (
	"encoding/json"
	"errors"

	"ts3news/internal/content"
)

const maxAbyssSkillPriority = 16

type abyssSkillPriorityItemView struct {
	ID              string
	Name            string
	Icon            string
	Type            string
	ManaCost        int
	CooldownRounds  int
	Color           string
	Priority        int
	DefaultPriority int
}

type abyssSkillPriorityView struct {
	Items      []abyssSkillPriorityItemView
	Customized bool
}

func abyssSkillPriorityIDs(skills []content.Skill) []string {
	ids := make([]string, 0, len(skills))
	for _, skill := range skills {
		ids = append(ids, skill.ID)
	}
	return ids
}

func applyAbyssSkillPriority(skills []content.Skill, priority []string) []content.Skill {
	byID := make(map[string]content.Skill, len(skills))
	for _, skill := range skills {
		byID[skill.ID] = skill
	}

	ordered := make([]content.Skill, 0, len(skills))
	seen := make(map[string]bool, len(skills))
	for _, id := range priority {
		skill, ok := byID[id]
		if !ok || seen[id] {
			continue
		}
		ordered = append(ordered, skill)
		seen[id] = true
	}
	for _, skill := range skills {
		if !seen[skill.ID] {
			ordered = append(ordered, skill)
		}
	}
	return ordered
}

func validateAbyssSkillPriority(priority []string, skills []content.Skill) ([]string, error) {
	if len(priority) > maxAbyssSkillPriority {
		return nil, errors.New("skill priority contains too many entries")
	}
	known := make(map[string]bool, len(skills))
	for _, skill := range skills {
		known[skill.ID] = true
	}
	seen := make(map[string]bool, len(priority))
	for _, id := range priority {
		if !known[id] {
			return nil, errors.New("skill priority contains an unknown skill")
		}
		if seen[id] {
			return nil, errors.New("skill priority contains a duplicate skill")
		}
		seen[id] = true
	}
	if len(priority) > len(skills) {
		return nil, errors.New("skill priority contains too many entries")
	}
	return abyssSkillPriorityIDs(applyAbyssSkillPriority(skills, priority)), nil
}

func (b *Bot) abyssPrioritizedSkills(uid string, skills []content.Skill) []content.Skill {
	var priority []string
	if err := json.Unmarshal([]byte(b.abyssCombatOption(uid, "skill_priority")), &priority); err != nil {
		priority = nil
	}
	return applyAbyssSkillPriority(skills, priority)
}

func (b *Bot) abyssSkillPriorityView(uid string) abyssSkillPriorityView {
	baseSkills := b.getSkills(uid)
	orderedSkills := b.abyssPrioritizedSkills(uid, baseSkills)
	return abyssSkillPriorityViewForSkills(baseSkills, orderedSkills)
}

func abyssSkillPriorityViewForSkills(baseSkills, orderedSkills []content.Skill) abyssSkillPriorityView {
	defaultRank := make(map[string]int, len(baseSkills))
	for i, skill := range baseSkills {
		defaultRank[skill.ID] = i + 1
	}

	view := abyssSkillPriorityView{
		Items: make([]abyssSkillPriorityItemView, 0, len(orderedSkills)),
	}
	for i, skill := range orderedSkills {
		view.Items = append(view.Items, abyssSkillPriorityItemView{
			ID: skill.ID, Name: skill.Name, Icon: abyssSkillPriorityIcon(skill),
			Type:     string(skill.Type),
			ManaCost: skill.ManaCost, CooldownRounds: skill.CooldownRounds,
			Color: skill.Rarity.Color(), Priority: i + 1, DefaultPriority: defaultRank[skill.ID],
		})
		if i+1 != defaultRank[skill.ID] {
			view.Customized = true
		}
	}
	return view
}

func abyssSkillPriorityIcon(skill content.Skill) string {
	switch skill.Type {
	case content.SkillPhysical:
		return "⚔"
	case content.SkillBuff:
		return "⬆"
	case content.SkillDebuff:
		return "⬇"
	default:
		return "✦"
	}
}

func firstReadyAffordableSkill(
	skills []content.Skill,
	cooldowns map[string]int,
	currentMana int,
	spellCost func(int) int,
) *content.Skill {
	for i := range skills {
		if cooldowns[skills[i].ID] == 0 && currentMana >= spellCost(skills[i].ManaCost) {
			return &skills[i]
		}
	}
	return nil
}
