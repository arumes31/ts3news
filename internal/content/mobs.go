package content

import (
	"fmt"
	"math/rand"
	"strings"
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
	RewardXP    int
	Effects     []MobEffect
	Spells      []Skill
	Equipped    []Gear
	DeathEffect *MobDeathEffect
}

func (m Mob) DisplayName() string {
	eff := ""
	if len(m.Effects) > 0 {
		eff = fmt.Sprintf(" (%s)", m.Effects[0])
	}
	if m.DeathEffect != nil {
		eff += fmt.Sprintf(" [💀 %s]", m.DeathEffect.Name)
	}
	return fmt.Sprintf("Lvl %d %s [%s]%s", m.Level, m.Name, m.Type, eff)
}

func (m Mob) Score() int {
	return m.Stats.HP/5 + m.Stats.STR + m.Stats.DEF + m.Stats.SPD + m.Level*10
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
	if isBoss && level >= 10 { // Bosses require level 10+
		idx = len(baseMobs) - 2 // Ancient Dragon
	}
	
	m := baseMobs[idx]
	if !isBoss {
		r := rand.Float64()
		if r < 0.01 && level >= 25 { // Legendaries require level 25+
			m = baseMobs[len(baseMobs)-1]
		} else if r < 0.05 && level >= 10 { // Bosses require level 10+
			m = baseMobs[len(baseMobs)-2]
		} else if r < 0.15 && level >= 5 { // Elites require level 5+
			m = baseMobs[len(baseMobs)-3]
		}
	}

	m.Level = level
	
	// --- BALANCED SCALING ---
	// 1. Level Scaling (Base power) - Flatter growth
	lvlScale := 1.0 + 0.05*float64(level-1)
	
	// 2. Difficulty Dampening
	// Instead of full multiplication, difficulty only affects 30% of the scaling
	// Example: difficulty 2.0 (Zone + Gear) becomes a 1.3x multiplier
	effectiveDiff := 1.0 + (difficulty-1.0)*0.3
	
	totalScale := lvlScale * effectiveDiff
	if totalScale < 0.1 { totalScale = 0.1 }

	m.Stats.HP = int(float64(m.Stats.HP) * totalScale)
	m.Stats.STR = int(float64(m.Stats.STR) * totalScale)
	m.Stats.DEF = int(float64(m.Stats.DEF) * totalScale)
	m.Stats.SPD = int(float64(m.Stats.SPD) * totalScale)

	// XP rewards still scale fully to reward the risk
	m.RewardXP = int(float64(m.RewardXP) * lvlScale * difficulty)

	// XP Scaling: Higher types provide even more rewards.
	switch m.Type {
	case MobElite:
		m.RewardXP = int(float64(m.RewardXP) * 1.5)
	case MobBoss:
		m.RewardXP = int(float64(m.RewardXP) * 2.5)
	case MobLegendary:
		m.RewardXP = int(float64(m.RewardXP) * 5.0) // Significant payout but balanced
	}

	// Random effect
	if rand.Float64() < 0.3 {
		effects := []MobEffect{EffectEnraged, EffectArmored, EffectFleet, EffectPoisoned, EffectWeakened, EffectBlinded, EffectRegen}
		eff := effects[rand.Intn(len(effects))]
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

	// Death Effect (Rare or based on type)
	chance := 0.1
	if m.Type == MobCommon {
		chance = 0.2 // Trash mobs often have effects
	}
	if rand.Float64() < chance {
		prefixes := []string{"Last", "Final", "Dying", "Bitter", "Vengeful", "Spiteful", "Desperate", "Echoing", "Ghostly", "Cursed"}
		actions := []string{"Roar", "Whimper", "Gasp", "Curse", "Blast", "Wail", "Howl", "Scream", "Sigh", "Command"}
		
		dType := DeathExplosion
		r := rand.Float64()
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
			Name: prefixes[rand.Intn(len(prefixes))] + " " + actions[rand.Intn(len(actions))],
			Type: dType,
		}
	}

	return m
}

func SpawnMobGroup(avgLevel int, zone Zone, difficulty float64) []Mob {
	// Difficulty affects count: higher difficulty = more mobs
	baseCount := 1 + rand.Intn(3)
	
	// Zone Special effect: extra mobs
	for _, eff := range zone.Effects {
		if eff.Type == ZoneSpecial && strings.Contains(eff.Name, "Surge") {
			baseCount += 1
		}
	}

	// Dampen count scaling
	count := int(float64(baseCount) * (1.0 + (difficulty-1.0)*0.2))
	if count < 1 { count = 1 }
	if count > 6 { count = 6 } // Reduced cap for balance

	var out []Mob
	hasBoss := rand.Float64() < 0.08 * difficulty
	for i := 0; i < count; i++ {
		out = append(out, SpawnMob(avgLevel, hasBoss && i == 0, difficulty))
	}
	return out
}
