package bot

import (
	"fmt"
	"strings"
	"time"

	"ts3news/internal/content"
)

func (c *abyssLiveCombat) optionsFor(au *activeUser) []abyssLiveOption {
	options := []abyssLiveOption{
		{Kind: "attack", Name: "Basic Attack", Target: "enemy", Power: 1},
		{Kind: "defend", Name: "Defend", Target: "self"},
	}
	spellCost := 20
	if chest, ok := au.u.Equipped[content.SlotChest]; ok && chest.ID == "ABYSS_ARCHMAGE_ROBES" {
		spellCost -= 5
	}
	if spellCost < 5 {
		spellCost = 5
	}
	for i := range au.u.Skills {
		skill := au.u.Skills[i]
		target := "enemy"
		if skill.HealPercent > 0 && skill.Power == 0 {
			target = "ally"
		}
		options = append(options, abyssLiveOption{
			Kind:        "skill",
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			Target:      target,
			Mana:        spellCost,
			Power:       skill.Power + skill.HealPercent*2,
		})
	}
	for _, ultimate := range au.u.Ultimates {
		if ultimate == nil {
			continue
		}
		options = append(options, abyssLiveOption{
			Kind:        "ultimate",
			ID:          ultimate.ID,
			Name:        ultimate.Name,
			Description: ultimate.Description,
			Target:      "enemy",
			Cooldown:    ultimate.CurrentCooldown,
			Power:       ultimate.Power,
		})
	}
	for _, item := range c.server.bot.getConsumables(au.u.UID) {
		if item.Type != content.ConsumableHealing && item.Type != content.ConsumableBuff {
			continue
		}
		if item.Duration <= 0 {
			continue
		}
		target := "self"
		if item.Type == content.ConsumableHealing {
			target = "ally"
		}
		options = append(options, abyssLiveOption{
			Kind:        "item",
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Target:      target,
			Count:       item.Duration,
			Power:       item.EffectValue,
		})
	}
	return options
}

func (c *abyssLiveCombat) bestActionLocked(uid string) abyssLiveAction {
	action := abyssLiveAction{Kind: "defend", Round: c.round, Automatic: true}
	if len(c.enemies) > 0 {
		target := c.enemies[0]
		for _, enemy := range c.enemies[1:] {
			if enemy.HP < target.HP {
				target = enemy
			}
		}
		action = abyssLiveAction{
			Kind:      "attack",
			TargetID:  target.ID,
			Round:     c.round,
			Automatic: true,
		}
	}

	ally := abyssLiveCombatantView{}
	for _, candidate := range c.allies {
		if candidate.HP <= 0 {
			continue
		}
		if ally.ID == "" || candidate.HP*ally.MaxHP < ally.HP*candidate.MaxHP {
			ally = candidate
		}
	}
	ratio := 1.0
	if ally.ID != "" && ally.MaxHP > 0 {
		ratio = float64(ally.HP) / float64(ally.MaxHP)
	}
	tactic := c.tactics[uid]
	healThreshold := 0.40
	itemThreshold := 0.28
	switch tactic {
	case "aggressive":
		healThreshold, itemThreshold = 0.22, 0.14
	case "defensive":
		healThreshold, itemThreshold = 0.68, 0.45
	case "conserve_items":
		healThreshold, itemThreshold = 0.42, 0.12
	}

	if ratio <= healThreshold {
		for _, option := range c.options[uid] {
			if option.Kind == "skill" && option.Target == "ally" {
				return abyssLiveAction{
					Kind: option.Kind, AbilityID: option.ID, TargetID: ally.ID,
					Round: c.round, Automatic: true,
				}
			}
		}
	}
	if ratio <= itemThreshold {
		for _, option := range c.options[uid] {
			if option.Kind == "item" && option.Target == "ally" && option.Count > 0 {
				return abyssLiveAction{
					Kind: option.Kind, AbilityID: option.ID, TargetID: ally.ID,
					Round: c.round, Automatic: true,
				}
			}
		}
	}

	best := abyssLiveOption{}
	currentMana := 0
	for _, candidate := range c.allies {
		if candidate.ID == "ally:"+uid {
			currentMana = candidate.Mana
			break
		}
	}
	for _, option := range c.options[uid] {
		if option.Target != "enemy" || option.Cooldown > 0 {
			continue
		}
		if option.Mana > currentMana {
			continue
		}
		if option.Kind != "skill" && option.Kind != "ultimate" {
			continue
		}
		if option.Power > best.Power {
			best = option
		}
	}
	if best.ID != "" && action.TargetID != "" {
		action.Kind = best.Kind
		action.AbilityID = best.ID
	}
	return action
}

