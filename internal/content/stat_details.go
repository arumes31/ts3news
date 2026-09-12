package content

import "math"

// StatDetail is the stable, ordered description of an item's attribute.
type StatDetail struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Value       int    `json:"value"`
	Combat      bool   `json:"combat"`
}

// Details includes zero values so consumers can compare removed attributes.
func (s Stats) Details() []StatDetail {
	return []StatDetail{
		{"HP", "HP", "Health contributed by the item.", s.HP, true},
		{"MNA", "MNA", "Mana available for abilities.", s.MNA, true},
		{"STR", "STR", "Strength used by physical attacks and class scaling.", s.STR, true},
		{"DEF", "DEF", "Defense used to reduce incoming damage.", s.DEF, true},
		{"SPD", "SPD", "Speed used by combat and class scaling.", s.SPD, true},
		{"CRT", "CRT%", "Critical rating; final chance depends on combat rules and caps.", s.CRT, true},
		{"DGE", "DGE%", "Dodge rating; final chance depends on combat rules and caps.", s.DGE, true},
		{"LCK", "LCK", "Luck used by loot and combat mechanics.", s.LCK, true},
		{"INT", "INT", "Intelligence used by magic and class scaling.", s.INT, true},
		{"STA", "STA", "Stamina helps resist durability loss.", s.STA, true},
		{"CHA", "CHA", "Charisma is a flavour attribute, excluded from stat power.", s.CHA, false},
		{"STN", "STN", "Stench is a flavour attribute, excluded from stat power.", s.STN, false},
		{"SHN", "SHN", "Shiny is a flavour attribute, excluded from stat power.", s.SHN, false},
		{"HGR", "HGR", "Hunger is a flavour attribute, excluded from stat power.", s.HGR, false},
	}
}

// Power is a rarity-independent item-stat estimate, including mana. It is not
// a prediction of character damage and deliberately excludes flavour stats.
func (s Stats) Power() float64 {
	power := float64(s.STR)*1.2 + float64(s.DEF)*.9 + float64(s.HP)*.3 +
		float64(s.SPD)*1.1 + float64(s.CRT)*1.5 + float64(s.DGE)*1.3 +
		float64(s.LCK)*.8 + float64(s.INT)*.7 + float64(s.STA)*.6 + float64(s.MNA)*.1
	return math.Round(power*10) / 10
}

type GearXPDetail struct {
	RawBonusPct       float64 `json:"raw_bonus_pct"`
	EffectiveBonusPct float64 `json:"effective_bonus_pct"`
	Explanation       string  `json:"explanation"`
	Valid             bool    `json:"valid"`
}

// XPDetail explains the exact eligibility/cap rule used by gameplay.
func (g Gear) XPDetail() GearXPDetail {
	if g.Unidentified {
		return GearXPDetail{Explanation: "Identify this item to reveal its XP contribution."}
	}
	if math.IsNaN(g.XPMultiplier) || math.IsInf(g.XPMultiplier, 0) || g.XPMultiplier < 0 {
		return GearXPDetail{Explanation: "XP data is invalid; automatic comparison is unavailable."}
	}
	detail := GearXPDetail{RawBonusPct: math.Round((g.XPMultiplier-1)*1000) / 10, EffectiveBonusPct: math.Round((g.EffectiveXPMultiplier()-1)*1000) / 10, Valid: true}
	switch {
	case IsPetGearSlot(g.Slot):
		detail.Explanation = "Pet equipment does not contribute to player XP."
	case g.Rarity < RarityRare:
		detail.Explanation = "Common and Uncommon equipment does not contribute an item XP bonus."
	case g.EffectiveXPMultiplier() < g.XPMultiplier:
		detail.Explanation = "This slot caps the item XP bonus at 2%."
	default:
		detail.Explanation = "The item's full XP multiplier applies; other equipment and character bonuses combine separately."
	}
	return detail
}
