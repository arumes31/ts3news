package content

import (
	"fmt"
	"math/rand"
	"time"
)

type GearSlot string

const (
	SlotHead      GearSlot = "Head"
	SlotNeck      GearSlot = "Neck"
	SlotShoulders GearSlot = "Shoulders"
	SlotBack      GearSlot = "Back"
	SlotChest     GearSlot = "Chest"
	SlotWrists    GearSlot = "Wrists"
	SlotHands     GearSlot = "Hands"
	SlotWaist     GearSlot = "Waist"
	SlotLegs      GearSlot = "Legs"
	SlotFeet      GearSlot = "Feet"
	SlotFinger1   GearSlot = "Finger1"
	SlotFinger2   GearSlot = "Finger2"
	SlotTrinket1  GearSlot = "Trinket1"
	SlotTrinket2  GearSlot = "Trinket2"
	SlotMainHand  GearSlot = "MainHand"
	SlotOffHand   GearSlot = "OffHand"
	SlotRanged    GearSlot = "Ranged"
	SlotRelic     GearSlot = "Relic"
	SlotArtifact  GearSlot = "Artifact"
	SlotSoul      GearSlot = "Soul"
	SlotAura      GearSlot = "Aura"
	SlotCharm     GearSlot = "Charm"
	SlotMount     GearSlot = "Mount"
	SlotCompanion GearSlot = "Companion"
)

var AllSlots = []GearSlot{
	SlotHead, SlotNeck, SlotShoulders, SlotBack, SlotChest, SlotWrists,
	SlotHands, SlotWaist, SlotLegs, SlotFeet, SlotFinger1, SlotFinger2,
	SlotTrinket1, SlotTrinket2, SlotMainHand, SlotOffHand, SlotRanged,
	SlotRelic, SlotArtifact, SlotSoul, SlotAura, SlotCharm, SlotMount, SlotCompanion,
}

type Gear struct {
	ID           string
	Name         string
	Slot         GearSlot
	XPMultiplier float64
	Duration     time.Duration
	Description  string
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

type Title struct {
	Name         string
	XPMultiplier float64
}

var allGear []Gear
var corruptedArtifacts []Artifact
var positiveTitles []Title
var negativeTitles []Title

func init() {
	// Generate base gear for each slot
	for _, slot := range AllSlots {
		allGear = append(allGear, Gear{
			ID:           fmt.Sprintf("B_%s", slot),
			Name:         fmt.Sprintf("Novice %s", slot),
			Slot:         slot,
			XPMultiplier: 1.01,
			Duration:     168 * time.Hour, // 7 days
			Description:  "+1% XP",
		})
	}

	// Add more interesting gear
	allGear = append(allGear, []Gear{
		{"W1", "Rusty Broadsword", SlotMainHand, 1.05, 72 * time.Hour, "+5% XP"},
		{"W2", "Silver Dagger", SlotMainHand, 1.10, 72 * time.Hour, "+10% XP"},
		{"A1", "Leather Tunic", SlotChest, 1.10, 120 * time.Hour, "+10% XP"},
		{"R1", "Lucky Rabbit's Foot", SlotRelic, 1.15, 168 * time.Hour, "+15% XP"},
	}...)

	// Generate 100 unique corrupted artifacts
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

	// Generate 100 Positive Titles
	posPrefixes := []string{"Divine", "Glorious", "Eternal", "Radiant", "Immortal", "Mythic", "Legendary", "Ancient", "Primal", "Celestial"}
	posNouns := []string{"Sovereign", "Overlord", "Godslayer", "Archon", "Paragon", "Vanguard", "Sentinel", "Oracle", "Exarch", "Titan"}
	for _, p := range posPrefixes {
		for _, n := range posNouns {
			mult := 2.0 + (rand.Float64() * 3.0) // 2x to 5x
			positiveTitles = append(positiveTitles, Title{Name: p + " " + n, XPMultiplier: mult})
		}
	}

	// Generate 20 Negative Titles
	negPrefixes := []string{"Wretched", "Damned", "Forlorn", "Forsaken"}
	negNouns := []string{"Peon", "Outcast", "Traitor", "Coward", "Scum"}
	for _, p := range negPrefixes {
		for _, n := range negNouns {
			mult := 0.05 + (rand.Float64() * 0.2) // 0.05x to 0.25x
			negativeTitles = append(negativeTitles, Title{Name: p + " " + n, XPMultiplier: mult})
		}
	}
}

func RandomArtifact() Artifact {
	return corruptedArtifacts[rand.Intn(len(corruptedArtifacts))]
}

func RandomGearDrop() Gear {
	return allGear[rand.Intn(len(allGear))]
}

func RandomStarterGear() Gear {
	// Pick a random novice gear
	return allGear[rand.Intn(len(AllSlots))]
}

func RandomTitle() Title {
	if rand.Float64() < 0.8 {
		return positiveTitles[rand.Intn(len(positiveTitles))]
	}
	return negativeTitles[rand.Intn(len(negativeTitles))]
}

func IsTitle(name string) bool {
	for _, t := range positiveTitles {
		if t.Name == name {
			return true
		}
	}
	for _, t := range negativeTitles {
		if t.Name == name {
			return true
		}
	}
	return false
}

func GetGearByID(id string) (Gear, bool) {
	for _, g := range allGear {
		if g.ID == id {
			return g, true
		}
	}
	return Gear{}, false
}
