package bot

import (
	"fmt"

	"ts3news/internal/content"
)

const (
	abyssWeakpointHead = "head"
	abyssWeakpointArms = "arms"
)

type abyssLiveWeakpoint struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type abyssBossWeakpointEffect struct {
	DamageMultiplier float64
	Silence          bool
	Log              string
}

func abyssBossWeakpointsAvailable(role string, hp, maxHP int) bool {
	return role == "boss" && hp > 0 && maxHP > 0 && hp*2 <= maxHP
}

func abyssBossWeakpointsFor(role string, hp, maxHP int) []abyssLiveWeakpoint {
	if !abyssBossWeakpointsAvailable(role, hp, maxHP) {
		return nil
	}
	return []abyssLiveWeakpoint{
		{ID: abyssWeakpointHead, Name: "Head", Description: "+20% damage to this hit"},
		{ID: abyssWeakpointArms, Name: "Arms", Description: "Silence the boss's next spell"},
	}
}

func validAbyssBossWeakpoint(weakpoint string, enemy abyssLiveCombatantView) bool {
	if weakpoint == "" {
		return true
	}
	if !abyssBossWeakpointsAvailable(enemy.Role, enemy.HP, enemy.MaxHP) {
		return false
	}
	return weakpoint == abyssWeakpointHead || weakpoint == abyssWeakpointArms
}

func resolveAbyssBossWeakpoint(weakpoint string, target *content.Mob) abyssBossWeakpointEffect {
	if target == nil || target.Type != content.MobBoss || target.Stats.HP <= 0 ||
		target.MaxHP <= 0 || target.Stats.HP*2 > target.MaxHP {
		return abyssBossWeakpointEffect{DamageMultiplier: 1}
	}

	switch weakpoint {
	case abyssWeakpointHead:
		return abyssBossWeakpointEffect{
			DamageMultiplier: 1.2,
			Log:              fmt.Sprintf("🎯 Weakpoint: the strike lands on %s's head! (+20%% damage)", target.Name),
		}
	case abyssWeakpointArms:
		return abyssBossWeakpointEffect{
			DamageMultiplier: 1,
			Silence:          true,
			Log:              fmt.Sprintf("🎯 Weakpoint: %s's arms are disabled — its next spell is silenced!", target.Name),
		}
	default:
		return abyssBossWeakpointEffect{DamageMultiplier: 1}
	}
}

func silenceAbyssBoss(target *content.Mob) {
	if target == nil {
		return
	}
	for _, effect := range target.Effects {
		if effect == content.EffectSilenced {
			return
		}
	}
	target.Effects = append(target.Effects, content.EffectSilenced)
}

func consumeAbyssBossSilence(target *content.Mob) bool {
	if target == nil {
		return false
	}
	kept := target.Effects[:0]
	consumed := false
	for _, effect := range target.Effects {
		if effect == content.EffectSilenced {
			consumed = true
			continue
		}
		kept = append(kept, effect)
	}
	target.Effects = kept
	return consumed
}
