package bot

import (
	"fmt"
	"strings"
	"time"

	"ts3news/internal/content"
)

func (c *abyssLiveCombat) optionsFor(
	au *activeUser,
	users []activeUser,
	mobs []*content.Mob,
) []abyssLiveOption {
	attackMin, attackMax := estimateLiveDamageRange(au.u, 1, 0, mobs)
	options := []abyssLiveOption{
		{
			Kind: "attack", Name: "Basic Attack", Target: "enemy", Power: 1,
			EffectLabel: "DMG", MinEffect: attackMin, MaxEffect: attackMax,
		},
		{
			Kind: "defend", Name: "Defend", Target: "self",
			EffectLabel: "DEF", MinEffect: 50, MaxEffect: 50,
		},
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
		effectLabel := "DMG"
		minEffect, maxEffect := estimateLiveDamageRange(au.u, skill.Power, skill.IgnoreDef, mobs)
		if skill.HealPercent > 0 && skill.Power == 0 {
			target = "ally"
			effectLabel = "HEAL"
			minEffect, maxEffect = estimateLiveSkillHeal(skill.HealPercent, users)
		}
		options = append(options, abyssLiveOption{
			Kind:        "skill",
			ID:          skill.ID,
			Name:        skill.Name,
			Description: skill.Description,
			Target:      target,
			Mana:        spellCost,
			Power:       skill.Power + skill.HealPercent*2,
			EffectLabel: effectLabel,
			MinEffect:   minEffect,
			MaxEffect:   maxEffect,
			Tags:        abyssSkillComboTags(skill),
		})
	}
	for _, ultimate := range au.u.Ultimates {
		if ultimate == nil {
			continue
		}
		minEffect, maxEffect := estimateLiveDamageRange(au.u, ultimate.Power, 0, mobs)
		options = append(options, abyssLiveOption{
			Kind:        "ultimate",
			ID:          ultimate.ID,
			Name:        ultimate.Name,
			Description: ultimate.Description,
			Target:      "enemy",
			Cooldown:    ultimate.CurrentCooldown,
			Power:       ultimate.Power,
			EffectLabel: "DMG",
			MinEffect:   minEffect,
			MaxEffect:   maxEffect,
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
		effectLabel := "BUFF"
		minEffect, maxEffect := estimateLiveBuff(item.EffectValue)
		if item.Type == content.ConsumableHealing {
			target = "ally"
			effectLabel = "HEAL"
			minEffect, maxEffect = estimateLiveItemHeal(item.EffectValue, users)
		}
		options = append(options, abyssLiveOption{
			Kind:        "item",
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Target:      target,
			Count:       item.Duration,
			Cooldown:    au.potionCooldown,
			Power:       item.EffectValue,
			EffectLabel: effectLabel,
			MinEffect:   minEffect,
			MaxEffect:   maxEffect,
		})
	}
	if relic, ok := au.u.Equipped[content.SlotRelic]; ok && au.relicCharges > 0 {
		options = append(options, abyssLiveOption{
			Kind: "relic", ID: relic.ID, Name: relic.Name, Target: "self",
			Description: "Once per run: restore 20% max HP and gain Guard.",
			Count:       au.relicCharges, EffectLabel: "HEAL+DEF",
			MinEffect: au.u.Stats.HP / 5, MaxEffect: au.u.Stats.HP / 5,
			Tags: []string{"relic", "emergency", "guard"},
		})
	}
	if len(au.u.Pets) > 0 && au.u.Pets[0] != nil {
		pet := au.u.Pets[0]
		options = append(options, abyssLiveOption{
			Kind: "companion", ID: pet.Name, Name: "Command " + pet.Name,
			Description: "Spend your action to order this companion to focus the selected enemy.",
			Target:      "enemy", Power: float64(pet.Stats.STR), EffectLabel: "FOCUS",
			Tags: []string{"companion", "focus", "combo"},
		})
	}
	return options
}

func (c *abyssLiveCombat) bestActionLocked(uid string) abyssLiveAction {
	action, _ := c.bestActionWithReasonLocked(uid)
	return action
}

func (c *abyssLiveCombat) bestActionWithReasonLocked(uid string) (abyssLiveAction, string) {
	action := abyssLiveAction{Kind: "defend", Round: c.round, Automatic: true}
	reason := "No valid target remains, so defend to reduce incoming damage."
	if len(c.enemies) > 0 {
		target := c.selectEnemyLocked(uid, "attack")
		action = abyssLiveAction{
			Kind:      "attack",
			TargetID:  target.ID,
			Round:     c.round,
			Automatic: true,
		}
		reason = fmt.Sprintf("Attack %s because it matches your attack target priority.", target.Name)
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
	if tactic == "balanced" && c.social.partyTactic != "" {
		tactic = c.social.partyTactic
	}
	policy := normalizeLivePolicy(c.policies[uid])
	if ratio <= 0.30 && policy.CriticalTactic != "same" {
		tactic = policy.CriticalTactic
	}
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
				}, fmt.Sprintf("Heal %s because they are below the %.0f%% safety threshold.", ally.Name, healThreshold*100)
			}
		}
	}
	if ratio <= itemThreshold {
		for _, option := range c.options[uid] {
			if option.Kind == "item" && option.Target == "ally" && option.Count > 0 && option.Cooldown == 0 {
				return abyssLiveAction{
					Kind: option.Kind, AbilityID: option.ID, TargetID: ally.ID,
					Round: c.round, Automatic: true,
				}, fmt.Sprintf("Use an emergency item on %s because they are critically low.", ally.Name)
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
		action.TargetID = c.selectEnemyLocked(uid, best.Kind).ID
		reason = fmt.Sprintf("Use %s, the strongest ready affordable action, on the skill-priority target.", best.Name)
	}
	return action, reason
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
	if len(action.IdempotencyKey) > abyssLiveMaxIdempotencyKeyLength {
		return fmt.Errorf("invalid idempotency key")
	}
	key := uid + ":" + action.IdempotencyKey
	if action.IdempotencyKey != "" {
		if accepted, ok := c.idempotency[key]; ok {
			if accepted.Action.sameIntent(action) {
				return nil
			}
			return errAbyssLiveIdempotencyConflict
		}
	}
	if c.phase != "planning" || action.Round != c.round || time.Now().After(c.deadline) {
		return errAbyssLiveStale
	}
	if action.IdempotencyKey != "" && c.idempotencyCountLocked(uid) >= abyssLiveMaxIdempotencyKeysPerRound {
		return errAbyssLiveIdempotencyLimit
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
	delete(c.ready, uid)
	if action.IdempotencyKey != "" {
		c.idempotency[key] = abyssLiveIdempotency{Round: c.round, Action: action}
	}
	c.version++
	return nil
}

func (c *abyssLiveCombat) setReady(uid, sessionID string, round int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.participants[uid] {
		return errAbyssLiveNotFound
	}
	if sessionID != c.id || c.phase != "planning" || round != c.round || time.Now().After(c.deadline) {
		return errAbyssLiveStale
	}
	if _, ok := c.queued[uid]; !ok {
		return fmt.Errorf("queue an action before marking ready")
	}
	if c.ready[uid] {
		return nil
	}
	c.ready[uid] = true
	c.version++
	return nil
}

func (c *abyssLiveCombat) releaseReadyRound() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.phase == "planning" && c.allReadyLocked() {
		select {
		case c.readySignal <- struct{}{}:
		default:
		}
	}
}

func (c *abyssLiveCombat) spendTimeBank(uid, sessionID string, round int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.participants[uid] {
		return errAbyssLiveNotFound
	}
	if sessionID != c.id || c.phase != "planning" || round != c.round || time.Now().After(c.deadline) {
		return errAbyssLiveStale
	}
	spend := min(c.timeBank[uid], abyssLiveTimeBankSpend)
	if spend <= 0 {
		return fmt.Errorf("decision-time bank is empty")
	}
	c.timeBank[uid] -= spend
	c.deadline = c.deadline.Add(spend)
	c.version++
	select {
	case c.deadlineSignal <- struct{}{}:
	default:
	}
	return nil
}

func (c *abyssLiveCombat) setPauseMode(uid, mode string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if uid != c.ownerUID {
		return fmt.Errorf("only the party owner can change pause triggers")
	}
	c.pauseMode = normalizeAbyssPauseMode(mode)
	c.version++
	return nil
}

func (c *abyssLiveCombat) setPolicy(uid string, policy abyssLivePolicy) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.participants[uid] {
		return errAbyssLiveNotFound
	}
	if c.policies == nil {
		c.policies = make(map[string]abyssLivePolicy)
	}
	c.policies[uid] = normalizeLivePolicy(policy)
	c.version++
	return nil
}

func (c *abyssLiveCombat) allReadyLocked() bool {
	aliveParticipants := 0
	for _, ally := range c.allies {
		if ally.HP <= 0 {
			continue
		}
		uid := strings.TrimPrefix(ally.ID, "ally:")
		if !c.participants[uid] {
			continue
		}
		aliveParticipants++
		if !c.ready[uid] {
			return false
		}
	}
	return aliveParticipants > 0
}

func (a abyssLiveAction) sameIntent(other abyssLiveAction) bool {
	return a.SessionID == other.SessionID &&
		a.Kind == other.Kind &&
		a.AbilityID == other.AbilityID &&
		a.TargetID == other.TargetID &&
		a.Round == other.Round
}

func (c *abyssLiveCombat) idempotencyCountLocked(uid string) int {
	count := 0
	for key, accepted := range c.idempotency {
		if accepted.Round == c.round && strings.HasPrefix(key, uid+":") {
			count++
		}
	}
	return count
}

func (c *abyssLiveCombat) pruneIdempotencyLocked() {
	for key, accepted := range c.idempotency {
		if accepted.Round != c.round {
			delete(c.idempotency, key)
		}
	}
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
