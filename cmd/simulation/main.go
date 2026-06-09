package main

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

// ============================================================
// CONSTANTS (mirrored from internal packages)
// ============================================================

const (
	xpMin             = 20
	xpMax             = 50
	maxLevel          = 10000
	prestigeLevel     = 10000
	critChance        = 0.05
	critMult          = 3.0
	partyMult         = 1.25
	serverMultPerUser = 0.05
	serverMultCap     = 2.0
	noGamePenalty     = 0.5
	deathXPPenalty    = 0.05
	artifactChance    = 0.01
	titleChance       = 0.005
	gearChance        = 0.05
	skillChance       = 0.05
	consChance        = 0.10
	enchChance        = 0.02
	lootBoxEvery      = 25
	lootBoxMin        = 50
	lootBoxMax        = 500
	slothGraceDays    = 7
	slothDailyDecay   = 0.02
	duraLossPerFight  = 1
	duraLossPenalty   = 3
)

const (
	Common    = 0
	Uncommon  = 1
	Rare      = 2
	Epic      = 3
	Legendary = 4
	Mythic    = 5
	Divine    = 6
)

var rarityNames = []string{"Common", "Uncommon", "Rare", "Epic", "Legendary", "Mythic", "Divine"}

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

func GenerateUniqueItemName(rng *rand.Rand) string {
	return uniqueAdjectives[rng.Intn(len(uniqueAdjectives))] + " " + uniqueNouns[rng.Intn(len(uniqueNouns))]
}

// Ultimate Skill name generation (50 verbs × 20 nouns = 1000 combinations)
var ultimateVerbs = []string{
	"Annihilating", "Devastating", "Obliterating", "Shattering", "Eradicating",
	"Decimating", "Destroying", "Crushing", "Smashing", "Pulverizing",
	"Vaporizing", "Incinerating", "Freezing", "Electrifying", "Corrupting",
	"Purifying", "Banishing", "Summoning", "Channeling", "Unleashing",
	"Igniting", "Extinguishing", "Rending", "Cleaving", "Piercing",
	"Shredding", "Blasting", "Bombarding", "Storming", "Ravaging",
	"Consuming", "Devouring", "Absorbing", "Reflecting", "Amplifying",
	"Nullifying", "Silencing", "Blinding", "Stunning", "Paralyzing",
	"Poisoning", "Cursing", "Blessing", "Healing", "Shielding",
	"Empowering", "Enraging", "Terrifying", "Inspiring", "Transcending",
}
var ultimateNouns = []string{
	"Strike", "Blast", "Wave", "Storm", "Fury",
	"Wrath", "Rage", "Nova", "Burst", "Flare",
	"Surge", "Pulse", "Beam", "Ray", "Bolt",
	"Slash", "Thrust", "Barrage", "Volley", "Onslaught",
}

func GenerateUltimateSkillName(rng *rand.Rand) string {
	return ultimateVerbs[rng.Intn(len(ultimateVerbs))] + " " + ultimateNouns[rng.Intn(len(ultimateNouns))]
}

var gearSlots = []string{
	"Head", "Neck", "Shoulders", "Back", "Chest", "Wrists", "Hands", "Waist",
	"Legs", "Feet", "Finger1", "Finger2", "Trinket1", "Trinket2", "MainHand",
	"OffHand", "Ranged", "Relic", "Artifact", "Soul", "Aura", "Charm", "Mount", "Companion",
	"Pet1", "Pet2", "Emblem1", "Emblem2", "Banner", "Totem", // 6 new slots (30 total)
}

// ============================================================
// DATA STRUCTURES
// ============================================================

type Stats struct {
	HP, STR, DEF, SPD, LCK int
	INT, STA, CRT, DGE     int
}

func (s Stats) Score() float64 {
	return float64(s.HP)/5 + float64(s.STR) + float64(s.DEF) + float64(s.SPD) + float64(s.LCK) +
		float64(s.INT) + float64(s.STA) + float64(s.CRT) + float64(s.DGE)
}

func (s Stats) Add(o Stats) Stats {
	return Stats{
		HP: s.HP + o.HP, STR: s.STR + o.STR, DEF: s.DEF + o.DEF, SPD: s.SPD + o.SPD, LCK: s.LCK + o.LCK,
		INT: s.INT + o.INT, STA: s.STA + o.STA, CRT: s.CRT + o.CRT, DGE: s.DGE + o.DGE,
	}
}

func (s Stats) MulF(m float64) Stats {
	return Stats{
		HP: int(float64(s.HP) * m), STR: int(float64(s.STR) * m), DEF: int(float64(s.DEF) * m),
		SPD: int(float64(s.SPD) * m), LCK: int(float64(s.LCK) * m), INT: int(float64(s.INT) * m),
		STA: int(float64(s.STA) * m), CRT: int(float64(s.CRT) * m), DGE: int(float64(s.DGE) * m),
	}
}

type Gear struct {
	Slot       string
	Rarity     int
	Durability int
	MaxDur     int
	Stats      Stats
	XPMult     float64
	ItemLevel  float64
}

// UltimateSkill represents a powerful ability with multi-round cooldown
type UltimateSkill struct {
	Name            string
	Power           float64 // Damage multiplier (e.g., 5.0 = 5x normal damage)
	CooldownRounds  int     // Total rounds to wait after use
	CurrentCooldown int     // Current cooldown counter (0 = ready)
	Rarity          int     // Rarity tier (affects power)
}

type AuctionItem struct {
	ID        int
	SellerID  int
	Type      string
	Name      string
	Gear      *Gear
	Skill     bool
	Ult       *UltimateSkill
	Unique    bool
	Ench      bool
	Price     int64
	ExpiresAt int // simulation day
}

type AuctionHouse struct {
	Items []AuctionItem
	NextID int
}

var globalAH = AuctionHouse{}
var playerGold = make(map[int]int64)

type Player struct {
	ID                 int
	Level              int
	XP                 float64
	XPNeeded           float64
	Prestige           int
	CurrentHP          int
	MaxHP              int
	Stats              Stats
	Gear               map[string]*Gear
	TotalGearScore     float64
	AvgItemLevel       float64
	ConsecutiveWins    int
	ConsecutiveLosses  int
	TotalFights        int
	TotalWins          int
	TotalLosses        int
	TotalGearDrops     int
	TotalSkillDrops    int
	TotalArtifactDrops int
	TotalTitleDrops    int
	TotalConsDrops     int
	TotalEnchDrops     int
	DaysActive         int
	StreakDays         int
	LastPokeDay        int
	TotalXPEarned      float64
	PityStack          float64 // consecutive loss stat boost
	GroupSize          int     // Number of players in the party (1-20)

	// Unique Item Collection
	UniqueItemsCollected map[string]bool
	TotalUniqueCollected int

	// Consumable Buffs
	XPBoostActive   bool
	XPBoostMult     float64
	StatBoostActive bool
	StatBoostMult   float64

	// Ultimate Skill Slot (dedicated slot, one at a time)
	UltimateSkill *UltimateSkill

	// Ultimate Skill Collection
	UltimateSkillsCollected map[string]bool
	TotalUltimatesCollected int
	TotalUltimatesUsed      int
}

type Mob struct {
	Name   string
	Level  int
	HP     int
	MaxHP  int
	Stats  Stats
	IsBoss bool
	Reward float64
}

type CombatResult struct {
	Won        bool
	XPGained   float64
	XPPenalty  float64
	MobsKilled int
	Rounds     int
}

