package content

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"ts3news/internal/i18n"
)

type Rarity int

const (
	RarityCommon Rarity = iota
	RarityUncommon
	RarityRare
	RarityEpic
	RarityLegendary
	RarityMythic
	RarityDivine
)

func (r Rarity) String() string {
	list := []string{i18n.T("rarity.common"), i18n.T("rarity.uncommon"), i18n.T("rarity.rare"), i18n.T("rarity.epic"), i18n.T("rarity.legendary"), i18n.T("rarity.mythic"), i18n.T("rarity.divine")}
	if int(r) < 0 || int(r) >= len(list) {
		return i18n.T("rarity.unknown", r)
	}
	return list[r]
}

// Color returns a BBCode color string for this rarity
func (r Rarity) Color() string {
	colors := []string{
		"#b0bec5", // Common (Gray)
		"#4caf50", // Uncommon (Green)
		"#2196f3", // Rare (Blue)
		"#9c27b0", // Epic (Purple)
		"#ff9800", // Legendary (Orange)
		"#f44336", // Mythic (Red)
		"#ffeb3b", // Divine (Gold)
	}
	if int(r) < 0 || int(r) >= len(colors) {
		return "#ffffff"
	}
	return colors[r]
}

type Stats struct {
	// Combat Stats
	HP  int
	STR int
	DEF int
	SPD int
	LCK int
	INT int // Intelligence (boosts XP slightly)
	STA int // Stamina (reduces durability loss chance)
	CRT int // Critical Chance %
	DGE int // Dodge Chance %
	MNA int // Mana

	// Useless / Flavour Stats
	CHA int // Charisma
	STN int // Stench
	SHN int // Shiny
	HGR int // Hunger
}

func (s Stats) Add(o Stats) Stats {
	return Stats{
		HP:  s.HP + o.HP,
		STR: s.STR + o.STR,
		DEF: s.DEF + o.DEF,
		SPD: s.SPD + o.SPD,
		LCK: s.LCK + o.LCK,
		INT: s.INT + o.INT,
		STA: s.STA + o.STA,
		CRT: s.CRT + o.CRT,
		DGE: s.DGE + o.DGE,
		MNA: s.MNA + o.MNA,
		CHA: s.CHA + o.CHA,
		STN: s.STN + o.STN,
		SHN: s.SHN + o.SHN,
		HGR: s.HGR + o.HGR,
	}
}

func (s Stats) Score() int {
	return s.HP/5 + s.STR + s.DEF + s.SPD + s.LCK + s.INT + s.STA + s.CRT + s.DGE + s.MNA/10
}

// CombatRating calculates a comprehensive Combat Rating (CR) for gear.
// Each stat is weighted by its combat effectiveness, then multiplied by a rarity factor.
func (g Gear) CombatRating() float64 {
	// Weight stats by their combat impact
	cr := float64(g.Stats.STR)*1.2 + // Direct damage dealer
		float64(g.Stats.DEF)*0.9 + // Survivability
		float64(g.Stats.HP)*0.3 + // Health pool (lower per-point value)
		float64(g.Stats.SPD)*1.1 + // Speed: dodge + attack speed
		float64(g.Stats.CRT)*1.5 + // Crit chance: high damage multiplier
		float64(g.Stats.DGE)*1.3 + // Dodge: pure avoidance
		float64(g.Stats.LCK)*0.8 + // Luck: loot + misc bonuses
		float64(g.Stats.INT)*0.7 + // Intelligence: XP + spell power
		float64(g.Stats.STA)*0.6 // Stamina: durability reduction

	// Rarity multiplier: ensures higher rarity items generally have higher CR
	rarityMult := map[Rarity]float64{
		RarityCommon:    1.0,
		RarityUncommon:  1.25,
		RarityRare:      1.55,
		RarityEpic:      1.9,
		RarityLegendary: 2.3,
		RarityMythic:    2.8,
	}
	if mult, ok := rarityMult[g.Rarity]; ok {
		cr *= mult
	}

	return math.Round(cr*10) / 10 // Round to 1 decimal
}

// Scaled multiplies the combat stats by f (flavour stats left unchanged). Used
// for the permanent per-prestige stat bonus.
func (s Stats) Scaled(f float64) Stats {
	return Stats{
		HP:  int(float64(s.HP) * f),
		STR: int(float64(s.STR) * f),
		DEF: int(float64(s.DEF) * f),
		SPD: int(float64(s.SPD) * f),
		LCK: int(float64(s.LCK) * f),
		INT: int(float64(s.INT) * f),
		STA: int(float64(s.STA) * f),
		CRT: int(float64(s.CRT) * f),
		DGE: int(float64(s.DGE) * f),
		MNA: int(float64(s.MNA) * f),
		CHA: s.CHA, STN: s.STN, SHN: s.SHN, HGR: s.HGR,
	}
}

// UserInCombat represents a user in combat
type UserInCombat struct {
	UID           string
	Nickname      string
	CLID          int
	Level         int
	Stats         Stats
	Skills        []Skill
	UltimateSkill *UltimateSkill
	CurrentHP     int
	RegenStacks   int
	Gold          int64
	Pets          []*Mob
	Equipped      map[GearSlot]Gear
	STRMod        float64
	DEFMod        float64
	SPDMod        float64
	LootFocus     string
	FloorModifier string
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
	SlotPet1      GearSlot = "Pet1"
	SlotPet2      GearSlot = "Pet2"
	SlotEmblem1   GearSlot = "Emblem1"
	SlotEmblem2   GearSlot = "Emblem2"
	SlotBanner    GearSlot = "Banner"
	SlotTotem     GearSlot = "Totem"
)

var AllSlots = []GearSlot{
	SlotHead, SlotNeck, SlotShoulders, SlotBack, SlotChest, SlotWrists,
	SlotHands, SlotWaist, SlotLegs, SlotFeet, SlotFinger1, SlotFinger2,
	SlotTrinket1, SlotTrinket2, SlotMainHand, SlotOffHand, SlotRanged,
	SlotRelic, SlotArtifact, SlotSoul, SlotAura, SlotCharm, SlotMount, SlotCompanion,
	SlotPet1, SlotPet2, SlotEmblem1, SlotEmblem2, SlotBanner, SlotTotem,
}

// slotIcons maps each equipment slot to a distinct, slot-appropriate emoji used
// across the web portal (armoury, inventory, shop, auction house).
var slotIcons = map[GearSlot]string{
	SlotHead: "🪖", SlotNeck: "📿", SlotShoulders: "🧥", SlotBack: "🧣",
	SlotChest: "🛡️", SlotWrists: "⌚", SlotHands: "🧤", SlotWaist: "🎗️",
	SlotLegs: "👖", SlotFeet: "🥾", SlotFinger1: "💍", SlotFinger2: "💍",
	SlotTrinket1: "🔱", SlotTrinket2: "🔮", SlotMainHand: "⚔️", SlotOffHand: "🗡️",
	SlotRanged: "🏹", SlotRelic: "🏺", SlotArtifact: "🗿", SlotSoul: "👻",
	SlotAura: "✨", SlotCharm: "🍀", SlotMount: "🐎", SlotCompanion: "🐕",
	SlotPet1: "🐉", SlotPet2: "🦅", SlotEmblem1: "🎖️", SlotEmblem2: "🏅",
	SlotBanner: "🚩", SlotTotem: "🪶",
}

