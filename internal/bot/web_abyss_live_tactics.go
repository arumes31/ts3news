package bot

import (
	"fmt"
	"strings"
	"time"

	"ts3news/internal/content"
)

const abyssLiveEncounterDuration = "encounter"

type abyssLiveEnemyPlan struct {
	Round      int
	Intent     abyssLiveEnemyIntent
	TargetUID  string
	SpellIndex int
}

func livePauseEnabled(mode, trigger string) bool {
	switch normalizeAbyssPauseMode(mode) {
	case "bosses":
		return trigger == "boss" || trigger == "phase"
	case "danger":
		return trigger == "critical" || trigger == "phase"
	case "fast":
		return false
	default:
		return true
	}
}

func normalizeLivePolicy(policy abyssLivePolicy) abyssLivePolicy {
	if policy.CriticalTactic == "" || policy.CriticalTactic == "same" {
		policy.CriticalTactic = "same"
	} else {
		policy.CriticalTactic = normalizeAbyssTactic(policy.CriticalTactic)
	}
	switch policy.AttackPriority {
	case "highest_hp", "weakness":
	default:
		policy.AttackPriority = "lowest_hp"
	}
	switch policy.SkillPriority {
	case "highest_hp", "weakness":
	default:
		policy.SkillPriority = "lowest_hp"
	}
	switch policy.TimeoutAction {
	case "attack", "defend":
	default:
		policy.TimeoutAction = "best"
	}
	return policy
}

func (c *abyssLiveCombat) selectEnemyLocked(uid, actionKind string) abyssLiveCombatantView {
	if len(c.enemies) == 0 {
		return abyssLiveCombatantView{}
	}
	policy := normalizeLivePolicy(c.policies[uid])
	priority := policy.AttackPriority
	if actionKind == "skill" || actionKind == "ultimate" {
		priority = policy.SkillPriority
	}
	target := c.enemies[0]
	for _, enemy := range c.enemies[1:] {
		switch priority {
		case "highest_hp":
			if enemy.HP > target.HP {
				target = enemy
			}
		case "weakness":
			selfElement := ""
			for _, ally := range c.allies {
				if ally.ID == "ally:"+uid {
					selfElement = ally.Element
					break
				}
			}
			if enemy.WeakTo == selfElement && target.WeakTo != selfElement {
				target = enemy
			}
		default:
			if enemy.HP < target.HP {
				target = enemy
			}
		}
	}
	return target
}

func (c *abyssLiveCombat) currentPauseMode() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return normalizeAbyssPauseMode(c.pauseMode)
}

func liveEnemyIndex(targetID string) int {
	index := -1
	if _, err := fmt.Sscanf(targetID, "enemy:%d", &index); err != nil {
		return -1
	}
	return index
}

func planLiveEnemyIntents(
	round int,
	users []activeUser,
	mobs []*content.Mob,
	logs []string,
) map[int]abyssLiveEnemyPlan {
	return planLiveEnemyIntentsWithRandom(
		round,
		users,
		mobs,
		logs,
		defaultCombatRandomSource{},
	)
}