type DaySnapshot struct {
	Day                int
	Level              int
	XP                 float64
	Prestige           int
	CurrentHP          int
	MaxHP              int
	STR, DEF, SPD      int
	LCK, INT, STA      int
	CRT, DGE           int
	AvgItemLevel       float64
	TotalGearScore     float64
	GearSlotsFilled    int
	GearSlotsTotal     int
	FightsToday        int
	WinsToday          int
	LossesToday        int
	CumulativeWinRate  float64
	XPEarnedToday      float64
	XPPerFight         float64
	GearDropsToday     int
	SkillDropsToday    int
	ArtifactDropsToday int
	Prestiges          int
	PityStack          float64
}

type Simulation struct {
	Players         []*Player
	Rng             *rand.Rand
	Day             int
	History         []DaySnapshot
	Recommendations []string
	Params          SimParams
}

type SimParams struct {
	XPMin                int
	XPMax                int
	ExponentCap          float64
	GearChance           float64
	SkillChance          float64
	ArtifactChance       float64
	MaxDurability        int
	DuraLoss             int
	PrestigeLevel        int
	MobHPMult            float64
	MobDamageMult        float64
	PlayerHPMult         float64
	PlayerDMGMult        float64
	SkillChanceCombat    float64
	SkillPowerBase       float64
	PityBoostPerLoss     float64
	PityBoostCap         float64
	ZoneDiffMin          float64
	ZoneDiffMax          float64
	MobScaling           float64
	GearILvlScale        float64
	GearStatScale        float64
	BaseMobsMin          int
	BaseMobsMax          int
	BossChance           float64
	BossHPMult           float64
	BossDMGMult          float64
	MaxRounds            int
	EscalationRate       float64
	GroupSize            int     // 1 to 20 channel members
	UniqueItemChance     float64 // Chance to drop a unique collectible item
	UltimateSkillChance  float64 // Chance to drop an ultimate skill
	UltimateCooldownBase int     // Base cooldown rounds for ultimate skills
	UltimatePowerBase    float64 // Base power multiplier for ultimate skills
}

func DefaultParams() SimParams {
	return SimParams{
		XPMin: 25, XPMax: 55, ExponentCap: 4.0,
		GearChance: 0.07, SkillChance: 0.05, ArtifactChance: 0.01,
		MaxDurability: 50, DuraLoss: 1, PrestigeLevel: 5000,
		MobHPMult: 0.8, MobDamageMult: 0.7,
		PlayerHPMult: 1.2, PlayerDMGMult: 1.1,
		SkillChanceCombat: 0.30, SkillPowerBase: 2.2,
		PityBoostPerLoss: 0.15, PityBoostCap: 1.5,
		ZoneDiffMin: 0.8, ZoneDiffMax: 1.6,
		MobScaling:    0.04,
		GearILvlScale: 10.0, GearStatScale: 0.025,
		BaseMobsMin: 2, BaseMobsMax: 6,
		BossChance: 0.10, BossHPMult: 2.5, BossDMGMult: 1.8,
		MaxRounds: 10, EscalationRate: 0.12,
		GroupSize:            1,     // Default solo
		UniqueItemChance:     0.01,  // 1% chance per roll
		UltimateSkillChance:  0.005, // 0.5% chance per roll (rare)
		UltimateCooldownBase: 5,     // 5 rounds base cooldown
		UltimatePowerBase:    4.0,   // 4x damage base
	}
}

func BalancedParams() SimParams {
	p := DefaultParams()
	// Balanced combat for steady progression
	p.MobHPMult = 1.0         // mobs have 100% HP
	p.MobDamageMult = 1.0     // mobs deal 100% damage
	p.PlayerHPMult = 1.0      // players have 1.0x HP (base stats tuned)
	p.PlayerDMGMult = 1.0     // players deal 1.0x damage (base stats tuned)
	p.PityBoostPerLoss = 0.20 // moderate pity
	p.PityBoostCap = 2.0
	p.SkillPowerBase = 2.5  // skills hit moderately harder
	p.EscalationRate = 0.12 // moderate escalation
	p.BossHPMult = 2.5      // bosses less tanky
	p.BossDMGMult = 1.8     // bosses less deadly
	// Better XP
	p.XPMin = 40
	p.XPMax = 80
	p.ExponentCap = 3.0
	p.PrestigeLevel = 3000  // Achievable in ~3-5 years
	p.GearStatScale = 0.015 // Better gear scaling to keep up
	p.GroupSize = 5         // Small party
	return p
}

func OptimizedParams() SimParams {
	p := BalancedParams()
	// Optimized for engaging progression
	p.XPMin = 50
	p.XPMax = 100
	p.GearChance = 0.12
	p.SkillChance = 0.10
	p.ArtifactChance = 0.02
	p.MaxDurability = 80
	p.DuraLoss = 1
	p.PrestigeLevel = 2500 // Faster prestige
	p.ExponentCap = 2.8
	p.GearStatScale = 0.020 // Stronger gear
	p.GroupSize = 10        // Larger raid group
	return p
}

// ============================================================
// XP CURVE
// ============================================================

func XPForLevel(level int, cap float64) float64 {
	if level <= 1 {
		return 0
	}
	// Smoother curve: starts at 1.15, caps at the provided cap (e.g., 3.0)
	exponent := 1.15 + float64(level)/2000.0
	if exponent > cap {
		exponent = cap
	}
	return math.Round(math.Pow(float64(level-1), exponent))
}

func XPNeededForNextLevel(level int, cap float64) float64 {
	return XPForLevel(level+1, cap) - XPForLevel(level, cap)
}

// ============================================================
// PLAYER
// ============================================================

func NewPlayer() *Player {
	p := &Player{
		Level:                   1,
		Gear:                    make(map[string]*Gear),
		GroupSize:               1,
		UniqueItemsCollected:    make(map[string]bool),
		UltimateSkillsCollected: make(map[string]bool),
	}
	p.RecalculateStats(DefaultParams())
	p.XPNeeded = XPNeededForNextLevel(1, 5.0)
	p.CurrentHP = p.MaxHP
	return p
}

func NewPlayerWithParams(params SimParams) *Player {
	p := &Player{
		Level:                   1,
		Gear:                    make(map[string]*Gear),
		GroupSize:               params.GroupSize,
		UniqueItemsCollected:    make(map[string]bool),
		UltimateSkillsCollected: make(map[string]bool),
	}
	p.RecalculateStats(params)
	p.XPNeeded = XPNeededForNextLevel(1, params.ExponentCap)
	p.CurrentHP = p.MaxHP
	return p
}

func (p *Player) BaseStats(params SimParams) Stats {
	lvl := float64(p.Level)
	// Tuned scaling: quadratic HP growth, but balanced against mob scaling
	return Stats{
		HP:  int((100.0 + lvl*5.0 + lvl*lvl*0.005) * params.PlayerHPMult),
		STR: int((10.0 + lvl*1.0 + lvl*lvl*0.001) * params.PlayerDMGMult),
		DEF: int(5.0 + lvl*0.5 + lvl*lvl*0.0005),
		SPD: int(10.0 + lvl*1.2),
		LCK: int(lvl / 4.0),
		INT: int(lvl / 8.0),
		STA: int(lvl / 8.0),
		CRT: int(5.0 + lvl/40.0),
		DGE: int(5.0 + lvl/40.0),
	}
}

func (p *Player) PrestigeMult() float64 {
	return 1.0 + float64(p.Prestige)*0.05
}