// SlotIcon returns the emoji icon for an equipment slot (a generic gem if the
// slot is unknown).
func SlotIcon(slot GearSlot) string {
	if ic, ok := slotIcons[slot]; ok {
		return ic
	}
	return "💎"
}

type ItemEffect string

const (
	EffectNone           ItemEffect = ""
	EffectThorns         ItemEffect = "Thorns"         // Reflect 10% damage
	EffectVampiric       ItemEffect = "Vampiric"       // 5% passive lifesteal
	EffectBerserk        ItemEffect = "Berserk"        // +20% STR when HP < 50%
	EffectLucky          ItemEffect = "Lucky"          // +10% LCK
	EffectTreasureHunter ItemEffect = "TreasureHunter" // +5% item find chance
	EffectQuick          ItemEffect = "Quick"          // +10% SPD
	EffectBulwark        ItemEffect = "Bulwark"        // +10% DEF
	EffectRadiant        ItemEffect = "Radiant"        // +10% XP bonus
	EffectFragile        ItemEffect = "Fragile"        // +30% STR but double durability loss
	EffectSteady         ItemEffect = "Steady"         // Reduces stun chance by 50%
	EffectMindControl    ItemEffect = "MindControl"    // Chance to capture low-health mobs
	EffectRegenStack     ItemEffect = "RegenStack"     // Adds permanent regen stack on victory
	EffectPhoenix        ItemEffect = "Phoenix"        // Revive once per fight with 50% HP
	EffectStealth        ItemEffect = "Stealth"        // Skip first round mob damage
	EffectParry          ItemEffect = "Parry"          // 10% chance to take 0 damage and counter for 50%
	EffectCleanse        ItemEffect = "Cleanse"        // Remove one negative effect/hazard at start of turn
)

type Element string

const (
	ElementPhysical Element = "Physical"
	ElementFire     Element = "Fire"
	ElementWater    Element = "Water"
	ElementEarth    Element = "Earth"
	ElementAir      Element = "Air"
)

type Position string

const (
	PositionFrontline Position = "Frontline"
	PositionBackline  Position = "Backline"
)

type Gear struct {
	ID            string
	Name          string
	Slot          GearSlot
	Rarity        Rarity
	XPMultiplier  float64
	MaxDurability int
	Stats         Stats
	Special       ItemEffect
	Element       Element

	// Custom progression fields
	Sockets      int      `json:"sockets,omitempty"`
	Gemstones    []string `json:"gemstones,omitempty"`
	Rune         string   `json:"rune,omitempty"`
	Cursed       bool     `json:"cursed,omitempty"`
	Eldritch     bool     `json:"eldritch,omitempty"`
	Unidentified bool     `json:"unidentified,omitempty"`
	GearLevel    int      `json:"gear_level,omitempty"`
	Insured      bool     `json:"insured,omitempty"`
}

type ConsumableType string

const (
	ConsumableHealing ConsumableType = "Healing"
	ConsumableRevive  ConsumableType = "Revive"
	ConsumableBuff    ConsumableType = "Buff"
	ConsumableRepair  ConsumableType = "Repair"
)

type Consumable struct {
	ID          string
	Name        string
	Type        ConsumableType
	EffectValue float64 // Changed to float64 for % scaling
	Duration    int     // Number of fights
	Description string
}

type Enchantment struct {
	ID           string
	Name         string
	Rarity       Rarity
	XPMultiplier float64
	Stats        Stats
	DuraBonus    int
	Description  string
	Special      ItemEffect
}

var allGear []Gear
var starterGear []Gear
var uniqueLegendaries []Gear
var abyssExclusiveGear []Gear
var allConsumables []Consumable

// buildConsumables (re)builds the consumable table. Names are intentionally
// literal English: the matching logic in hazards.go keys off English substrings
// (e.g. "antidote", "warmth"), and the content.consumable.* translation keys do
// not exist, so routing through i18n.T would only leak raw keys.
func buildConsumables() []Consumable {
	return []Consumable{
		{"small_health_potion", "Small Health Potion", ConsumableHealing, 50, 0, "Restores a small amount of HP in battle."},
		{"great_health_potion", "Great Health Potion", ConsumableHealing, 200, 0, "Restores a large amount of HP in battle."},
		{"strength_elixir", "Strength Elixir", ConsumableBuff, 15, 3, "Boosts Strength for several fights."},
		{"iron_skin_brew", "Iron Skin Brew", ConsumableBuff, 10, 3, "Boosts Defense for several fights."},
		{"phoenix_feather", "Phoenix Feather", ConsumableRevive, 50, 0, "Revives you once when you fall in battle."},
		{"repair_kit", "Repair Kit", ConsumableRepair, 30, 0, "Restores durability to your equipment."},
		{"master_repair_kit", "Master Repair Kit", ConsumableRepair, 75, 0, "Fully restores durability to your equipment."},
		{"speed_elixir", "Speed Elixir", ConsumableBuff, 25, 3, "Boosts Speed by +25 for several fights."},
		{"intellect_elixir", "Intellect Elixir", ConsumableBuff, 20, 3, "Boosts Intellect by +20 for several fights."},
		{"lucky_draught", "Lucky Draught", ConsumableBuff, 20, 3, "Boosts Luck by +20 for several fights."},
		{"giant_strength_elixir", "Giant Strength Elixir", ConsumableBuff, 40, 3, "Massively boosts Strength for several fights."},
		{"rejuvenation_potion", "Rejuvenation Potion", ConsumableHealing, 0.6, 0, "Restores 60% of Max HP."},
		{"elixir_of_life", "Elixir of Life", ConsumableHealing, 1.0, 0, "Fully restores your health (100%)."},
	}
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
	Special       ItemEffect
}

func (a Artifact) IsBoon() bool {
	return a.Mult > 1.0
}

func (a Artifact) XPBonusDesc() string {
	if a.Mult > 1.0 {
		return i18n.T("content.artifact.format.xp_bonus_desc_positive", (a.Mult-1.0)*100)
	}
	return i18n.T("content.artifact.format.xp_bonus_desc_negative", (1.0-a.Mult)*100)
}

func (a Artifact) Score() int {
	return a.Stats.Score() + int(a.Mult*100)
}

type Title struct {
	Name         string
	XPMultiplier float64
	Stats        Stats
	ExtraSkills  int  // +X more skill slots
	Lifesteal    int  // % of damage dealt healed
	MultiStrike  int  // % chance to hit twice
	DoubleLoot   bool // Chance to double all mob drops
}

