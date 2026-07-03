// Command sim prints leveling-curve and economy figures for quick sanity checks.
package main

import (
	"fmt"
	"ts3news/internal/leveling"
	"ts3news/internal/content"
)

func main() {
	fmt.Println("=== TS3 RPG Economy Simulation ===")
	
	levels := []int{1, 10, 50, 100, 200, 500, 1000}
	
	fmt.Println("\n1. Leveling Curve Requirements")
	for _, lvl := range levels {
		xp := leveling.XPForLevel(lvl)
		fmt.Printf("Level %4d -> %12d XP\n", lvl, xp)
	}

	fmt.Println("\n2. Mob Scaling at Key Levels")
	// simulate avg mob stats at different levels
	for _, lvl := range levels {
		// Common Mob
		common := content.SpawnMob(lvl, false, 1.0)
		boss := content.SpawnMob(lvl, true, 1.0)
		
		fmt.Printf("Level %4d | Common: %5d HP, %4d STR, %4d XP | Boss: %6d HP, %4d STR, %5d XP\n",
			lvl, common.Stats.HP, common.Stats.STR, common.RewardXP,
			boss.Stats.HP, boss.Stats.STR, boss.RewardXP)
	}

	fmt.Println("\n3. Simulated Progression to Level 10,000")
	xp := 0
	gold := int64(0)
	level := 1
	fights := 0
	
	for level < 10000 {
		fights++
		// Suppose they fight mobs of their own level
		mobs := content.SpawnMobGroup(level, content.Zone{Difficulty: 1.0}, 1.0, 1, false)
		combatXP := 0
		for _, m := range mobs {
			combatXP += m.RewardXP
			// gold drop (New standard base is m.Level * 2 to 10)
			baseGold := m.Level * 2
			if m.Type == content.MobBoss {
				baseGold = m.Level * 10
			}
			gold += int64(float64(baseGold) * 1.0)
		}
		
		// Add poke XP
		xp += leveling.XPPerPoke() + combatXP
		
		// Level up check
		for leveling.XPForLevel(level+1) <= xp {
			level++
			if level%1000 == 0 {
				fmt.Printf("Reached Level %d after %d fights. XP: %d, Gold: %d\n", level, fights, xp, gold)
			}
		}
		
		// Safety break to avoid infinite loops if XP curve is truly broken
		if fights > 5000000 {
			fmt.Println("Simulation aborted: Reached 5,000,000 fights without hitting level 10,000.")
			break
		}
	}
	
	fmt.Printf("\nFinal Result: Level %d achieved after %d Pokes+Fights.\nTotal XP: %d\nTotal Gold: %d\n", level, fights, xp, gold)
}
