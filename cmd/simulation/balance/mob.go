package main

import (
	"fmt"
	"math"
	"math/rand"
)

// SimMob represents a mob in the simulation, mirroring content.Mob.
type SimMob struct {
	Name        string
	Type        SimMobType
	Level       int
	HP          int
	MaxHP       int
	Stats       SimStats
	Element     SimElement
	Effects     []SimMobEffect
	Spells      []SimSkill
	DeathEffect *SimDeathEffect
	RewardXP    int
	RewardGold  int64
	STRMod      float64
	DEFMod      float64
	SPDMod      float64
}

type SimDeathEffect struct {
	Name string
	Type SimDeathEffectType
}

// Clone returns a deep copy of the mob.
func (m *SimMob) Clone() *SimMob {
	c := *m
	if m.Effects != nil {
		c.Effects = make([]SimMobEffect, len(m.Effects))
		copy(c.Effects, m.Effects)
	}
	if m.Spells != nil {
		c.Spells = make([]SimSkill, len(m.Spells))
		copy(c.Spells, m.Spells)
	}
	if m.DeathEffect != nil {
		de := *m.DeathEffect
		c.DeathEffect = &de
	}
	c.STRMod = m.STRMod
	c.DEFMod = m.DEFMod
	c.SPDMod = m.SPDMod
	return &c
}

// DisplayName returns a short combat description.
func (m *SimMob) DisplayName() string {
	typeNames := map[SimMobType]string{
		MobCommon: "Common", MobEliteMinion: "EliteMinion", MobElite: "Elite",
		MobMiniboss: "Miniboss", MobBoss: "Boss", MobLegendary: "Legendary",
	}
	tName := "Unknown"
	if n, ok := typeNames[m.Type]; ok {
		tName = n
	}
	eff := ""
	if len(m.Effects) > 0 {
		eff = fmt.Sprintf(" (%v)", m.Effects[0])
	}
	return fmt.Sprintf("Lvl%d %s [%s]%s (%d/%d HP)", m.Level, m.Name, tName, eff, m.HP, m.MaxHP)
}

// Score returns a combat rating for the mob.
func (m *SimMob) Score() int {
	return m.MaxHP/5 + m.Stats.STR + m.Stats.DEF + m.Stats.SPD + m.Level*10
}

// === Base Mob Templates ===
// Mirrors the baseMobs table in content/mobs.go

type baseMobTemplate struct {
	Name     string
	Type     SimMobType
	Stats    SimStats
	RewardXP int
}

var baseMobTemplates []baseMobTemplate

func init() {
	prefixes := []string{"Snotty", "Angry", "Undead", "Shadow", "Fiery", "Ice-Cold", "Toxic", "Ghostly", "Metallic", "Giant"}
	nouns := []string{"Rat", "Slime", "Goblin", "Spider", "Zombie", "Wolf", "Skeleton", "Bat", "Orc", "Troll"}

	// Common mobs (indices 0-99)
	for _, p := range prefixes {
		for _, n := range nouns {
			baseMobTemplates = append(baseMobTemplates, baseMobTemplate{
				Name: p + " " + n, Type: MobCommon,
				Stats:    SimStats{HP: 20, STR: 12, DEF: 2, SPD: 5, LCK: 0},
				RewardXP: 5,
			})
		}
	}

	// EliteMinions (indices 100-101)
	baseMobTemplates = append(baseMobTemplates,
		baseMobTemplate{Name: "Corrupted Guard", Type: MobEliteMinion, Stats: SimStats{HP: 60, STR: 25, DEF: 10, SPD: 7, LCK: 2}, RewardXP: 12},
		baseMobTemplate{Name: "Shadow Assassin", Type: MobEliteMinion, Stats: SimStats{HP: 50, STR: 35, DEF: 5, SPD: 15, LCK: 5}, RewardXP: 15},
	)

	// Elites (indices 102-103)
	baseMobTemplates = append(baseMobTemplates,
		baseMobTemplate{Name: "Dread Knight", Type: MobElite, Stats: SimStats{HP: 150, STR: 45, DEF: 20, SPD: 10, LCK: 5}, RewardXP: 25},
		baseMobTemplate{Name: "Frost Lich", Type: MobElite, Stats: SimStats{HP: 120, STR: 60, DEF: 15, SPD: 12, LCK: 8}, RewardXP: 30},
	)

	// Minibosses (indices 104-105)
	baseMobTemplates = append(baseMobTemplates,
		baseMobTemplate{Name: "Gatekeeper", Type: MobMiniboss, Stats: SimStats{HP: 400, STR: 80, DEF: 35, SPD: 15, LCK: 7}, RewardXP: 60},
		baseMobTemplate{Name: "Raging Behemoth", Type: MobMiniboss, Stats: SimStats{HP: 600, STR: 100, DEF: 20, SPD: 5, LCK: 3}, RewardXP: 70},
	)

	// Bosses (indices 106-107)
	baseMobTemplates = append(baseMobTemplates,
		baseMobTemplate{Name: "Ancient Dragon", Type: MobBoss, Stats: SimStats{HP: 1000, STR: 150, DEF: 50, SPD: 20, LCK: 10}, RewardXP: 100},
		baseMobTemplate{Name: "Kraken of the Deep", Type: MobBoss, Stats: SimStats{HP: 1200, STR: 130, DEF: 40, SPD: 15, LCK: 12}, RewardXP: 120},
	)

	// Legendaries (indices 108-109)
	baseMobTemplates = append(baseMobTemplates,
		baseMobTemplate{Name: "THE VOID LORD", Type: MobLegendary, Stats: SimStats{HP: 5000, STR: 450, DEF: 100, SPD: 50, LCK: 25}, RewardXP: 500},
		baseMobTemplate{Name: "CHRONOS, TIME EATER", Type: MobLegendary, Stats: SimStats{HP: 4500, STR: 500, DEF: 80, SPD: 100, LCK: 50}, RewardXP: 600},
	)
}