func (t Title) Score() int {
	score := t.Stats.Score() + int(t.XPMultiplier*100)
	score += t.ExtraSkills * 500
	score += t.Lifesteal * 50
	score += t.MultiStrike * 30
	if t.DoubleLoot {
		score += 2000
	}
	return score
}

func init() {
	buildContent()
}

// InitLocalized rebuilds all i18n-dependent content (gear, consumables, titles,
// artifacts, enchantments) so their names resolve to the active locale. It must
// be called once after i18n.InitWithLocale; otherwise the names baked at package
// init time (when i18n is not yet loaded) leak raw translation keys such as
// "content.gear.novice". Gear IDs are seeded deterministically, so rebuilding
// keeps IDs stable while refreshing the display names. Safe to call repeatedly.
func InitLocalized() {
	buildContent()
}

func buildContent() {
	// Reset so repeated calls (init + InitLocalized) don't duplicate entries.
	allGear = nil
	starterGear = nil
	uniqueLegendaries = nil
	corruptedArtifacts = nil
	positiveTitles = nil
	negativeTitles = nil
	allEnchantments = nil
	allConsumables = buildConsumables()

	// Use a fixed seed for procedural generation to ensure Gear IDs (G1, G2...)
	// are stable across bot restarts/rebuilds.
	r := rand.New(rand.NewPCG(42, 42)) // #nosec G404

	// XP Multiplier Logic:
	// - Novice/Common items: NO XP bonus (1.0x)
	// - Uncommon: +5% XP (1.05x)
	// - Rare: +10% XP (1.10x)
	// - Epic: +20% XP (1.20x)
	// - Legendary: +30% XP (1.30x) - maximum
	getXPMult := func(rar Rarity) float64 {
		switch rar {
		case RarityCommon:
			return 1.0 // No XP bonus for basic items
		case RarityUncommon:
			return 1.05 // +5% XP
		case RarityRare:
			return 1.10 // +10% XP
		case RarityEpic:
			return 1.20 // +20% XP
		case RarityLegendary:
			return 1.30 // +30% XP (max)
		default:
			return 1.0
		}
	}

	// Pools for procedural generation
	prefixes := i18n.Pool("pool.prefix")
	suffixes := i18n.Pool("pool.suffix")

	// Safety check for empty pools (can happen during init before i18n is fully loaded)
	if len(prefixes) == 0 {
		prefixes = []string{"Ancient", "Eternal", "Celestial"}
	}
	if len(suffixes) == 0 {
		suffixes = []string{"of Power", "of Wisdom", "of Valor"}
	}

	// 1. Generate ~1200 unique gear variants
	idx := 1
	for _, slot := range AllSlots {
		// Starter Novice gear
		noviceGear := Gear{
			ID:            fmt.Sprintf("B_%s", slot),
			Name:          i18n.T("content.gear.novice", i18n.T("content.gear.slot."+strings.ToLower(string(slot)))),
			Slot:          slot,
			Rarity:        RarityCommon,
			XPMultiplier:  getXPMult(RarityCommon),
			MaxDurability: 50,
			Stats:         Stats{HP: 10, STR: 2, DEF: 2, SPD: 2, CHA: 1, STN: r.IntN(5)},
		}
		allGear = append(allGear, noviceGear)
		starterGear = append(starterGear, noviceGear)

		// Procedural variants
		for _, rar := range []Rarity{RarityUncommon, RarityRare, RarityEpic, RarityLegendary} {
			for i := 0; i < 10; i++ { // 24 slots * 4 rarities * 10 variants = 960 items
				p := prefixes[r.IntN(len(prefixes))]
				s := suffixes[r.IntN(len(suffixes))]
				name := i18n.T("content.gear.procedural", p, i18n.T("content.gear.slot."+strings.ToLower(string(slot))), s)

				mul := float64(rar) + 1.0
				allGear = append(allGear, Gear{
					ID:            fmt.Sprintf("G%d", idx),
					Name:          name,
					Slot:          slot,
					Rarity:        rar,
					XPMultiplier:  getXPMult(rar),
					MaxDurability: 50 + int(rar)*30,
					Stats: Stats{
						HP:  int(10 * mul),
						STR: int(5 * mul),
						DEF: int(3 * mul),
						SPD: int(4 * mul),
						LCK: int(2 * mul),
						INT: int(rar),
						STA: int(rar),
						CHA: r.IntN(10),
						SHN: r.IntN(20),
					},
				})
				idx++
			}
		}
	}

	// Two solid "Novice+" starter items every new player is equipped with, so the
	// early game is not painfully weak (better than Novice, far below endgame gear).
	allGear = append(allGear,
		Gear{ID: "B_GOOD_1", Name: i18n.T("content.gear.trusty_longsword"), Slot: SlotMainHand, Rarity: RarityUncommon, XPMultiplier: getXPMult(RarityUncommon), MaxDurability: 120, Stats: Stats{HP: 25, STR: 14, DEF: 4, SPD: 7, CRT: 5, LCK: 3}},
		Gear{ID: "B_GOOD_2", Name: i18n.T("content.gear.reinforced_breastplate"), Slot: SlotChest, Rarity: RarityUncommon, XPMultiplier: getXPMult(RarityUncommon), MaxDurability: 150, Stats: Stats{HP: 70, STR: 4, DEF: 20, SPD: 2, STA: 6}},
	)

	// Add some Unique Legendaries with massive stats but very low durability
	uniqueLegendaries = append(uniqueLegendaries, []Gear{
		{ID: "U_LEG_1", Name: i18n.T("content.gear.god_slayers_heart"), Slot: SlotChest, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 5, Stats: Stats{HP: 1000, STR: 500, DEF: 200, SPD: 200, LCK: 100}},
		{ID: "U_LEG_2", Name: i18n.T("content.gear.infinity_edge"), Slot: SlotMainHand, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 8, Stats: Stats{STR: 1000, SPD: 300, CRT: 50}},
		{ID: "U_LEG_3", Name: i18n.T("content.gear.chrono_guard"), Slot: SlotWrists, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 6, Stats: Stats{SPD: 500, DGE: 80, INT: 100}},
		{ID: "U_LEG_4", Name: i18n.T("content.gear.eye_of_the_storm"), Slot: SlotHead, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 7, Stats: Stats{HP: 800, INT: 200, STR: 100}},
		{ID: "U_LEG_5", Name: i18n.T("content.gear.titans_pillar"), Slot: SlotLegs, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 10, Stats: Stats{HP: 1500, DEF: 400, STA: 100}},
	}...)

	abyssExclusiveGear = []Gear{
		{ID: "ABYSS_WEAPON", Name: i18n.T("content.gear.abyss_weapon"), Slot: SlotMainHand, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 100, Stats: Stats{HP: 150, STR: 80, DEF: 20, SPD: 40, CRT: 12, LCK: 10}},
		{ID: "ABYSS_CHEST", Name: i18n.T("content.gear.abyss_chest"), Slot: SlotChest, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 120, Stats: Stats{HP: 300, STR: 30, DEF: 80, SPD: 10, STA: 20}},
		{ID: "ABYSS_HEAD", Name: i18n.T("content.gear.abyss_head"), Slot: SlotHead, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 90, Stats: Stats{HP: 120, STR: 25, DEF: 35, SPD: 15, INT: 30}},
		{ID: "ABYSS_LEGS", Name: i18n.T("content.gear.abyss_legs"), Slot: SlotLegs, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 110, Stats: Stats{HP: 200, STR: 20, DEF: 50, SPD: 20, STA: 15}},
		{ID: "ABYSS_HANDS", Name: i18n.T("content.gear.abyss_hands"), Slot: SlotHands, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 100, STR: 40, DEF: 15, SPD: 25, CRT: 8}},
		{ID: "ABYSS_FEET", Name: i18n.T("content.gear.abyss_feet"), Slot: SlotFeet, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 85, Stats: Stats{HP: 90, STR: 15, DEF: 20, SPD: 50, DGE: 10}},
		{ID: "ABYSS_RING", Name: i18n.T("content.gear.abyss_ring"), Slot: SlotFinger1, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 80, STR: 20, DEF: 10, SPD: 20, LCK: 25}},
		{ID: "ABYSS_NECK", Name: i18n.T("content.gear.abyss_neck"), Slot: SlotNeck, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 110, STR: 10, DEF: 15, SPD: 15, INT: 25}},
		// Wave-4 Abyss exclusives: fill the slots the original eight left bare, plus a
		// Legendary Relic the deepest banks can chase. Literal names (no i18n key) so
		// new content ships without touching all 20 locale files.
		{ID: "ABYSS_SHOULDERS", Name: "Mantle of the Abyss", Slot: SlotShoulders, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 95, Stats: Stats{HP: 160, STR: 28, DEF: 45, SPD: 12, STA: 18}},
		{ID: "ABYSS_BACK", Name: "Shroud of the Deep", Slot: SlotBack, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 110, STR: 15, DEF: 25, SPD: 35, DGE: 12}},
		{ID: "ABYSS_WAIST", Name: "Girdle of Echoes", Slot: SlotWaist, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 85, Stats: Stats{HP: 140, STR: 22, DEF: 30, SPD: 18, STA: 14}},
		{ID: "ABYSS_WRISTS", Name: "Voidsteel Bracers", Slot: SlotWrists, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 75, Stats: Stats{HP: 95, STR: 30, DEF: 18, SPD: 22, CRT: 9}},
		{ID: "ABYSS_RANGED", Name: "Whisper of the Dark", Slot: SlotRanged, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 90, Stats: Stats{HP: 90, STR: 55, DEF: 12, SPD: 45, CRT: 18, LCK: 8}},
		// Signature relics with a fixed combat Special
		{ID: "ABYSS_OFFHAND", Name: "Aegis of the Nadir", Slot: SlotOffHand, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 100, Stats: Stats{HP: 220, STR: 18, DEF: 65, SPD: 8, STA: 20}, Special: EffectThorns},
		{ID: "ABYSS_AURA", Name: "Aura of the Drowned", Slot: SlotAura, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 130, STR: 14, DEF: 20, SPD: 16, INT: 35, CHA: 40}, Special: EffectVampiric},
		{ID: "ABYSS_BAND", Name: "Bloodrage Band", Slot: SlotFinger2, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 120, STR: 45, DEF: 10, SPD: 20, CRT: 12}, Special: EffectBerserk},
		{ID: "ABYSS_TRINKET", Name: "Stillshadow Charm", Slot: SlotTrinket1, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 65, Stats: Stats{HP: 100, STR: 20, DEF: 15, SPD: 40, DGE: 14}, Special: EffectStealth},
		{ID: "ABYSS_TALISMAN", Name: "Warding Talisman", Slot: SlotTrinket2, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 65, Stats: Stats{HP: 140, STR: 16, DEF: 35, SPD: 18, STA: 12}, Special: EffectParry},
		{ID: "ABYSS_RELIC", Name: "Heart of the Abyss", Slot: SlotRelic, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 60, Stats: Stats{HP: 400, STR: 90, DEF: 90, SPD: 50, INT: 50, LCK: 40, CRT: 15}, Special: EffectPhoenix},

		// New Abyss exclusive items
		{ID: "ABYSS_LUCKY_COIN", Name: "Lucky Coin", Slot: SlotTrinket1, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 75, Stats: Stats{HP: 50, LCK: 30, CHA: 50}},
		{ID: "ABYSS_POUCH", Name: "Consumable Pouch", Slot: SlotWaist, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 80, DEF: 15, STA: 10}},
		{ID: "ABYSS_PHOENIX_PIN", Name: "Phoenix Pin", Slot: SlotCharm, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 50, Stats: Stats{HP: 150, LCK: 20}},
		{ID: "ABYSS_CHAMELEON_CLOAK", Name: "Chameleon Cloak", Slot: SlotBack, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 65, Stats: Stats{HP: 100, SPD: 25, DGE: 10}},
		{ID: "ABYSS_VAMP_NECKLACE", Name: "Vampire Tooth Necklace", Slot: SlotNeck, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 120, STR: 30}},
		{ID: "ABYSS_MANA_BATTERY", Name: "Mana Battery", Slot: SlotTrinket2, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 75, Stats: Stats{HP: 100, INT: 20, MNA: 100}},
		{ID: "ABYSS_BERSERKER_RING", Name: "Berserker Ring", Slot: SlotFinger1, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 150, STR: 35, CRT: 10}},
		{ID: "ABYSS_TITAN_BELT", Name: "Titan Belt", Slot: SlotWaist, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 90, Stats: Stats{HP: 250, STR: 60, DEF: 40}},
		{ID: "ABYSS_LEECH_SPORES", Name: "Leech Spores", Slot: SlotRelic, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 60, Stats: Stats{HP: 130, INT: 15}},
		{ID: "ABYSS_STATIC_SPARK", Name: "Static Spark Ring", Slot: SlotFinger2, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 80, SPD: 20, DGE: 5}},
		{ID: "ABYSS_FROSTBITE_GLOVES", Name: "Frostbite Gauntlets", Slot: SlotHands, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 140, STR: 25, DEF: 20}},
		{ID: "ABYSS_FIREBRAND_SWORD", Name: "Firebrand Greatsword", Slot: SlotMainHand, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 100, Stats: Stats{HP: 200, STR: 100, DEF: 20, SPD: -10}},
		{ID: "ABYSS_TIDAL_SCEPTER", Name: "Tidal Wave Scepter", Slot: SlotMainHand, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 85, Stats: Stats{HP: 120, INT: 40, MNA: 80}},
		{ID: "ABYSS_EARTHSHAKER_HAMMER", Name: "Earthshaker Warhammer", Slot: SlotMainHand, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 95, Stats: Stats{HP: 250, STR: 90, DEF: 30, CRT: 15}},
		{ID: "ABYSS_ZEPHYR_DAGGER", Name: "Zephyr Dagger", Slot: SlotMainHand, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 100, STR: 40, SPD: 35}},
		{ID: "ABYSS_LIFEBLOOM_STAFF", Name: "Lifebloom Staff", Slot: SlotMainHand, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 90, Stats: Stats{HP: 180, INT: 35, MNA: 50}},
		{ID: "ABYSS_NECROTIC_DAGGER", Name: "Necrotic Dagger", Slot: SlotMainHand, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 100, STR: 45, SPD: 15}},
		{ID: "ABYSS_DIVINE_AEGIS", Name: "Divine Aegis Shield", Slot: SlotOffHand, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 110, Stats: Stats{HP: 300, DEF: 80, MNA: 40}},
		{ID: "ABYSS_ASSASSIN_HOOD", Name: "Shadow Assassin Hood", Slot: SlotHead, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 75, Stats: Stats{HP: 100, STR: 20, SPD: 25, CRT: 10}},
		{ID: "ABYSS_ARCHMAGE_ROBES", Name: "Archmage Robes", Slot: SlotChest, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 90, Stats: Stats{HP: 150, DEF: 30, INT: 50, MNA: 150}},
		{ID: "ABYSS_GLADIATOR_CHEST", Name: "Gladiator Chestplate", Slot: SlotChest, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 105, Stats: Stats{HP: 250, STR: 20, DEF: 60}},
		{ID: "ABYSS_RANGER_BOOTS", Name: "Ranger Swift-Boots", Slot: SlotFeet, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 100, SPD: 30, DGE: 10}},
		{ID: "ABYSS_BEASTMASTER_HARNESS", Name: "Beastmaster Harness", Slot: SlotChest, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 95, Stats: Stats{HP: 200, STR: 15, DEF: 30, LCK: 20}},
		{ID: "ABYSS_DEMONIC_PACT", Name: "Demonic Pact Ring", Slot: SlotFinger1, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 60, Stats: Stats{HP: -100, INT: 60, MNA: 100}},
		{ID: "ABYSS_GUARDIAN_WARD", Name: "Guardian Angels Ward", Slot: SlotTrinket1, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 140, DEF: 25, STA: 15}},
		{ID: "ABYSS_ALCHEMIST_BELT", Name: "Alchemist Belt", Slot: SlotWaist, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 120, LCK: 20, STA: 15}},
		{ID: "ABYSS_STORMBRINGER_CLOAK", Name: "Stormbringer Cloak", Slot: SlotBack, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 75, Stats: Stats{HP: 110, DEF: 15, SPD: 20}},
		{ID: "ABYSS_SUNFIRE_PENDANT", Name: "Sunfire Pendant", Slot: SlotNeck, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 130, INT: 25}},
		{ID: "ABYSS_VOID_ESSENCE", Name: "Void Essence Relic", Slot: SlotRelic, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 60, Stats: Stats{HP: 200, STR: 50, INT: 30}},
		{ID: "ABYSS_TOMB_RAIDER", Name: "Tomb Raider Boots", Slot: SlotFeet, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 100, LCK: 25, SPD: 15}},
		{ID: "ABYSS_DRAGON_SCALE", Name: "Dragon Scale Mail", Slot: SlotChest, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 120, Stats: Stats{HP: 400, DEF: 90, STA: 30}},
		{ID: "ABYSS_KRAKEN_HIDE", Name: "Kraken Hide Leather", Slot: SlotChest, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 110, Stats: Stats{HP: 350, DEF: 60, SPD: 25, STA: 25}},
		{ID: "ABYSS_WYRM_TOOTH", Name: "Wyrm Tooth Spear", Slot: SlotMainHand, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 85, Stats: Stats{HP: 110, STR: 50, SPD: 10}},
		{ID: "ABYSS_VALKYRIE_HELM", Name: "Valkyrie Helm", Slot: SlotHead, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 130, STR: 30, DEF: 25}},
		{ID: "ABYSS_SOUL_REAPER", Name: "Soul Reaper Scythe", Slot: SlotMainHand, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 95, Stats: Stats{HP: 220, STR: 80, INT: 40, MNA: 60}},
		{ID: "ABYSS_GORGON_SHIELD", Name: "Gorgon Shield", Slot: SlotOffHand, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 100, Stats: Stats{HP: 200, DEF: 70, STA: 15}},
		{ID: "ABYSS_PEGASUS_BOOTS", Name: "Pegasus Boots", Slot: SlotFeet, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 85, Stats: Stats{HP: 110, SPD: 40, DGE: 8}},
		{ID: "ABYSS_MIDAS_GLOVES", Name: "Midas Touch Gloves", Slot: SlotHands, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 75, Stats: Stats{HP: 90, LCK: 40, CHA: 100}},
		{ID: "ABYSS_HELLFIRE_RING", Name: "Hellfire Ring", Slot: SlotFinger2, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 65, Stats: Stats{HP: 100, INT: 30}},
		{ID: "ABYSS_BLIZZARD_AMULET", Name: "Blizzard Amulet", Slot: SlotNeck, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 120, INT: 30}},
		{ID: "ABYSS_THUNDERSTRIKE", Name: "Thunderstrike Bracers", Slot: SlotWrists, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 75, Stats: Stats{HP: 90, INT: 25, SPD: 10}},
		{ID: "ABYSS_VINE_WHIP", Name: "Vine-Whip Belt", Slot: SlotWaist, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 130, DEF: 20, STA: 10}},
		{ID: "ABYSS_PLAGUE_DOCTOR", Name: "Plague Doctor Mask", Slot: SlotHead, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 140, DEF: 20, INT: 20}},
		{ID: "ABYSS_HOLY_GRAIL", Name: "Holy Grail Relic", Slot: SlotRelic, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 65, Stats: Stats{HP: 250, INT: 30, STA: 20}},
		{ID: "ABYSS_SHADOW_ORB", Name: "Shadow Orb Accessory", Slot: SlotTrinket2, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 100, INT: 35, MNA: 40}},
		{ID: "ABYSS_IRON_WILL", Name: "Iron Will Ring", Slot: SlotFinger1, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 75, Stats: Stats{HP: 150, DEF: 30, STA: 25}},
		{ID: "ABYSS_LUCKY_CLOVER", Name: "Lucky Clover Charm", Slot: SlotCharm, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 60, Stats: Stats{HP: 80, LCK: 35}},
		{ID: "ABYSS_CURSED_COMPASS", Name: "Cursed Compass", Slot: SlotTrinket1, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 65, Stats: Stats{HP: 90, LCK: 20}},
		{ID: "ABYSS_STARLIGHT_TIARA", Name: "Starlight Tiara", Slot: SlotHead, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 80, Stats: Stats{HP: 150, INT: 45, MNA: 60}},
		{ID: "ABYSS_GRAVEDIGGER_SPADE", Name: "Grave-Digger Spade", Slot: SlotMainHand, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 110, STR: 40, LCK: 15}},
		{ID: "ABYSS_SIREN_SHELL", Name: "Siren Shell Horn", Slot: SlotTrinket2, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 65, Stats: Stats{HP: 100, INT: 20, SPD: 15}},
		{ID: "ABYSS_CHRONO_WATCH", Name: "Chrono Pocketwatch", Slot: SlotTrinket1, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 110, SPD: 25}},
		{ID: "ABYSS_SPIRIT_LINK", Name: "Spirit Link Totem", Slot: SlotTotem, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 160, DEF: 25}},
		{ID: "ABYSS_GOLIATH_GLOVES", Name: "Goliath Gauntlets", Slot: SlotHands, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 85, Stats: Stats{HP: 150, STR: 45, DEF: 20}},
		{ID: "ABYSS_FEATHERWEIGHT", Name: "Featherweight Cloak", Slot: SlotBack, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 65, Stats: Stats{HP: 90, SPD: 25, DGE: 5}},
		{ID: "ABYSS_ASHEN_URN", Name: "Ashen Urn Relic", Slot: SlotRelic, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 60, Stats: Stats{HP: 120, DEF: 20, STA: 30}},
		{ID: "ABYSS_MERCURIAL_GREAVES", Name: "Mercurial Greaves", Slot: SlotFeet, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 130, SPD: 20, DEF: 20}},
		{ID: "ABYSS_RAGEBORN", Name: "Rageborn Pauldrons", Slot: SlotShoulders, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 85, Stats: Stats{HP: 170, STR: 30, DEF: 25}},
		{ID: "ABYSS_FOCUSING_MONOCLE", Name: "Focusing Monocle", Slot: SlotHead, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 80, INT: 20, CRT: 5}},
		{ID: "ABYSS_SKELETAL_KEY", Name: "Skeletal Key Accessory", Slot: SlotTrinket2, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 65, Stats: Stats{HP: 90, LCK: 25}},
		{ID: "ABYSS_BLIGHTED_RING", Name: "Blighted Ring", Slot: SlotFinger2, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 65, Stats: Stats{HP: -50, STR: 40, INT: 40}},
		{ID: "ABYSS_VESTA_HEART", Name: "Vesta Heart Jewel", Slot: SlotFinger1, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 80, Stats: Stats{HP: 200, DEF: 30}},
		{ID: "ABYSS_CRYSTALLINE_DAGGER", Name: "Crystalline Dagger", Slot: SlotMainHand, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 100, STR: 50, DGE: 10}},
		{ID: "ABYSS_ABYSSAL_PEARL", Name: "Abyssal Pearl Pendant", Slot: SlotNeck, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 120, LCK: 15}},
		{ID: "ABYSS_DREADNOUGHT", Name: "Dreadnought Plate", Slot: SlotChest, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 130, Stats: Stats{HP: 500, DEF: 120, SPD: -20}},
		{ID: "ABYSS_NINJA_TABI", Name: "Ninja Tabi", Slot: SlotFeet, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 100, SPD: 25, DGE: 10}},
		{ID: "ABYSS_SQUIRE_SHIELD", Name: "Squire Shield", Slot: SlotOffHand, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 95, Stats: Stats{HP: 180, DEF: 50, STA: 15}},
		{ID: "ABYSS_WARLORD_BANNER", Name: "Warlord Flag Banner", Slot: SlotBanner, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 85, Stats: Stats{HP: 150, STR: 25}},
		{ID: "ABYSS_SAGE_RING", Name: "Sage Ring", Slot: SlotFinger1, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 70, Stats: Stats{HP: 100, INT: 30}},
		{ID: "ABYSS_THIEF_BANDANA", Name: "Thief Bandana", Slot: SlotHead, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 75, Stats: Stats{HP: 110, LCK: 30, SPD: 15}},
		{ID: "ABYSS_MIRROR_SHIELD", Name: "Mirror Shield", Slot: SlotOffHand, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 100, Stats: Stats{HP: 250, DEF: 75}},
		{ID: "ABYSS_SLAYER_BOOTS", Name: "Slayer Boots", Slot: SlotFeet, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 85, Stats: Stats{HP: 120, STR: 20, SPD: 15}},
		{ID: "ABYSS_RUNE_CLAYMORE", Name: "Rune-Carved Claymore", Slot: SlotMainHand, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 105, Stats: Stats{HP: 180, STR: 80, DEF: 15}},
		{ID: "ABYSS_VOODOO_DOLL", Name: "Voodoo Doll Relic", Slot: SlotRelic, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 60, Stats: Stats{HP: 150, LCK: 15}},
		{ID: "ABYSS_STAR_METAL", Name: "Star-Metal Helm", Slot: SlotHead, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 140, DEF: 30, INT: 20}},
		{ID: "ABYSS_WANDERING_BOOTS", Name: "Wandering Boots", Slot: SlotFeet, Rarity: RarityEpic, XPMultiplier: 1.20, MaxDurability: 80, Stats: Stats{HP: 100, LCK: 20, SPD: 20}},
		{ID: "ABYSS_SUN_KING", Name: "Sun-King Crown", Slot: SlotHead, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 90, Stats: Stats{HP: 200, INT: 40}},
		{ID: "ABYSS_LICH_CROWN", Name: "Lich Crown", Slot: SlotHead, Rarity: RarityLegendary, XPMultiplier: getXPMult(RarityLegendary), MaxDurability: 85, Stats: Stats{HP: -150, INT: 80, MNA: 150}},
	}
	allGear = append(allGear, abyssExclusiveGear...)

	// 2. Generate 100 Corrupted Artifacts
	idx = 1
	prefixesArt := i18n.Pool("pool.artifact.corrupted_prefix")
	nounsArt := i18n.Pool("pool.artifact.corrupted_noun")

	// Safety check for empty pools (can happen during init before i18n is fully loaded)
	if len(prefixesArt) == 0 {
		prefixesArt = []string{"Corrupted", "Tainted", "Cursed"}
	}
	if len(nounsArt) == 0 {
		nounsArt = []string{"Soul", "Heart", "Essence"}
	}

	for _, p := range prefixesArt {
		for _, n := range nounsArt {
			name := i18n.T("content.artifact.corrupted", p, n)
			var mult float64
			var s Stats
			if idx%2 == 0 {
				mult = 1.5 + (r.Float64() * 2.5)
				s = Stats{HP: 150, STR: 60, DEF: 30, SPD: 45, LCK: 30, CRT: 15, CHA: 50}
			} else {
				mult = 0.1 + (r.Float64() * 0.4)
				s = Stats{HP: -100, STR: -40, DEF: -20, SPD: -20, LCK: -30, STN: 100, HGR: 50}
			}
			corruptedArtifacts = append(corruptedArtifacts, Artifact{Name: name, Mult: mult, Stats: s, MaxDurability: 15})
			idx++
		}
	}

	// 3. Titles
	posPrefixes := i18n.Pool("pool.title.positive_prefix")
	posNouns := i18n.Pool("pool.title.positive_noun")

	// Safety check for empty pools (can happen during init before i18n is fully loaded)
	if len(posPrefixes) == 0 {
		posPrefixes = []string{"Mighty", "Glorious", "Noble"}
	}
	if len(posNouns) == 0 {
		posNouns = []string{"Champion", "Warrior", "Hero"}
	}

	for _, p := range posPrefixes {
		for _, n := range posNouns {
			positiveTitles = append(positiveTitles, Title{
				Name:         i18n.T("content.title.positive", p, n),
				XPMultiplier: 3.0 + r.Float64()*7.0,
				Stats:        Stats{HP: 500, STR: 200, DEF: 100, SPD: 100, LCK: 80, INT: 50, STA: 50, CHA: 1000},
			})
		}
	}

	// 100 Extreme Titles
	extremePrefixes := i18n.Pool("pool.title.extreme_prefix")
	extremeNouns := i18n.Pool("pool.title.extreme_noun")

	// Safety check for empty pools (can happen during init before i18n is fully loaded)
	if len(extremePrefixes) == 0 {
		extremePrefixes = []string{"Apocalyptic", "Cosmic", "Galactic"}
	}
	if len(extremeNouns) == 0 {
		extremeNouns = []string{"Destroyer", "Annihilator", "Obliterator"}
	}

	for _, p := range extremePrefixes {
		for _, n := range extremeNouns {
			t := Title{
				Name:         i18n.T("content.title.extreme", p, n),
				XPMultiplier: 5.0 + r.Float64()*10.0,
				Stats:        Stats{HP: 1000, STR: 500, DEF: 250, SPD: 200, LCK: 150, INT: 100, STA: 100, CHA: 5000},
			}
			switch p {
			case "Skill-Master":
				t.ExtraSkills = 5
			case "Vampiric":
				t.Lifesteal = 50
			case "Time-Warp":
				t.MultiStrike = 100
			case "Loot-Hoarder":
				t.DoubleLoot = true
			case "One-Punch":
				t.Stats.STR = 10000
				t.Stats.CRT = 100
			case "Invincible":
				t.Stats.DEF = 10000
				t.Stats.HP = 5000
			case "Berserker":
				t.MultiStrike = 50
				t.Stats.STR = 2000
			case "Ghost":
				t.Stats.DGE = 90
				t.Stats.SPD = 1000
			case "Unbreakable":
				t.Stats.STA = 100
				t.Stats.DEF = 1000
			case "God-Mode":
				t.ExtraSkills = 5
				t.Lifesteal = 100
				t.DoubleLoot = true
				t.MultiStrike = 100
			}
			positiveTitles = append(positiveTitles, t)
		}
	}

	negPrefixes := i18n.Pool("pool.title.negative_prefix")
	negNouns := i18n.Pool("pool.title.negative_noun")

	// Safety check for empty pools (can happen during init before i18n is fully loaded)
	if len(negPrefixes) == 0 {
		negPrefixes = []string{"Weak", "Feeble", "Pathetic"}
	}
	if len(negNouns) == 0 {
		negNouns = []string{"Peasant", "Beggar", "Failure"}
	}

	for _, p := range negPrefixes {
		for _, n := range negNouns {
			negativeTitles = append(negativeTitles, Title{
				Name:         i18n.T("content.title.negative", p, n),
				XPMultiplier: 0.01 + r.Float64()*0.1,
				Stats:        Stats{HP: -300, STR: -150, DEF: -80, SPD: -80, LCK: -100, STN: 500, HGR: 100},
			})
		}
	}

	// 3.5. 100 New Temporary Titles (programmatic, single-word)
	tempTitles := []string{
		"Sentinel", "Specter", "Wraith", "Crusader", "Sage", "Stalker", "Archon", "Vanguard", "Harbinger", "Outcast",
		"Gladiator", "Paladin", "Ranger", "Sorcerer", "Warlock", "Necromancer", "Assassin", "Rogue", "Cleric", "Druid",
		"Bard", "Monk", "Barbarian", "Fighter", "Wizard", "Alchemist", "Beastmaster", "Berserker", "Champion", "Defender",
		"Guardian", "Protector", "Warden", "Templar", "Zealot", "Inquisitor", "Executioner", "Slayer", "Hunter", "Tracker",
		"Scout", "Pathfinder", "Pioneer", "Explorer", "Nomad", "Wanderer", "Pilgrim", "Traveler", "Voyager", "Adventurer",
		"Hero", "Legend", "Myth", "Fable", "Shadow", "Ghost", "Phantom", "Spirit", "Soul", "Mind",
		"Heart", "Blade", "Shield", "Staff", "Wand", "Scroll", "Tome", "Grimoire", "Relic", "Artifact",
		"Catalyst", "Conduit", "Medium", "Oracle", "Prophet", "Seer", "Mystic", "Occultist", "Ritualist", "Summoner",
		"Conjurer", "Elementalist", "Pyromancer", "Cryomancer", "Electromancer", "Geomancer", "Aeromancer", "Hydromancer", "Chronomancer", "Illusionist",
		"Enchanter", "Spellbinder", "Runesmith", "Blacksmith", "Artificer", "Engineer", "Scholar", "Philosopher", "Tactician", "Strategist",
	}
	for _, name := range tempTitles {
		positiveTitles = append(positiveTitles, Title{
			Name:         name,
			XPMultiplier: 1.5 + r.Float64()*3.5,
			Stats:        Stats{HP: 300, STR: 100, DEF: 50, SPD: 50, LCK: 40, INT: 30, STA: 30, CHA: 500},
		})
	}

	// 4. Generate Enchantments
	enchPrefixes := i18n.Pool("pool.enchantment.prefix")

	// Safety check for empty pools (can happen during init before i18n is fully loaded)
	if len(enchPrefixes) == 0 {
		enchPrefixes = []string{"Fiery", "Icy", "Shocking"}
	}

	for i, p := range enchPrefixes {
		rarity := RarityRare
		if i > 6 {
			rarity = RarityEpic
		}
		if i > 10 {
			rarity = RarityLegendary
		}

		duraBonus := 0
		if strings.Contains(p, "Reinforced") || strings.Contains(p, "Unbreakable") || strings.Contains(p, "Diamond") {
			duraBonus = 50 * (int(rarity) - 1)
			if duraBonus < 20 {
				duraBonus = 20
			}
		}

		allEnchantments = append(allEnchantments, Enchantment{
			ID:           fmt.Sprintf("E%d", i),
			Name:         i18n.T("content.enchantment.name", p),
			Rarity:       rarity,
			XPMultiplier: getXPMult(rarity) - 0.1,
			Stats:        Stats{STR: 15 * (int(rarity) + 1), SPD: 10 * (int(rarity) + 1), CRT: 5 * (int(rarity) + 1)},
			DuraBonus:    duraBonus,
			Description:  i18n.T("content.enchantment.description", p),
		})
	}
}