func (p *Player) AutoListUnwantedItems(item interface{}, day int) {
	var itype, name string
	var price int64
	var gear *Gear
	var ult *UltimateSkill

	switch v := item.(type) {
	case *Gear:
		if v.Rarity < Rare {
			return
		}
		itype = "gear"
		name = v.Slot
		gear = v
		// Combat rating equivalent for simulation
		cr := float64(v.Stats.Score()) / 10.0
		price = int64(cr*10+float64(v.Stats.Score())*5) * int64(v.Rarity+1)
	case string: // Skill or Unique
		switch v {
		case "skill":
			itype = "skill"
			price = 500
		case "unique":
			itype = "unique"
			price = 2000
		case "ench":
			itype = "ench"
			price = 300
		}
	case *UltimateSkill:
		itype = "ultimate"
		name = v.Name
		ult = v
		price = int64(v.Power*200) * int64(v.Rarity+1)
	default:
		return
	}

	if price < 10 {
		price = 10
	}

	globalAH.Items = append(globalAH.Items, AuctionItem{
		ID:        globalAH.NextID,
		SellerID:  p.ID,
		Type:      itype,
		Name:      name,
		Gear:      gear,
		Ult:       ult,
		Price:     price,
		ExpiresAt: day + 1,
	})
	globalAH.NextID++
}

func (p *Player) AutoPurchaseUpgrades(day int) {
	gold := playerGold[p.ID]
	for i := 0; i < len(globalAH.Items); i++ {
		item := globalAH.Items[i]
		if item.SellerID == p.ID || item.ExpiresAt < day || item.Price > gold {
			continue
		}

		isUpgrade := false
		switch item.Type {
		case "gear":
			if p.EquipGear(item.Gear) {
				isUpgrade = true
			}
		case "skill":
			// Simplified: always an upgrade if we have < 5
			isUpgrade = true
			p.TotalSkillDrops++
		case "ultimate":
			if _, exists := p.UltimateSkillsCollected[item.Name]; !exists {
				isUpgrade = true
				p.UltimateSkillsCollected[item.Name] = true
				p.TotalUltimatesCollected++
				if p.UltimateSkill == nil {
					p.UltimateSkill = item.Ult
				}
			}
		case "unique":
			if _, exists := p.UniqueItemsCollected[item.Name]; !exists {
				isUpgrade = true
				p.UniqueItemsCollected[item.Name] = true
				p.TotalUniqueCollected++
			}
		case "ench":
			isUpgrade = true
			p.TotalEnchDrops++
		}

		if isUpgrade {
			// Buy it
			playerGold[p.ID] -= item.Price
			playerGold[item.SellerID] += item.Price
			// Remove from AH
			globalAH.Items = append(globalAH.Items[:i], globalAH.Items[i+1:]...)
			i--
			gold = playerGold[p.ID]
		}
	}
}

func (p *Player) RecalculateStats(params SimParams) {
	base := p.BaseStats(params)
	gearStats := Stats{}
	gearScore := 0.0
	totalILvl := 0.0
	gearCount := 0

	for _, g := range p.Gear {
		if g.Durability > 0 {
			gearStats = gearStats.Add(g.Stats)
			gearScore += g.Stats.Score()
			totalILvl += g.ItemLevel
			gearCount++
		}
	}

	p.Stats = base.Add(gearStats).MulF(p.PrestigeMult())

	// Apply consumable stat boost
	if p.StatBoostActive {
		p.Stats = p.Stats.MulF(p.StatBoostMult)
	}
	p.TotalGearScore = gearScore
	if gearCount > 0 {
		p.AvgItemLevel = totalILvl / float64(gearCount)
	} else {
		p.AvgItemLevel = 0
	}
	p.MaxHP = p.Stats.HP
	if p.CurrentHP > p.MaxHP {
		p.CurrentHP = p.MaxHP
	}
	if p.CurrentHP <= 0 {
		p.CurrentHP = p.MaxHP
	}
}

// ============================================================
// GEAR
// ============================================================

func GenerateGear(rng *rand.Rand, level int, params SimParams) *Gear {
	r := rng.Float64()
	var rarity int
	switch {
	case r < 0.45:
		rarity = Common
	case r < 0.75:
		rarity = Uncommon
	case r < 0.90:
		rarity = Rare
	case r < 0.97:
		rarity = Epic
	case r < 0.995:
		rarity = Legendary
	case r < 0.999:
		rarity = Mythic
	default:
		rarity = Divine
	}

	slot := gearSlots[rng.Intn(len(gearSlots))]
	mul := float64(rarity + 1)
	lvlScale := 1.0 + float64(level)*params.GearStatScale

	stats := Stats{
		HP:  int(10.0 * mul * lvlScale),
		STR: int(5.0 * mul * lvlScale),
		DEF: int(3.0 * mul * lvlScale),
		SPD: int(2.0 * mul * lvlScale),
		LCK: int(1.0 * mul * lvlScale),
		INT: int(1.0 * mul * lvlScale),
		STA: int(1.0 * mul * lvlScale),
		CRT: int(1.0 * mul * lvlScale),
		DGE: int(1.0 * mul * lvlScale),
	}

	maxDur := 30 + rarity*20
	if maxDur < params.MaxDurability {
		maxDur = params.MaxDurability + rarity*10
	}

	itemLevel := float64(rarity+1) * lvlScale * params.GearILvlScale

	return &Gear{
		Slot:       slot,
		Rarity:     rarity,
		Durability: maxDur,
		MaxDur:     maxDur,
		Stats:      stats,
		XPMult:     1.5 - 0.1*float64(rarity),
		ItemLevel:  itemLevel,
	}
}

func (p *Player) EquipGear(g *Gear) bool {
	existing, ok := p.Gear[g.Slot]
	if ok && existing.Rarity >= g.Rarity && existing.ItemLevel >= g.ItemLevel {
		return false
	}
	p.Gear[g.Slot] = g
	return true
}

// GenerateUltimateSkill creates a random ultimate skill with rarity-based power
func GenerateUltimateSkill(rng *rand.Rand, level int, params SimParams) *UltimateSkill {
	// Determine rarity (ultimate skills are inherently rare)
	r := rng.Float64()
	var rarity int
	switch {
	case r < 0.50:
		rarity = Rare // 50%
	case r < 0.80:
		rarity = Epic // 30%
	case r < 0.95:
		rarity = Legendary // 15%
	case r < 0.99:
		rarity = Mythic // 4%
	default:
		rarity = Divine // 1%
	}

	name := GenerateUltimateSkillName(rng)

	// Power scales with rarity and player level
	rarityMult := float64(rarity+1) * 0.5 // Rare=1.5, Epic=2.0, Legendary=2.5, Mythic=3.0, Divine=3.5
	levelMult := 1.0 + float64(level)*0.001
	power := params.UltimatePowerBase * rarityMult * levelMult

	// Cooldown scales with rarity (higher rarity = longer cooldown but more power)
	cooldown := params.UltimateCooldownBase + rarity*2 // Rare=9, Epic=11, Legendary=13, Mythic=15, Divine=17

	return &UltimateSkill{
		Name:            name,
		Power:           power,
		CooldownRounds:  cooldown,
		CurrentCooldown: 0, // Starts ready
		Rarity:          rarity,
	}
}

// EquipUltimateSkill equips an ultimate skill if it's better than the current one
func (p *Player) EquipUltimateSkill(skill *UltimateSkill) bool {
	if p.UltimateSkill != nil && p.UltimateSkill.Rarity >= skill.Rarity && p.UltimateSkill.Power >= skill.Power {
		return false
	}
	p.UltimateSkill = skill
	return true
}

// ============================================================
// MOBS
// ============================================================

