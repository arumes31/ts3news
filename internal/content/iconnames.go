package content

import "strings"

// This file maps game concepts (equipment slots, item effects, stats) to the
// basename of a game-icons.net SVG served by the web portal at
// /static/icons/<name>.svg. These are display-only and used by the web layer;
// the emoji SlotIcon is kept for plain-text (TeamSpeak) output.

var slotIconNames = map[GearSlot]string{
	SlotHead: "visored-helm", SlotNeck: "gem-pendant", SlotShoulders: "spiked-shoulder-armor",
	SlotBack: "cloak", SlotChest: "breastplate", SlotWrists: "bracers", SlotHands: "gauntlet",
	SlotWaist: "belt-armor", SlotLegs: "leg-armor", SlotFeet: "leather-boot",
	SlotFinger1: "ring", SlotFinger2: "ring", SlotTrinket1: "crystal-ball", SlotTrinket2: "orb-wand",
	SlotMainHand: "broadsword", SlotOffHand: "checked-shield", SlotRanged: "pocket-bow",
	SlotRelic: "relic-blade", SlotArtifact: "crystal-cluster", SlotSoul: "spectre", SlotAura: "aura",
	SlotCharm: "clover", SlotMount: "wolf-head", SlotCompanion: "eagle-emblem",
	SlotPet1: "dragon-head", SlotPet2: "bird-mask", SlotEmblem1: "medal", SlotEmblem2: "medal",
	SlotBanner: "flying-flag", SlotTotem: "totem",
}

// SlotIconName returns the game-icons.net icon basename for an equipment slot.
func SlotIconName(slot GearSlot) string {
	if n, ok := slotIconNames[slot]; ok {
		return n
	}
	return "crystal-cluster"
}

var effectIconNames = map[ItemEffect]string{
	EffectThorns: "thorny-vine", EffectVampiric: "fangs", EffectBerserk: "enrage",
	EffectLucky: "clover", EffectTreasureHunter: "open-treasure-chest", EffectQuick: "sprint",
	EffectBulwark: "checked-shield", EffectRadiant: "sunbeams", EffectFragile: "cracked-shield",
	EffectSteady: "stone-block", EffectMindControl: "psychic-waves", EffectRegenStack: "regeneration",
	EffectPhoenix: "fire-silhouette", EffectStealth: "hood", EffectParry: "sword-clash", EffectCleanse: "holy-water",
}

// EffectIconName returns the icon basename for a special item effect ("" if none).
func EffectIconName(e ItemEffect) string {
	return effectIconNames[e]
}

var statIconNames = map[string]string{
	"STR": "biceps", "DEF": "checked-shield", "SPD": "run", "HP": "health-normal",
	"CRT": "bullseye", "DGE": "dodging", "LCK": "clover", "INT": "brain", "STA": "lungs",
}

// StatIconName returns the icon basename for a stat code (e.g. "STR", "CRT%").
func StatIconName(code string) string {
	return statIconNames[strings.TrimSuffix(code, "%")]
}
