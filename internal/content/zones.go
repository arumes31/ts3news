package content

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

type ZoneEffectType string

const (
	ZoneBuff    ZoneEffectType = "Buff"
	ZoneDebuff  ZoneEffectType = "Debuff"
	ZoneHazard  ZoneEffectType = "Hazard"
	ZoneSpecial ZoneEffectType = "Special"
)

type ZoneEffect struct {
	ID          string
	Name        string
	Type        ZoneEffectType
	Power       float64 // Multiplier or magnitude
	Description string
}

type Zone struct {
	Name       string
	Difficulty float64 // Multiplier on top of base scaling
	Effects    []ZoneEffect
}

var allZoneEffects []ZoneEffect

func init() {
	// Generate 100+ unique zone effects
	prefixes := []string{
		"Volcanic", "Glacial", "Static", "Void", "Celestial", "Abyssal", "Primal", "Ancient", "Hallowed", "Cursed",
		"Toxic", "Metallic", "Glass", "Luminous", "Shadowy", "Arcane", "Raging", "Silent", "Eternal", "Blighted",
	}
	suffixes := []string{
		"Eruption", "Chill", "Shock", "Leak", "Surge", "Pressure", "Breeze", "Echo", "Grace", "Spite",
		"Vapors", "Plating", "Shatter", "Glow", "Mist", "Current", "Wind", "Stillness", "Bloom", "Rot",
	}

	idx := 1
	for _, p := range prefixes {
		for _, s := range suffixes {
			if idx > 120 { break }
			name := p + " " + s
			zType := ZoneBuff
			if idx%2 == 0 { zType = ZoneDebuff }
			if idx%5 == 0 { zType = ZoneHazard }
			if idx%10 == 0 { zType = ZoneSpecial }

			ze := ZoneEffect{
				ID:    fmt.Sprintf("ZE%d", idx),
				Name:  name,
				Type:  zType,
// #nosec G404
				Power: 0.1 + (0.05 * float64(rand.IntN(10))), // #nosec G404
			}

			switch zType {
			case ZoneBuff:
				ze.Description = fmt.Sprintf("Provides +%.0f%% to random combat stat.", ze.Power*100)
			case ZoneDebuff:
				ze.Description = fmt.Sprintf("Reduces random enemy combat stat by %.0f%%.", ze.Power*100)
			case ZoneHazard:
				ze.Description = fmt.Sprintf("Deals %.0f damage per round to everyone.", ze.Power*50)
			case ZoneSpecial:
				ze.Description = "Increases rare loot drop rates or spawns extra mobs."
			}

			allZoneEffects = append(allZoneEffects, ze)
			idx++
		}
	}
}

func GetRandomZone(partyAvgLvl int, partyGearScore int) Zone {
	// Tiered Zone Selection: Common (70%), Rare (20%), Legendary (10%)
	commonZones := []string{"Elwynn Forest", "Westfall", "Durotar", "Mulgore", "Teldrassil", "Loch Modan", "Silverpine"}
	rareZones := []string{"Stranglethorn Vale", "Tanaris", "Un'Goro Crater", "Winterspring", "Searing Gorge", "Burning Steppes"}
	legendaryZones := []string{"Molten Core", "Sunwell Plateau", "Icecrown Citadel", "Void Rift", "The Maelstrom"}

// #nosec G404
	r := rand.Float64() // #nosec G404
	var name string
	var baseDiff float64

	switch {
	case r < 0.70: // Common
// #nosec G404
		name = commonZones[rand.IntN(len(commonZones))] // #nosec G404
		baseDiff = 0.8 // Easier than average
	case r < 0.90: // Rare
// #nosec G404
		name = rareZones[rand.IntN(len(rareZones))] // #nosec G404
		baseDiff = 1.2
	default: // Legendary
// #nosec G404
		name = legendaryZones[rand.IntN(len(legendaryZones))] // #nosec G404
		baseDiff = 1.8 // Dangerous
	}

	z := Zone{
		Name: name,
	}

	// Scaling Difficulty: harder zones for better players, dampened by tier
	z.Difficulty = baseDiff + (float64(partyAvgLvl) * 0.001) + (float64(partyGearScore) * 0.00005)
	
	// Add 1-3 stacking effects (Legendary zones have more)
// #nosec G404
	effectCount := 1 + rand.IntN(2) // #nosec G404
	if r >= 0.90 { effectCount = 3 }
	
	for i := 0; i < effectCount; i++ {
// #nosec G404
		z.Effects = append(z.Effects, allZoneEffects[rand.IntN(len(allZoneEffects))]) // #nosec G404
	}

	return z
}

func (z Zone) Display() string {
	var effs []string
	for _, e := range z.Effects {
		effs = append(effs, fmt.Sprintf("%s (%s)", e.Name, e.Type))
	}
	return fmt.Sprintf("📍 %s [Diff: %.2f] — Effects: %s", z.Name, z.Difficulty, strings.Join(effs, ", "))
}
