package bot

import (
	"fmt"
	"strings"
	"unicode"

	"ts3news/internal/content"
)

const abyssLivePresentationLimit = 128

// Equipment has slots and names, rather than a weapon-class stat. This hint is
// only for art selection; combat continues to use its existing equipped stats.
func abyssPresentationWeapon(user *UserInCombat) (weaponType, name string) {
	if user == nil {
		return "", ""
	}
	weapon := user.Equipped[content.SlotMainHand]
	if ranged, ok := user.Equipped[content.SlotRanged]; ok && user.Position == content.PositionBackline {
		return "ranged", ranged.Name
	}
	name = weapon.Name
	for _, word := range strings.FieldsFunc(strings.ToLower(name), func(r rune) bool { return !unicode.IsLetter(r) }) {
		switch word {
		case "crossbow", "bow", "staff", "wand", "dagger", "sword", "axe", "hammer", "spear":
			return word, name
		case "longbow", "shortbow":
			return "bow", name
		}
	}
	return "", name
}

func abyssPresentationPetAbilityID(name string) string {
	switch name {
	case "Pounce":
		return "pet_pounce"
	case "Healing Spell":
		return "pet_healing_spell"
	case "Mending Cry":
		return "pet_mending_cry"
	default:
		return "pet_ability"
	}
}

// Presentation events describe resolved engine actions. They never schedule or
// influence combat, and intentionally contain no absolute or maximum HP values.
type combatManaChange struct {
	Before int `json:"before"`
	After  int `json:"after"`
	Max    int `json:"max"`
}

type abyssLivePresentationEvent struct {
	Mana        *combatManaChange              `json:"mana,omitempty"`
	Seq         int64                          `json:"seq"`
	Round       int                            `json:"round"`
	Kind        string                         `json:"kind"`
	ActorID     string                         `json:"actor_id"`
	AbilityID   string                         `json:"ability_id,omitempty"`
	AbilityName string                         `json:"ability_name,omitempty"`
	Element     string                         `json:"element,omitempty"`
	Targets     []abyssLivePresentationOutcome `json:"targets,omitempty"`
}

type abyssLivePresentationOutcome struct {
	TargetID string `json:"target_id"`
	Damage   int    `json:"damage,omitempty"`
	Healing  int    `json:"healing,omitempty"`
	Absorbed int    `json:"absorbed,omitempty"`
	Damaged  bool   `json:"damaged,omitempty"`
	Healed   bool   `json:"healed,omitempty"`
	Blocked  bool   `json:"blocked,omitempty"`
	Dodged   bool   `json:"dodged,omitempty"`
	Critical bool   `json:"critical,omitempty"`
	Status   string `json:"status,omitempty"`
	Defeated bool   `json:"defeated,omitempty"`
}

func (c *abyssLiveCombat) presentationEntity(mob *content.Mob, prefix string) string {
	if c == nil || mob == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.presentationEntities == nil {
		c.presentationEntities = make(map[*content.Mob]map[string]string)
		c.presentationEntityCounts = make(map[string]int)
	}
	if c.presentationEntities[mob] == nil {
		c.presentationEntities[mob] = make(map[string]string)
	}
	if id := c.presentationEntities[mob][prefix]; id != "" {
		return id
	}
	id := fmt.Sprintf("%s:%d", prefix, c.presentationEntityCounts[prefix])
	c.presentationEntityCounts[prefix]++
	c.presentationEntities[mob][prefix] = id
	return id
}

func (c *abyssLiveCombat) present(round int, kind, actor, abilityID, name string, element content.Element, outcomes ...abyssLivePresentationOutcome) {
	c.presentWithMana(round, kind, actor, abilityID, name, element, nil, outcomes...)
}

func (c *abyssLiveCombat) presentWithMana(round int, kind, actor, abilityID, name string, element content.Element, mana *combatManaChange, outcomes ...abyssLivePresentationOutcome) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if round <= 0 {
		round = max(1, c.round)
	}
	c.presentationCursor++
	c.presentationEvents = append(c.presentationEvents, abyssLivePresentationEvent{
		Mana: mana,
		Seq:  c.presentationCursor, Round: round, Kind: kind, ActorID: actor,
		AbilityID: abilityID, AbilityName: name, Element: string(element),
		Targets: append([]abyssLivePresentationOutcome(nil), outcomes...),
	})
	if len(c.presentationEvents) > abyssLivePresentationLimit {
		c.presentationEvents = append([]abyssLivePresentationEvent(nil), c.presentationEvents[len(c.presentationEvents)-abyssLivePresentationLimit:]...)
	}
}

