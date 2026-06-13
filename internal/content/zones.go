package content

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"ts3news/internal/i18n"
)

type ZoneEffectType string

const (
	ZoneBuff    ZoneEffectType = "Buff"
	ZoneDebuff  ZoneEffectType = "Debuff"
	ZoneHazard  ZoneEffectType = "Hazard"
	ZoneSpecial ZoneEffectType = "Special"
)

// ZoneEffect represents a temporary environmental effect tied to a specific zone instance.
// Note: These are distinct from Hazards in hazards.go; ZoneEffects are simpler round-based
// modifications while Hazards are status effects that can be resisted and have durations.
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
var zoneEffectsInitialized bool

func initZoneEffects() {
	if zoneEffectsInitialized {
		return
	}
	zoneEffectsInitialized = true

	// Generate 100+ unique zone effects
	prefixes := i18n.Pool("pool.zone.prefix")
	suffixes := i18n.Pool("pool.zone.suffix")

	// Safety check for empty pools (can happen during init before i18n is fully loaded)
	if len(prefixes) == 0 {
		prefixes = []string{"Mystic", "Ancient", "Dark", "Light", "Frozen", "Burning", "Shadow", "Holy", "Cursed", "Blessed"}
	}
	if len(suffixes) == 0 {
		suffixes = []string{"Forest", "Cave", "Mountain", "Valley", "Ruins", "Temple", "Crypt", "Tower", "Lake", "Wastes"}
	}

	idx := 1
	for _, p := range prefixes {
		for _, s := range suffixes {
			if idx > 120 {
				break
			}
			name := p + " " + s
			zType := ZoneBuff
			if idx%2 == 0 {
				zType = ZoneDebuff
			}
			if idx%5 == 0 {
				zType = ZoneHazard
			}
			if idx%10 == 0 {
				zType = ZoneSpecial
			}

			ze := ZoneEffect{
				ID:   fmt.Sprintf("ZE%d", idx),
				Name: name,
				Type: zType,
				// #nosec G404
				Power: 0.1 + (0.05 * float64(rand.IntN(10))), // #nosec G404
			}

			switch zType {
			case ZoneBuff:
				ze.Description = i18n.T("content.zone.effect.buff_desc", ze.Power*100)
			case ZoneDebuff:
				ze.Description = i18n.T("content.zone.effect.debuff_desc", ze.Power*100)
			case ZoneHazard:
				ze.Description = i18n.T("content.zone.effect.hazard_desc", ze.Power*50)
			case ZoneSpecial:
				ze.Description = i18n.T("content.zone.effect.special_desc")
			}

			allZoneEffects = append(allZoneEffects, ze)
			idx++
		}
	}

	// Add specific hazards (Renamed to distinguish from Hazard system)
	allZoneEffects = append(allZoneEffects, ZoneEffect{
		ID: "ZE_LAVA_POOLS", Name: i18n.T("content.zone.lava_pools.name"), Type: ZoneHazard, Power: 0.8, Description: i18n.T("content.zone.lava_pools.desc"),
	})
	allZoneEffects = append(allZoneEffects, ZoneEffect{
		ID: "ZE_GAS_FUMES", Name: i18n.T("content.zone.toxic_fumes.name"), Type: ZoneHazard, Power: 0.6, Description: i18n.T("content.zone.toxic_fumes.desc"),
	})
	allZoneEffects = append(allZoneEffects, ZoneEffect{
		ID: "ZE_SAND_GUSTS", Name: i18n.T("content.zone.sandstorm_gusts.name"), Type: ZoneHazard, Power: 0.4, Description: i18n.T("content.zone.sandstorm_gusts.desc"),
	})
	allZoneEffects = append(allZoneEffects, ZoneEffect{
		ID: "ZE_BLIZ_WINDS", Name: i18n.T("content.zone.blizzard_winds.name"), Type: ZoneHazard, Power: 0.5, Description: i18n.T("content.zone.blizzard_winds.desc"),
	})
}

func GetRandomZone(partyAvgLvl int, partyGearScore float64) Zone {
	initZoneEffects()

	// ... (common/rare/legendary selection) ...
	commonZones := []string{i18n.T("content.zone.elwynn_forest"), i18n.T("content.zone.westfall"), i18n.T("content.zone.durotar"), i18n.T("content.zone.mulgore"), i18n.T("content.zone.teldrassil"), i18n.T("content.zone.loch_modan"), i18n.T("content.zone.silverpine"), i18n.T("content.zone.desolace")}
	rareZones := []string{i18n.T("content.zone.stranglethorn_vale"), i18n.T("content.zone.tanaris"), i18n.T("content.zone.ungoro_crater"), i18n.T("content.zone.winterspring"), i18n.T("content.zone.searing_gorge"), i18n.T("content.zone.burning_steppes"), i18n.T("content.zone.deadwind_pass"), i18n.T("content.zone.eastern_plaguelands")}
	legendaryZones := []string{i18n.T("content.zone.molten_core"), i18n.T("content.zone.sunwell_plateau"), i18n.T("content.zone.icecrown_citadel"), i18n.T("content.zone.void_rift"), i18n.T("content.zone.maelstrom"), i18n.T("content.zone.firelands"), i18n.T("content.zone.shadowlands")}

	// #nosec G404
	r := rand.Float64() // #nosec G404
	var name string
	var baseDiff float64

	switch {
	case r < 0.70: // Common
		// #nosec G404
		name = commonZones[rand.IntN(len(commonZones))] // #nosec G404
		baseDiff = 0.8                                  // Easier than average
	case r < 0.90: // Rare
		// #nosec G404
		name = rareZones[rand.IntN(len(rareZones))] // #nosec G404
		baseDiff = 1.2
	default: // Legendary
		// #nosec G404
		name = legendaryZones[rand.IntN(len(legendaryZones))] // #nosec G404
		baseDiff = 1.8                                        // Dangerous
	}

	z := Zone{
		Name: name,
	}

	// Scaling Difficulty: harder zones for better players.
	// Since GearScore is now an average (e.g. 50-500), we use a larger multiplier.
	z.Difficulty = baseDiff + (float64(partyAvgLvl) * 0.001) + (partyGearScore * 0.001)

	// Add 1-3 stacking effects (Legendary zones have more)
	// #nosec G404
	effectCount := 1 + rand.IntN(2) // #nosec G404
	if r >= 0.90 {
		effectCount = 3
	}

	// Safety check for empty zone effects
	if len(allZoneEffects) == 0 {
		allZoneEffects = []ZoneEffect{
			{ID: "ZE_DEFAULT", Name: "Mystic Aura", Type: ZoneBuff, Power: 0.1, Description: "Provides +10% to random combat stat."},
		}
	}

	for i := 0; i < effectCount; i++ {
		// #nosec G404
		z.Effects = append(z.Effects, allZoneEffects[rand.IntN(len(allZoneEffects))]) // #nosec G404
	}

	return z
}

func (z Zone) Display() string {
	var effs []string
	for _, e := range z.Effects {
		effs = append(effs, i18n.T("content.zone.effect_display", e.Name, e.Type))
	}
	return i18n.T("content.zone.display", z.Name, z.Difficulty, strings.Join(effs, ", "))
}
