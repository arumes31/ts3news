package content

import (
	"fmt"
	"math/rand"
	"time"
)

type GearSlot string

const (
	SlotWeapon GearSlot = "Weapon"
	SlotArmor  GearSlot = "Armor"
	SlotRelic  GearSlot = "Relic"
)

type Gear struct {
	ID          string
	Name        string
	Slot        GearSlot
	XPMultiplier float64
	Duration    time.Duration
	Description string
}

type Artifact struct {
	Name string
	Mult float64
}

func (a Artifact) IsBoon() bool {
	return a.Mult > 1.0
}

func (a Artifact) Effect() string {
	if a.Mult > 1.0 {
		return fmt.Sprintf("+%.0f%%", (a.Mult-1.0)*100)
	}
	return fmt.Sprintf("-%.0f%%", (1.0-a.Mult)*100)
}

var allGear = []Gear{
	// Weapons (Small buffs, 3 days)
	{"W1", "Rusty Broadsword", SlotWeapon, 1.05, 72 * time.Hour, "+5% XP"},
	{"W2", "Silver Dagger", SlotWeapon, 1.10, 72 * time.Hour, "+10% XP"},
	{"W3", "Golden Halberd", SlotWeapon, 1.15, 72 * time.Hour, "+15% XP"},
	{"W4", "Void Bow", SlotWeapon, 1.20, 72 * time.Hour, "+20% XP"},
	{"W5", "Cursed Blade", SlotWeapon, 0.90, 72 * time.Hour, "-10% XP"},

	// Armor (Medium buffs, 5 days)
	{"A1", "Leather Tunic", SlotArmor, 1.10, 120 * time.Hour, "+10% XP"},
	{"A2", "Iron Plate", SlotArmor, 1.15, 120 * time.Hour, "+15% XP"},
	{"A3", "Mithril Chainmail", SlotArmor, 1.25, 120 * time.Hour, "+25% XP"},
	{"A4", "Abyssal Cloak", SlotArmor, 1.30, 120 * time.Hour, "+30% XP"},
	{"A5", "Heavy Shackles", SlotArmor, 0.80, 120 * time.Hour, "-20% XP"},

	// Relics (Utility, 7 days)
	{"R1", "Lucky Rabbit's Foot", SlotRelic, 1.15, 168 * time.Hour, "+15% XP"},
	{"R2", "Amulet of Haste", SlotRelic, 1.20, 168 * time.Hour, "+20% XP"},
	{"R3", "Crown of the Sovereign", SlotRelic, 1.50, 168 * time.Hour, "+50% XP"},
	{"R4", "Broken Compass", SlotRelic, 0.75, 168 * time.Hour, "-25% XP"},
}

var corruptedArtifacts []Artifact

func init() {
	prefixes := []string{"Cursed", "Blighted", "Tainted", "Demonic", "Shadow", "Void", "Ruined", "Shattered", "Forbidden", "Malevolent"}
	nouns := []string{"Chalice", "Orb", "Scepter", "Tome", "Crown", "Amulet", "Skull", "Idol", "Heart", "Eye"}
	
	idx := 1
	for _, p := range prefixes {
		for _, n := range nouns {
			name := p + " " + n
			var mult float64
			if idx%2 == 0 {
				mult = 1.5 + (rand.Float64() * 2.5) // 1.5x to 4x
			} else {
				mult = 0.1 + (rand.Float64() * 0.4) // 0.1x to 0.5x
			}
			corruptedArtifacts = append(corruptedArtifacts, Artifact{
				Name: name,
				Mult: mult,
			})
			idx++
		}
	}
}

func RandomArtifact() Artifact {
	return corruptedArtifacts[rand.Intn(len(corruptedArtifacts))]
}

func RandomGearDrop() Gear {
	return allGear[rand.Intn(len(allGear))]
}

func GetGearByID(id string) (Gear, bool) {
	for _, g := range allGear {
		if g.ID == id {
			return g, true
		}
	}
	return Gear{}, false
}
