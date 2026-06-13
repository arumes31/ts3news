package content

import (
	"math/rand/v2"
	"ts3news/internal/i18n"
)

// Unique item name generation (50 adjectives × 20 nouns = 1000 combinations)
var uniqueAdjectives []string
var uniqueNouns []string
var uniqueItemsInitialized bool

func initUniqueItems() {
	if uniqueItemsInitialized {
		return
	}
	uniqueItemsInitialized = true
	uniqueAdjectives = i18n.Pool("pool.unique.adjective")
	uniqueNouns = i18n.Pool("pool.unique.noun")
}

// UniqueItem represents a collectible item with a unique name
type UniqueItem struct {
	Name   string
	Rarity Rarity
	Power  float64
}

// GenerateUniqueItemName creates a random unique item name
func GenerateUniqueItemName() string {
	initUniqueItems()
	adjectives := uniqueAdjectives
	nouns := uniqueNouns

	// Safety check for empty pools (can happen during init before i18n is fully loaded)
	if len(adjectives) == 0 {
		adjectives = []string{"Ancient", "Cursed", "Blessed", "Radiant", "Shadow"}
	}
	if len(nouns) == 0 {
		nouns = []string{"Blade", "Shield", "Helm", "Armor", "Boots"}
	}

	// #nosec G404
	adj := adjectives[rand.IntN(len(adjectives))] // #nosec G404
	// #nosec G404
	noun := nouns[rand.IntN(len(nouns))] // #nosec G404
	return adj + " " + noun
}

// RandomUniqueItem generates a random unique collectible item
func RandomUniqueItem() UniqueItem {
	initUniqueItems()
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
	return i18n.T("content.unique_item.description", i18n.R(int(item.Rarity)), item.Name, item.Power)
}
