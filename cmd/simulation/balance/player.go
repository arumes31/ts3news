package main

import (
	"fmt"
	"math"
	"math/rand"
	"ts3news/internal/content"
)

// === Enum Types ===

type SimPosition int

const (
	PositionFrontline SimPosition = iota
	PositionBackline
)

type SimElement int

const (
	ElementPhysical SimElement = iota
	ElementFire
	ElementWater
	ElementEarth
	ElementAir
)

func (e SimElement) String() string {
	names := []string{"Physical", "Fire", "Water", "Earth", "Air"}
	if int(e) < len(names) {
		return names[e]
	}
	return "Unknown"
}

type SimRarity int

const (
	RarityCommon SimRarity = iota
	RarityUncommon
	RarityRare
	RarityEpic
	RarityLegendary
	RarityMythic
	RarityDivine
)

func (r SimRarity) String() string {
	names := []string{"Common", "Uncommon", "Rare", "Epic", "Legendary", "Mythic", "Divine"}
	if int(r) < len(names) {
		return names[r]
	}
	return "Unknown"
}

type SimItemEffect int

const (
	EffectNone SimItemEffect = iota
	EffectThorns
	EffectVampiric
	EffectBerserk
	EffectLucky
	EffectTreasureHunter
	EffectQuick
	EffectBulwark
	EffectRadiant
	EffectFragile
	EffectSteady
	EffectMindControl
	EffectRegenStack
	EffectCleanse
	EffectStealth
	EffectParry
	EffectPhoenix
)

type SimMobEffect int

const (
	MobEffectNone     SimMobEffect = iota
	MobEffectEnraged               // +50% STR
	MobEffectArmored               // +50% DEF
	MobEffectFleet                 // +50% SPD
	MobEffectPoisoned              // Loses 5% HP per round
	MobEffectWeakened              // -50% STR
	MobEffectBlinded               // 50% miss chance
	MobEffectRegen                 // Heals 5% HP per round
)

type SimDeathEffectType int

const (
	DeathNone SimDeathEffectType = iota
	DeathSummon
	DeathExplosion
	DeathCurse
	DeathXP
	DeathLoot
)

type SimMobType int

const (
	MobCommon SimMobType = iota
	MobEliteMinion
	MobElite
	MobMiniboss
	MobBoss
	MobLegendary
)

type SimSkillType int

const (
	SkillPhysical SimSkillType = iota
	SkillMagic
	SkillBuff
	SkillDebuff
)

type SimConsumableType int

const (
	ConsumableHealing SimConsumableType = iota
	ConsumableRevive
	ConsumableRepair
)

// === Data Structures ===

type SimStats struct {
	HP, STR, DEF, SPD, LCK int
	INT, STA, CRT, DGE     int
}

func (s SimStats) Add(o SimStats) SimStats {
	return SimStats{
		HP: s.HP + o.HP, STR: s.STR + o.STR, DEF: s.DEF + o.DEF,
		SPD: s.SPD + o.SPD, LCK: s.LCK + o.LCK, INT: s.INT + o.INT,
		STA: s.STA + o.STA, CRT: s.CRT + o.CRT, DGE: s.DGE + o.DGE,
	}
}

func (s SimStats) MulF(m float64) SimStats {
	return SimStats{
		HP: int(float64(s.HP) * m), STR: int(float64(s.STR) * m), DEF: int(float64(s.DEF) * m),
		SPD: int(float64(s.SPD) * m), LCK: int(float64(s.LCK) * m), INT: int(float64(s.INT) * m),
		STA: int(float64(s.STA) * m), CRT: int(float64(s.CRT) * m), DGE: int(float64(s.DGE) * m),
	}
}

func (s SimStats) Score() int {
	return s.HP/5 + s.STR + s.DEF + s.SPD + s.LCK + s.INT + s.STA + s.CRT + s.DGE
}

