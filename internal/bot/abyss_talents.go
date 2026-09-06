package bot

import (
	"errors"
	"fmt"
	"math"
	"ts3news/internal/content"
)

type abyssClassProgress struct {
	XP           int64    `json:"xp"`
	Clears       int64    `json:"clears"`
	BestDepth    int      `json:"best_depth"`
	Foundation   []string `json:"foundation"`
	LegacyCredit bool     `json:"legacy_credit,omitempty"`
}

func normalizeAbyssClassState(state *abyssClassState) error {
	if state.Profiles == nil {
		state.Profiles = map[string]abyssClassProfile{}
	}
	if state.Progress == nil {
		state.Progress = map[string]abyssClassProgress{}
	}
	if state.Version == 1 {
		ids := map[string]bool{state.Selected: true}
		for id := range state.Profiles {
			ids[id] = true
		}
		for id := range ids {
			if sub, ok := content.AbyssSubclassByID(id); ok {
				p := state.Progress[sub.ClassID]
				p.XP = max(p.XP, 75000)
				p.LegacyCredit = true
				if len(p.Foundation) == 0 {
					for i := 1; i <= 5; i++ {
						p.Foundation = append(p.Foundation, fmt.Sprintf("%s_t%d_1", sub.ClassID, i))
					}
				}
				state.Progress[sub.ClassID] = p
			}
		}
		if sub, ok := content.AbyssSubclassByID(state.Selected); ok {
			state.Class = sub.ClassID
		}
		state.Version = 2
	}
	if state.Version != 2 {
		return errors.New("unsupported class build version")
	}
	if state.Class != "" {
		if _, ok := content.AbyssClassByID(state.Class); !ok {
			return errors.New("unknown saved class")
		}
	}
	for id, p := range state.Progress {
		if _, ok := content.AbyssClassByID(id); !ok || p.XP < 0 || p.XP > math.MaxInt64/2 || p.Clears < 0 {
			return errors.New("invalid saved class progression")
		}
		if err := validateAbyssTalents(id, p.Foundation, min(5, content.AbyssClassPoints(p.XP))); err != nil {
			return err
		}
	}
	for id, p := range state.Profiles {
		if len(p.Talents) > 0 {
			sub, ok := content.AbyssSubclassByID(id)
			if !ok {
				return errors.New("unknown saved subclass tree")
			}
			if err := validateAbyssTalents(id, p.Talents, max(0, content.AbyssClassPoints(state.Progress[sub.ClassID].XP)-5)); err != nil {
				return err
			}
		}
	}
	return nil
}
func validateAbyssTalents(id string, ids []string, points int) error {
	tree := content.AbyssTalents(id)
	if len(tree.Nodes) == 0 {
		return errors.New("unknown talent tree")
	}
	if len(ids) > min(points, tree.Budget) {
		return errors.New("not enough class talent points")
	}
	nodes := map[string]content.AbyssTalent{}
	for _, n := range tree.Nodes {
		nodes[n.ID] = n
	}
	seen := map[string]bool{}
	tiers := [6]int{}
	for _, id := range ids {
		n, ok := nodes[id]
		if !ok || seen[id] {
			return errors.New("unknown or duplicate talent")
		}
		seen[id] = true
		tiers[n.Tier]++
	}
	for tier, count := range tiers {
		limit := 1
		if tree.Subclass != "" && tier < 5 {
			limit = 2
		}
		if count > limit {
			return errors.New("competing talent choices exceed this tier's limit")
		}
		if count > 0 && tier > 0 && tiers[tier-1] == 0 {
			return errors.New("complete the previous tier first")
		}
	}
	if tiers[5] > 0 && len(ids)-tiers[5] < 9 {
		return errors.New("choose nine subclass talents before a capstone")
	}
	return nil
}
func abyssClassID(state abyssClassState) string {
	if sub, ok := content.AbyssSubclassByID(state.Selected); ok {
		return sub.ClassID
	}
	return state.Class
}
func abyssTalentEffects(state abyssClassState) map[string]float64 {
	id := abyssClassID(state)
	effects := map[string]float64{}
	for treeID, ids := range map[string][]string{id: state.Progress[id].Foundation, state.Selected: state.Profiles[state.Selected].Talents} {
		for _, node := range content.AbyssTalents(treeID).Nodes {
			for _, chosen := range ids {
				if chosen == node.ID {
					effects[node.Effect] += node.Amount
				}
			}
		}
	}
	return effects
}
func applyAbyssTalentStats(u *UserInCombat, state abyssClassState) {
	u.classTalents = abyssTalentEffects(state)
	style, _ := content.AbyssCombatStyle(u.AbyssSubclass)
	if style.ID == "" {
		style, _ = content.AbyssFoundationStyle(u.AbyssClass)
	}
	scale := func(v int, p float64) int {
		if p == 0 {
			return v
		}
		return max(1, int(float64(v)*(1+p)))
	}
	switch style.Scaling {
	case "STR":
		u.Stats.STR = scale(u.Stats.STR, u.classTalents["primary"])
	case "INT":
		u.Stats.INT = scale(u.Stats.INT, u.classTalents["primary"])
	case "DEF":
		u.Stats.DEF = scale(u.Stats.DEF, u.classTalents["primary"])
	}
	u.Stats.HP = scale(u.Stats.HP, u.classTalents["health"])
	u.CurrentHP = min(u.CurrentHP, u.Stats.HP)
	u.Stats.DEF = scale(u.Stats.DEF, u.classTalents["defense"])
	u.Stats.MNA = scale(u.Stats.MNA, u.classTalents["mana"])
	for i := range u.Skills {
		s := &u.Skills[i]
		if s.Source != "class_signature" {
			continue
		}
		s.ManaCost = max(5, s.ManaCost-int(u.classTalents["mana_discount"]))
		if abyssClassSkillRole(*s) == "finisher" {
			if u.classTalents["cap_burst"] > 0 {
				s.CooldownRounds++
			}
			if u.classTalents["cap_siphon"] > 0 {
				s.ManaCost += 10
			}
		}
	}
}