// SpawnMob creates a scaled mob for the given level and difficulty.
// Mirrors content.SpawnMob in mobs.go.
func SpawnMob(rng *rand.Rand, level int, isBoss bool, difficulty float64, params SimParams) SimMob {
	// Select base template
	idx := rng.Intn(100) // common mobs
	if isBoss && level >= 10 {
		idx = 106 + rng.Intn(2)
	} else if !isBoss {
		r := rng.Float64()
		switch {
		case r < 0.01 && level >= 25:
			idx = 108 + rng.Intn(2) // Legendary
		case r < 0.05 && level >= 10:
			idx = 106 + rng.Intn(2) // Boss
		case r < 0.12 && level >= 8:
			idx = 104 + rng.Intn(2) // Miniboss
		case r < 0.25 && level >= 5:
			idx = 102 + rng.Intn(2) // Elite
		case r < 0.40 && level >= 3:
			idx = 100 + rng.Intn(2) // EliteMinion
		}
	}

	tmpl := baseMobTemplates[idx]
	m := SimMob{
		Name:     tmpl.Name,
		Type:     tmpl.Type,
		Level:    level,
		Stats:    tmpl.Stats,
		RewardXP: tmpl.RewardXP,
		STRMod:   1.0,
		DEFMod:   1.0,
		SPDMod:   1.0,
	}

	// === BALANCED SCALING — mirrors content.SpawnMob ===
	lvlScale := 1.0 + params.MobLevelScale*float64(level-1)

	// Difficulty dampening: only 30% of difficulty applies
	effectiveDiff := 1.0 + (difficulty-1.0)*params.DifficultyDampen
	totalScale := lvlScale * effectiveDiff
	if totalScale < 0.1 {
		totalScale = 0.1
	}

	// Apply mob multipliers
	m.Stats.HP = int(float64(m.Stats.HP) * totalScale * params.MobHPMult)
	m.Stats.STR = int(float64(m.Stats.STR) * totalScale * params.MobSTRMult)

	// DEF scales at 50% the rate of STR/HP to prevent "DEF Wall"
	defScale := 1.0 + (totalScale-1.0)*0.5
	m.Stats.DEF = int(float64(m.Stats.DEF) * defScale * params.MobDEFMult)

	m.Stats.SPD = int(float64(m.Stats.SPD) * totalScale)

	// XP rewards scale fully
	m.RewardXP = int(float64(m.RewardXP) * lvlScale * difficulty)

	// Type XP bonuses
	switch m.Type {
	case MobEliteMinion:
		m.RewardXP = int(float64(m.RewardXP) * 1.2)
	case MobElite:
		m.RewardXP = int(float64(m.RewardXP) * 1.5)
	case MobMiniboss:
		m.RewardXP = int(float64(m.RewardXP) * 2.0)
	case MobBoss:
		m.RewardXP = int(float64(m.RewardXP) * 2.5)
	case MobLegendary:
		m.RewardXP = int(float64(m.RewardXP) * 5.0)
	}

	// Random mob effect (30% chance)
	if rng.Float64() < params.MobEffectChance {
		effects := []SimMobEffect{MobEffectEnraged, MobEffectArmored, MobEffectFleet, MobEffectPoisoned, MobEffectWeakened, MobEffectBlinded, MobEffectRegen}
		eff := effects[rng.Intn(len(effects))]
		m.Effects = append(m.Effects, eff)

		switch eff {
		case MobEffectEnraged, MobEffectArmored, MobEffectRegen:
			m.RewardXP = int(float64(m.RewardXP) * 1.3)
		case MobEffectFleet:
			m.RewardXP = int(float64(m.RewardXP) * 1.1)
		}
	}

	// Mob spells
	spellCount := params.MobSpellCount
	if isBoss || m.Type == MobLegendary || m.Type == MobMiniboss {
		spellCount = 2
	}
	for i := 0; i < spellCount; i++ {
		m.Spells = append(m.Spells, GenerateSkill(rng, level))
	}

	// Death effect
	deathChance := params.DeathEffectChance
	if m.Type == MobCommon {
		deathChance = 0.20
	}
	if rng.Float64() < deathChance {
		prefixes := []string{"Last", "Final", "Dying", "Bitter", "Vengeful", "Spiteful", "Desperate", "Echoing", "Ghostly", "Cursed"}
		actions := []string{"Roar", "Whimper", "Gasp", "Curse", "Blast", "Wail", "Howl", "Scream", "Sigh", "Command"}

		r := rng.Float64()
		var dType SimDeathEffectType
		switch {
		case r < params.DeathEffectSummonWeight:
			dType = DeathSummon
		case r < params.DeathEffectSummonWeight+params.DeathEffectExploWeight:
			dType = DeathExplosion
		case r < params.DeathEffectSummonWeight+params.DeathEffectExploWeight+params.DeathEffectCurseWeight:
			dType = DeathCurse
		case r < params.DeathEffectSummonWeight+params.DeathEffectExploWeight+params.DeathEffectCurseWeight+params.DeathEffectXPWeight:
			dType = DeathXP
		default:
			dType = DeathLoot
		}

		m.DeathEffect = &SimDeathEffect{
			Name: prefixes[rng.Intn(len(prefixes))] + " " + actions[rng.Intn(len(actions))],
			Type: dType,
		}
	}

	m.MaxHP = m.Stats.HP
	m.HP = m.MaxHP

	// Assign random element
	elements := []SimElement{ElementFire, ElementWater, ElementEarth, ElementAir}
	if rng.Float64() < params.MobElementChance {
		m.Element = elements[rng.Intn(len(elements))]
	} else {
		m.Element = ElementPhysical
	}

	// Gold reward proportional to XP
	m.RewardGold = int64(float64(m.RewardXP) * 0.75)

	return m
}

