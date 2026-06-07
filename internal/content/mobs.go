package content

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

type MobType string

const (
	MobCommon      MobType = "Common"
	MobEliteMinion MobType = "EliteMinion"
	MobElite       MobType = "Elite"
	MobMiniboss    MobType = "Miniboss"
	MobBoss        MobType = "Boss"
	MobLegendary   MobType = "Legendary"
)

type MobEffect string

const (
	EffectEnraged  MobEffect = "Enraged"      // +50% STR
	EffectArmored  MobEffect = "Armored"      // +50% DEF
	EffectFleet    MobEffect = "Fleet-foot"   // +50% SPD
	EffectPoisoned MobEffect = "Poisoned"     // Loses 5% HP per round
	EffectWeakened MobEffect = "Weakened"     // -50% STR
	EffectBlinded  MobEffect = "Blinded"      // 50% miss chance
	EffectRegen    MobEffect = "Regenerative" // Heals 5% HP per round
)

type DeathEffectType string

const (
	DeathSummon    DeathEffectType = "Summon"
	DeathExplosion DeathEffectType = "Explosion"
	DeathCurse     DeathEffectType = "Curse"
	DeathXP        DeathEffectType = "Bonus XP"
	DeathLoot      DeathEffectType = "Loot Rain"
)

type MobDeathEffect struct {
	Name string
	Type DeathEffectType
}

type Mob struct {
	Name        string
	Type        MobType
	Level       int
	Stats       Stats
	CurrentHP   int
	MaxHP       int
	RewardXP    int
	Effects     []MobEffect
	Spells      []Skill
	Equipped    []Gear
	DeathEffect *MobDeathEffect
}

func (m Mob) Clone() *Mob {
	newMob := m
	// Deep copy slices
	if m.Effects != nil {
		newMob.Effects = make([]MobEffect, len(m.Effects))
		copy(newMob.Effects, m.Effects)
	}
	if m.Spells != nil {
		newMob.Spells = make([]Skill, len(m.Spells))
		copy(newMob.Spells, m.Spells)
	}
	if m.Equipped != nil {
		newMob.Equipped = make([]Gear, len(m.Equipped))
		copy(newMob.Equipped, m.Equipped)
	}
	return &newMob
}

func (m Mob) DisplayName() string {
	eff := ""
	if len(m.Effects) > 0 {
		eff = fmt.Sprintf(" (%s)", m.Effects[0])
	}
	if m.DeathEffect != nil {
		eff += fmt.Sprintf(" [death:%s]", m.DeathEffect.Name)
	}
	return fmt.Sprintf("Lvl %d %s [%s]%s (%d/%d HP)", m.Level, m.Name, m.Type, eff, m.CurrentHP, m.MaxHP)
}

func (m Mob) Score() int {
	return m.MaxHP/5 + m.Stats.STR + m.Stats.DEF + m.Stats.SPD + m.Level*10
}

var baseMobs []Mob

func init() {
	prefixes := []string{"Snotty", "Angry", "Undead", "Shadow", "Fiery", "Ice-Cold", "Toxic", "Ghostly", "Metallic", "Giant"}
	nouns := []string{"Rat", "Slime", "Goblin", "Spider", "Zombie", "Wolf", "Skeleton", "Bat", "Orc", "Troll"}

	for _, p := range prefixes {
		for _, n := range nouns {
			name := p + " " + n
			baseMobs = append(baseMobs, Mob{
				Name:     name,
				Type:     MobCommon,
				Stats:    Stats{HP: 20, STR: 12, DEF: 2, SPD: 5, LCK: 0},
				RewardXP: 5,
			})
		}
	}

	// EliteMinions (stronger common)
	baseMobs = append(baseMobs, Mob{Name: "Corrupted Guard", Type: MobEliteMinion, Stats: Stats{HP: 60, STR: 25, DEF: 10, SPD: 7, LCK: 2}, RewardXP: 12})
	baseMobs = append(baseMobs, Mob{Name: "Shadow Assassin", Type: MobEliteMinion, Stats: Stats{HP: 50, STR: 35, DEF: 5, SPD: 15, LCK: 5}, RewardXP: 15})

	// Elites
	baseMobs = append(baseMobs, Mob{Name: "Dread Knight", Type: MobElite, Stats: Stats{HP: 150, STR: 45, DEF: 20, SPD: 10, LCK: 5}, RewardXP: 25})
	baseMobs = append(baseMobs, Mob{Name: "Frost Lich", Type: MobElite, Stats: Stats{HP: 120, STR: 60, DEF: 15, SPD: 12, LCK: 8}, RewardXP: 30})

	// Minibosses (between Elite and Boss)
	baseMobs = append(baseMobs, Mob{Name: "Gatekeeper", Type: MobMiniboss, Stats: Stats{HP: 400, STR: 80, DEF: 35, SPD: 15, LCK: 7}, RewardXP: 60})
	baseMobs = append(baseMobs, Mob{Name: "Raging Behemoth", Type: MobMiniboss, Stats: Stats{HP: 600, STR: 100, DEF: 20, SPD: 5, LCK: 3}, RewardXP: 70})

	// Bosses
	baseMobs = append(baseMobs, Mob{Name: "Ancient Dragon", Type: MobBoss, Stats: Stats{HP: 1000, STR: 150, DEF: 50, SPD: 20, LCK: 10}, RewardXP: 100})
	baseMobs = append(baseMobs, Mob{Name: "Kraken of the Deep", Type: MobBoss, Stats: Stats{HP: 1200, STR: 130, DEF: 40, SPD: 15, LCK: 12}, RewardXP: 120})

	// Legendaries
	baseMobs = append(baseMobs, Mob{Name: "THE VOID LORD", Type: MobLegendary, Stats: Stats{HP: 5000, STR: 450, DEF: 100, SPD: 50, LCK: 25}, RewardXP: 500})
	baseMobs = append(baseMobs, Mob{Name: "CHRONOS, TIME EATER", Type: MobLegendary, Stats: Stats{HP: 4500, STR: 500, DEF: 80, SPD: 100, LCK: 50}, RewardXP: 600})
}

