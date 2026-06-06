package content

import (
	"fmt"
	"math/rand"
)

type MobType string

const (
	MobCommon    MobType = "Common"
	MobElite     MobType = "Elite"
	MobBoss      MobType = "Boss"
	MobLegendary MobType = "Legendary"
)

type MobEffect string

const (
	EffectEnraged    MobEffect = "Enraged"     // +50% STR
	EffectArmored    MobEffect = "Armored"     // +50% DEF
	EffectFleet      MobEffect = "Fleet-foot"  // +50% SPD
	EffectPoisoned   MobEffect = "Poisoned"    // Loses 5% HP per round
	EffectWeakened   MobEffect = "Weakened"    // -50% STR
	EffectBlinded    MobEffect = "Blinded"     // 50% miss chance
	EffectRegen      MobEffect = "Regenerative" // Heals 5% HP per round
)

type Mob struct {
	Name     string
	Type     MobType
	Level    int
	Stats    Stats
	RewardXP int
	Effects  []MobEffect
	Spells   []Skill
	Equipped []Gear
}

func (m Mob) DisplayName() string {
	eff := ""
	if len(m.Effects) > 0 {
		eff = fmt.Sprintf(" (%s)", m.Effects[0])
	}
	return fmt.Sprintf("Lvl %d %s [%s]%s", m.Level, m.Name, m.Type, eff)
}

var baseMobs []Mob

func init() {
	prefixes := []string{"Snotty", "Angry", "Undead", "Shadow", "Fiery", "Ice-Cold", "Toxic", "Ghostly", "Metallic", "Giant"}
	nouns := []string{"Rat", "Slime", "Goblin", "Spider", "Zombie", "Wolf", "Skeleton", "Bat", "Orc", "Troll"}

	for _, p := range prefixes {
		for _, n := range nouns {
			name := p + " " + n
			baseMobs = append(baseMobs, Mob{
				Name:  name,
				Type:  MobCommon,
				Stats: Stats{HP: 20, STR: 5, DEF: 2, SPD: 5, LCK: 0},
				RewardXP: 5,
			})
		}
	}

	baseMobs = append(baseMobs, Mob{Name: "Dread Knight", Type: MobElite, Stats: Stats{HP: 150, STR: 30, DEF: 20, SPD: 10, LCK: 5}, RewardXP: 25})
	baseMobs = append(baseMobs, Mob{Name: "Ancient Dragon", Type: MobBoss, Stats: Stats{HP: 1000, STR: 100, DEF: 50, SPD: 20, LCK: 10}, RewardXP: 100})
	baseMobs = append(baseMobs, Mob{Name: "THE VOID LORD", Type: MobLegendary, Stats: Stats{HP: 5000, STR: 300, DEF: 100, SPD: 50, LCK: 25}, RewardXP: 500})
}

// SpawnMob scales a mob to the given level and difficulty factor (0.1 to 1.0+)
func SpawnMob(level int, isBoss bool, difficulty float64) Mob {
	idx := rand.Intn(100) // index for common mobs
	if isBoss {
		idx = len(baseMobs) - 2 // Ancient Dragon
	}
	
	m := baseMobs[idx]
	if !isBoss {
		r := rand.Float64()
		if r < 0.01 {
			m = baseMobs[len(baseMobs)-1] // Legendary
		} else if r < 0.05 {
			m = baseMobs[len(baseMobs)-2] // Boss
		} else if r < 0.15 {
			m = baseMobs[len(baseMobs)-3] // Elite
		}
	}

	m.Level = level
	// Scaling logic: base level scaling * gear-aware difficulty factor
	scale := (1.0 + 0.1*float64(level-1)) * difficulty
	if scale < 0.1 { scale = 0.1 }

	m.Stats.HP = int(float64(m.Stats.HP) * scale)
	m.Stats.STR = int(float64(m.Stats.STR) * scale)
	m.Stats.DEF = int(float64(m.Stats.DEF) * scale)
	m.Stats.SPD = int(float64(m.Stats.SPD) * scale)
	m.RewardXP = int(float64(m.RewardXP) * scale)

	// Random effect
	if rand.Float64() < 0.3 {
		effects := []MobEffect{EffectEnraged, EffectArmored, EffectFleet, EffectPoisoned, EffectWeakened, EffectBlinded, EffectRegen}
		m.Effects = append(m.Effects, effects[rand.Intn(len(effects))])
	}

	// 1-2 Spells for mobs
	spellCount := 1
	if isBoss || m.Type == MobLegendary {
		spellCount = 2
	}
	for i := 0; i < spellCount; i++ {
		m.Spells = append(m.Spells, RandomSkill())
	}

	// 1-2 Equipped items that drop as loot
	itemCount := 1
	if rand.Float64() < 0.3 {
		itemCount = 2
	}
	for i := 0; i < itemCount; i++ {
		m.Equipped = append(m.Equipped, RandomGearDrop())
	}

	return m
}

func SpawnMobGroup(avgLevel int, difficulty float64) []Mob {
	count := 1 + rand.Intn(4) // 1 to 4 mobs
	var out []Mob
	hasBoss := rand.Float64() < 0.1
	for i := 0; i < count; i++ {
		out = append(out, SpawnMob(avgLevel, hasBoss && i == 0, difficulty))
	}
	return out
}