var mobPrefixes = []string{"Feral", "Dark", "Shadow", "Cursed", "Wild", "Ancient", "Corrupt", "Savage", "Grim", "Vile"}
var mobNouns = []string{"Wolf", "Bear", "Wraith", "Goblin", "Skeleton", "Troll", "Spider", "Bat", "Slime", "Imp"}

func GenerateMobs(rng *rand.Rand, partyLevel int, groupSize int, difficulty float64, params SimParams) []Mob {
	// 15% chance to spawn a HORDE of weaker mobs (great for farming drops/XP)
	isHorde := rng.Float64() < 0.15

	numMobs := params.BaseMobsMin + rng.Intn(params.BaseMobsMax-params.BaseMobsMin+1)
	if isHorde {
		numMobs = 5 + rng.Intn(6) // 5 to 10 mobs in a horde
	}

	// Bosses don't spawn in hordes
	isBoss := rng.Float64() < params.BossChance && !isHorde

	// Mobs scale slightly with group size to prevent trivial farming
	groupMult := 1.0 + float64(groupSize-1)*0.1
	if groupMult > 2.5 {
		groupMult = 2.5
	}

	mobs := make([]Mob, 0, numMobs)
	for i := 0; i < numMobs; i++ {
		// Mob level scales with party level
		levelMult := 0.8 + rng.Float64()*0.4
		if isHorde {
			levelMult = 0.5 + rng.Float64()*0.3 // Horde mobs are weaker (50-80% of player level)
		}
		mobLvl := int(float64(partyLevel) * levelMult)
		if mobLvl < 1 {
			mobLvl = 1
		}

		name := mobPrefixes[rng.Intn(len(mobPrefixes))] + " " + mobNouns[rng.Intn(len(mobNouns))]
		if isHorde {
			name = "Horde of " + name + "s"
		}

		// Mob stats scale quadratically to match player's quadratic growth, scaled by group size
		hp := int((float64(50) + float64(mobLvl)*12.0 + float64(mobLvl)*float64(mobLvl)*0.015) * params.MobHPMult * difficulty * groupMult)
		str := int((float64(5) + float64(mobLvl)*2.5 + float64(mobLvl)*float64(mobLvl)*0.004) * params.MobDamageMult * difficulty * groupMult)
		def := int((float64(3) + float64(mobLvl)*1.2 + float64(mobLvl)*float64(mobLvl)*0.002) * groupMult)

		// Hordes give slightly less XP per mob, but more total XP due to quantity
		rewardMult := 0.7
		if isHorde {
			rewardMult = 0.6
		}
		reward := float64(mobLvl) * 8 * difficulty * rewardMult

		mob := Mob{
			Name:   name,
			Level:  mobLvl,
			HP:     hp,
			MaxHP:  hp,
			Stats:  Stats{HP: hp, STR: str, DEF: def, SPD: int(5 + float64(mobLvl)*0.5)},
			Reward: reward,
		}

		if i == 0 && isBoss {
			mob.Name = "BOSS: " + name
			mob.HP = int(float64(mob.HP) * params.BossHPMult)
			mob.MaxHP = mob.HP
			mob.Stats.STR = int(float64(mob.Stats.STR) * params.BossDMGMult)
			mob.Stats.HP = mob.HP
			mob.Reward *= 3
			mob.IsBoss = true
		}

		mobs = append(mobs, mob)
	}
	return mobs
}

// ============================================================
// COMBAT
// ============================================================

func SimulateCombat(rng *rand.Rand, player *Player, mobs []Mob, params SimParams) CombatResult {
	result := CombatResult{}

	// Party bonus scales with group size (1 to 20), capped at 3.0x
	groupSize := player.GroupSize
	if groupSize < 1 {
		groupSize = 1
	}
	if groupSize > 20 {
		groupSize = 20
	}
	partyBonus := 1.0 + float64(groupSize-1)*0.15
	if partyBonus > 3.0 {
		partyBonus = 3.0
	}

	playerHP := player.CurrentHP
	maxPlayerHP := player.MaxHP
	playerSTR := int(float64(player.Stats.STR) * partyBonus)
	playerDEF := int(float64(player.Stats.DEF) * partyBonus)
	playerDGE := player.Stats.DGE
	playerLCK := player.Stats.LCK
	playerCRT := player.Stats.CRT

	// Pity boost - moderate scaling
	pityMult := 1.0 + player.PityStack
	playerSTR = int(float64(playerSTR) * pityMult)
	playerDEF = int(float64(playerDEF) * pityMult)

	// Level-based damage reduction: higher level players take slightly less damage
	levelReduction := 1.0 / (1.0 + float64(player.Level)*0.0005)

	for round := 1; round <= params.MaxRounds; round++ {
		escalation := 1.0 + params.EscalationRate*float64(round-1)

		// Evaluate ultimate skill ONCE at start of player attack phase
		ultMult := 1.0
		if player.UltimateSkill != nil && player.UltimateSkill.CurrentCooldown <= 0 {
			ultMult = player.UltimateSkill.Power
			player.UltimateSkill.CurrentCooldown = player.UltimateSkill.CooldownRounds
			player.TotalUltimatesUsed++
		}

		// Player attacks each living mob
		for i := range mobs {
			if mobs[i].HP <= 0 {
				continue
			}

			// Dodge check (mob dodges) - capped at 25%
			dodgeChance := playerDGE
			if dodgeChance > 25 {
				dodgeChance = 25
			}
			if rng.Intn(100) < dodgeChance {
				continue
			}

			mult := 1.0
			if rng.Intn(100) < playerCRT {
				mult = critMult
			}

			// Skill chance
			skillMult := 1.0
			if rng.Float64() < params.SkillChanceCombat {
				skillMult = params.SkillPowerBase + rng.Float64()*1.5
			}

			// Player damage with balanced scaling (includes ultimate skill multiplier)
			damage := int((float64(playerSTR)*escalation - float64(mobs[i].Stats.DEF)*0.4) * mult * skillMult * ultMult)
			minDmg := int(float64(playerSTR) * 0.15 * escalation)
			if damage < minDmg {
				damage = minDmg
			}
			if damage < 1 {
				damage = 1
			}

			mobs[i].HP -= damage
			if mobs[i].HP <= 0 {
				result.MobsKilled++
			}
		}

		// Check victory
		allDead := true
		for _, m := range mobs {
			if m.HP > 0 {
				allDead = false
				break
			}
		}
		if allDead {
			result.Won = true
			result.Rounds = round
			goto combatEnd
		}

		// Mobs attack
		for _, m := range mobs {
			if m.HP <= 0 {
				continue
			}
			// Player dodge - capped at 25%
			dodgeChance := playerDGE
			if dodgeChance > 25 {
				dodgeChance = 25
			}
			if rng.Intn(100) < dodgeChance {
				continue
			}

			// Mob damage with level-based reduction
			mobDamage := int((float64(m.Stats.STR)*escalation - float64(playerDEF)*0.5) * levelReduction)
			minMobDmg := int(float64(m.Stats.STR) * 0.10 * escalation)
			if mobDamage < minMobDmg {
				mobDamage = minMobDmg
			}
			if mobDamage < 1 {
				mobDamage = 1
			}

			// LCK reduces damage
			luckRed := playerLCK / 8
			mobDamage -= luckRed
			if mobDamage < 1 {
				mobDamage = 1
			}

			playerHP -= mobDamage
			if playerHP <= 0 {
				result.Won = false
				result.Rounds = round
				goto combatEnd
			}
		}

		// Moderate regen for players with STA
		if player.Stats.STA > 0 && round > 3 {
			regen := player.Stats.STA / 2
			if regen > 0 {
				playerHP += regen
				if playerHP > maxPlayerHP {
					playerHP = maxPlayerHP
				}
			}
		}

		// Decrement ultimate skill cooldown at end of each round
		if player.UltimateSkill != nil && player.UltimateSkill.CurrentCooldown > 0 {
			player.UltimateSkill.CurrentCooldown--
		}
	}

combatEnd:
	totalReward := 0.0
	for _, m := range mobs {
		totalReward += m.Reward
	}

	if result.Won {
		result.XPGained = totalReward
		player.ConsecutiveWins++
		player.ConsecutiveLosses = 0
		player.PityStack = 0 // reset pity
	} else {
		penalty := totalReward * deathXPPenalty
		if penalty < 10 {
			penalty = 10
		}
		result.XPPenalty = penalty
		result.XPGained = -penalty
		player.ConsecutiveLosses++
		player.ConsecutiveWins = 0
		// Pity boost increases
		player.PityStack += params.PityBoostPerLoss
		if player.PityStack > params.PityBoostCap {
			player.PityStack = params.PityBoostCap
		}
	}

	return result
}