func (s SimStats) CombatRating() float64 {
	return float64(s.STR)*1.2 + float64(s.DEF)*0.9 + float64(s.HP)*0.3 +
		float64(s.SPD)*1.1 + float64(s.CRT)*1.5 + float64(s.DGE)*1.3 +
		float64(s.LCK)*0.8 + float64(s.INT)*0.7 + float64(s.STA)*0.6
}

type SimGear struct {
	Slot       string
	Rarity     SimRarity
	Durability int
	MaxDur     int
	Stats      SimStats
	XPMult     float64
	ItemLevel  float64
	Element    SimElement
	Effect     SimItemEffect
	Name       string
}

func (g SimGear) CombatRating() float64 {
	return g.Stats.CombatRating() * float64(g.Rarity+1)
}

type SimSkill struct {
	ID         string
	Name       string
	Type       SimSkillType
	Rarity     SimRarity
	Power      float64
	IgnoreDef  float64
	StunChance float64
	HealPct    float64
	Effect     SimItemEffect
}

type SimUltimate struct {
	Name            string
	Power           float64
	CooldownRounds  int
	CurrentCooldown int
	Rarity          SimRarity
}

type SimConsumable struct {
	ID          string
	Name        string
	Type        SimConsumableType
	EffectValue float64 // e.g. 0.3 = 30% HP heal
	Remaining   int     // fights remaining
}

type SimPet struct {
	Name  string
	Level int
	Stats SimStats
}

type SimPlayer struct {
	ID        int
	Level     int
	XP        float64
	Prestige  int
	CurrentHP int
	MaxHP     int
	Stats     SimStats

	Gear          map[string]*SimGear // slot -> gear
	Skills        []SimSkill
	UltimateSkill *SimUltimate
	TreeBonus     content.TreeBonus // Simulated passive tree bonus (Item 91)

	Gold              int64
	PityStack         float64
	ConsecutiveLosses int
	ConsecutiveWins   int

	TotalFights int
	TotalWins   int
	TotalLosses int

	Position    SimPosition
	Element     SimElement
	UniqueItems map[string]bool
	Pets        []*SimPet
	Consumables []SimConsumable
	ItemEffects []SimItemEffect
	LastSkillID string // combo tracking

	// Tracking
	TotalXPEarned   float64
	TotalGoldEarned int64
	TotalGearDrops  int
	TotalSkillDrops int
}

// NewSimPlayer creates a level-1 player with starter gear.
func NewSimPlayer(id int, params SimParams) *SimPlayer {
	p := &SimPlayer{
		ID:          id,
		Level:       1,
		Gear:        make(map[string]*SimGear),
		UniqueItems: make(map[string]bool),
		Position:    PositionFrontline,
		Element:     ElementPhysical,
		Gold:        2000,
	}
	p.RecalculateStats(params)
	p.CurrentHP = p.MaxHP
	return p
}

// BaseStats computes the player's base stats from level + params.
// Mirrors bot.calculateTotalStats: integer division like the real engine.
//
//	HP: BaseHP + level*5, STR: BaseSTR + level, DEF: BaseDEF + level/2
//	SPD: 10 + level, LCK: level/5, INT: level/10, STA: level/10
//	CRT: 5 + level/50, DGE: 5 + level/50
func (p *SimPlayer) BaseStats(params SimParams) SimStats {
	lvl := p.Level
	return SimStats{
		HP:  int((float64(params.BaseHP) + float64(lvl*5)) * params.PlayerHPMult),
		STR: int((float64(params.BaseSTR) + float64(lvl)) * params.PlayerSTRMult),
		DEF: int((float64(params.BaseDEF) + float64(lvl/2)) * params.PlayerDEFMult),
		SPD: 10 + lvl,
		LCK: lvl / 5,
		INT: lvl / 10,
		STA: lvl / 10,
		CRT: 5 + lvl/50,
		DGE: 5 + lvl/50,
	}
}