func RandomItemEffect() ItemEffect {
	effects := []ItemEffect{
		EffectThorns, EffectVampiric, EffectBerserk, EffectLucky, EffectTreasureHunter,
		EffectQuick, EffectBulwark, EffectRadiant, EffectFragile, EffectSteady,
		EffectMindControl, EffectRegenStack, EffectPhoenix, EffectStealth, EffectParry, EffectCleanse,
	}
	// #nosec G404
	if rand.Float64() < 0.2 { // #nosec G404
		// #nosec G404
		return effects[rand.IntN(len(effects))] // #nosec G404
	}
	return EffectNone
}

// #nosec G404
func RandomConsumable() Consumable { return allConsumables[rand.IntN(len(allConsumables))] } // #nosec G404
func RandomGearDrop() Gear {
	var g Gear
	// Loop until we get a non-Abyss-exclusive item for standard drops
	for {
		// #nosec G404
		if rand.Float64() < 0.05 { // #nosec G404
			// #nosec G404
			g = uniqueLegendaries[rand.IntN(len(uniqueLegendaries))] // #nosec G404
		} else {
			// #nosec G404
			g = allGear[rand.IntN(len(allGear))] // #nosec G404
		}
		if !strings.HasPrefix(g.ID, "ABYSS_") {
			break
		}
	}
	g.Special = RandomItemEffect()
	return g
}