// SpawnMob scales a mob to the given level and difficulty factor (0.1 to 1.0+)
func SpawnMob(level int, isBoss bool, difficulty float64) Mob {
	// #nosec G404
	idx := rand.IntN(100)      // index for common mobs // #nosec G404
	if isBoss && level >= 10 { // Bosses require level 10+
		idx = 106 + rand.IntN(2) // Bosses: 106-107
	}

	m := baseMobs[idx]
	if !isBoss {
		// #nosec G404
		r := rand.Float64()          // #nosec G404
		if r < 0.01 && level >= 25 { // Legendaries require level 25+
			m = baseMobs[108+rand.IntN(2)]
		} else if r < 0.05 && level >= 10 { // Bosses require level 10+
			m = baseMobs[106+rand.IntN(2)]
		} else if r < 0.12 && level >= 8 { // Minibosses require level 8+
			m = baseMobs[104+rand.IntN(2)]
		} else if r < 0.25 && level >= 5 { // Elites require level 5+
			m = baseMobs[102+rand.IntN(2)]
		} else if r < 0.40 && level >= 3 { // EliteMinions require level 3+
			m = baseMobs[100+rand.IntN(2)]
		}
	}

	m.Level = level

	// --- BALANCED SCALING ---
	// 1. Level Scaling (Base power) - Flatter growth
	lvlScale := 1.0 + 0.005*float64(level-1)

	// 2. Difficulty Dampening
	// Instead of full multiplication, difficulty only affects 30% of the scaling
	// Example: difficulty 2.0 (Zone + Gear) becomes a 1.3x multiplier
	effectiveDiff := 1.0 + (difficulty-1.0)*0.3

	totalScale := lvlScale * effectiveDiff
	if totalScale < 0.1 {
		totalScale = 0.1
	}

	m.Stats.HP = int(float64(m.Stats.HP) * totalScale)
	m.Stats.STR = int(float64(m.Stats.STR) * totalScale)

	// Flatten DEF scaling: 50% slower growth than STR/HP to prevent 'DEF Wall'
	defScale := 1.0 + (totalScale-1.0)*0.5
	m.Stats.DEF = int(float64(m.Stats.DEF) * defScale)

	m.Stats.SPD = int(float64(m.Stats.SPD) * totalScale)

	// XP rewards still scale fully to reward the risk
	m.RewardXP = int(float64(m.RewardXP) * lvlScale * difficulty)

	// XP Scaling: Higher types provide even more rewards.
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
		m.RewardXP = int(float64(m.RewardXP) * 5.0) // Significant payout but balanced
	}

	// Random effect
	// #nosec G404
	if rand.Float64() < 0.3 { // #nosec G404
		effects := []MobEffect{EffectEnraged, EffectArmored, EffectFleet, EffectPoisoned, EffectWeakened, EffectBlinded, EffectRegen}
		// #nosec G404
		eff := effects[rand.IntN(len(effects))] // #nosec G404
		m.Effects = append(m.Effects, eff)

		// Harder effects give more XP
		switch eff {
		case EffectEnraged, EffectArmored, EffectRegen:
			m.RewardXP = int(float64(m.RewardXP) * 1.3)
		case EffectFleet:
			m.RewardXP = int(float64(m.RewardXP) * 1.1)
		}
	}

	// 1-2 Spells for mobs
	spellCount := 1
	if isBoss || m.Type == MobLegendary || m.Type == MobMiniboss {
		spellCount = 2
	}
	for i := 0; i < spellCount; i++ {
		m.Spells = append(m.Spells, RandomSkill())
	}

	// 1-2 Equipped items that drop as loot
	itemCount := 1
	// #nosec G404
	if rand.Float64() < 0.3 { // #nosec G404
		itemCount = 2
	}
	for i := 0; i < itemCount; i++ {
		m.Equipped = append(m.Equipped, RandomGearDrop())
	}

	// Death Effect (Rare or based on type)
	chance := 0.1
	if m.Type == MobCommon {
		chance = 0.2 // Trash mobs often have effects
	}
	// #nosec G404
	if rand.Float64() < chance { // #nosec G404
		prefixes := []string{"Last", "Final", "Dying", "Bitter", "Vengeful", "Spiteful", "Desperate", "Echoing", "Ghostly", "Cursed"}
		actions := []string{"Roar", "Whimper", "Gasp", "Curse", "Blast", "Wail", "Howl", "Scream", "Sigh", "Command"}

		dType := DeathExplosion
		// #nosec G404
		r := rand.Float64() // #nosec G404
		if r < 0.4 {
			dType = DeathSummon
		} else if r < 0.6 {
			dType = DeathCurse
		} else if r < 0.8 {
			dType = DeathXP
		} else if r < 0.9 {
			dType = DeathLoot
		}

		m.DeathEffect = &MobDeathEffect{
			// #nosec G404
			Name: prefixes[rand.IntN(len(prefixes))] + " " + actions[rand.IntN(len(actions))], // #nosec G404
			Type: dType,
		}
	}

	m.MaxHP = m.Stats.HP
	m.CurrentHP = m.MaxHP

	return m
}

