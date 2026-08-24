package content

import (
	"fmt"
	"sort"
	"strings"
)

type SkillTargetMode string

const (
	SkillTargetEnemy    SkillTargetMode = "enemy"
	SkillTargetAlly     SkillTargetMode = "ally"
	SkillTargetSelf     SkillTargetMode = "self"
	SkillTargetAllEnemy SkillTargetMode = "all_enemies"
	SkillTargetAllAlly  SkillTargetMode = "all_allies"
)

type SkillDetail struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Type                 SkillType       `json:"type"`
	Rarity               Rarity          `json:"rarity"`
	TargetMode           SkillTargetMode `json:"target_mode"`
	ManaCost             int             `json:"mana_cost"`
	CooldownRounds       int             `json:"cooldown_rounds"`
	Element              Element         `json:"element"`
	Tags                 []string        `json:"tags"`
	EffectDurationRounds int             `json:"effect_duration_rounds"`
	StackLimit           int             `json:"stack_limit"`
	ScalingStat          string          `json:"scaling_stat"`
	PreviewMin           float64         `json:"preview_min"`
	PreviewMax           float64         `json:"preview_max"`
	UpgradeRank          int             `json:"upgrade_rank"`
	Archetype            string          `json:"archetype"`
	Role                 string          `json:"role"`
	Source               string          `json:"source"`
	Description          string          `json:"description"`
	Mechanics            string          `json:"mechanics"`
}

func finalizeSkillCatalog(skills []Skill) {
	for i := range skills {
		enrichSkillMetadata(&skills[i])
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].ID < skills[j].ID })
}

func enrichSkillMetadata(skill *Skill) {
	skill.ManaCost = 20
	skill.CooldownRounds = 0
	skill.EffectDurationRounds = 0
	skill.StackLimit = 1
	skill.UpgradeRank = int(skill.Rarity) + 1
	skill.Element = skillElementFromName(skill.Name)
	skill.TargetMode = SkillTargetEnemy
	skill.ScalingStat = "STR"
	skill.Archetype = "martial"
	skill.Role = "damage"
	if skill.Type != SkillPhysical {
		skill.ScalingStat = "INT"
		skill.Archetype = "arcane"
	}
	if skill.HealPercent > 0 && skill.Power == 0 {
		skill.TargetMode = SkillTargetAlly
		skill.Archetype = "restoration"
		skill.Role = "healing"
		skill.CooldownRounds = 2
	}
	if skill.StunChance > 0 || skill.Type == SkillDebuff {
		skill.EffectDurationRounds = 1
		skill.CooldownRounds = 2
		skill.Archetype = "control"
		skill.Role = "control"
	}
	if skill.Type == SkillBuff {
		skill.EffectDurationRounds = 2
		skill.CooldownRounds = max(skill.CooldownRounds, 2)
		skill.Archetype = "support"
		if skill.Power == 0 {
			skill.Role = "support"
		}
	}
	skill.PreviewMin = skill.Power * 0.9
	skill.PreviewMax = skill.Power * 1.1
	if skill.HealPercent > 0 && skill.Power == 0 {
		skill.PreviewMin = skill.HealPercent
		skill.PreviewMax = skill.HealPercent
	}
	skill.Source = "procedural_catalog"
	if strings.HasPrefix(skill.ID, "S0_") {
		skill.Source = "starter"
	} else if skill.ID == "S_EQ" || skill.ID == "S_AS" {
		skill.Source = "signature_catalog"
	}
	skill.Tags = []string{"spender", strings.ToLower(string(skill.Type)), strings.ToLower(skill.Role), strings.ToLower(string(skill.Element))}
	if skill.HealPercent > 0 {
		skill.Tags = append(skill.Tags, "heal")
	}
	if skill.StunChance > 0 {
		skill.Tags = append(skill.Tags, "stun")
	}
	if skill.IgnoreDef > 0 {
		skill.Tags = append(skill.Tags, "defense-penetration")
	}
	if skill.Special != EffectNone {
		skill.Tags = append(skill.Tags, strings.ToLower(string(skill.Special)))
	}
	skill.Tags = uniqueSortedStrings(skill.Tags)
	skill.Mechanics = skillMechanicalDescription(*skill)
	if !strings.Contains(skill.Description, skill.Mechanics) {
		skill.Description = strings.TrimSpace(skill.Description + " " + skill.Mechanics)
	}
}

func skillElementFromName(name string) Element {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "fire"), strings.Contains(lower, "flame"), strings.Contains(lower, "ember"), strings.Contains(lower, "inferno"):
		return ElementFire
	case strings.Contains(lower, "water"), strings.Contains(lower, "ice"), strings.Contains(lower, "icy"), strings.Contains(lower, "frost"), strings.Contains(lower, "tidal"):
		return ElementWater
	case strings.Contains(lower, "earth"), strings.Contains(lower, "stone"), strings.Contains(lower, "quake"):
		return ElementEarth
	case strings.Contains(lower, "air"), strings.Contains(lower, "wind"), strings.Contains(lower, "storm"), strings.Contains(lower, "lightning"):
		return ElementAir
	default:
		return ElementPhysical
	}
}

