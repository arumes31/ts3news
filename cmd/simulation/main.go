package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// ============================================================
// CONSTANTS (mirrored from internal packages)
// ============================================

const (
	xpMin             = 20
	xpMax             = 50
	maxLevel          = 10000
	prestigeLevel     = 5000 // Lowered to be achievable
	critChance        = 0.05
	critMult          = 3.0
	partyMult         = 1.25
	serverMultPerUser = 0.05
	serverMultCap     = 2.0
	noGamePenalty     = 0.5
	deathXPPenalty    = 0.05
	artifactChance    = 0.01
	titleChance       = 0.005
	gearChance        = 0.10 // Increased
	skillChance       = 0.08 // Increased
	consChance        = 0.10
	enchChance        = 0.03 // Increased
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
	"Pet1", "Pet2", "Emblem1", "Emblem2", "Banner", "Totem",
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

type UltimateSkill struct {
	Name            string
	Power           float64
	CooldownRounds  int
	CurrentCooldown int
	Rarity          int
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
	ExpiresAt int
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
	PityStack          float64
	GroupSize          int

	UniqueItemsCollected map[string]bool
	TotalUniqueCollected int

	XPBoostActive   bool
	XPBoostMult     float64
	StatBoostActive bool
	StatBoostMult   float64

	UltimateSkill *UltimateSkill
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
	GroupSize            int
	UniqueItemChance     float64
	UltimateSkillChance  float64
	UltimateCooldownBase int
	UltimatePowerBase    float64
}

func DefaultParams() SimParams {
	return SimParams{
		XPMin: 30, XPMax: 65, ExponentCap: 4.5,
		GearChance: 0.08, SkillChance: 0.06, ArtifactChance: 0.015,
		MaxDurability: 60, DuraLoss: 1, PrestigeLevel: 5000,
		MobHPMult: 1.0, MobDamageMult: 1.0,
		PlayerHPMult: 1.0, PlayerDMGMult: 1.0,
		SkillChanceCombat: 0.35, SkillPowerBase: 2.5,
		PityBoostPerLoss: 0.20, PityBoostCap: 2.0,
		ZoneDiffMin: 0.8, ZoneDiffMax: 2.0,
		MobScaling:    0.05, // 5% New scaling
		GearILvlScale: 12.0, GearStatScale: 0.03,
		BaseMobsMin: 2, BaseMobsMax: 5,
		BossChance: 0.12, BossHPMult: 3.0, BossDMGMult: 2.0,
		MaxRounds: 20, EscalationRate: 0.15,
		GroupSize:            1,
		UniqueItemChance:     0.012,
		UltimateSkillChance:  0.006,
		UltimateCooldownBase: 6,
		UltimatePowerBase:    4.5,
	}
}

func BalancedParams() SimParams {
	p := DefaultParams()
	p.MobScaling = 0.05
	p.ExponentCap = 3.5
	p.GearStatScale = 0.04
	p.PityBoostCap = 3.0
	return p
}

func OptimizedParams() SimParams {
	p := BalancedParams()
	p.XPMin = 45
	p.XPMax = 90
	p.GearChance = 0.12
	p.MaxDurability = 100
	p.PrestigeLevel = 3000
	p.ExponentCap = 3.2
	return p
}

func XPForLevel(level int, cap float64) float64 {
	if level <= 1 { return 0 }
	exponent := 1.2 + float64(level)/1500.0
	if exponent > cap { exponent = cap }
	return math.Round(math.Pow(float64(level-1), exponent) * 1.5)
}

func XPNeededForNextLevel(level int, cap float64) float64 {
	return XPForLevel(level+1, cap) - XPForLevel(level, cap)
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
	return Stats{
		HP:  int((100.0 + lvl*5.0) * params.PlayerHPMult),
		STR: int((10.0 + lvl*1.0) * params.PlayerDMGMult),
		DEF: int(5.0 + lvl*0.5),
		SPD: int(10.0 + lvl*1.0),
		LCK: int(lvl / 5.0),
		INT: int(lvl / 10.0),
		STA: int(lvl / 10.0),
		CRT: int(5.0 + lvl/50.0),
		DGE: int(5.0 + lvl/50.0),
	}
}

func (p *Player) PrestigeMult() float64 {
	return 1.0 + float64(p.Prestige)*0.10 // 10% per prestige
}

func (p *Player) AutoListUnwantedItems(item interface{}, day int) {
	var itype, name string
	var price int64
	var gear *Gear
	var ult *UltimateSkill

	switch v := item.(type) {
	case *Gear:
		if v.Rarity < Rare { return }
		itype = "gear"
		name = v.Slot
		gear = v
		price = int64(v.Stats.Score()*5) * int64(v.Rarity+1)
	case string:
		switch v {
		case "skill": itype, price = "skill", 500
		case "unique": itype, price = "unique", 3000
		case "ench": itype, price = "ench", 400
		}
	case *UltimateSkill:
		itype = "ultimate"
		name, ult = v.Name, v
		price = int64(v.Power*150) * int64(v.Rarity+1)
	default: return
	}

	if price < 10 { price = 10 }
	globalAH.Items = append(globalAH.Items, AuctionItem{
		ID: globalAH.NextID, SellerID: p.ID, Type: itype, Name: name, Gear: gear, Ult: ult, Price: price, ExpiresAt: day + 7,
	})
	globalAH.NextID++
}

func (p *Player) AutoPurchaseUpgrades(day int) {
	gold := playerGold[p.ID]
	for i := 0; i < len(globalAH.Items); i++ {
		item := globalAH.Items[i]
		if item.SellerID == p.ID || item.ExpiresAt < day || item.Price > gold { continue }

		isUpgrade := false
		switch item.Type {
		case "gear":
			if p.EquipGear(item.Gear) { isUpgrade = true }
		case "skill":
			isUpgrade = true // Simulation simplification
		case "ultimate":
			if _, exists := p.UltimateSkillsCollected[item.Name]; !exists {
				isUpgrade = true
				p.UltimateSkillsCollected[item.Name] = true
				p.TotalUltimatesCollected++
				if p.UltimateSkill == nil { p.UltimateSkill = item.Ult }
			}
		case "unique":
			if _, exists := p.UniqueItemsCollected[item.Name]; !exists {
				isUpgrade = true
				p.UniqueItemsCollected[item.Name] = true
				p.TotalUniqueCollected++
			}
		case "ench": isUpgrade = true
		}

		if isUpgrade {
			playerGold[p.ID] -= item.Price
			playerGold[item.SellerID] += item.Price
			globalAH.Items = append(globalAH.Items[:i], globalAH.Items[i+1:]...)
			i--
			gold = playerGold[p.ID]
		}
	}
}

func (p *Player) RecalculateStats(params SimParams) {
	base := p.BaseStats(params)
	gearStats := Stats{}
	gearScore, totalILvl, gearCount := 0.0, 0.0, 0
	for _, g := range p.Gear {
		if g.Durability > 0 {
			gearStats = gearStats.Add(g.Stats)
			gearScore += g.Stats.Score()
			totalILvl += g.ItemLevel
			gearCount++
		}
	}
	p.Stats = base.Add(gearStats).MulF(p.PrestigeMult())
	p.TotalGearScore = gearScore / 30.0 // Average score
	if gearCount > 0 { p.AvgItemLevel = totalILvl / float64(gearCount) }
	p.MaxHP = p.Stats.HP
	if p.CurrentHP > p.MaxHP || p.CurrentHP <= 0 { p.CurrentHP = p.MaxHP }
}

func GenerateGear(rng *rand.Rand, level int, params SimParams) *Gear {
	r := rng.Float64()
	var rarity int
	switch {
	case r < 0.40: rarity = Common
	case r < 0.70: rarity = Uncommon
	case r < 0.88: rarity = Rare
	case r < 0.96: rarity = Epic
	case r < 0.992: rarity = Legendary
	case r < 0.998: rarity = Mythic
	default: rarity = Divine
	}
	slot := gearSlots[rng.Intn(len(gearSlots))]
	mul := float64(rarity + 1)
	lvlScale := 1.0 + float64(level)*params.GearStatScale * 2.0
	stats := Stats{
		HP: int(200*mul*lvlScale), STR: int(100*mul*lvlScale), DEF: int(80*mul*lvlScale),
		SPD: int(20*mul*lvlScale), LCK: int(10*mul*lvlScale), INT: int(10*mul*lvlScale),
		STA: int(10*mul*lvlScale), CRT: int(5*mul*lvlScale), DGE: int(5*mul*lvlScale),
	}
	maxDur := 40 + rarity*25
	return &Gear{Slot: slot, Rarity: rarity, Durability: maxDur, MaxDur: maxDur, Stats: stats, XPMult: 1.0 + float64(rarity)*0.05, ItemLevel: float64(rarity+1)*lvlScale}
}

func (p *Player) EquipGear(g *Gear) bool {
	existing, ok := p.Gear[g.Slot]
	if ok && existing.Rarity >= g.Rarity && existing.ItemLevel >= g.ItemLevel { return false }
	p.Gear[g.Slot] = g
	return true
}

func (p *Player) RollUnique() {
	name := GenerateUniqueItemName(rand.New(rand.NewSource(time.Now().UnixNano())))
	if !p.UniqueItemsCollected[name] {
		p.UniqueItemsCollected[name] = true
		p.TotalUniqueCollected++
	} else {
		p.AutoListUnwantedItems("unique", 0)
	}
}

func (p *Player) RollUltimate() {
	// Dummy for now
	p.TotalUltimatesCollected++
}

var mobPrefixes = []string{"Feral", "Dark", "Shadow", "Cursed", "Wild", "Ancient", "Corrupt", "Savage", "Grim", "Vile"}
var mobNouns = []string{"Wolf", "Bear", "Wraith", "Goblin", "Skeleton", "Troll", "Spider", "Bat", "Slime", "Imp"}

func GenerateMobs(rng *rand.Rand, partyLevel int, groupSize int, difficulty float64, params SimParams) []Mob {
	isHorde := rng.Float64() < 0.15
	numMobs := params.BaseMobsMin + rng.Intn(params.BaseMobsMax-params.BaseMobsMin+1)
	if isHorde { numMobs = 6 + rng.Intn(6) }
	
	// Group scaling
	numMobs = int(float64(numMobs) * (1.0 + float64(groupSize-1)*0.2))
	if numMobs > 20 { numMobs = 20 }

	isBoss := rng.Float64() < params.BossChance && !isHorde
	mobs := make([]Mob, 0, numMobs)
	for i := 0; i < numMobs; i++ {
		mobLvl := int(float64(partyLevel) * (0.8 + rng.Float64()*0.4))
		if mobLvl < 1 { mobLvl = 1 }
		
		// Scaled lvlScale for simulation to be more forgiving at low levels
		lvlScale := 1.0 + params.MobScaling*float64(mobLvl-1)
		if mobLvl < 10 { lvlScale *= 0.5 } // Buff early game progression

		name := mobPrefixes[rng.Intn(len(mobPrefixes))] + " " + mobNouns[rng.Intn(len(mobNouns))]
		hp := int(float64(100+mobLvl*5) * lvlScale * params.MobHPMult * difficulty)
		str := int(float64(10+mobLvl) * lvlScale * params.MobDamageMult * difficulty)
		def := int(float64(5+mobLvl/2) * lvlScale * difficulty)

		mob := Mob{Name: name, Level: mobLvl, HP: hp, MaxHP: hp, Stats: Stats{HP: hp, STR: str, DEF: def, SPD: 10 + mobLvl}, Reward: float64(mobLvl)*15*difficulty}
		if i == 0 && isBoss {
			mob.Name, mob.HP, mob.Stats.STR, mob.Reward = "BOSS: "+name, mob.HP*3, int(float64(mob.Stats.STR)*1.5), mob.Reward*5
			mob.MaxHP, mob.Stats.HP, mob.IsBoss = mob.HP, mob.HP, true
		}
		mobs = append(mobs, mob)
	}
	return mobs
}

func SimulateCombat(rng *rand.Rand, players []*Player, mobs []Mob, params SimParams) CombatResult {
	result := CombatResult{}
	
	// Group Synergy: Bonus power based on party diversity/cooperation
	groupSynergy := 1.0 + float64(len(players)-1)*0.05
	partyBonus := (1.0 + float64(len(players)-1)*0.15) * groupSynergy
	if partyBonus > 5.0 { partyBonus = 5.0 }

	type combatPlayer struct { p *Player; hp, str, def int }
	cp := make([]combatPlayer, len(players))
	for i, p := range players {
		pm := 1.0 + p.PityStack
		cp[i] = combatPlayer{p: p, hp: p.CurrentHP, str: int(float64(p.Stats.STR)*partyBonus*pm), def: int(float64(p.Stats.DEF)*partyBonus*pm)}
	}

	for round := 1; round <= params.MaxRounds; round++ {
		if round == 1 && players[0].Level == 1 && rng.Intn(100) == 0 {
			fmt.Printf("DEBUG: Round 1 | Player[0] STR: %d HP: %d | Mob[0] STR: %d HP: %d\n", cp[0].str, cp[0].hp, mobs[0].Stats.STR, mobs[0].HP)
		}
		intensify := 1.0 + float64(round-1)*0.15
		fatigue := 1.0
		if round > 10 {
			fatigue = 1.0 - float64(round-10)*0.1
			if fatigue < 0.1 { fatigue = 0.1 }
		}

		// Players Turn
		for i := range cp {
			if cp[i].hp <= 0 { continue }
			// Find random target
			targetIdx := -1
			for j := range mobs {
				if mobs[j].HP > 0 {
					if targetIdx == -1 || rng.Intn(2) == 0 { targetIdx = j }
				}
			}
			if targetIdx == -1 { break }

			skillMult := 1.0
			if rng.Float64() < params.SkillChanceCombat { skillMult = params.SkillPowerBase }
			dmg := int((float64(cp[i].str)*skillMult - float64(mobs[targetIdx].Stats.DEF)*0.5) * intensify * fatigue)
			minDmg := int(float64(cp[i].str) * 0.15 * intensify)
			if dmg < minDmg { dmg = minDmg }
			mobs[targetIdx].HP -= dmg
			if mobs[targetIdx].HP <= 0 { result.MobsKilled++ }
		}

		// Check Victory
		allDead := true
		for _, m := range mobs { if m.HP > 0 { allDead = false; break } }
		if allDead { result.Won, result.Rounds = true, round; goto end }

		// Mobs Turn
		for j := range mobs {
			if mobs[j].HP <= 0 { continue }
			targetIdx := -1
			for i := range cp {
				if cp[i].hp > 0 {
					if targetIdx == -1 || rng.Intn(2) == 0 { targetIdx = i }
				}
			}
			if targetIdx == -1 { break }

			mobDmg := int((float64(mobs[j].Stats.STR)*intensify - float64(cp[targetIdx].def)*0.5) * fatigue)
			minMobDmg := int(float64(mobs[j].Stats.STR) * 0.25 * intensify)
			if mobDmg < minMobDmg { mobDmg = minMobDmg }
			cp[targetIdx].hp -= mobDmg
			if cp[targetIdx].hp < 0 { cp[targetIdx].hp = 0 }
		}

		// Check Defeat
		anyAlive := false
		for _, p := range cp { if p.hp > 0 { anyAlive = true; break } }
		if !anyAlive { result.Won, result.Rounds = false, round; goto end }

		// Regen
		for i := range cp {
			if cp[i].hp > 0 {
				cp[i].hp += cp[i].p.Stats.STA / 2
				if cp[i].hp > cp[i].p.MaxHP { cp[i].hp = cp[i].p.MaxHP }
			}
		}
	}

end:
	totalXP := 0.0
	for _, m := range mobs { totalXP += m.Reward }
	for i, p := range players {
		p.CurrentHP = cp[i].hp
		if result.Won {
			result.XPGained = totalXP / float64(len(players))
			p.ConsecutiveWins++; p.ConsecutiveLosses = 0; p.PityStack = 0
		} else {
			result.XPPenalty = (totalXP / float64(len(players))) * deathXPPenalty
			p.ConsecutiveWins = 0; p.ConsecutiveLosses++
			p.PityStack = math.Min(params.PityBoostCap, p.PityStack+params.PityBoostPerLoss)
		}
	}
	return result
}

func CalculateXPGain(rng *rand.Rand, p *Player, combatXP float64, params SimParams) float64 {
	xp := combatXP
	if xp < 0 { return xp } // penalty is raw
	
	// Multipliers
	mult := 1.0
	if p.StreakDays >= 7 { mult *= 2.0 } else if p.StreakDays >= 3 { mult *= 1.25 }
	if rng.Float64() < critChance { mult *= critMult }
	mult *= partyMult
	
	gearMult := 1.0
	for _, g := range p.Gear { if g.Durability > 0 { gearMult *= g.XPMult } }
	xp *= mult * math.Pow(gearMult, 1.0/30.0) // Average gear mult
	xp += 50 // Base activity XP
	return xp
}

func (sim *Simulation) SimulateDay() DaySnapshot {
	sim.Day++
	params := sim.Params
	var totalWins, totalLosses, totalGearDrops int
	var totalXP float64

	for _, p := range sim.Players {
		if sim.Day == p.LastPokeDay+1 { p.StreakDays++ } else if sim.Day > p.LastPokeDay+1 { p.StreakDays = 1 }
		p.LastPokeDay, p.DaysActive = sim.Day, p.DaysActive+1
		p.AutoPurchaseUpgrades(sim.Day)
	}

	for poke := 0; poke < 5; poke++ {
		partySize := 3
		for i := 0; i < len(sim.Players); i += partySize {
			end := i + partySize
			if end > len(sim.Players) { end = len(sim.Players) }
			party := sim.Players[i:end]
			if len(party) == 0 { continue }

			avgLvl := 0
			for _, pp := range party { avgLvl += pp.Level }
			avgLvl /= len(party)
			for _, pp := range party { pp.CurrentHP = pp.MaxHP }

			// Dynamic Mob Count: Spawn more if party is strong
			extraMobs := 0
			if len(party) > 0 && party[0].ConsecutiveWins > 5 {
				extraMobs = party[0].ConsecutiveWins / 5
				if extraMobs > 10 { extraMobs = 10 }
			}

			diff := params.ZoneDiffMin + sim.Rng.Float64()*(params.ZoneDiffMax-params.ZoneDiffMin)
			mobs := GenerateMobs(sim.Rng, avgLvl, len(party), diff, params)
			if extraMobs > 0 {
				added := GenerateMobs(sim.Rng, avgLvl, 1, diff, params)
				if len(added) > extraMobs { added = added[:extraMobs] }
				mobs = append(mobs, added...)
			}
			res := SimulateCombat(sim.Rng, party, mobs, params)

			if res.Won { 
				totalWins += len(party) 
				for _, pp := range party { 
					pp.TotalWins++
					playerGold[pp.ID] += 500 // Increased reward
				}
			} else { 
				totalLosses += len(party) 
			}

			for _, pp := range party {
				pp.TotalFights++
				gain := CalculateXPGain(sim.Rng, pp, res.XPGained, params)
				if !res.Won { gain = -res.XPPenalty }
				totalXP += gain
				pp.XP += gain
				pp.TotalXPEarned += gain

				// Leveling
				for pp.XP >= pp.XPNeeded && pp.Level < maxLevel {
					pp.XP -= pp.XPNeeded; pp.Level++; pp.XPNeeded = XPNeededForNextLevel(pp.Level, params.ExponentCap)
					pp.RecalculateStats(params)
					if pp.Level >= params.PrestigeLevel {
						pp.Prestige++; pp.Level = 1; pp.XP = 0; pp.XPNeeded = XPNeededForNextLevel(1, params.ExponentCap)
						pp.RecalculateStats(params)
					}
				}

				// Loot
				if res.Won {
					for roll := 0; roll < res.MobsKilled/len(party)+1; roll++ {
						if sim.Rng.Float64() < params.GearChance {
							g := GenerateGear(sim.Rng, pp.Level, params)
							if pp.EquipGear(g) { totalGearDrops++; pp.TotalGearDrops++ } else { pp.AutoListUnwantedItems(g, sim.Day) }
						}
					}
				}
				
				// Durability & Repair
				for slot, g := range pp.Gear {
					if g.Durability > 0 {
						g.Durability -= 1
						if !res.Won { g.Durability -= 2 }
						if g.Durability <= 0 { 
							if playerGold[pp.ID] >= 1 {
								playerGold[pp.ID] -= 1

								g.Durability = g.MaxDur
							} else {
								delete(pp.Gear, slot); pp.RecalculateStats(params) 
							}
						}
					}
				}
			}
		}
	}

	p := sim.Players[0]
	return DaySnapshot{
		Day: sim.Day, Level: p.Level, Prestige: p.Prestige, STR: p.Stats.STR, DEF: p.Stats.DEF, 
		TotalGearScore: p.TotalGearScore, GearSlotsFilled: len(p.Gear), 
		WinsToday: totalWins / len(sim.Players), LossesToday: totalLosses / len(sim.Players),
		XPEarnedToday: totalXP / float64(len(sim.Players)),
	}
}

func (sim *Simulation) Analyze(label string) {
	p := sim.Players[0]
	fmt.Printf("\nSIMULATION: %s\nLevel: %d (P%d) | Fights: %d | Wins: %d | WinRate: %.1f%% | Gear: %d/30\n",
		label, p.Level, p.Prestige, p.TotalFights, p.TotalWins, float64(p.TotalWins)/float64(p.TotalFights)*100, len(p.Gear))
}

func runSimulation(days int, seed int64, params SimParams, label string) *Simulation {
	rng := rand.New(rand.NewSource(seed))
	sim := &Simulation{Rng: rng, Params: params}
	for i := 0; i < 15; i++ {
		player := NewPlayerWithParams(params); player.ID = i
		// Starter Gear
		for _, s := range []string{"MainHand", "Chest", "Legs"} {
			g := GenerateGear(rng, 1, params)
			g.Slot = s
			g.Rarity = Uncommon
			player.EquipGear(g)
		}
		player.RecalculateStats(params)
		sim.Players = append(sim.Players, player)
		playerGold[i] = 1000
	}
	for i := 0; i < days; i++ {
		sim.History = append(sim.History, sim.SimulateDay())
		if sim.Players[0].Prestige > 10 { break }
	}
	sim.Analyze(label)
	return sim
}

func main() {
	fmt.Println("TS3NEWS RPG GROUP PROGRESSION SIMULATION")
	seed := time.Now().UnixNano()
	days := 365 * 10
	runSimulation(days, seed, DefaultParams(), "BASE - 5% Scaling & Party Combat")
	runSimulation(days, seed, BalancedParams(), "BALANCED - 5% Scaling & Party Combat")
	runSimulation(days, seed, OptimizedParams(), "OPTIMIZED - 5% Scaling & Party Combat")
}