// RandomArcadeGearDrop returns a random gear drop capped at RarityRare for arcade/daily spin rewards.
func RandomArcadeGearDrop() Gear {
	for {
		g := RandomGearDrop()
		if g.Rarity <= RarityRare {
			return g
		}
	}
}

// GearByMinRarity returns all gear items (excluding Abyss-exclusive) at or above
// the given rarity floor. Used as a guaranteed fallback in deep-bank rewards.
func GearByMinRarity(floor Rarity) []Gear {
	var out []Gear
	for _, g := range allGear {
		if !strings.HasPrefix(g.ID, "ABYSS_") && g.Rarity >= floor {
			out = append(out, g)
		}
	}
	return out
}

// abyssSetTiers define the cumulative set bonuses granted for equipping N pieces
// of Abyss-exclusive gear (IDs prefixed "ABYSS_"). Bonuses stack: 6 equipped
// pieces grant the 2-, 4-, and 6-piece tiers together.
var abyssSetTiers = []struct {
	Pieces int
	Bonus  Stats
}{
	{2, Stats{HP: 200, STR: 30, DEF: 30}},
	{4, Stats{HP: 400, STR: 60, DEF: 60, SPD: 20}},
	{6, Stats{HP: 800, STR: 120, DEF: 120, SPD: 40, CRT: 30}},
}