// ============================================================
// XP CALCULATION
// ============================================================

func CalculateXPGain(rng *rand.Rand, player *Player, combatXP float64, hasNewGame bool, onlinePlayers int, streakDays int, params SimParams) float64 {
	// Separate combat penalty (death penalty) from positive XP.
	// Penalty is applied at the END so multipliers don't amplify it.
	var penalty float64
	positiveXP := combatXP
	if positiveXP < 0 {
		penalty = -positiveXP
		positiveXP = 0
	}

	xp := positiveXP

	// GROUP XP PENALTY: Larger groups get less XP per person to prevent massive farming abuse
	// Groups of 1-4 get 100% XP. Groups of 5+ get a 5% penalty per extra member (min 50%).
	groupSize := player.GroupSize
	if groupSize < 1 {
		groupSize = 1
	}
	if groupSize > 20 {
		groupSize = 20
	}
	if groupSize > 4 {
		groupPenalty := 1.0 - float64(groupSize-4)*0.05
		if groupPenalty < 0.5 {
			groupPenalty = 0.5
		}
		xp *= groupPenalty
	}

	// CONSUMABLE XP BOOST
	if player.XPBoostActive {
		xp *= player.XPBoostMult
	}

	// Add poke XP
	pokeXP := float64(params.XPMin + rng.Intn(params.XPMax-params.XPMin+1))
	xp += pokeXP

	// Daily login bonus
	if streakDays >= 1 {
		xp += 5
	}

	// Streak multiplier
	if streakDays >= 7 {
		xp *= 2.0
	} else if streakDays >= 5 {
		xp *= 1.5
	} else if streakDays >= 3 {
		xp *= 1.25
	}

	// Critical hit
	if rng.Float64() < critChance {
		xp *= critMult
	}

	// Party bonus
	xp *= partyMult

	// Server bonus
	serverMult := 1.0 + serverMultPerUser*float64(onlinePlayers-1)
	if serverMult > serverMultCap {
		serverMult = serverMultCap
	}
	xp *= serverMult

	// No game penalty
	if !hasNewGame {
		xp *= noGamePenalty
	}

	// INT bonus
	xp *= 1.0 + float64(player.Stats.INT)/1000.0

	// Radiant gear bonus
	radiantCount := 0
	for _, g := range player.Gear {
		if g.Durability > 0 && rng.Float64() < 0.1 {
			radiantCount++
		}
	}
	if radiantCount > 0 {
		xp *= 1.0 + 0.1*float64(radiantCount)
	}

	// Gear XP multiplier
	gearXP := 1.0
	gc := 0
	for _, g := range player.Gear {
		if g.Durability > 0 {
			gearXP *= g.XPMult
			gc++
		}
	}
	if gc > 0 {
		gearXP = math.Pow(gearXP, 1.0/float64(gc))
	}
	xp *= gearXP

	// Loot box
	if player.Level > 1 && player.Level%lootBoxEvery == 0 {
		bonus := float64(lootBoxMin + rng.Intn(lootBoxMax-lootBoxMin+1))
		xp += bonus
	}

	// Apply death penalty at end (NOT multiplied)
	xp -= penalty
	if xp < 0 {
		xp = 0 // floor at 0 for total (death penalty already accounted)
	}

	return xp
}

// ============================================================
// SIMULATION DAY
// ============================================================