// PrestigeMult returns the permanent stat multiplier from prestige.
func (p *SimPlayer) PrestigeMult(prestigeStatBonus float64) float64 {
	return 1.0 + float64(p.Prestige)*prestigeStatBonus
}

// RecalculateStats rebuilds the player's effective stats from base + gear + prestige.
func getSimulatedTreeBonus(level int) content.TreeBonus {
	tree := content.AbyssTree()
	allocated := []int{}
	visited := map[int]bool{0: true}
	queue := []int{0}

	for len(queue) > 0 && len(allocated) < level-1 {
		curr := queue[0]
		queue = queue[1:]

		for _, nb := range tree.Adj[curr] {
			if nb > 0 && !visited[nb] && len(allocated) < level-1 {
				visited[nb] = true
				allocated = append(allocated, nb)
				queue = append(queue, nb)
			}
		}
	}
	return tree.BonusFor(allocated)
}

func (p *SimPlayer) RecalculateStats(params SimParams) {
	base := p.BaseStats(params)

	// Apply Skill Tree Simulator Integration (Item 91)
	p.TreeBonus = getSimulatedTreeBonus(p.Level)

	// Mirror live combat (buildAbyssUser): ApplyCombatPct runs on the full
	// base+gear+tree-flat total, so gear must be folded in before the
	// percent modifiers, not after.
	gearStats := SimStats{}
	for _, g := range p.Gear {
		if g.Durability > 0 {
			gearStats = gearStats.Add(g.Stats)
		}
	}
	base = base.Add(gearStats)

	base.HP += p.TreeBonus.Stats.HP
	base.STR += p.TreeBonus.Stats.STR
	base.DEF += p.TreeBonus.Stats.DEF
	base.SPD += p.TreeBonus.Stats.SPD
	base.LCK += p.TreeBonus.Stats.LCK
	base.INT += p.TreeBonus.Stats.INT
	base.STA += p.TreeBonus.Stats.STA
	base.CRT += p.TreeBonus.Stats.CRT
	base.DGE += p.TreeBonus.Stats.DGE

	// Apply percent multipliers
	if v := p.TreeBonus.Pct["str_pct"]; v != 0 {
		base.STR = int(float64(base.STR) * (1 + v))
	}
	if v := p.TreeBonus.Pct["hp_pct"]; v != 0 {
		base.HP = int(float64(base.HP) * (1 + v))
	}
	if v := p.TreeBonus.Pct["spd_pct"]; v != 0 {
		base.SPD = int(float64(base.SPD) * (1 + v))
	}
	if v := p.TreeBonus.Pct["int_pct"]; v != 0 {
		base.INT = int(float64(base.INT) * (1 + v))
	}

	// Apply Conversions
	if v := p.TreeBonus.Pct["str_to_spd"]; v != 0 {
		converted := int(float64(base.STR) * v)
		base.SPD += converted
		base.STR -= converted
	}
	if v := p.TreeBonus.Pct["hp_to_def"]; v != 0 {
		converted := int(float64(base.HP) * v)
		base.DEF += converted / 10
		base.HP -= converted
	}
	if v := p.TreeBonus.Pct["spd_to_dge"]; v != 0 {
		converted := int(float64(base.SPD) * v)
		base.DGE += converted
		base.SPD -= converted
	}


	// Apply Limit Break (Item 73) — driven by the node's configured value,
	// matching TreeBonus.ApplyCombatPct's 1+v.
	if v := p.TreeBonus.Pct["limit_break"]; v != 0 {
		mult := 1 + v
		base.STR = int(float64(base.STR) * mult)
		base.HP = int(float64(base.HP) * mult)
		base.DEF = int(float64(base.DEF) * mult)
		base.SPD = int(float64(base.SPD) * mult)
		base.INT = int(float64(base.INT) * mult)
	}

	prestigeMult := 1.0 + float64(p.Prestige)*params.PrestigeStatBonus
	p.Stats = base.MulF(prestigeMult)
	p.MaxHP = p.Stats.HP
	if p.CurrentHP > p.MaxHP || p.CurrentHP <= 0 {
		p.CurrentHP = p.MaxHP
	}
}