// AbyssSetBonus returns the cumulative Abyss-set bonus stats for the given number
// of equipped Abyss pieces, plus the highest piece-threshold actually reached (0
// if none), so callers can surface "N-piece Abyss set" in the loot notes.
func AbyssSetBonus(equipped int) (Stats, int) {
	var total Stats
	reached := 0
	for _, t := range abyssSetTiers {
		if equipped >= t.Pieces {
			total = total.Add(t.Bonus)
			reached = t.Pieces
		}
	}
	return total, reached
}

// IsAbyssGearID reports whether a gear ID belongs to the Abyss-exclusive set.
func IsAbyssGearID(id string) bool { return strings.HasPrefix(id, "ABYSS_") }

func RandomAbyssGearDrop() Gear {
	if len(abyssExclusiveGear) == 0 {
		return RandomGearDrop()
	}
	// #nosec G404
	g := abyssExclusiveGear[rand.IntN(len(abyssExclusiveGear))]
	// Signature relics carry a fixed Special (Phoenix, Thorns, …); only items that
	// define none get a random roll, so their authored combat effect is preserved.
	if g.Special == EffectNone {
		g.Special = RandomItemEffect()
	}
	return g
}

// #nosec G404
func RandomStarterGear() Gear {
	if len(starterGear) == 0 {
		return Gear{}
	}
	return starterGear[rand.IntN(len(starterGear))] // #nosec G404
}
func RandomArtifact() Artifact {
	// #nosec G404
	a := corruptedArtifacts[rand.IntN(len(corruptedArtifacts))] // #nosec G404
	a.Special = RandomItemEffect()
	return a
}
func RandomEnchantment() Enchantment {
	// #nosec G404
	e := allEnchantments[rand.IntN(len(allEnchantments))] // #nosec G404
	e.Special = RandomItemEffect()
	return e
}
func RandomTitle() Title {
	// #nosec G404
	if rand.Float64() < 0.8 {
		return positiveTitles[rand.IntN(len(positiveTitles))]
	} // #nosec G404
	// #nosec G404
	return negativeTitles[rand.IntN(len(negativeTitles))] // #nosec G404
}

