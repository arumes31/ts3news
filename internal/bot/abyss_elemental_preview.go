package bot

import "ts3news/internal/content"

type abyssElementalPreviewView struct {
	Active           bool
	EnemyName        string
	EnemyElement     string
	EnemyIcon        string
	PlayerElement    string
	PlayerIcon       string
	StrongElement    string
	StrongIcon       string
	ResistedElement  string
	ResistedIcon     string
	Outcome          string
	OutcomeIcon      string
	OutcomeClass     string
	Multiplier       string
	TargetDepth      int
	TwinBossDepth    int
	SecretEncounter  bool
	NeutralEncounter bool
}

func abyssElementalPreview(
	forecast abyssBossAffinityForecastView,
	equipped map[content.GearSlot]content.Gear,
) abyssElementalPreviewView {
	enemyElement, enemyIcon := abyssElementDisplay(content.Element(forecast.Element))
	playerElement, playerIcon := abyssElementDisplay(abyssEquippedAttackElement(equipped))
	multiplier := getElementMult(playerElement, enemyElement)

	view := abyssElementalPreviewView{
		Active:           forecast.TargetDepth > 0 && forecast.Element != "",
		EnemyName:        forecast.Name,
		EnemyElement:     string(enemyElement),
		EnemyIcon:        enemyIcon,
		PlayerElement:    string(playerElement),
		PlayerIcon:       playerIcon,
		Multiplier:       abyssElementMultiplierLabel(multiplier),
		TargetDepth:      forecast.TargetDepth,
		TwinBossDepth:    abyssDoubleBossDepth,
		SecretEncounter:  forecast.Secret,
		NeutralEncounter: enemyElement == content.ElementPhysical,
	}

	switch {
	case multiplier > 1:
		view.Outcome, view.OutcomeIcon, view.OutcomeClass = "ADVANTAGE", "▲", "strong"
	case multiplier < 1:
		view.Outcome, view.OutcomeIcon, view.OutcomeClass = "RESISTED", "▼", "resisted"
	default:
		view.Outcome, view.OutcomeIcon, view.OutcomeClass = "NEUTRAL", "◆", "neutral"
	}

	if !view.NeutralEncounter {
		strongElement, strongIcon := abyssElementDisplay(content.Element(forecast.WeakTo))
		resistedElement, resistedIcon := abyssElementDisplay(content.Element(forecast.StrongAgainst))
		view.StrongElement, view.StrongIcon = string(strongElement), strongIcon
		view.ResistedElement, view.ResistedIcon = string(resistedElement), resistedIcon
	}
	return view
}

func abyssEquippedAttackElement(equipped map[content.GearSlot]content.Gear) content.Element {
	weapon, ok := equipped[content.SlotMainHand]
	if !ok || !abyssGearActiveForCombat(weapon) {
		return content.ElementPhysical
	}
	element, _ := abyssElementDisplay(weapon.Element)
	return element
}

func abyssElementDisplay(element content.Element) (content.Element, string) {
	switch element {
	case content.ElementFire:
		return element, "🔥"
	case content.ElementWater:
		return element, "💧"
	case content.ElementEarth:
		return element, "🜃"
	case content.ElementAir:
		return element, "🌪"
	default:
		return content.ElementPhysical, "⚔"
	}
}

func abyssElementMultiplierLabel(multiplier float64) string {
	switch multiplier {
	case 2:
		return "2×"
	case 0.5:
		return "½×"
	default:
		return "1×"
	}
}
