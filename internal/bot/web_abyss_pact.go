package bot

import "strings"

// Pacts
// -----------------------------------------------------------------------------
// Pacts are optional, self-imposed challenge modifiers a player toggles on at run
// start (à la Hades' Pact of Punishment). Each makes the descent harder and, in
// return, lifts the escrow reward by a fixed amount. They stack on top of the
// difficulty tier and the daily affix, and persist for the whole run.
//
// They deliberately reuse the existing combat hooks: a pact's Token is folded into
// the player's FloorModifier (read in the engine via strings.Contains), Danger
// multiplies floor difficulty, and Enrage adds EffectEnraged to the spawned mobs —
// the same levers the daily affixes already use.

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
}

func abyssPactByKey(key string) (abyssPact, bool) {
	for _, p := range abyssPactCatalog {
		if p.Key == key {
			return p, true
		}
	}
	return abyssPact{}, false
}

// abyssValidatePacts filters a requested pact list down to known keys, de-duplicated
// and in catalog order, and returns the canonical space-separated string to persist.
func abyssValidatePacts(req []string) string {
	want := make(map[string]bool, len(req))
	for _, k := range req {
		want[k] = true
	}
	var keys []string
	for _, p := range abyssPactCatalog { // catalog order → stable storage
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

// abyssPactRewardMult is the escrow-reward multiplier from the active pacts (1.0 if
// none): the base 1.0 plus each pact's additive bonus.
func abyssPactRewardMult(pacts []string) float64 {
	mult := 1.0
	for _, k := range pacts {
		if p, ok := abyssPactByKey(k); ok {
			mult += p.Reward
		}
	}
	return mult
}

// abyssPactDangerMult is the floor-difficulty multiplier from the active pacts.
func abyssPactDangerMult(pacts []string) float64 {
	mult := 1.0
	for _, k := range pacts {
		if p, ok := abyssPactByKey(k); ok && p.Danger > 1.0 {
			mult *= p.Danger
		}
	}
	return mult
}

// abyssPactsEnrage reports whether any active pact enrages the spawned mobs.
func abyssPactsEnrage(pacts []string) bool {
	for _, k := range pacts {
		if p, ok := abyssPactByKey(k); ok && p.Enrage {
			return true
		}
	}
	return false
}

// abyssPactCombatTokens returns the FloorModifier tokens contributed by the active
// pacts (for folding into the combatant's modifier).
func abyssPactCombatTokens(pacts []string) []string {
	var toks []string
	for _, k := range pacts {
		if p, ok := abyssPactByKey(k); ok && p.Token != "" {
			toks = append(toks, p.Token)
		}
	}
	return toks
}
