package bot

import (
	"fmt"
	"strings"

	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

type abyssPetCommand string

const (
	abyssPetCommandFocus abyssPetCommand = "focus"
	abyssPetCommandGuard abyssPetCommand = "guard"
	abyssPetCommandFree  abyssPetCommand = "free"

	abyssPetGuardPercent = 15
)

func parseAbyssPetCommand(value string) (abyssPetCommand, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(abyssPetCommandFocus):
		return abyssPetCommandFocus, true
	case string(abyssPetCommandGuard):
		return abyssPetCommandGuard, true
	case string(abyssPetCommandFree):
		return abyssPetCommandFree, true
	default:
		return abyssPetCommandFree, false
	}
}

func normalizeAbyssPetCommand(value string) abyssPetCommand {
	command, _ := parseAbyssPetCommand(value)
	return command
}

func abyssPetCommandLabel(command abyssPetCommand) string {
	switch normalizeAbyssPetCommand(string(command)) {
	case abyssPetCommandFocus:
		return "Focus Target"
	case abyssPetCommandGuard:
		return "Guard Me"
	default:
		return "Free-for-All"
	}
}

func (b *Bot) loadAbyssPetCommand(uid string) abyssPetCommand {
	return normalizeAbyssPetCommand(b.abyssCombatOption(uid, "pet_command"))
}

func recordAbyssPetFocus(au *activeUser, target *content.Mob) {
	if au == nil || target == nil || normalizeAbyssPetCommand(string(au.petCommand)) != abyssPetCommandFocus {
		return
	}
	if au.petFocus != target.Name {
		au.petFocus = target.Name
		au.petFocusLogged = false
	}
}

func abyssPetRetaliationTarget(alive []*content.Mob, attackers map[*content.Mob]bool) *content.Mob {
	for _, mob := range alive {
		if mob != nil && attackers[mob] {
			return mob
		}
	}
	return nil
}

func selectAbyssPetTarget(
	alive []*content.Mob,
	au *activeUser,
	random combatRandomSource,
) *content.Mob {
	if len(alive) == 0 || au == nil {
		return nil
	}
	switch normalizeAbyssPetCommand(string(au.petCommand)) {
	case abyssPetCommandFocus:
		if target := petFocusTarget(alive, au.petFocus); target != nil {
			return target
		}
	case abyssPetCommandGuard:
		if target := abyssPetRetaliationTarget(alive, au.lastAttackers); target != nil {
			return target
		}
	}
	return alive[random.IntN(len(alive))]
}

func abyssGuardingPet(au *activeUser) *content.Mob {
	if au == nil || au.u == nil || !abyssCombatant(au.u) || normalizeAbyssPetCommand(string(au.petCommand)) != abyssPetCommandGuard {
		return nil
	}
	for _, pet := range au.u.Pets {
		if pet != nil && pet.Stats.HP > 0 {
			return pet
		}
	}
	return nil
}

func mitigateAbyssPetGuard(au *activeUser, damage int) (remaining, guarded int) {
	if damage <= 0 || abyssGuardingPet(au) == nil {
		return damage, 0
	}
	guarded = damage/100*abyssPetGuardPercent + damage%100*abyssPetGuardPercent/100
	return damage - guarded, guarded
}

func applyAbyssLivePetCommand(
	au *activeUser,
	action abyssLiveAction,
	mobs []*content.Mob,
	logs *[]string,
) bool {
	if abyssLivingPet(au) == nil {
		return false
	}
	command, known := parseAbyssPetCommand(action.AbilityID)
	if !known {
		// Rolling clients used the pet name as the focus command ID.
		command = abyssPetCommandFocus
	}
	switch command {
	case abyssPetCommandFocus:
		target := liveMobFromTarget(action.TargetID, mobs)
		if target == nil {
			return false
		}
		au.petCommand = command
		au.petFocus = target.Name
		au.petFocusLogged = false
		*logs = append(*logs, fmt.Sprintf("🐾 %s orders FOCUS TARGET: %s. Companions act now.", au.u.Nickname, target.Name))
	case abyssPetCommandGuard:
		au.petCommand = command
		*logs = append(*logs, fmt.Sprintf("🐾 %s orders GUARD ME: companions intercept %d%% of direct-hit damage.", au.u.Nickname, abyssPetGuardPercent))
	case abyssPetCommandFree:
		au.petCommand = command
		au.petFocus = ""
		au.petFocusLogged = false
		*logs = append(*logs, fmt.Sprintf("🐾 %s orders FREE-FOR-ALL: companions choose independent targets.", au.u.Nickname))
	}
	return true
}