func (c *abyssLiveCombat) submit(uid string, action abyssLiveAction) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.participants[uid] {
		return errAbyssLiveNotFound
	}
	if action.SessionID != c.id {
		return errAbyssLiveStale
	}
	if action.IdempotencyKey != "" && c.idempotency[uid+":"+action.IdempotencyKey] {
		return nil
	}
	if c.phase != "planning" || action.Round != c.round || time.Now().After(c.deadline) {
		return errAbyssLiveStale
	}
	validOption := false
	for _, option := range c.options[uid] {
		if option.Kind == action.Kind && option.ID == action.AbilityID {
			validOption = option.Cooldown == 0
			if option.Kind == "item" && option.Count <= 0 {
				validOption = false
			}
			if option.Mana > 0 {
				for _, ally := range c.allies {
					if ally.ID == "ally:"+uid && ally.Mana < option.Mana {
						validOption = false
					}
				}
			}
			break
		}
	}
	if !validOption {
		return fmt.Errorf("invalid or unavailable action")
	}
	if !c.validTargetLocked(uid, action) {
		return fmt.Errorf("invalid target")
	}
	action.Automatic = false
	c.queued[uid] = action
	if action.IdempotencyKey != "" {
		c.idempotency[uid+":"+action.IdempotencyKey] = true
	}
	c.version++
	return nil
}

func (c *abyssLiveCombat) validTargetLocked(uid string, action abyssLiveAction) bool {
	if action.Kind == "defend" {
		return action.TargetID == "" || action.TargetID == "ally:"+uid
	}
	var targetType string
	for _, option := range c.options[uid] {
		if option.Kind == action.Kind && option.ID == action.AbilityID {
			targetType = option.Target
			break
		}
	}
	switch targetType {
	case "self":
		return action.TargetID == "" || action.TargetID == "ally:"+uid
	case "ally":
		for _, ally := range c.allies {
			if ally.ID == action.TargetID && ally.HP > 0 {
				return true
			}
		}
	case "enemy":
		for _, enemy := range c.enemies {
			if enemy.ID == action.TargetID && enemy.HP > 0 {
				return true
			}
		}
	}
	return false
}

func liveMobFromTarget(targetID string, mobs []*content.Mob) *content.Mob {
	var index int
	if _, err := fmt.Sscanf(targetID, "enemy:%d", &index); err != nil {
		return nil
	}
	if index < 0 || index >= len(mobs) || mobs[index] == nil || mobs[index].Stats.HP <= 0 {
		return nil
	}
	return mobs[index]
}

func liveAllyFromTarget(targetID string, users []activeUser) *UserInCombat {
	uid := strings.TrimPrefix(targetID, "ally:")
	for i := range users {
		if users[i].u != nil && users[i].u.UID == uid && users[i].u.CurrentHP > 0 {
			return users[i].u
		}
	}
	return nil
}

func findLiveSkill(u *UserInCombat, id string) *content.Skill {
	for i := range u.Skills {
		if u.Skills[i].ID == id {
			return &u.Skills[i]
		}
	}
	return nil
}

func findLiveUltimate(u *UserInCombat, id string) *content.UltimateSkill {
	for _, ultimate := range u.Ultimates {
		if ultimate != nil && ultimate.ID == id {
			return ultimate
		}
	}
	return nil
}

func (b *Bot) useLiveConsumable(
	actor *UserInCombat,
	target *UserInCombat,
	consumableID string,
	logs *[]string,
) bool {
	consumable, ok := content.GetConsumableByID(consumableID)
	if !ok || (consumable.Type != content.ConsumableHealing && consumable.Type != content.ConsumableBuff) {
		return false
	}
	result, err := b.DB.Exec(
		`UPDATE user_consumables
		    SET remaining_fights=remaining_fights-1
		  WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights>0`,
		actor.UID,
		consumableID,
	)
	if err != nil {
		return false
	}
	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return false
	}
	_, _ = b.DB.Exec(
		"DELETE FROM user_consumables WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights<=0",
		actor.UID,
		consumableID,
	)
	b.abyssSpendLoadout(actor.UID, consumableID)
	_, _ = b.DB.Exec("UPDATE abyss_active SET momentum=0 WHERE client_uid=$1", actor.UID)

	switch consumable.Type {
	case content.ConsumableHealing:
		if target == nil {
			target = actor
		}
		heal := int(consumable.EffectValue)
		if consumable.EffectValue <= 1 {
			heal = int(float64(target.Stats.HP) * consumable.EffectValue)
		}
		if heal < 1 {
			heal = 1
		}
		target.CurrentHP = min(target.Stats.HP, target.CurrentHP+heal)
		*logs = append(*logs, fmt.Sprintf(
			"🧪 %s uses %s on %s, restoring %d HP.",
			actor.Nickname,
			consumable.Name,
			target.Nickname,
			heal,
		))
	case content.ConsumableBuff:
		amount := int(consumable.EffectValue)
		switch consumableID {
		case "iron_skin_brew":
			actor.Stats.DEF += amount
		case "speed_elixir":
			actor.Stats.SPD += amount
		case "intellect_elixir":
			actor.Stats.INT += amount
		default:
			actor.Stats.STR += amount
		}
		*logs = append(*logs, fmt.Sprintf("🧪 %s uses %s and surges with power.", actor.Nickname, consumable.Name))
	}
	return true
}