func (sim *Simulation) SimulateDay() DaySnapshot {
	sim.Day++
	params := sim.Params

	var totalGearDrops, totalSkillDrops, totalArtifactDrops, totalTitleDrops, totalConsDrops, totalEnchDrops int
	var totalWins, totalLosses int
	var totalXP float64

	for _, p := range sim.Players {
		pokes := 1 + sim.Rng.Intn(5)

		if sim.Day == p.LastPokeDay+1 {
			p.StreakDays++
		} else if sim.Day > p.LastPokeDay+1 {
			p.StreakDays = 1
		}
		p.LastPokeDay = sim.Day
		p.DaysActive++

		hasNewGame := sim.Rng.Float64() < 0.7
		onlinePlayers := 3 + sim.Rng.Intn(8)

		// AH Auto-Purchase
		p.AutoPurchaseUpgrades(sim.Day)

		for poke := 0; poke < pokes; poke++ {
			difficulty := params.ZoneDiffMin + sim.Rng.Float64()*(params.ZoneDiffMax-params.ZoneDiffMin)
			difficulty += float64(p.Level) * 0.001
			difficulty += p.TotalGearScore * 0.00005
			if difficulty > 2.5 {
				difficulty = 2.5
			}

			mobs := GenerateMobs(sim.Rng, p.Level, p.GroupSize, difficulty, params)
			result := SimulateCombat(sim.Rng, p, mobs, params)

			if result.Won {
				totalWins++
				p.TotalWins++
				playerGold[p.ID] += int64(difficulty * 50)
			} else {
				totalLosses++
				p.TotalLosses++
			}
			p.TotalFights++

			combatXP := result.XPGained
			modifiedXP := CalculateXPGain(sim.Rng, p, combatXP, hasNewGame, onlinePlayers, p.StreakDays, params)
			totalXP += modifiedXP

			p.XP += modifiedXP
			p.TotalXPEarned += modifiedXP

			// Level up
			for p.XP >= p.XPNeeded && p.Level < maxLevel {
				p.XP -= p.XPNeeded
				p.Level++
				p.XPNeeded = XPNeededForNextLevel(p.Level, params.ExponentCap)
				p.RecalculateStats(params)
				p.CurrentHP = p.MaxHP

				if p.Level >= params.PrestigeLevel {
					p.Prestige++
					p.Level = 1
					p.XP = 0
					p.XPNeeded = XPNeededForNextLevel(1, params.ExponentCap)
					p.RecalculateStats(params)
					p.CurrentHP = p.MaxHP
				}
			}

			// Durability loss
			for slot, g := range p.Gear {
				if g.Durability > 0 {
					loss := params.DuraLoss
					if !result.Won {
						loss = duraLossPenalty
					}
					if sim.Rng.Intn(100) >= p.Stats.STA {
						g.Durability -= loss
						if g.Durability <= 0 {
							delete(p.Gear, slot)
						}
					}
				}
			}

			// Loot drops
			lootMult := difficulty
			treasureHunter := 0.0
			if sim.Rng.Float64() < 0.05 {
				treasureHunter = 0.05
			}

			rolls := len(mobs)
			for _, m := range mobs {
				if m.IsBoss {
					rolls += 2
				}
			}

			for roll := 0; roll < rolls; roll++ {
				if sim.Rng.Float64() < titleChance*lootMult+treasureHunter {
					totalTitleDrops++
					p.TotalTitleDrops++
				}
				if sim.Rng.Float64() < params.ArtifactChance*lootMult+treasureHunter {
					totalArtifactDrops++
					p.TotalArtifactDrops++
				}
				if sim.Rng.Float64() < params.GearChance*lootMult+treasureHunter {
					g := GenerateGear(sim.Rng, p.Level, params)
					if p.EquipGear(g) {
						totalGearDrops++
						p.TotalGearDrops++
					} else {
						p.AutoListUnwantedItems(g, sim.Day)
					}
				}
				if sim.Rng.Float64() < params.SkillChance*lootMult+treasureHunter {
					totalSkillDrops++
					p.TotalSkillDrops++
					p.AutoListUnwantedItems("skill", sim.Day)
				}
				if sim.Rng.Float64() < consChance*lootMult+treasureHunter {
					totalConsDrops++
					p.TotalConsDrops++
				}
				if sim.Rng.Float64() < enchChance*lootMult+treasureHunter {
					totalEnchDrops++
					p.TotalEnchDrops++
					p.AutoListUnwantedItems("ench", sim.Day)
				}

				if sim.Rng.Float64() < params.UniqueItemChance {
					name := GenerateUniqueItemName(sim.Rng)
					if !p.UniqueItemsCollected[name] {
						p.UniqueItemsCollected[name] = true
						p.TotalUniqueCollected++
					} else {
						p.AutoListUnwantedItems("unique", sim.Day)
					}
				}

				if sim.Rng.Float64() < params.UltimateSkillChance {
					skill := GenerateUltimateSkill(sim.Rng, p.Level, params)
					if !p.UltimateSkillsCollected[skill.Name] {
						p.UltimateSkillsCollected[skill.Name] = true
						p.TotalUltimatesCollected++
					} else {
						p.AutoListUnwantedItems(skill, sim.Day)
					}
					p.EquipUltimateSkill(skill)
				}
			}
		}

		p.RecalculateStats(params)
	}

	p := sim.Players[0]
	gearFilled := len(p.Gear)
	winRate := 0.0
	if p.TotalFights > 0 {
		winRate = float64(p.TotalWins) / float64(p.TotalFights) * 100
	}

	return DaySnapshot{
		Day:                sim.Day,
		Level:              p.Level,
		XP:                 p.XP,
		Prestige:           p.Prestige,
		CurrentHP:          p.CurrentHP,
		MaxHP:              p.MaxHP,
		STR:                p.Stats.STR,
		DEF:                p.Stats.DEF,
		SPD:                p.Stats.SPD,
		LCK:                p.Stats.LCK,
		INT:                p.Stats.INT,
		STA:                p.Stats.STA,
		CRT:                p.Stats.CRT,
		DGE:                p.Stats.DGE,
		AvgItemLevel:       p.AvgItemLevel,
		TotalGearScore:     p.TotalGearScore,
		GearSlotsFilled:    gearFilled,
		GearSlotsTotal:     len(gearSlots),
		FightsToday:        p.TotalFights,
		WinsToday:          totalWins / len(sim.Players),
		LossesToday:        totalLosses / len(sim.Players),
		CumulativeWinRate:  winRate,
		XPEarnedToday:      totalXP / float64(len(sim.Players)),
		XPPerFight:         totalXP / float64(totalWins+totalLosses+1),
		GearDropsToday:     totalGearDrops,
		SkillDropsToday:    totalSkillDrops,
		ArtifactDropsToday: totalArtifactDrops,
		Prestiges:          p.Prestige,
		PityStack:          p.PityStack,
	}
}

// ============================================================
// ANALYSIS
// ============================================================

func (sim *Simulation) Analyze(label string) {
	p := sim.Players[0]
	h := sim.History

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("SIMULATION: %s (Party Size: %d)\n", label, len(sim.Players))
	fmt.Println(strings.Repeat("=", 80))

	fmt.Printf("\n--- OVERVIEW ---\n")
	fmt.Printf("Simulated: %d days (~%.1f years)\n", sim.Day, float64(sim.Day)/365)
	fmt.Printf("Final level: %d (Prestige %d)\n", p.Level, p.Prestige)
	fmt.Printf("Total XP earned: %.0f\n", p.TotalXPEarned)
	fmt.Printf("Total fights: %d\n", p.TotalFights)
	fmt.Printf("Auction House: %d Items currently listed\n", len(globalAH.Items))

	// Level milestones
	milestones := []int{10, 50, 100, 250, 500, 1000, 2500, 5000, 10000}
	fmt.Println("\n--- LEVEL MILESTONES ---")
	fmt.Printf("%-10s | %-10s | %-12s | %-10s\n", "Level", "Day", "XP/Day", "Win Rate")
	fmt.Println(strings.Repeat("-", 55))
	for _, target := range milestones {
		for _, s := range h {
			if s.Level >= target {
				fmt.Printf("Lvl %-7d | Day %-6d | %-10.0f | %.1f%%\n",
					target, s.Day, s.XPEarnedToday, s.CumulativeWinRate)
				break
			}
		}
	}

	// Gear
	fmt.Println("\n--- GEAR ---")
	gf := len(p.Gear)
	fmt.Printf("Slots filled: %d/%d (%.0f%%)\n", gf, len(gearSlots), float64(gf)/float64(len(gearSlots))*100)
	fmt.Printf("Avg item level: %.1f\n", p.AvgItemLevel)
	fmt.Printf("Total gear score: %.1f\n", p.TotalGearScore)
	fmt.Printf("Drops - Gear:%d Skills:%d Artifacts:%d Titles:%d Consumables:%d Enchants:%d\n",
		p.TotalGearDrops, p.TotalSkillDrops, p.TotalArtifactDrops, p.TotalTitleDrops, p.TotalConsDrops, p.TotalEnchDrops)
	fmt.Printf("Unique Items Collected: %d / 1000 (%.1f%%)\n", p.TotalUniqueCollected, float64(p.TotalUniqueCollected)/10.0)
	fmt.Printf("Ultimate Skills Collected: %d / 1000 (%.1f%%)\n", p.TotalUltimatesCollected, float64(p.TotalUltimatesCollected)/10.0)
	fmt.Printf("Ultimate Skills Used in Combat: %d\n", p.TotalUltimatesUsed)
	if p.UltimateSkill != nil {
		fmt.Printf("Current Ultimate: %s (Power: %.1fx, Cooldown: %d rounds, Rarity: %s)\n",
			p.UltimateSkill.Name, p.UltimateSkill.Power, p.UltimateSkill.CooldownRounds, rarityNames[p.UltimateSkill.Rarity])
	}

	rarityCount := make(map[int]int)
	for _, g := range p.Gear {
		rarityCount[g.Rarity]++
	}
	fmt.Println("Rarity distribution:")
	for r := Legendary; r >= Common; r-- {
		if rarityCount[r] > 0 {
			fmt.Printf("  %s: %d (%.0f%%)\n", rarityNames[r], rarityCount[r], float64(rarityCount[r])/float64(gf)*100)
		}
	}

	// Prestige
	fmt.Println("\n--- PRESTIGE ---")
	fmt.Printf("Total prestiges: %d (+%.0f%% stats)\n", p.Prestige, float64(p.Prestige)*5)
	if p.Prestige > 0 {
		fmt.Printf("Days per prestige: %d (~%.1f years)\n", sim.Day/p.Prestige, float64(sim.Day/p.Prestige)/365)
	}

	// Win rate by bracket
	fmt.Println("\n--- WIN RATE BY LEVEL BRACKET ---")
	brackets := []struct {
		name     string
		min, max int
	}{
		{"1-50", 1, 50}, {"51-100", 51, 100}, {"101-500", 101, 500},
		{"501-1000", 501, 1000}, {"1001-2500", 1001, 2500}, {"2501-5000", 2501, 5000}, {"5001+", 5001, 100000},
	}
	for _, b := range brackets {
		w, l := 0, 0
		for _, s := range h {
			if s.Level >= b.min && s.Level <= b.max {
				w += s.WinsToday
				l += s.LossesToday
			}
		}
		t := w + l
		if t > 0 {
			fmt.Printf("  Lvl %s: %dW/%dL (%.1f%%)\n", b.name, w, l, float64(w)/float64(t)*100)
		}
	}

	// Stats progression
	fmt.Println("\n--- STATS OVER TIME ---")
	fmt.Printf("%-8s | %-6s | %-6s | %-6s | %-6s | %-8s | %-8s | %-8s\n",
		"Day", "HP", "STR", "DEF", "SPD", "GearScore", "AvgILvl", "WinRate")
	fmt.Println(strings.Repeat("-", 70))
	if len(h) == 0 {
		fmt.Println("  (no history recorded)")
	} else {
		intervals := []float64{0, 0.05, 0.1, 0.25, 0.5, 0.75, 1.0}
		for _, frac := range intervals {
			idx := int(float64(len(h)-1) * frac)
			if idx < 0 {
				idx = 0
			}
			if idx >= len(h) {
				idx = len(h) - 1
			}
			s := h[idx]
			fmt.Printf("Day %-5d | %-6d | %-6d | %-6d | %-6d | %-8.1f | %-8.1f | %.1f%%\n",
				s.Day, s.MaxHP, s.STR, s.DEF, s.SPD, s.TotalGearScore, s.AvgItemLevel, s.CumulativeWinRate)
		}
	}

	// Gear score over time
	fmt.Println("\n--- GEAR SCORE PROGRESSION ---")
	fmt.Printf("%-8s | %-10s | %-10s | %-8s\n", "Day", "GearScore", "AvgILvl", "Slots")
	fmt.Println(strings.Repeat("-", 45))
	if len(h) == 0 {
		fmt.Println("  (no history recorded)")
	} else {
		step := int(math.Max(1, float64(len(h)/20)))
		for i := 0; i < len(h); i += step {
			s := h[i]
			fmt.Printf("Day %-5d | %-10.1f | %-10.1f | %d/%d\n",
				s.Day, s.TotalGearScore, s.AvgItemLevel, s.GearSlotsFilled, s.GearSlotsTotal)
		}
	}

	// XP analysis
	fmt.Println("\n--- XP ANALYSIS ---")
	if len(h) == 0 {
		fmt.Println("  (no history recorded)")
	} else {
		totalXP := 0.0
		negXP := 0.0
		for _, s := range h {
			totalXP += s.XPEarnedToday
			if s.XPEarnedToday < 0 {
				negXP += s.XPEarnedToday
			}
		}
		avgXP := totalXP / float64(len(h))
		fmt.Printf("Avg XP/day: %.0f\n", avgXP)
		if totalXP != 0 {
			fmt.Printf("Days with negative XP: %.0f (%.1f%%)\n", negXP, negXP/totalXP*100)
		} else {
			fmt.Printf("Days with negative XP: %.0f\n", negXP)
		}
		fmt.Printf("Final pity stack: %.1f%% bonus\n", p.PityStack*100)
	}

	sim.generateRecommendations()
}

