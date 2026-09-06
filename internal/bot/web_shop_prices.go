package bot

import (
	"math"

	"ts3news/internal/content"
)

// shopGearPrice is shared by regular, buff-promoted and featured offers.
// Premium rarity prices are shop-only: gearPrice still determines resale value.
func shopGearPrice(g content.Gear) int64 {
	var floor int64
	switch g.Rarity {
	case content.RarityMythic:
		floor = 3_000_000
	case content.RarityDivine:
		floor = 6_000_000
	case content.RarityCelestial:
		floor = 8_000_000
	case content.RarityEternal:
		floor = 10_000_000
	default:
		return gearPrice(g)
	}

	// Keep the showcase's combat-power premium for every Mythic+ offer.
	// Check before conversion/addition so endless buffs cannot wrap into a discount.
	premium := max(g.CombatRating()*1000+float64(g.Stats.Score())*500, 0)
	if premium >= float64(math.MaxInt64-floor) {
		return math.MaxInt64
	}
	return max(gearPrice(g), floor+int64(premium))
}