// EquipGear equips gear if it's an upgrade. Returns true if equipped.
func (p *SimPlayer) EquipGear(g *SimGear) bool {
	existing, ok := p.Gear[g.Slot]
	if ok && existing.Rarity >= g.Rarity && existing.ItemLevel >= g.ItemLevel {
		return false
	}
	p.Gear[g.Slot] = g
	return true
}

// HasEffect checks if the player has a specific item effect.
func (p *SimPlayer) HasEffect(eff SimItemEffect) bool {
	for _, e := range p.ItemEffects {
		if e == eff {
			return true
		}
	}
	return false
}

// Lifesteal returns the lifesteal percentage from item effects.
func (p *SimPlayer) Lifesteal() int {
	ls := 0
	for _, e := range p.ItemEffects {
		if e == EffectVampiric {
			ls += 5
		}
	}
	// Lifesteal from Defensive Conversion (Item 37)
	if v := p.TreeBonus.Pct["def_to_lifesteal"]; v > 0 {
		ls += int(float64(p.Stats.DEF) * v)
	}
	return ls
}

// MultiStrike returns the multi-strike chance from item effects.
func (p *SimPlayer) MultiStrike() int {
	// Simplified: 10% per berserk effect
	ms := 0
	for _, e := range p.ItemEffects {
		if e == EffectBerserk {
			ms += 10
		}
	}
	return ms
}

// MindControlLevel returns the mind control level from gear + skills.
func (p *SimPlayer) MindControlLevel() int {
	lvl := 0
	for _, g := range p.Gear {
		if g.Effect == EffectMindControl {
			lvl += int(g.Rarity) + 1
		}
	}
	for _, s := range p.Skills {
		if s.Effect == EffectMindControl {
			lvl += int(s.Rarity) + 1
		}
	}
	return lvl
}

// PartyBonus computes the group synergy + party multiplier.
// Mirrors: groupSynergy = 1 + (n-1)*0.05, partyBonus = (1 + (n-1)*0.15) * groupSynergy, cap 5.0
func PartyBonus(groupSize int) float64 {
	if groupSize <= 1 {
		return 1.0
	}
	groupSynergy := 1.0 + float64(groupSize-1)*0.05
	partyBonus := (1.0 + float64(groupSize-1)*0.15) * groupSynergy
	if partyBonus > 5.0 {
		partyBonus = 5.0
	}
	return partyBonus
}

// GetElementMult returns the elemental damage multiplier.
// Fire > Air > Earth > Water > Fire
func GetElementMult(attacker, defender SimElement) float64 {
	switch attacker {
	case ElementFire:
		if defender == ElementAir {
			return 2.0
		}
		if defender == ElementWater {
			return 0.5
		}
	case ElementAir:
		if defender == ElementEarth {
			return 2.0
		}
		if defender == ElementFire {
			return 0.5
		}
	case ElementEarth:
		if defender == ElementWater {
			return 2.0
		}
		if defender == ElementAir {
			return 0.5
		}
	case ElementWater:
		if defender == ElementFire {
			return 2.0
		}
		if defender == ElementEarth {
			return 0.5
		}
	}
	return 1.0
}

// RollRarity generates a random gear rarity using the weighted distribution.
func RollRarity(rng *rand.Rand) SimRarity {
	r := rng.Float64()
	switch {
	case r < 0.40:
		return RarityCommon
	case r < 0.70:
		return RarityUncommon
	case r < 0.88:
		return RarityRare
	case r < 0.96:
		return RarityEpic
	case r < 0.992:
		return RarityLegendary
	case r < 0.998:
		return RarityMythic
	default:
		return RarityDivine
	}
}