func skillMechanicalDescription(skill Skill) string {
	parts := []string{fmt.Sprintf("Targets %s; costs %d mana", strings.ReplaceAll(string(skill.TargetMode), "_", " "), skill.ManaCost)}
	if skill.CooldownRounds > 0 {
		parts = append(parts, fmt.Sprintf("%d-round cooldown", skill.CooldownRounds))
	}
	if skill.Power > 0 {
		parts = append(parts, fmt.Sprintf("%.2fx power", skill.Power))
	}
	if skill.HealPercent > 0 {
		parts = append(parts, fmt.Sprintf("heals %.0f%% max HP", skill.HealPercent*100))
	}
	if skill.IgnoreDef > 0 {
		parts = append(parts, fmt.Sprintf("ignores %.0f%% defense", skill.IgnoreDef*100))
	}
	if skill.StunChance > 0 {
		parts = append(parts, fmt.Sprintf("%.0f%% stun chance", skill.StunChance*100))
	}
	return strings.Join(parts, "; ") + "."
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func ValidateSkillCatalog() error {
	initSkills()
	return validateSkillCatalog(allSkills)
}

func validateSkillCatalog(skills []Skill) error {
	seen := make(map[string]bool, len(skills))
	validTargets := map[SkillTargetMode]bool{SkillTargetEnemy: true, SkillTargetAlly: true, SkillTargetSelf: true, SkillTargetAllEnemy: true, SkillTargetAllAlly: true}
	validElements := map[Element]bool{ElementPhysical: true, ElementFire: true, ElementWater: true, ElementEarth: true, ElementAir: true}
	for _, skill := range skills {
		if skill.ID == "" || skill.Name == "" || seen[skill.ID] {
			return fmt.Errorf("skill has empty or duplicate identity %q", skill.ID)
		}
		seen[skill.ID] = true
		if !validTargets[skill.TargetMode] || !validElements[skill.Element] {
			return fmt.Errorf("skill %q has invalid target or element", skill.ID)
		}
		if skill.ManaCost < 0 || skill.CooldownRounds < 0 || skill.EffectDurationRounds < 0 || skill.StackLimit < 1 {
			return fmt.Errorf("skill %q has invalid cost, cooldown, duration, or stack limit", skill.ID)
		}
		if skill.IgnoreDef < 0 || skill.IgnoreDef > 1 || skill.StunChance < 0 || skill.StunChance > 1 || skill.HealPercent < 0 || skill.HealPercent > 1 {
			return fmt.Errorf("skill %q has an invalid probability", skill.ID)
		}
		if skill.PreviewMin < 0 || skill.PreviewMax < skill.PreviewMin || skill.UpgradeRank < 1 || skill.UpgradeRank > 9 {
			return fmt.Errorf("skill %q has invalid preview or rank metadata", skill.ID)
		}
		if skill.HealPercent > 0 && skill.Power == 0 && skill.TargetMode != SkillTargetAlly && skill.TargetMode != SkillTargetSelf && skill.TargetMode != SkillTargetAllAlly {
			return fmt.Errorf("healing skill %q has incompatible target %q", skill.ID, skill.TargetMode)
		}
		if skill.Description == "" || skill.Mechanics == "" || !strings.Contains(skill.Description, skill.Mechanics) {
			return fmt.Errorf("skill %q description does not match mechanical metadata", skill.ID)
		}
		if skill.Archetype == "" || skill.Role == "" || skill.Source == "" || skill.ScalingStat == "" || len(skill.Tags) == 0 {
			return fmt.Errorf("skill %q has incomplete classification metadata", skill.ID)
		}
	}
	return nil
}

func SkillDetailsByID(ids []string) []SkillDetail {
	initSkills()
	details := make([]SkillDetail, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		skill, ok := GetSkillByID(id)
		if !ok {
			continue
		}
		details = append(details, SkillDetail{
			ID: skill.ID, Name: skill.Name, Type: skill.Type, Rarity: skill.Rarity,
			TargetMode: skill.TargetMode, ManaCost: skill.ManaCost, CooldownRounds: skill.CooldownRounds,
			Element: skill.Element, Tags: append([]string(nil), skill.Tags...), EffectDurationRounds: skill.EffectDurationRounds,
			StackLimit: skill.StackLimit, ScalingStat: skill.ScalingStat, PreviewMin: skill.PreviewMin,
			PreviewMax: skill.PreviewMax, UpgradeRank: skill.UpgradeRank, Archetype: skill.Archetype,
			Role: skill.Role, Source: skill.Source, Description: skill.Description, Mechanics: skill.Mechanics,
		})
	}
	sort.Slice(details, func(i, j int) bool { return details[i].ID < details[j].ID })
	return details
}
