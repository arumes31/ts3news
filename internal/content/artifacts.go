package content

import (
	"fmt"
	"math/rand"
	"time"
)

type Stats struct {
	HP  int
	STR int
	DEF int
	SPD int
	LCK int
}

func (s Stats) Add(o Stats) Stats {
	return Stats{
		HP:  s.HP + o.HP,
		STR: s.STR + o.STR,
		DEF: s.DEF + o.DEF,
		SPD: s.SPD + o.SPD,
		LCK: s.LCK + o.LCK,
	}
}

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
	Stats        Stats
}

type Artifact struct {
	Name  string
	Mult  float64
	Stats Stats
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
	Stats        Stats
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
			Duration:     168 * time.Hour,
			Description:  "+1% XP",
			Stats:        Stats{HP: 5, STR: 1, DEF: 1, SPD: 1, LCK: 0},
		})
	}

	// Add 100 Corrupted Artifacts with Stats
	prefixes := []string{"Cursed", "Blighted", "Tainted", "Demonic", "Shadow", "Void", "Ruined", "Shattered", "Forbidden", "Malevolent"}
	nouns := []string{"Chalice", "Orb", "Scepter", "Tome", "Crown", "Amulet", "Skull", "Idol", "Heart", "Eye"}
	idx := 1
	for _, p := range prefixes {
		for _, n := range nouns {
			name := p + " " + n
			var mult float64
			var s Stats
			if idx%2 == 0 {
				mult = 1.5 + (rand.Float64() * 2.5)
				s = Stats{HP: 50, STR: 20, DEF: 10, SPD: 15, LCK: 10}
			} else {
				mult = 0.1 + (rand.Float64() * 0.4)
				s = Stats{HP: -20, STR: -10, DEF: -5, SPD: -5, LCK: -10}
			}
			corruptedArtifacts = append(corruptedArtifacts, Artifact{Name: name, Mult: mult, Stats: s})
			idx++
		}
	}

	// Generate 100 Positive Titles with Stats
	posPrefixes := []string{"Divine", "Glorious", "Eternal", "Radiant", "Immortal", "Mythic", "Legendary", "Ancient", "Primal", "Celestial"}
	posNouns := []string{"Sovereign", "Overlord", "Godslayer", "Archon", "Paragon", "Vanguard", "Sentinel", "Oracle", "Exarch", "Titan"}
	for _, p := range posPrefixes {
		for _, n := range posNouns {
			mult := 2.0 + (rand.Float64() * 3.0)
			s := Stats{HP: 100, STR: 50, DEF: 30, SPD: 25, LCK: 20}
			positiveTitles = append(positiveTitles, Title{Name: p + " " + n, XPMultiplier: mult, Stats: s})
		}
	}

	// Generate 20 Negative Titles with Stats
	negPrefixes := []string{"Wretched", "Damned", "Forlorn", "Forsaken"}
	negNouns := []string{"Peon", "Outcast", "Traitor", "Coward", "Scum"}
	for _, p := range negPrefixes {
		for _, n := range negNouns {
			mult := 0.05 + (rand.Float64() * 0.2)
			s := Stats{HP: -50, STR: -20, DEF: -15, SPD: -10, LCK: -15}
			negativeTitles = append(negativeTitles, Title{Name: p + " " + n, XPMultiplier: mult, Stats: s})
		}
	}
}

func RandomArtifact() Artifact { return corruptedArtifacts[rand.Intn(len(corruptedArtifacts))] }
func RandomGearDrop() Gear     { return allGear[rand.Intn(len(allGear))] }
func RandomStarterGear() Gear  { return allGear[rand.Intn(len(AllSlots))] }
func RandomTitle() Title {
	if rand.Float64() < 0.8 {
		return positiveTitles[rand.Intn(len(positiveTitles))]
	}
	return negativeTitles[rand.Intn(len(negativeTitles))]
}

func IsTitle(name string) bool {
	for _, t := range positiveTitles {
		if t.Name == name { return true }
	}
	for _, t := range negativeTitles {
		if t.Name == name { return true }
	}
	return false
}

func GetGearByID(id string) (Gear, bool) {
	for _, g := range allGear {
		if g.ID == id { return g, true }
	}
	return Gear{}, false
}

func GetTitleByName(name string) (Title, bool) {
	for _, t := range positiveTitles {
		if t.Name == name { return t, true }
	}
	for _, t := range negativeTitles {
		if t.Name == name { return t, true }
	}
	return Title{}, false
}