func planLiveEnemyIntentsWithRandom(
	round int,
	users []activeUser,
	mobs []*content.Mob,
	logs []string,
	source combatRandomSource,
) map[int]abyssLiveEnemyPlan {
	plans := make(map[int]abyssLiveEnemyPlan, len(mobs))
	potentialTargets := livePotentialTargets(users)
	for i, mob := range mobs {
		if mob == nil || mob.Stats.HP <= 0 {
			continue
		}
		plan := abyssLiveEnemyPlan{
			Round:      round,
			SpellIndex: -1,
			Intent: abyssLiveEnemyIntent{
				EnemyID:   fmt.Sprintf("enemy:%d", i),
				EnemyName: mob.Name,
				Kind:      "attack",
				Ability:   "Basic attack",
			},
		}
		if mob.Stats.SPD == 0 {
			if liveMobChanneling(mob.Name, logs) {
				plan.Intent.Kind = "interruptible"
				plan.Intent.Ability = "Channeling · interrupt with Ultimate"
			} else {
				plan.Intent.Kind = "stunned"
				plan.Intent.Ability = "Skips this turn"
			}
			plans[i] = plan
			continue
		}
		if len(potentialTargets) > 0 {
			// #nosec G404 -- non-cryptographic combat plan selection
			target := potentialTargets[source.IntN(len(potentialTargets))]
			plan.TargetUID = target.u.UID
			plan.Intent.TargetID = "ally:" + target.u.UID
			plan.Intent.Target = target.u.Nickname
		}
		if len(mob.Spells) > 0 && source.Float64() < 0.2 {
			// #nosec G404 -- non-cryptographic combat plan selection
			plan.SpellIndex = source.IntN(len(mob.Spells))
			plan.Intent.Kind = "cast"
			plan.Intent.Ability = mob.Spells[plan.SpellIndex].Name
		}
		if mob.Type == content.MobTreasureGoblin && round >= 3 && source.Float64() < 0.3 {
			plan.SpellIndex = -1
			plan.Intent.Kind = "flee"
			plan.Intent.Ability = "Escape attempt"
			plan.Intent.TargetID = ""
			plan.Intent.Target = ""
		}
		plans[i] = plan
	}
	return plans
}

func liveMobChanneling(name string, logs []string) bool {
	for i := len(logs) - 1; i >= 0 && i >= len(logs)-6; i-- {
		if strings.Contains(logs[i], name) && strings.Contains(logs[i], "summoning ritual") {
			return true
		}
	}
	return false
}

func livePotentialTargets(users []activeUser) []activeUser {
	potentialTargets := make([]activeUser, 0, len(users))
	for _, au := range users {
		if au.u != nil && au.u.CurrentHP > 0 && au.u.Position == content.PositionFrontline {
			potentialTargets = append(potentialTargets, au)
		}
	}
	if len(potentialTargets) > 0 {
		return potentialTargets
	}
	for _, au := range users {
		if au.u != nil && au.u.CurrentHP > 0 {
			potentialTargets = append(potentialTargets, au)
		}
	}
	return potentialTargets
}

func lowestHealthMobExcept(mobs []*content.Mob, excluded *content.Mob) *content.Mob {
	var target *content.Mob
	for _, mob := range mobs {
		if mob == nil || mob == excluded || mob.Stats.HP <= 0 {
			continue
		}
		if target == nil || mob.Stats.HP < target.Stats.HP {
			target = mob
		}
	}
	return target
}

func liveSkillComboFollowup(previousTarget string, action abyssLiveAction) bool {
	return action.Kind == "skill" && strings.HasPrefix(action.TargetID, "enemy:") &&
		previousTarget != "" && previousTarget == action.TargetID
}

func liveInitiative(
	users []activeUser,
	mobs []*content.Mob,
	playerStarts bool,
) []abyssLiveInitiativeEntry {
	allies := make([]abyssLiveInitiativeEntry, 0, len(users))
	for _, au := range users {
		if au.u == nil || au.u.CurrentHP <= 0 {
			continue
		}
		allies = append(allies, abyssLiveInitiativeEntry{
			ID: "ally:" + au.u.UID, Name: au.u.Nickname, Side: "ally", Speed: au.u.Stats.SPD,
		})
	}
	if _, support := abyssRescueSupportForUsers(users); support != nil {
		allies = append(allies, abyssLiveInitiativeEntry{
			ID: "support:explorer", Name: support.Name, Side: "ally", Speed: support.Speed,
		})
	}
	enemies := make([]abyssLiveInitiativeEntry, 0, len(mobs))
	for i, mob := range mobs {
		if mob == nil || mob.Stats.HP <= 0 {
			continue
		}
		enemies = append(enemies, abyssLiveInitiativeEntry{
			ID: fmt.Sprintf("enemy:%d", i), Name: mob.Name, Side: "enemy", Speed: mob.Stats.SPD,
		})
	}
	if playerStarts {
		return append(allies, enemies...)
	}
	return append(enemies, allies...)
}

