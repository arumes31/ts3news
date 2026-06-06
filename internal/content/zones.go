package content

import (
	"fmt"
	"math/rand"
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
				Power: 0.1 + (0.05 * float64(rand.Intn(10))),
			}

			// Custom logic for effects
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
	zoneNames := []string{
		"Obsidian Sanctum", "Azure Plateau", "Emerald Dreamscape", "Searing Gorge", "Whispering Woods",
		"Frozen Tundra", "Twilight Highlands", "Molten Core", "Shadowfen", "Thunder Bluff",
		"Astral Nexus", "Void Rift", "Hellfire Peninsula", "Deadwind Pass", "Sunwell Plateau",
	}

	z := Zone{
		Name: zoneNames[rand.Intn(len(zoneNames))],
	}

	// Scaling Difficulty: harder zones for better players
	// Base difficulty 1.0, increases with level and GS
	z.Difficulty = 1.0 + (float64(partyAvgLvl) * 0.02) + (float64(partyGearScore) * 0.0005)
	
	// Add 1-3 stacking effects
	effectCount := 1 + rand.Intn(3)
	for i := 0; i < effectCount; i++ {
		z.Effects = append(z.Effects, allZoneEffects[rand.Intn(len(allZoneEffects))])
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
