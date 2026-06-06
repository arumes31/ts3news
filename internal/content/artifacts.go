package content

import (
	"fmt"
	"math/rand"
)

type Rarity int

const (
	RarityCommon Rarity = iota
	RarityUncommon
	RarityRare
	RarityEpic
	RarityLegendary
)

func (r Rarity) String() string {
	return []string{"Common", "Uncommon", "Rare", "Epic", "Legendary"}[r]
}

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

func (s Stats) Score() int {
	return s.HP/5 + s.STR + s.DEF + s.SPD + s.LCK
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
	ID            string
	Name          string
	Slot          GearSlot
	Rarity        Rarity
	XPMultiplier  float64
	MaxDurability int
	Stats         Stats
}

type ConsumableType string

const (
	ConsumableHealing ConsumableType = "Healing"
	ConsumableBuff    ConsumableType = "Buff"
)

type Consumable struct {
	ID          string
	Name        string
	Type        ConsumableType
	EffectValue int
	Duration    int // Number of fights
	Description string
}

type Enchantment struct {
	ID          string
	Name        string
	Rarity      Rarity
	Stats       Stats
	Description string
}

var allGear []Gear
var allConsumables = []Consumable{
	{"P1", "Small Health Potion", ConsumableHealing, 50, 0, "Restores 50 HP instantly"},
	{"P2", "Great Health Potion", ConsumableHealing, 200, 0, "Restores 200 HP instantly"},
	{"P3", "Strength Elixir", ConsumableBuff, 15, 3, "+15 STR for 3 fights"},
	{"P4", "Iron Skin Brew", ConsumableBuff, 10, 3, "+10 DEF for 3 fights"},
}

var allEnchantments []Enchantment

// Global pools
var corruptedArtifacts []Artifact
var positiveTitles []Title
var negativeTitles []Title

type Artifact struct {
	Name          string
	Mult          float64
	Stats         Stats
	MaxDurability int
}

type Title struct {
	Name         string
	XPMultiplier float64
	Stats        Stats
}

func init() {
	// 1. Generate Gear
	for _, slot := range AllSlots {
		allGear = append(allGear, Gear{
			ID:            fmt.Sprintf("B_%s", slot),
			Name:          fmt.Sprintf("Novice %s", slot),
			Slot:          slot,
			Rarity:        RarityCommon,
			XPMultiplier:  1.01,
			MaxDurability: 50,
			Stats:         Stats{HP: 10, STR: 2, DEF: 2, SPD: 2},
		})
	}
	// Add Rares
	allGear = append(allGear, Gear{ID: "W_EPIC_1", Name: "Soul-Eater Blade", Slot: SlotMainHand, Rarity: RarityEpic, XPMultiplier: 1.5, MaxDurability: 100, Stats: Stats{STR: 50, SPD: 20}})

	// 2. Generate 100 Corrupted Artifacts
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
				s = Stats{HP: 150, STR: 60, DEF: 30, SPD: 45, LCK: 30}
			} else {
				mult = 0.1 + (rand.Float64() * 0.4)
				s = Stats{HP: -100, STR: -40, DEF: -20, SPD: -20, LCK: -30}
			}
			corruptedArtifacts = append(corruptedArtifacts, Artifact{Name: name, Mult: mult, Stats: s, MaxDurability: 15})
			idx++
		}
	}

	// 3. Titles (Buffed massively)
	posPrefixes := []string{"Divine", "Glorious", "Eternal", "Radiant", "Immortal", "Mythic", "Legendary", "Ancient", "Primal", "Celestial"}
	posNouns := []string{"Sovereign", "Overlord", "Godslayer", "Archon", "Paragon", "Vanguard", "Sentinel", "Oracle", "Exarch", "Titan"}
	for _, p := range posPrefixes {
		for _, n := range posNouns {
			positiveTitles = append(positiveTitles, Title{
				Name:         p + " " + n,
				XPMultiplier: 3.0 + rand.Float64()*7.0, // 3x to 10x
				Stats:        Stats{HP: 500, STR: 200, DEF: 100, SPD: 100, LCK: 80},
			})
		}
	}
	negPrefixes := []string{"Wretched", "Damned", "Forlorn", "Forsaken"}
	negNouns := []string{"Peon", "Outcast", "Traitor", "Coward", "Scum"}
	for _, p := range negPrefixes {
		for _, n := range negNouns {
			negativeTitles = append(negativeTitles, Title{
				Name:         p + " " + n,
				XPMultiplier: 0.01 + rand.Float64()*0.1, // 0.01x to 0.11x
				Stats:        Stats{HP: -300, STR: -150, DEF: -80, SPD: -80, LCK: -100},
			})
		}
	}

	// 4. Generate Enchantments
	enchPrefixes := []string{"Fiery", "Icy", "Shocking", "Venomous", "Holy", "Vampiric", "Arcane", "Stone", "Wind", "Shadow"}
	for i, p := range enchPrefixes {
		rarity := RarityRare
		if i > 6 { rarity = RarityEpic }
		if i > 8 { rarity = RarityLegendary }
		
		allEnchantments = append(allEnchantments, Enchantment{
			ID:          fmt.Sprintf("E%d", i),
			Name:        p,
			Rarity:      rarity,
			Stats:       Stats{STR: 15 * (int(rarity) + 1), SPD: 10 * (int(rarity) + 1)},
			Description: fmt.Sprintf("Adds %s power", p),
		})
	}
}

func RandomConsumable() Consumable { return allConsumables[rand.Intn(len(allConsumables))] }
func RandomGearDrop() Gear         { return allGear[rand.Intn(len(allGear))] }
func RandomStarterGear() Gear      { return allGear[rand.Intn(len(AllSlots))] }
func RandomArtifact() Artifact     { return corruptedArtifacts[rand.Intn(len(corruptedArtifacts))] }
func RandomEnchantment() Enchantment { return allEnchantments[rand.Intn(len(allEnchantments))] }
func RandomTitle() Title {
	if rand.Float64() < 0.8 { return positiveTitles[rand.Intn(len(positiveTitles))] }
	return negativeTitles[rand.Intn(len(negativeTitles))]
}

func GetGearByID(id string) (Gear, bool) {
	for _, g := range allGear {
		if g.ID == id { return g, true }
	}
	return Gear{}, false
}

func GetEnchantmentByID(id string) (Enchantment, bool) {
	for _, e := range allEnchantments {
		if e.ID == id { return e, true }
	}
	return Enchantment{}, false
}

func GetConsumableByID(id string) (Consumable, bool) {
	for _, c := range allConsumables {
		if c.ID == id { return c, true }
	}
	return Consumable{}, false
}

func GetTitleByName(name string) (Title, bool) {
	for _, t := range positiveTitles { if t.Name == name { return t, true } }
	for _, t := range negativeTitles { if t.Name == name { return t, true } }
	return Title{}, false
}

func GetArtifactByName(name string) (Artifact, bool) {
	for _, a := range corruptedArtifacts { if a.Name == name { return a, true } }
	return Artifact{}, false
}

func IsTitle(name string) bool {
	_, ok := GetTitleByName(name)
	return ok
}

func IsGearOrArtifact(name string) bool {
	for _, g := range allGear { if g.Name == name { return true } }
	for _, a := range corruptedArtifacts { if a.Name == name { return true } }
	return false
}