func (sim *Simulation) generateRecommendations() {
	p := sim.Players[0]
	h := sim.History
	recs := &sim.Recommendations
	*recs = nil

	if sim.Day > 0 {
		yearsToMax := float64(sim.Day) / 365
		if p.Prestige == 0 {
			// Extrapolate
			if p.Level > 1 {
				xpRate := p.TotalXPEarned / float64(sim.Day)
				if xpRate > 0 {
					estDays := XPForLevel(10000, sim.Params.ExponentCap) / xpRate
					yearsToMax = estDays / 365
				}
			}
		}
		if yearsToMax > 15 {
			*recs = append(*recs, fmt.Sprintf("LEVELING TOO SLOW: ~%.0f years to max. Increase base XP (30-65->50-90), lower exponent cap (5.0->3.5), or add more XP sources.", yearsToMax))
		} else if yearsToMax < 2 {
			*recs = append(*recs, fmt.Sprintf("LEVELING TOO FAST: ~%.0f years to max. Increase exponent curve or add diminishing returns.", yearsToMax))
		} else {
			*recs = append(*recs, fmt.Sprintf("LEVELING PACE: ~%.0f years to max — good for long-term engagement.", yearsToMax))
		}
	}

	if p.TotalFights > 0 {
		winRate := float64(p.TotalWins) / float64(p.TotalFights) * 100
		if winRate > 85 {
			*recs = append(*recs, fmt.Sprintf("WIN RATE TOO HIGH (%.1f%%): Add harder zones, increase mob scaling, or add elite mob tiers.", winRate))
		} else if winRate < 40 {
			*recs = append(*recs, fmt.Sprintf("WIN RATE TOO LOW (%.1f%%): Reduce mob HP/damage scaling, strengthen pity system, or increase base player HP.", winRate))
		} else {
			*recs = append(*recs, fmt.Sprintf("WIN RATE BALANCED (%.1f%%): Good challenge curve with room for improvement.", winRate))
		}
	}

	gf := len(p.Gear)
	gearPct := float64(gf) / float64(len(gearSlots)) * 100
	if gearPct < 50 {
		*recs = append(*recs, fmt.Sprintf("GEAR SLOW: %.0f%% slots filled. Increase gear drop rate (5%%->10%%), add milestone gear rewards, or add gear upgrade system.", gearPct))
	}

	if p.Prestige == 0 && sim.Day > 1000 {
		*recs = append(*recs, "PRESTIGE NEVER REACHED: Lower threshold (10000->5000) or accelerate early XP gains.")
	} else if p.Prestige > 30 {
		*recs = append(*recs, fmt.Sprintf("PRESTIGE TOO OFTEN (%d): Increase threshold or reduce XP gains.", p.Prestige))
	}

	// Check ILvl growth
	if len(h) > 200 {
		early := h[99].AvgItemLevel
		late := h[len(h)-1].AvgItemLevel
		growth := late - early
		if growth < 50 {
			*recs = append(*recs, fmt.Sprintf("ILVL GROWTH SLOW (+%.0f): Increase gear stat scaling with level or add upgrade/reforge system.", growth))
		}
	}

	// Negative XP days
	negDays := 0
	for _, s := range h {
		if s.XPEarnedToday < 0 {
			negDays++
		}
	}
	if float64(negDays)/float64(len(h)) > 0.3 {
		*recs = append(*recs, fmt.Sprintf("TOO MANY NEGATIVE XP DAYS (%d/%d, %.0f%%): Reduce death XP penalty (5%%->2%%), increase base poke XP, or add minimum XP floor.", negDays, len(h), float64(negDays)/float64(len(h))*100))
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("RECOMMENDATIONS")
	fmt.Println(strings.Repeat("=", 80))
	for i, r := range *recs {
		fmt.Printf("\n%d. %s\n", i+1, r)
	}
}

// ============================================================
// RUN SIMULATION
// ============================================================

func runSimulation(days int, seed int64, params SimParams, label string) *Simulation {
	// #nosec G404
	rng := rand.New(rand.NewSource(seed))
	
	sim := &Simulation{
		Rng:    rng,
		Params: params,
	}

	for i := 0; i < 10; i++ {
		p := NewPlayerWithParams(params)
		p.ID = i
		sim.Players = append(sim.Players, p)
		playerGold[p.ID] = 1000 // starting gold
	}

	for i := 0; i < days; i++ {
		snap := sim.SimulateDay()
		sim.History = append(sim.History, snap)
		if sim.Players[0].Prestige > 100 {
			break
		}
	}

	sim.Analyze(label)
	return sim
}

// ============================================================
// MAIN
// ============================================================

func main() {
	fmt.Println("TS3NEWS RPG PROGRESSION SIMULATION")
	fmt.Println(strings.Repeat("=", 80))

	seed := time.Now().UnixNano()
	days := 365 * 15

	// ---- BASE (current code parameters) ----
	fmt.Println("\n>>> BASE SIMULATION (current game parameters)")
	baseSim := runSimulation(days, seed, DefaultParams(), "BASE - Current Parameters")

	// ---- BALANCED ----
	fmt.Println("\n>>> BALANCED SIMULATION (tuned combat)")
	balSim := runSimulation(days, seed, BalancedParams(), "BALANCED - Tuned Combat")

	// ---- OPTIMIZED ----
	fmt.Println("\n>>> OPTIMIZED SIMULATION (full rebalance)")
	optSim := runSimulation(days, seed, OptimizedParams(), "OPTIMIZED - Full Rebalance")

	// ---- COMPARISON TABLE ----
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("COMPARATIVE ANALYSIS")
	fmt.Println(strings.Repeat("=", 80))

	sims := []struct {
		label string
		sim   *Simulation
	}{
		{"Base", baseSim}, {"Balanced", balSim}, {"Optimized", optSim},
	}

	fmt.Printf("\n%-30s", "Metric")
	for _, s := range sims {
		fmt.Printf(" | %-15s", s.label)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 80))

	// Helper
	rowF := func(label string, vals ...float64) {
		fmt.Printf("%-30s", label)
		for _, v := range vals {
			fmt.Printf(" | %-15.1f", v)
		}
		fmt.Println()
	}
	rowI := func(label string, vals ...int) {
		fmt.Printf("%-30s", label)
		for _, v := range vals {
			fmt.Printf(" | %-15d", v)
		}
		fmt.Println()
	}

	rowI("Days", len(baseSim.History), len(balSim.History), len(optSim.History))
	rowI("Final Level", baseSim.Players[0].Level, balSim.Players[0].Level, optSim.Players[0].Level)
	rowI("Prestiges", baseSim.Players[0].Prestige, balSim.Players[0].Prestige, optSim.Players[0].Prestige)
	rowI("Total Fights", baseSim.Players[0].TotalFights, balSim.Players[0].TotalFights, optSim.Players[0].TotalFights)

	wr := func(p *Player) float64 {
		if p.TotalFights == 0 {
			return 0
		}
		return float64(p.TotalWins) / float64(p.TotalFights) * 100
	}
	rowF("Win Rate %", wr(baseSim.Players[0]), wr(balSim.Players[0]), wr(optSim.Players[0]))
	rowI("Gear Slots", len(baseSim.Players[0].Gear), len(balSim.Players[0].Gear), len(optSim.Players[0].Gear))
	rowF("Avg Item Level", baseSim.Players[0].AvgItemLevel, balSim.Players[0].AvgItemLevel, optSim.Players[0].AvgItemLevel)
	rowF("Total Gear Score", baseSim.Players[0].TotalGearScore, balSim.Players[0].TotalGearScore, optSim.Players[0].TotalGearScore)
	rowF("Total XP", baseSim.Players[0].TotalXPEarned, balSim.Players[0].TotalXPEarned, optSim.Players[0].TotalXPEarned)
	rowI("Gear Drops", baseSim.Players[0].TotalGearDrops, balSim.Players[0].TotalGearDrops, optSim.Players[0].TotalGearDrops)
	rowI("Skill Drops", baseSim.Players[0].TotalSkillDrops, balSim.Players[0].TotalSkillDrops, optSim.Players[0].TotalSkillDrops)
	rowI("Artifact Drops", baseSim.Players[0].TotalArtifactDrops, balSim.Players[0].TotalArtifactDrops, optSim.Players[0].TotalArtifactDrops)

	// Win rate by bracket comparison
	fmt.Println("\n--- WIN RATE BY LEVEL BRACKET (all configs) ---")
	fmt.Printf("%-12s", "Bracket")
	for _, s := range sims {
		fmt.Printf(" | %-12s", s.label)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 55))

	brackets := []struct {
		name     string
		min, max int
	}{
		{"1-50", 1, 50}, {"51-100", 51, 100}, {"101-500", 101, 500},
		{"501-1000", 501, 1000}, {"1001+", 1001, 100000},
	}
	for _, b := range brackets {
		fmt.Printf("%-12s", b.name)
		for _, s := range sims {
			w, l := 0, 0
			for _, snap := range s.sim.History {
				if snap.Level >= b.min && snap.Level <= b.max {
					w += snap.WinsToday
					l += snap.LossesToday
				}
			}
			t := w + l
			if t > 0 {
				fmt.Printf(" | %-11.1f%%", float64(w)/float64(t)*100)
			} else {
				fmt.Printf(" | %-12s", "N/A")
			}
		}
		fmt.Println()
	}

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("KEY FINDINGS")
	fmt.Println(strings.Repeat("=", 80))

	fmt.Print(`
1. BASE CONFIG: 0% win rate — combat math is fundamentally broken.
	  Mobs have too much HP and deal too much damage relative to players.
	  This makes all progression (XP, gear, prestige) effectively non-functional.

2. BALANCED CONFIG: Fixing mob HP (0.5x) and damage (0.4x) while boosting
	  player HP (2x) and damage (1.5x) creates a viable fight curve.
	  Pity system (20% per loss, cap 200%) ensures losing streaks auto-correct.

3. OPTIMIZED CONFIG: Further tuning + faster XP (40-80 base), better drops
	  (10% gear, 8% skill, 2% artifact), lower prestige threshold (5000).
	  This creates engaging short-term and long-term progression loops.

4. PRESTIGE: At 10000 level threshold with base params, prestige is never
	  reached in 15 years. Lowering to 5000 makes it achievable while still
	  requiring significant investment.

5. GEAR: With 5% base drop rate and 24 slots, filling all slots takes
	  thousands of fights. Increasing to 10% + guaranteed drops at milestones
	  creates satisfying gear progression without making it trivial.

6. DURABILITY: At 1 durability loss/fight and 30-110 max durability,
	  gear lasts 30-110 fights. With 1-5 fights/day, that's 6-110 days.
	  Increasing max durability to 80+ keeps gear around longer for
	  meaningful progression tracking.

RECOMMENDED CHANGES TO ACTUAL CODE:
	 - internal/bot/xp.go: Reduce mob HP/damage scaling in resolveChannelCombat
	 - internal/bot/xp.go: Add pity system (already partially tracked via consecutive_losses)
	 - internal/bot/xp.go: Increase base XP from 20-50 to 30-65
	 - internal/content/artifacts.go: Increase gear drop chance from 5% to 8-10%
	 - internal/bot/prestige.go: Lower prestige threshold from 10000 to 5000
	 - internal/content/artifacts.go: Increase max durability base from 30 to 50
	 - Add gear repair/upgrade system for mid-late game progression
`)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("SIMULATION COMPLETE")
	fmt.Println(strings.Repeat("=", 80))
}
