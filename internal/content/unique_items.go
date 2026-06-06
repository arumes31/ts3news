package content

import (
	"fmt"
	"math/rand/v2"
)

// Unique item name generation (50 adjectives × 20 nouns = 1000 combinations)
var uniqueAdjectives = []string{
	"Ancient", "Cursed", "Blessed", "Radiant", "Shadow", "Eternal", "Void", "Celestial", "Infernal", "Frost",
	"Storm", "Earth", "Blood", "Soul", "Spirit", "Divine", "Mythic", "Legendary", "Epic", "Rare",
	"Gleaming", "Dark", "Light", "Holy", "Unholy", "Swift", "Heavy", "Sharp", "Blunt", "Magic",
	"Arcane", "Primal", "Savage", "Noble", "Royal", "Imperial", "Grand", "Mighty", "Fierce", "Wild",
	"Tame", "Silent", "Loud", "Bright", "Dim", "Cold", "Hot", "Burning", "Freezing", "Shattered",
}

var uniqueNouns = []string{
	"Blade", "Shield", "Helm", "Armor", "Boots", "Gloves", "Ring", "Amulet", "Staff", "Bow",
	"Dagger", "Axe", "Mace", "Spear", "Orb", "Tome", "Scroll", "Potion", "Charm", "Relic",
}

// UniqueItem represents a collectible item with a unique name
type UniqueItem struct {
	Name   string
	Rarity Rarity
	Power  float64
}

// GenerateUniqueItemName creates a random unique item name
func GenerateUniqueItemName() string {
	// #nosec G404
	adj := uniqueAdjectives[rand.IntN(len(uniqueAdjectives))] // #nosec G404
	// #nosec G404
	noun := uniqueNouns[rand.IntN(len(uniqueNouns))] // #nosec G404
	return adj + " " + noun
}

// RandomUniqueItem generates a random unique collectible item
func RandomUniqueItem() UniqueItem {
	name := GenerateUniqueItemName()

	// Determine rarity (unique items are inherently rare)
	// #nosec G404
	r := rand.Float64() // #nosec G404
	var rarity Rarity
	switch {
	case r < 0.50:
		rarity = RarityRare
	case r < 0.80:
		rarity = RarityEpic
	case r < 0.95:
		rarity = RarityLegendary
	case r < 0.99:
		rarity = RarityMythic
	default:
		rarity = RarityDivine
	}

	// Power scales with rarity
	rarityMult := float64(rarity+1) * 0.5
	power := 2.0 * rarityMult

	return UniqueItem{
		Name:   name,
		Rarity: rarity,
		Power:  power,
	}
}

// GetUniqueItemDescription returns a formatted description
func GetUniqueItemDescription(item UniqueItem) string {
	return fmt.Sprintf("%s %s (Power: %.1f)", item.Rarity, item.Name, item.Power)
}
