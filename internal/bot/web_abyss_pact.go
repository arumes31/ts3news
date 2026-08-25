package bot

import (
	"strings"

	"ts3news/internal/content"
)

// Pacts are optional, self-imposed challenge modifiers selected at run start.
// Each challenge increases danger or removes a safety system in exchange for a
// fixed cache bonus. The server validates and persists the canonical set.
type abyssPact struct {
	Key    string
	Label  string
	Desc   string
	Reward float64 // additive escrow-reward bonus (0.20 = +20%)
	Token  string  // FloorModifier token folded into combat ("" = none)
	Danger float64 // floor-difficulty multiplier (1.0 = none)
	Enrage bool    // adds EffectEnraged to every spawned mob
}

// abyssPactCatalog is the fixed set of selectable pacts.
var abyssPactCatalog = []abyssPact{
	{"double_hazards", "Doubled Hazards", "Floor hazard damage is doubled.", 0.15, "double_hazards", 1.0, false},
	{"vampiric_mobs", "Vampiric Host", "Enemies heal for 15% of the damage they deal.", 0.20, "vampiric_mobs", 1.0, false},
	{"enraged", "Enraged Host", "Every enemy enters combat enraged.", 0.20, "", 1.0, true},
	{"glass_cannon", "Glass Cannon", "Floors are 30% deadlier.", 0.30, "", 1.3, false},
	{"abstinence", "Abstinence", "Consumables cannot be carried or used.", 0.15, "", 1.0, false},
	{"pauper", "Pauper", "Only Common, Uncommon, and Rare gear may be equipped.", 0.30, "", 1.0, false},
	{"anemic", "Anemic", "Maximum HP is halved for every fight.", 0.25, "", 1.0, false},
	{"cursed_horde", "Cursed Horde", "Every enemy gains one additional beneficial affix.", 0.20, "", 1.0, false},
	{"deep_drums", "Deep Drums", "A boss answers every third floor.", 0.35, "", 1.0, false},
	{"uninsured", "Uninsured", "Cache insurance is disabled for the entire run.", 0.15, "", 1.0, false},
	{"blind", "Blind", "Floor choices, route forecasts, and the threat meter are concealed.", 0.10, "", 1.0, false},
	{"brittle", "Brittle", "Equipped gear suffers a second durability-loss pass after combat.", 0.10, "", 1.0, false},
	{"famine", "Famine", "Sanctuary rest floors never appear.", 0.20, "", 1.0, false},
}

// abyssPactIndex maps pact key → catalog entry once for constant-time lookups.
var abyssPactIndex = func() map[string]abyssPact {
	m := make(map[string]abyssPact, len(abyssPactCatalog))
	for _, p := range abyssPactCatalog {
		m[p.Key] = p
	}
	return m
}()

func abyssPactByKey(key string) (abyssPact, bool) {
	p, ok := abyssPactIndex[key]
	return p, ok
}

func abyssHasPact(pacts []string, key string) bool {
	for _, pact := range pacts {
		if pact == key {
			return true
		}
	}
	return false
}

func abyssPactBossFloor(pacts []string, depth int) bool {
	return abyssHasPact(pacts, "deep_drums") && depth > 0 && depth%3 == 0
}

func abyssPactAllowsRest(pacts []string) bool {
	return !abyssHasPact(pacts, "famine")
}

func abyssPactMaxHP(pacts []string, maxHP int) int {
	if abyssHasPact(pacts, "anemic") {
		return max(1, maxHP/2)
	}
	return maxHP
}

func abyssPactDurabilityPasses(pacts []string) int {
	if abyssHasPact(pacts, "brittle") {
		return 2
	}
	return 1
}

func abyssPactEquipmentError(pacts []string, equipped map[content.GearSlot]content.Gear) string {
	if !abyssHasPact(pacts, "pauper") {
		return ""
	}
	for _, gear := range equipped {
		if gear.Rarity > content.RarityRare {
			return "Pauper allows only Rare-or-lower equipped gear"
		}
	}
	return ""
}

func abyssApplyCursedHorde(mobs []content.Mob, source combatRandomSource) {
	if len(mobs) == 0 || source == nil {
		return
	}
	pool := []content.MobEffect{
		content.EffectEnraged,
		content.EffectArmored,
		content.EffectFleet,
		content.EffectRegen,
	}
	for index := range mobs {
		start := source.IntN(len(pool))
		for offset := range pool {
			effect := pool[(start+offset)%len(pool)]
			if !mobHasEffect(mobs[index], effect) {
				mobs[index].Effects = append(mobs[index].Effects, effect)
				break
			}
		}
	}
}

func mobHasEffect(mob content.Mob, want content.MobEffect) bool {
	for _, effect := range mob.Effects {
		if effect == want {
			return true
		}
	}
	return false
}

// abyssValidatePacts filters a requested pact list down to known keys, de-duplicated
// and in catalog order, and returns the canonical space-separated storage value.
func abyssValidatePacts(req []string) string {
	want := make(map[string]bool, len(req))
	for _, k := range req {
		want[k] = true
	}
	var keys []string
	for _, p := range abyssPactCatalog {
		if want[p.Key] {
			keys = append(keys, p.Key)
		}
	}
	return strings.Join(keys, " ")
}

// abyssRunPacts reads the active run's stored pact set as a slice of keys.
func (b *Bot) abyssRunPacts(uid string) []string {
	var s string
	_ = b.DB.QueryRow("SELECT COALESCE(pacts, '') FROM abyss_active WHERE client_uid=$1", uid).Scan(&s)
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}

// abyssPactRewardMult is one plus every active pact's additive cache bonus.
func abyssPactRewardMult(pacts []string) float64 {
	mult := 1.0
	for _, k := range pacts {
		if p, ok := abyssPactByKey(k); ok {
			mult += p.Reward
		}
	}
	return mult
}

// abyssPactDangerMult multiplies the explicit danger scalars of active pacts.
func abyssPactDangerMult(pacts []string) float64 {
	mult := 1.0
	for _, k := range pacts {
		if p, ok := abyssPactByKey(k); ok && p.Danger > 1.0 {
			mult *= p.Danger
		}
	}
	return mult
}

func abyssPactsEnrage(pacts []string) bool {
	for _, k := range pacts {
		if p, ok := abyssPactByKey(k); ok && p.Enrage {
			return true
		}
	}
	return false
}

func abyssPactCombatTokens(pacts []string) []string {
	var tokens []string
	for _, k := range pacts {
		if p, ok := abyssPactByKey(k); ok && p.Token != "" {
			tokens = append(tokens, p.Token)
		}
	}
	return tokens
}