// SpawnMobGroup creates a group of mobs for a party encounter.
// Mirrors content.SpawnMobGroup in mobs.go.
func SpawnMobGroup(rng *rand.Rand, avgLevel int, difficulty float64, groupSize int, params SimParams) []SimMob {
	isHorde := rng.Float64() < params.HordeChance

	baseCount := params.BaseMobsMin + rng.Intn(params.BaseMobsMax-params.BaseMobsMin+1)

	if isHorde {
		baseCount = 5 + rng.Intn(6) // 5-10 mobs
	}

	// Dampen count scaling with difficulty
	count := int(float64(baseCount) * (1.0 + (difficulty-1.0)*0.3))
	if count < 1 {
		count = 1
	}
	if count > 12 {
		count = 12
	}

	// Group scaling multiplier
	groupMult := 1.0 + float64(groupSize-1)*params.GroupSizeScale
	if groupMult > params.GroupSizeCap {
		groupMult = params.GroupSizeCap
	}

	hasBoss := rng.Float64() < params.BossChance*difficulty && !isHorde

	var out []SimMob
	for i := 0; i < count; i++ {
		mob := SpawnMob(rng, avgLevel, hasBoss && i == 0, difficulty, params)

		// Apply group scaling
		mob.Stats.HP = int(float64(mob.Stats.HP) * groupMult)
		mob.Stats.STR = int(float64(mob.Stats.STR) * groupMult)
		mob.Stats.DEF = int(float64(mob.Stats.DEF) * groupMult)
		mob.Stats.SPD = int(float64(mob.Stats.SPD) * groupMult)

		if isHorde {
			levelMult := 0.5 + rng.Float64()*0.3
			mob.Level = maxInt(1, int(float64(mob.Level)*levelMult))
			mob.Stats.HP = int(float64(mob.Stats.HP) * levelMult)
			mob.Stats.STR = int(float64(mob.Stats.STR) * levelMult)
			mob.Stats.DEF = int(float64(mob.Stats.DEF) * levelMult)
			mob.Stats.SPD = int(float64(mob.Stats.SPD) * levelMult)
			mob.Name = "Horde " + mob.Name
			mob.RewardXP = int(float64(mob.RewardXP) * 0.6)
			mob.RewardGold = int64(float64(mob.RewardGold) * 0.6)
		}

		mob.MaxHP = mob.Stats.HP
		mob.HP = mob.MaxHP
		out = append(out, mob)
	}

	return out
}