func abyssLiveCombatFor(users []activeUser) *abyssLiveCombat {
	for i := range users {
		if users[i].u != nil && users[i].u.live != nil {
			return users[i].u.live
		}
	}
	return nil
}

func (c *abyssLiveCombat) enemyPlansForRound(round int) map[int]abyssLiveEnemyPlan {
	c.mu.Lock()
	defer c.mu.Unlock()
	plans := make(map[int]abyssLiveEnemyPlan, len(c.enemyPlans))
	for index, plan := range c.enemyPlans {
		if plan.Round == round {
			plans[index] = plan
		}
	}
	return plans
}

func (c *abyssLiveCombat) waitForPlanningDeadline() {
	for {
		c.mu.Lock()
		if c.phase != "planning" {
			c.mu.Unlock()
			return
		}
		wait := time.Until(c.deadline)
		readySignal := c.readySignal
		deadlineSignal := c.deadlineSignal
		c.mu.Unlock()
		if wait <= 0 {
			return
		}
		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
			return
		case <-readySignal:
			stopLiveTimer(timer)
			return
		case <-deadlineSignal:
			stopLiveTimer(timer)
		}
	}
}

func stopLiveTimer(timer *time.Timer) {
	if timer.Stop() {
		return
	}
	select {
	case <-timer.C:
	default:
	}
}

func liveUserElement(u *UserInCombat) content.Element {
	if u != nil {
		if weapon, ok := u.Equipped[content.SlotMainHand]; ok && weapon.Element != "" {
			return weapon.Element
		}
	}
	return content.ElementPhysical
}

func liveThreat(u *UserInCombat) int {
	if u == nil || u.CurrentHP <= 0 {
		return 0
	}
	if u.Position == content.PositionFrontline {
		return 100
	}
	return 25
}

func liveRoundRecap(
	round int,
	previousAllies, allies, previousEnemies, enemies []abyssLiveCombatantView,
) string {
	if round <= 1 || len(previousAllies) == 0 {
		return ""
	}
	partyLoss := liveHPDelta(previousAllies, allies)
	enemyLoss := liveHPDelta(previousEnemies, enemies)
	defeated := max(0, liveAliveCount(previousEnemies)-liveAliveCount(enemies))
	return fmt.Sprintf(
		"Round %d recap: party lost %d HP; enemies lost %d HP; %d enemies defeated.",
		round-1, partyLoss, enemyLoss, defeated,
	)
}

func liveHPDelta(previous, current []abyssLiveCombatantView) int {
	currentHP := make(map[string]int, len(current))
	for _, unit := range current {
		currentHP[unit.ID] = unit.HP
	}
	delta := 0
	for _, unit := range previous {
		delta += max(0, unit.HP-currentHP[unit.ID])
	}
	return delta
}

func liveAliveCount(units []abyssLiveCombatantView) int {
	count := 0
	for _, unit := range units {
		if unit.HP > 0 {
			count++
		}
	}
	return count
}

func liveElementWeakness(element content.Element) content.Element {
	switch element {
	case content.ElementFire:
		return content.ElementWater
	case content.ElementWater:
		return content.ElementEarth
	case content.ElementEarth:
		return content.ElementAir
	case content.ElementAir:
		return content.ElementFire
	default:
		return ""
	}
}

func liveAllyEffects(au *activeUser) []abyssLiveEffect {
	if au == nil {
		return nil
	}
	effects := make([]abyssLiveEffect, 0, len(au.effects)+1)
	seen := make(map[string]bool, len(au.effects)+1)
	for _, effect := range au.effects {
		name := string(effect)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		effects = append(effects, abyssLiveEffect{
			Name:     name,
			Duration: abyssLiveEncounterDuration,
		})
	}
	if au.Stunned && !seen["Stunned"] {
		effects = append(effects, abyssLiveEffect{Name: "Stunned", RemainingRounds: 1})
	}
	if au.defendingRound > 0 && !seen["Guarded"] {
		effects = append(effects, abyssLiveEffect{Name: "Guarded", Duration: "next enemy phase"})
	}
	if au.u != nil && len(au.u.Pets) > 0 {
		effects = append(effects, abyssLiveEffect{
			Name:     "Companion: " + abyssPetCommandLabel(au.petCommand),
			Duration: abyssLiveEncounterDuration,
		})
	}
	return effects
}