func abyssLivingPet(au *activeUser) *content.Mob {
	if au == nil || au.u == nil {
		return nil
	}
	for _, pet := range au.u.Pets {
		if pet != nil && pet.Stats.HP > 0 {
			return pet
		}
	}
	return nil
}

func abyssPetCommandOpeningLog(u *UserInCombat, command abyssPetCommand) string {
	if u == nil || len(u.Pets) == 0 {
		return ""
	}
	switch normalizeAbyssPetCommand(string(command)) {
	case abyssPetCommandFocus:
		return fmt.Sprintf("🐾 Companion order — FOCUS TARGET: %s's pets follow each direct target.", u.Nickname)
	case abyssPetCommandGuard:
		return fmt.Sprintf("🐾 Companion order — GUARD ME: %d%% direct-hit interception is ready for %s.", abyssPetGuardPercent, u.Nickname)
	default:
		return fmt.Sprintf("🐾 Companion order — FREE-FOR-ALL: %s's pets choose independently.", u.Nickname)
	}
}

type abyssPetTurnContext struct {
	activeUsers     []activeUser
	mobs            *[]*content.Mob
	zone            content.Zone
	intensify       float64
	logs            *[]string
	totalUserDamage *int
	totalMobDamage  *int
	avgLevel        int
	difficulty      float64
	originalUsers   []UserInCombat
	loots           *[]LootResult
	track           *abyssFightTrack
	random          combatRandomSource
}

