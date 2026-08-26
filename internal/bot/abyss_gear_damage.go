package bot

import "ts3news/internal/content"

type abyssGearDamageView struct {
	Available     bool
	BasePower     int
	BaseSharePct  int
	SpellPower    int
	SpellSharePct int
	AttackType    string
	HasDirect     bool
}

func abyssGearDamageContributions(equipped map[content.GearSlot]content.Gear) map[content.GearSlot]abyssGearDamageView {
	positiveSTR, positiveINT := 0, 0
	for _, gear := range equipped {
		if gear.Unidentified {
			continue
		}
		positiveSTR += max(0, gear.Stats.STR)
		positiveINT += max(0, gear.Stats.INT)
	}

	out := make(map[content.GearSlot]abyssGearDamageView, len(equipped))
	for slot, gear := range equipped {
		if gear.Unidentified {
			continue
		}
		view := abyssGearDamageView{
			Available:     true,
			BasePower:     gear.Stats.STR,
			BaseSharePct:  abyssGearContributionPct(gear.Stats.STR, positiveSTR),
			SpellPower:    gear.Stats.INT,
			SpellSharePct: abyssGearContributionPct(gear.Stats.INT, positiveINT),
		}
		if slot == content.SlotMainHand {
			view.AttackType = string(gear.Element)
			if view.AttackType == "" {
				view.AttackType = string(content.ElementPhysical)
			}
		}
		view.HasDirect = view.BasePower != 0 || view.SpellPower != 0 || view.AttackType != ""
		out[slot] = view
	}
	return out
}

func abyssGearContributionPct(value, positiveTotal int) int {
	if value <= 0 || positiveTotal <= 0 {
		return 0
	}
	percent := int((int64(value)*100 + int64(positiveTotal)/2) / int64(positiveTotal))
	return min(100, max(1, percent))
}