func SpawnMobGroup(avgLevel int, zone Zone, difficulty float64, groupSize int) []Mob {
	// 15% chance to spawn a HORDE of weaker mobs (great for farming drops/XP)
	// #nosec G404
	isHorde := rand.Float64() < 0.15 // #nosec G404

	// Difficulty affects count: higher difficulty = more mobs
	// #nosec G404
	baseCount := 2 + rand.IntN(3) // Increased base from 1 to 2

	// Zone Special effect: extra mobs
	for _, eff := range zone.Effects {
		if eff.Type == ZoneSpecial && strings.Contains(eff.Name, "Surge") {
			baseCount += 2 // Increased surge from 1 to 2
		}
	}

	// Horde spawns: 5-10 weaker mobs
	if isHorde {
		baseCount = 5 + rand.IntN(6) // 5 to 10 mobs in a horde // #nosec G404
	}

	// Dampen count scaling
	count := int(float64(baseCount) * (1.0 + (difficulty-1.0)*0.3))
	if count < 1 {
		count = 1
	}
	if count > 12 {
		count = 12
	} // Increased cap for hordes

	// Mobs scale slightly with group size to prevent trivial farming
	groupMult := 1.0 + float64(groupSize-1)*0.1
	if groupMult > 2.5 {
		groupMult = 2.5
	}

	var out []Mob
	// #nosec G404
	hasBoss := rand.Float64() < 0.1*difficulty && !isHorde // Bosses don't spawn in hordes // #nosec G404
	for i := 0; i < count; i++ {
		mob := SpawnMob(avgLevel, hasBoss && i == 0, difficulty)
		// Apply group scaling
		mob.Stats.HP = int(float64(mob.Stats.HP) * groupMult)
		mob.Stats.STR = int(float64(mob.Stats.STR) * groupMult)
		mob.Stats.DEF = int(float64(mob.Stats.DEF) * groupMult)
		mob.Stats.SPD = int(float64(mob.Stats.SPD) * groupMult)

		if isHorde {
			// Horde mobs are weaker (50-80% of normal level)
			// #nosec G404
			levelMult := 0.5 + rand.Float64()*0.3 // #nosec G404
			mob.Level = int(float64(mob.Level) * levelMult)
			if mob.Level < 1 {
				mob.Level = 1
			}
			// Scale stats down proportionally
			mob.Stats.HP = int(float64(mob.Stats.HP) * levelMult)
			mob.Stats.STR = int(float64(mob.Stats.STR) * levelMult)
			mob.Stats.DEF = int(float64(mob.Stats.DEF) * levelMult)
			mob.Stats.SPD = int(float64(mob.Stats.SPD) * levelMult)
			// Rename to indicate horde
			mob.Name = "Horde " + mob.Name
			// Hordes give slightly less XP per mob
			mob.RewardXP = int(float64(mob.RewardXP) * 0.6)
		}

		mob.MaxHP = mob.Stats.HP
		mob.CurrentHP = mob.MaxHP
		out = append(out, mob)
	}
	return out
}