func (b *Bot) runAbyssPetTurns(au *activeUser, ctx abyssPetTurnContext) {
	if au == nil || au.u == nil || ctx.random == nil || ctx.logs == nil ||
		ctx.mobs == nil || ctx.totalUserDamage == nil || ctx.totalMobDamage == nil {
		return
	}
	if au.petNervousLogged == nil {
		au.petNervousLogged = make(map[*content.Mob]bool)
	}
	if au.petCooldowns == nil {
		au.petCooldowns = make(map[int]int)
	}
	u := au.u
	for petIndex, pet := range u.Pets {
		if pet == nil || pet.Stats.HP <= 0 {
			continue
		}
		if abyssCombatant(u) && abyssPetNervous(pet.Loyalty) && !au.petNervousLogged[pet] {
			au.petNervousLogged[pet] = true
			*ctx.logs = append(*ctx.logs, fmt.Sprintf("🐾 %s hangs back, eyes darting toward the exit. (Loyalty %d%% — betrayal risk)", pet.Name, pet.Loyalty))
		}

		betrayalChance := abyssPetBetrayalChance(pet.Loyalty, au.treeBonus.Pct["pet_betrayal_reduce"])
		if len(ctx.activeUsers) > 0 && ctx.random.Float64() < betrayalChance { // #nosec G404
			pet.Loyalty = max(0, pet.Loyalty-5)
			targetUser := ctx.activeUsers[ctx.random.IntN(len(ctx.activeUsers))].u // #nosec G404
			if targetUser.CurrentHP > 0 {
				damage := int(float64(pet.Stats.STR-targetUser.Stats.DEF) * ctx.intensify)
				if damage < 1 {
					damage = 1
				}
				targetUser.DamageTaken += damage
				targetUser.CurrentHP -= damage
				*ctx.logs = append(*ctx.logs, i18n.T("bot.combat.rogue_pet_bite", pet.Name, targetUser.Nickname, damage))
				*ctx.totalMobDamage += damage
				if targetUser.CurrentHP <= 0 {
					targetUser.CurrentHP = 0
					if !b.checkUserRevive(targetUser, ctx.logs) {
						*ctx.logs = append(*ctx.logs, i18n.T("bot.combat.slain_by_pet", targetUser.Nickname, pet.Name))
					}
				}
				continue
			}
		}

		ability, hasAbility := abyssPetAbilityForClass(petIndex+1, pet.PetClass)
		abilityReady := hasAbility && au.petCooldowns[petIndex] == 0
		if abilityReady && ability.Kind == "heal" && abyssPetAutoskillEnabled(u.petHealEnabled, pet.Name) {
			var bestTarget *UserInCombat
			lowestHealth := 1.0
			for i := range ctx.activeUsers {
				targetUser := ctx.activeUsers[i].u
				if targetUser.Stats.HP <= 0 || targetUser.CurrentHP <= 0 ||
					targetUser.CurrentHP >= targetUser.Stats.HP {
					continue
				}
				ratio := float64(targetUser.CurrentHP) / float64(targetUser.Stats.HP)
				if ratio < lowestHealth {
					lowestHealth = ratio
					bestTarget = targetUser
				}
			}
			if bestTarget != nil {
				heal := int(float64(bestTarget.Stats.HP)*ability.PowerScale) + pet.Level*3
				if heal < 10 {
					heal = 10
				}
				bestTarget.CurrentHP += heal
				if bestTarget.CurrentHP > bestTarget.Stats.HP {
					heal -= bestTarget.CurrentHP - bestTarget.Stats.HP
					bestTarget.CurrentHP = bestTarget.Stats.HP
				}
				setAbyssPetAbilityCooldown(au, petIndex, ability.Cooldown)
				*ctx.logs = append(*ctx.logs, fmt.Sprintf("✨ [color=#4caf50]%s's Pet %s casts %s on %s, restoring %d HP! (Cooldown: %d rounds)[/color]", u.Nickname, pet.Name, ability.Name, bestTarget.Nickname, heal, ability.Cooldown))
				if bark := abyssPetBark(pet.PetBark, pet.Name, "heal"); bark != "" {
					*ctx.logs = append(*ctx.logs, bark)
				}
				continue
			}
		}

		aliveMobs := b.getAliveMobs(*ctx.mobs)
		if len(aliveMobs) == 0 {
			break
		}
		target := selectAbyssPetTarget(aliveMobs, au, ctx.random)
		if target == nil {
			break
		}
		if normalizeAbyssPetCommand(string(au.petCommand)) == abyssPetCommandFocus && !au.petFocusLogged {
			au.petFocusLogged = true
			*ctx.logs = append(*ctx.logs, fmt.Sprintf("🎯 %s focuses %s on %s.", u.Nickname, pet.Name, target.Name))
		}

		damageMultiplier := 1.0
		if bonus := au.treeBonus.Pct["pet_damage_pct"]; bonus > 0 {
			damageMultiplier += bonus
		}
		damageMultiplier *= abyssTreeActionMultiplier(au.treeBonus, "companion_skill_power")
		usesAttackAbility := abilityReady && ability.Kind == "attack"
		if usesAttackAbility {
			damageMultiplier *= ability.PowerScale
		}
		damage := int(float64(pet.Stats.STR-target.Stats.DEF) * damageMultiplier * ctx.intensify)
		if damage < 1 {
			damage = 1
		}
		damage = abyssKillerDamage(damage, u, target)
		remainingHP := target.Stats.HP
		overkill := abyssOverkillHit(damage, remainingHP)
		target.Stats.HP -= damage
		appendAbyssExecuteThresholdLog(ctx.logs, target, remainingHP, abyssCombatant(u))
		applyAbyssBreakDamage(target, damage, ctx.logs)
		*ctx.totalUserDamage += damage
		if usesAttackAbility {
			setAbyssPetAbilityCooldown(au, petIndex, ability.Cooldown)
			*ctx.logs = append(*ctx.logs, fmt.Sprintf("🦷 %s uses %s on %s for %d damage! (Cooldown: %d rounds)", pet.Name, ability.Name, target.Name, damage, ability.Cooldown))
		}
		if target.Stats.HP > 0 {
			continue
		}

		killLog := i18n.T("bot.combat.killed_by_pet", target.Name, pet.Name)
		*ctx.logs = append(*ctx.logs, markAbyssOverkillLog(killLog, abyssCombatant(u) && overkill))
		if bark := abyssPetBark(pet.PetBark, pet.Name, "kill"); bark != "" {
			*ctx.logs = append(*ctx.logs, bark)
		}
		if winner := randomLootEligibleUser(ctx.originalUsers, ctx.random); winner != nil {
			b.awardCombatLoot(winner, *target, ctx.zone, ctx.logs, ctx.loots)
		}
		b.handleDeathEffects(target, ctx.mobs, ctx.logs, ctx.avgLevel, ctx.difficulty, ctx.activeUsers, ctx.random)
		if ctx.track != nil {
			if finalOverkill := abyssTerminalOverkillDamage(*ctx.mobs, target, damage, remainingHP); finalOverkill > 0 {
				ctx.track.overkill = finalOverkill
			}
		}
	}
}