// GearSlots mirrors the real engine's 30 gear slots.
var GearSlots = []string{
	"Head", "Neck", "Shoulders", "Back", "Chest", "Wrists", "Hands", "Waist",
	"Legs", "Feet", "Finger1", "Finger2", "Trinket1", "Trinket2", "MainHand",
	"OffHand", "Ranged", "Relic", "Artifact", "Soul", "Aura", "Charm", "Mount", "Companion",
	"Pet1", "Pet2", "Emblem1", "Emblem2", "Banner", "Totem",
}

// GenerateGear creates a random gear item for the given level.
func GenerateGear(rng *rand.Rand, level int, _ SimParams) *SimGear {
	rarity := RollRarity(rng)
	slot := GearSlots[rng.Intn(len(GearSlots))]
	// Mirrors real bot gear stats from internal/content/artifacts.go:
	//   Common (Novice): HP=10, STR=2, DEF=2, SPD=2
	//   Uncommon:        HP=20, STR=10, DEF=6, SPD=8
	//   Rare:            HP=30, STR=15, DEF=9, SPD=12
	//   Epic:            HP=40, STR=20, DEF=12, SPD=16
	//   Legendary:       HP=50, STR=25, DEF=15, SPD=20
	// mul = rarity+1 gives: Common=1, Uncommon=2, Rare=3, Epic=4, Legendary=5
	mul := float64(rarity + 1)
	// Gentle level scaling: gear improves ~1% per level, capped at 3x at level 200+
	lvlScale := 1.0 + float64(level)*0.01
	if lvlScale > 3.0 {
		lvlScale = 3.0
	}

	stats := SimStats{
		HP:  int(10 * mul * lvlScale),
		STR: int(5 * mul * lvlScale),
		DEF: int(3 * mul * lvlScale),
		SPD: int(4 * mul * lvlScale),
		LCK: int(2 * mul * lvlScale),
		INT: int(float64(rarity) * lvlScale),
		STA: int(float64(rarity) * lvlScale),
		CRT: int(2 * mul * lvlScale),
		DGE: int(2 * mul * lvlScale),
	}

	maxDur := 50 + int(rarity)*30

	// Random element for weapons
	element := ElementPhysical
	if slot == "MainHand" || slot == "OffHand" || slot == "Ranged" {
		elements := []SimElement{ElementFire, ElementWater, ElementEarth, ElementAir}
		if rng.Float64() < 0.3 {
			element = elements[rng.Intn(len(elements))]
		}
	}

	// Random item effect for rare+ gear
	effect := EffectNone
	if rarity >= RarityRare {
		effects := []SimItemEffect{
			EffectThorns, EffectVampiric, EffectBerserk, EffectLucky,
			EffectTreasureHunter, EffectQuick, EffectBulwark, EffectRadiant,
			EffectFragile, EffectSteady, EffectMindControl, EffectRegenStack,
			EffectCleanse, EffectStealth, EffectParry, EffectPhoenix,
		}
		effect = effects[rng.Intn(len(effects))]
	}

	// Generate name
	adjectives := []string{"Sturdy", "Fierce", "Ancient", "Glowing", "Shadow", "Frost", "Flame", "Storm", "Void", "Holy"}
	nouns := []string{"Blade", "Shield", "Helm", "Plate", "Boots", "Ring", "Amulet", "Staff", "Bow", "Cloak"}
	name := adjectives[rng.Intn(len(adjectives))] + " " + nouns[rng.Intn(len(nouns))]

	return &SimGear{
		Slot:       slot,
		Rarity:     rarity,
		Durability: maxDur,
		MaxDur:     maxDur,
		Stats:      stats,
		XPMult:     1.0 + float64(rarity)*0.05,
		ItemLevel:  float64(rarity+1) * lvlScale,
		Element:    element,
		Effect:     effect,
		Name:       name,
	}
}