func abyssUserStyle(u *UserInCombat) (content.AbyssSubclass, bool) {
	id := u.AbyssSubclass
	if id == "" {
		id = u.AbyssClass
	}
	return content.AbyssCombatStyle(id)
}
func previewAbyssTalentSkill(au *activeUser, s content.Skill, target *content.Mob) content.Skill {
	if au == nil || au.u == nil || s.Source != "class_signature" {
		return s
	}
	e := au.u.classTalents
	role := abyssClassSkillRole(s)
	if role == "builder" {
		s.Power *= 1 + e["builder_power"]
		s.HealPercent += e["builder_heal"]
	} else {
		s.Power *= 1 + e["finisher_power"] + e["cap_burst"]
		s.HealPercent += e["finisher_heal"]
		if e["cap_aegis"] > 0 {
			s.Power *= .85
		}
		if au.classResource > 0 {
			s.Power *= 1 + e["resource_power"]*float64(min(au.classResource, 3))
			s.HealPercent += e["cap_siphon"]
		}
		if target != nil && target == au.classMarkedTarget {
			s.Power *= 1 + e["marked_power"]
		}
	}
	if au.u.CurrentHP < au.u.Stats.HP/2 {
		s.Power *= 1 + e["low_health"]
	}
	s.IgnoreDef = min(1, s.IgnoreDef+e["penetration"])
	return s
}
func abyssClassShield(au *activeUser, s content.Skill) int {
	base := abyssClassBaseShield(au, s)
	if au == nil || au.u == nil || s.Source != "class_signature" {
		return base
	}
	if abyssClassSkillRole(s) == "builder" {
		base += int(float64(au.u.Stats.DEF) * au.u.classTalents["builder_shield"])
	} else if au.classResource > 0 {
		base += int(float64(au.u.Stats.HP) * au.u.classTalents["cap_aegis"])
	}
	return base
}
