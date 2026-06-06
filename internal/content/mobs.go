package content

import (
	"math/rand"
)

type MobType string

const (
	MobCommon    MobType = "Common"
	MobElite     MobType = "Elite"
	MobBoss      MobType = "Boss"
	MobLegendary MobType = "Legendary"
)

type Mob struct {
	Name  string
	Type  MobType
	Stats Stats
	RewardXP int // base XP reward/penalty
}

var allMobs []Mob

func init() {
	prefixes := []string{"Snotty", "Angry", "Undead", "Shadow", "Fiery", "Ice-Cold", "Toxic", "Ghostly", "Metallic", "Giant"}
	nouns := []string{"Rat", "Slime", "Goblin", "Spider", "Zombie", "Wolf", "Skeleton", "Bat", "Orc", "Troll"}

	for _, p := range prefixes {
		for _, n := range nouns {
			name := p + " " + n
			allMobs = append(allMobs, Mob{
				Name:  name,
				Type:  MobCommon,
				Stats: Stats{HP: 20, STR: 5, DEF: 2, SPD: 5, LCK: 0},
				RewardXP: 5,
			})
		}
	}

	// Add some Elites
	allMobs = append(allMobs, Mob{
		Name: "Dread Knight", Type: MobElite, Stats: Stats{HP: 150, STR: 30, DEF: 20, SPD: 10, LCK: 5}, RewardXP: 25,
	})
	allMobs = append(allMobs, Mob{
		Name: "Shadow Assassin", Type: MobElite, Stats: Stats{HP: 80, STR: 50, DEF: 5, SPD: 40, LCK: 15}, RewardXP: 25,
	})

	// Add Bosses
	allMobs = append(allMobs, Mob{
		Name: "Ancient Dragon", Type: MobBoss, Stats: Stats{HP: 1000, STR: 100, DEF: 50, SPD: 20, LCK: 10}, RewardXP: 100,
	})

	// Add Legendary
	allMobs = append(allMobs, Mob{
		Name: "THE VOID LORD", Type: MobLegendary, Stats: Stats{HP: 5000, STR: 300, DEF: 100, SPD: 50, LCK: 25}, RewardXP: 500,
	})
}

func SpawnRandomMob() Mob {
	r := rand.Float64()
	if r < 0.01 {
		return allMobs[len(allMobs)-1] // Legendary
	}
	if r < 0.05 {
		return allMobs[len(allMobs)-2] // Boss
	}
	if r < 0.15 {
		return allMobs[len(allMobs)-3] // Elite
	}
	return allMobs[rand.Intn(100)] // Common
}