func liveMobEffects(mob *content.Mob) []abyssLiveEffect {
	if mob == nil {
		return nil
	}
	effects := make([]abyssLiveEffect, 0, len(mob.Effects)+2)
	seen := make(map[string]bool, len(mob.Effects)+2)
	for _, effect := range mob.Effects {
		name := string(effect)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		effects = append(effects, abyssLiveEffect{
			Name:     name,
			Duration: abyssLiveEncounterDuration,
		})
	}
	if mob.Stats.SPD == 0 && !seen["Stunned"] {
		effects = append(effects, abyssLiveEffect{Name: "Stunned", RemainingRounds: max(1, mob.StunRounds)})
	}
	if mob.WeaknessWindow && !seen["Weakness Window"] {
		effects = append(effects, abyssLiveEffect{
			Name: "Weakness Window", Duration: "Next player hit · guaranteed critical",
		})
	}
	return effects
}

func estimateLiveDamageRange(
	u *UserInCombat,
	power float64,
	ignoreDef float64,
	mobs []*content.Mob,
) (int, int) {
	if u == nil || power <= 0 {
		return 0, 0
	}
	minDamage, maxDamage := 0, 0
	for _, mob := range mobs {
		if mob == nil || mob.Stats.HP <= 0 {
			continue
		}
		low, high := estimateLiveDamageAgainst(u, power, ignoreDef, mob)
		if minDamage == 0 || low < minDamage {
			minDamage = low
		}
		if high > maxDamage {
			maxDamage = high
		}
	}
	if minDamage == 0 {
		return estimateLiveDamageAgainst(u, power, ignoreDef, &content.Mob{})
	}
	return minDamage, maxDamage
}

func estimateLiveDamageAgainst(
	u *UserInCombat,
	power float64,
	ignoreDef float64,
	mob *content.Mob,
) (int, int) {
	strengthMod := u.STRMod
	if strengthMod <= 0 {
		strengthMod = 1
	}
	strength := float64(max(1, u.Stats.STR)) * strengthMod
	damageMult := power * getElementMult(liveUserElement(u), mob.Element)
	if u.Position == content.PositionBackline {
		damageMult *= 1.10
	}
	defenseMod := mob.DEFMod
	if defenseMod <= 0 {
		defenseMod = 1
	}
	defense := float64(mob.Stats.DEF) * defenseMod * (1 - ignoreDef)
	base := strength*damageMult - defense
	damageFloor := strength * 0.15
	if base < damageFloor {
		base = damageFloor
	}
	low := max(1, int(base*0.5))
	high := max(low, int(base*2))
	return low, high
}

func estimateLiveSkillHeal(percent float64, users []activeUser) (int, int) {
	if percent <= 0 {
		return 0, 0
	}
	return estimateLiveHeal(users, func(maxHP int) int {
		return int(float64(maxHP) * percent)
	})
}

func estimateLiveItemHeal(value float64, users []activeUser) (int, int) {
	return estimateLiveHeal(users, func(maxHP int) int {
		if value <= 1 {
			return int(float64(maxHP) * value)
		}
		return int(value)
	})
}

func estimateLiveHeal(users []activeUser, amount func(int) int) (int, int) {
	minHeal, maxHeal := 0, 0
	for i := range users {
		if users[i].u == nil || users[i].u.CurrentHP <= 0 {
			continue
		}
		heal := max(1, amount(max(1, users[i].u.Stats.HP)))
		if minHeal == 0 || heal < minHeal {
			minHeal = heal
		}
		if heal > maxHeal {
			maxHeal = heal
		}
	}
	return minHeal, maxHeal
}

func estimateLiveBuff(value float64) (int, int) {
	amount := int(value)
	if value > 0 && value <= 1 {
		amount = int(value * 100)
	}
	amount = max(1, amount)
	return amount, amount
}
