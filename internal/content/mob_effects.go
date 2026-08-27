package content

// MobEffectInfo is the stable presentation contract for a combat effect.
// Mechanics remain authoritative in the combat engine; this metadata lets
// clients explain those mechanics without maintaining a duplicate name map.
type MobEffectInfo struct {
	Key         string
	Icon        string
	Description string
	Tone        string
}

var mobEffectInfo = map[MobEffect]MobEffectInfo{
	EffectEnraged: {
		Key: "enraged", Icon: "💢", Tone: "offense",
		Description: "Deals damage with 50% more Strength.",
	},
	EffectArmored: {
		Key: "armored", Icon: "🛡️", Tone: "defense",
		Description: "Carries reinforced defenses and increased threat.",
	},
	EffectFleet: {
		Key: "fleet", Icon: "💨", Tone: "speed",
		Description: "Carries heightened Speed and initiative pressure.",
	},
	EffectPoisoned: {
		Key: "poisoned", Icon: "☠️", Tone: "damage-over-time",
		Description: "Loses 5% of current HP per stack each round.",
	},
	EffectWeakened: {
		Key: "weakened", Icon: "🥀", Tone: "debuff",
		Description: "Deals damage with 50% less Strength.",
	},
	EffectBlinded: {
		Key: "blinded", Icon: "🌫️", Tone: "control",
		Description: "Has a 50% chance for each attack to miss.",
	},
	EffectRegen: {
		Key: "regenerative", Icon: "💖", Tone: "sustain",
		Description: "Restores 5% of current HP per stack each round.",
	},
	EffectSilenced: {
		Key: "silenced", Icon: "🔇", Tone: "control",
		Description: "Its next spell is suppressed, then Silence expires.",
	},
}

var mobEffectOrder = []MobEffect{
	EffectEnraged,
	EffectArmored,
	EffectFleet,
	EffectPoisoned,
	EffectWeakened,
	EffectBlinded,
	EffectRegen,
	EffectSilenced,
}

// MobEffectDetails returns presentation metadata for a declared effect.
func MobEffectDetails(effect MobEffect) (MobEffectInfo, bool) {
	info, ok := mobEffectInfo[effect]
	return info, ok
}

// MobEffectCatalog returns all documented monster affixes in stable display
// order. Values are copied, so callers cannot mutate the canonical table.
func MobEffectCatalog() []MobEffectInfo {
	catalog := make([]MobEffectInfo, 0, len(mobEffectOrder))
	for _, effect := range mobEffectOrder {
		catalog = append(catalog, mobEffectInfo[effect])
	}
	return catalog
}