// GenerateSkill creates a random skill.
func GenerateSkill(rng *rand.Rand, level int) SimSkill {
	prefixes := []string{"Mortal", "Heroic", "Flash", "Greater", "Chaos", "Shadow", "Holy", "Frost", "Fire", "Arcane"}
	actions := []string{"Strike", "Blast", "Roar", "Slash", "Burst", "Heal", "Shield", "Nova", "Drain", "Bolt"}

	rarity := RollRarity(rng)
	power := 1.1 + float64(rarity)*0.3 + float64(level)*0.002
	if power > 5.0 {
		power = 5.0
	}

	name := prefixes[rng.Intn(len(prefixes))] + " " + actions[rng.Intn(len(actions))]

	// Determine type
	sType := SkillPhysical
	if rng.Float64() < 0.3 {
		sType = SkillMagic
	}
	if rng.Float64() < 0.15 {
		sType = SkillBuff
	}

	ignoreDef := 0.0
	if sType == SkillMagic && rng.Float64() < 0.3 {
		ignoreDef = 0.25 + rng.Float64()*0.25
	}

	stunChance := 0.0
	if sType == SkillPhysical && rng.Float64() < 0.2 {
		stunChance = 0.1 + rng.Float64()*0.15
	}

	healPct := 0.0
	if sType == SkillBuff {
		healPct = 0.05 + rng.Float64()*0.1
	}

	return SimSkill{
		ID:         fmt.Sprintf("SIM_%d_%s", level, name),
		Name:       name,
		Type:       sType,
		Rarity:     rarity,
		Power:      power,
		IgnoreDef:  ignoreDef,
		StunChance: stunChance,
		HealPct:    healPct,
	}
}

// GenerateUltimate creates a random ultimate skill.
func GenerateUltimate(rng *rand.Rand, level int) *SimUltimate {
	verbs := []string{"Annihilating", "Devastating", "Obliterating", "Shattering", "Eradicating"}
	nouns := []string{"Strike", "Blast", "Wave", "Storm", "Fury"}

	rarity := RollRarity(rng)
	if rarity < RarityEpic {
		rarity = RarityEpic // Ultimates are always Epic+
	}

	power := 3.0 + float64(rarity)*1.0 + float64(level)*0.001
	if power > 10.0 {
		power = 10.0
	}

	return &SimUltimate{
		Name:            verbs[rng.Intn(len(verbs))] + " " + nouns[rng.Intn(len(nouns))],
		Power:           power,
		CooldownRounds:  5,
		CurrentCooldown: 0,
		Rarity:          rarity,
	}
}

// GenerateConsumable creates a random consumable.
func GenerateConsumable(rng *rand.Rand, _ int) SimConsumable {
	types := []SimConsumableType{ConsumableHealing, ConsumableHealing, ConsumableHealing, ConsumableRevive, ConsumableRepair}
	cType := types[rng.Intn(len(types))]

	var id, name string
	var effectValue float64

	switch cType {
	case ConsumableHealing:
		id = "P_HEAL"
		name = "Health Potion"
		effectValue = 0.2 + rng.Float64()*0.3 // 20-50% HP
	case ConsumableRevive:
		id = "P_REVIVE"
		name = "Phoenix Feather"
		effectValue = 0.5 // Revive at 50% HP
	case ConsumableRepair:
		id = "P_REPAIR"
		name = "Repair Kit"
		effectValue = 30 // 30 durability restored
	}

	return SimConsumable{
		ID:          id,
		Name:        name,
		Type:        cType,
		EffectValue: effectValue,
		Remaining:   5 + rng.Intn(10),
	}
}

// PlayerDisplay returns a short description of the player.
func (p *SimPlayer) PlayerDisplay() string {
	return fmt.Sprintf("P%d Lvl%d(P%d) HP:%d/%d STR:%d DEF:%d Gear:%d/%d Gold:%d",
		p.ID, p.Level, p.Prestige, p.CurrentHP, p.MaxHP,
		p.Stats.STR, p.Stats.DEF, len(p.Gear), len(GearSlots), p.Gold)
}

// maxInt returns the larger of two ints.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// clampFloat clamps a float between min and max.
func clampFloat(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