// ShopStock returns a deterministic list of purchasable gear for the given seed
// (e.g. a day number), excluding the basic Novice starter items so the shop
// always offers a meaningful upgrade path. Used by the web shop.
func ShopStock(seed int64, count int) []Gear {
	r := rand.New(rand.NewPCG(uint64(seed), uint64(seed)+1)) // #nosec G404 G115 -- deterministic shop rotation, seed always non-negative
	var pool []Gear
	for _, g := range allGear {
		if strings.HasPrefix(g.ID, "B_") { // skip Novice/starter junk
			continue
		}
		pool = append(pool, g)
	}
	out := make([]Gear, 0, count)
	if len(pool) == 0 {
		return out
	}
	for i := 0; i < count; i++ {
		out = append(out, pool[r.IntN(len(pool))])
	}
	return out
}

func GetGearByID(id string) (Gear, bool) {
	for _, g := range allGear {
		if g.ID == id {
			return g, true
		}
	}
	for _, g := range uniqueLegendaries {
		if g.ID == id {
			return g, true
		}
	}
	return Gear{}, false
}

func GetEnchantmentByID(id string) (Enchantment, bool) {
	for _, e := range allEnchantments {
		if e.ID == id {
			return e, true
		}
	}
	return Enchantment{}, false
}

func GetConsumableByID(id string) (Consumable, bool) {
	for _, c := range allConsumables {
		if c.ID == id {
			return c, true
		}
	}
	return Consumable{}, false
}

func GetTitleByName(name string) (Title, bool) {
	for _, t := range positiveTitles {
		if t.Name == name {
			return t, true
		}
	}
	for _, t := range negativeTitles {
		if t.Name == name {
			return t, true
		}
	}
	return Title{}, false
}

func GetArtifactByName(name string) (Artifact, bool) {
	for _, a := range corruptedArtifacts {
		if a.Name == name {
			return a, true
		}
	}
	return Artifact{}, false
}

func IsTitle(name string) bool {
	_, ok := GetTitleByName(name)
	return ok
}

func IsGearOrArtifact(name string) bool {
	for _, g := range allGear {
		if g.Name == name {
			return true
		}
	}
	for _, g := range uniqueLegendaries {
		if g.Name == name {
			return true
		}
	}
	for _, a := range corruptedArtifacts {
		if a.Name == name {
			return true
		}
	}
	return false
}