func cloneAbyssPresentationEvents(events []abyssLivePresentationEvent) []abyssLivePresentationEvent {
	cloned := append([]abyssLivePresentationEvent{}, events...)
	for i := range cloned {
		if cloned[i].Mana != nil {
			mana := *cloned[i].Mana
			cloned[i].Mana = &mana
		}
		cloned[i].Targets = append([]abyssLivePresentationOutcome(nil), cloned[i].Targets...)
	}
	return cloned
}

func (c *abyssLiveCombat) presentationForLocked() []abyssLivePresentationEvent {
	events := cloneAbyssPresentationEvents(c.presentationEvents)
	hidden := make(map[string]bool)
	for _, views := range [][]abyssLiveCombatantView{c.allies, c.enemies} {
		for _, view := range views {
			if view.HPHidden {
				hidden[view.EntityID], hidden[view.ID] = true, true
			}
		}
	}
	darkness := hasAbyssFloorModifier(c.modifier, "darkness")
	for i := range events {
		for j := range events[i].Targets {
			outcome := &events[i].Targets[j]
			if hidden[outcome.TargetID] || darkness && strings.HasPrefix(outcome.TargetID, "enemy:") {
				// Preserve the public outcome type without exposing concealed HP.
				outcome.Damaged, outcome.Healed = outcome.Damage > 0, outcome.Healing > 0
				outcome.Damage, outcome.Healing, outcome.Absorbed = 0, 0, 0
			}
		}
	}
	return events
}

func abyssPresentationDamage(targetID string, damage, previousHP, remainingHP int) abyssLivePresentationOutcome {
	return abyssLivePresentationOutcome{
		TargetID: targetID, Damage: max(0, min(damage, previousHP)),
		Defeated: previousHP > 0 && remainingHP <= 0,
	}
}

func (c *abyssLiveCombat) presentMobDamage(round int, kind, actor, abilityID, name string, element content.Element, mob *content.Mob, damage, previousHP int) {
	if c == nil || mob == nil {
		return
	}
	c.present(round, kind, actor, abilityID, name, element,
		abyssPresentationDamage(c.presentationEntity(mob, "enemy"), damage, previousHP, mob.Stats.HP))
}

func (c *abyssLiveCombat) presentPhase(round int, mob *content.Mob, id, name string, outcomes ...abyssLivePresentationOutcome) {
	if c == nil || mob == nil {
		return
	}
	actor := c.presentationEntity(mob, "enemy")
	outcomes = append([]abyssLivePresentationOutcome{{TargetID: actor, Status: id}}, outcomes...)
	c.present(round, "phase", actor, id, name, mob.Element, outcomes...)
}

func abyssPresentationStunnedUsers(users []activeUser) []abyssLivePresentationOutcome {
	var outcomes []abyssLivePresentationOutcome
	for _, user := range users {
		if user.u != nil && user.Stunned {
			outcomes = append(outcomes, abyssLivePresentationOutcome{TargetID: "ally:" + user.u.UID, Status: "stunned"})
		}
	}
	return outcomes
}

// Complete snapshots must contain the terminal engine state, rather than the
// last planning state, so bars cannot undo a defeat after its animation.
func (c *abyssLiveCombat) capturePresentationFinalState(users []activeUser, mobs []*content.Mob) {
	if c == nil {
		return
	}
	enemyHP := make(map[string]int, len(mobs))
	for _, mob := range mobs {
		if mob != nil {
			enemyHP[c.presentationEntity(mob, "enemy")] = max(0, mob.Stats.HP)
		}
	}
	petHP := make(map[string]int)
	for _, user := range users {
		if user.u == nil {
			continue
		}
		for _, pet := range user.u.Pets {
			if pet != nil {
				petHP[c.presentationEntity(pet, "pet:"+user.u.UID)] = max(0, pet.Stats.HP)
			}
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.enemies {
		c.enemies[i].HP = enemyHP[c.enemies[i].EntityID]
	}
	for i := range c.allies {
		if hp, ok := petHP[c.allies[i].EntityID]; ok {
			c.allies[i].HP = hp
		}
		for _, user := range users {
			if user.u != nil && c.allies[i].ID == "ally:"+user.u.UID {
				c.allies[i].HP = max(0, user.u.CurrentHP)
				c.allies[i].Mana = max(0, user.CurrentMana)
				c.allies[i].Shield = max(0, user.shield)
			}
		}
	}
}