// ApplyDeathEffect handles mob death effects.
// Mirrors bot.handleDeathEffects in xp.go.
func ApplyDeathEffects(mob *SimMob, mobs *[]*SimMob, players []*SimPlayer, avgLvl int, difficulty float64, rng *rand.Rand, params SimParams, logs *[]string) {
	if mob.DeathEffect == nil {
		return
	}

	*logs = append(*logs, fmt.Sprintf("⚠️ %s triggers %s: %s!", mob.Name, deathEffectTypeName(mob.DeathEffect.Type), mob.DeathEffect.Name))

	switch mob.DeathEffect.Type {
	case DeathSummon:
		count := 1
		if mob.Type == MobCommon {
			count = 3
		}
		for i := 0; i < count; i++ {
			lvl := avgLvl - 5
			if lvl < 1 {
				lvl = 1
			}
			newMob := SpawnMob(rng, lvl, false, difficulty*0.7, params)
			newMob.Name = "Summoned " + newMob.Name
			*mobs = append(*mobs, &newMob)
		}
		*logs = append(*logs, fmt.Sprintf("📢 %d reinforcements have arrived!", count))

	case DeathExplosion:
		dmg := mob.Level * 10
		*logs = append(*logs, fmt.Sprintf("💥 Explosion dealt %d damage to everyone!", dmg))
		for _, p := range players {
			if p.CurrentHP <= 0 {
				continue
			}
			p.CurrentHP -= dmg
			if p.CurrentHP <= 0 {
				p.CurrentHP = 0
				*logs = append(*logs, fmt.Sprintf("💀 %s was slain by explosion!", p.PlayerDisplay()))
			}
		}

	case DeathCurse:
		for _, p := range players {
			p.Stats.STR -= 10
			p.Stats.DEF -= 5
		}
		*logs = append(*logs, "🥀 A dark curse weakens the party!")

	case DeathXP:
		*logs = append(*logs, "✨ A pulse of pure energy provides bonus XP!")

	case DeathLoot:
		*logs = append(*logs, "💰 Shiny items scatter across the floor!")
	}
}

func deathEffectTypeName(t SimDeathEffectType) string {
	switch t {
	case DeathSummon:
		return "Summon"
	case DeathExplosion:
		return "Explosion"
	case DeathCurse:
		return "Curse"
	case DeathXP:
		return "Bonus XP"
	case DeathLoot:
		return "Loot Rain"
	default:
		return "Unknown"
	}
}

// GetAliveMobs returns all mobs with HP > 0.
func GetAliveMobs(mobs []*SimMob) []*SimMob {
	var out []*SimMob
	for _, m := range mobs {
		if m.HP > 0 {
			out = append(out, m)
		}
	}
	return out
}

// ApplyMobEffects processes per-round mob effects (poison, regen).
func ApplyMobEffects(mobs []*SimMob, round int, intensify float64, logs *[]string) {
	for _, m := range mobs {
		if m.HP <= 0 {
			continue
		}

		poisonStacks := 0
		regenStacks := 0
		for _, eff := range m.Effects {
			if eff == MobEffectPoisoned {
				poisonStacks++
			}
			if eff == MobEffectRegen {
				regenStacks++
			}
		}

		if poisonStacks > 0 {
			delta := int(float64(m.HP/20)*float64(poisonStacks)*intensify) * 1 // 5% per stack
			if delta < 1 {
				delta = 1
			}
			m.HP -= delta
			if round%3 == 0 {
				*logs = append(*logs, fmt.Sprintf("🤢 %s takes %d poison damage (%d stacks)!", m.Name, delta, poisonStacks))
			}
		}
		if regenStacks > 0 {
			delta := int(float64(m.MaxHP) * 0.05 * float64(regenStacks))
			if delta < 1 {
				delta = 1
			}
			m.HP += delta
			if m.HP > m.MaxHP {
				m.HP = m.MaxHP
			}
		}
	}
}

// ComputeEffectiveMobSTR applies mob effect modifiers to STR.
func ComputeEffectiveMobSTR(m *SimMob, fatigueMult float64) int {
	mSTR := int(float64(m.Stats.STR) * m.STRMod * fatigueMult)
	for _, eff := range m.Effects {
		switch eff {
		case MobEffectEnraged:
			mSTR = int(float64(mSTR) * 1.5)
		case MobEffectWeakened:
			mSTR = int(float64(mSTR) * 0.5)
		}
	}
	return mSTR
}

// ComputeEffectiveMobDEF applies mob effect modifiers to DEF.
func ComputeEffectiveMobDEF(m *SimMob) int {
	mDEF := int(float64(m.Stats.DEF) * m.DEFMod)
	for _, eff := range m.Effects {
		if eff == MobEffectArmored {
			mDEF = int(float64(mDEF) * 1.5)
		}
	}
	return mDEF
}

// RoundFloat rounds to nearest int.
func RoundFloat(f float64) int {
	return int(math.Round(f))
}
