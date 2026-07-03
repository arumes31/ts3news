// Command chaos runs a concurrent load-test simulation of many users hammering
// the bot's XP/combat paths at once, to shake out race conditions.
package main

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ts3news/internal/bot"
	"ts3news/internal/content"
	"ts3news/internal/leveling"
)

// ChaosSim simulates a high-concurrency environment with multiple users
// interacting with the bot logic simultaneously.
func runChaosSim(userCount int, cycles int) {
	fmt.Printf("Starting Chaos Simulation: %d users, %d cycles\n", userCount, cycles)
	
	var wg sync.WaitGroup
	var totalGold int64
	var totalXP int64
	var totalWins int64
	var totalLosses int64
	var totalPrestiges int64

	// Mocking a basic "Bot" state for logic testing
	// We'll simulate the core XP and Combat paths without a real DB
	// by using atomic counters for global stats.

	start := time.Now()

	for i := 0; i < userCount; i++ {
		wg.Add(1)
		go func(_ int) {
			defer wg.Done()
			
			// Local user state
			uLvl := 1
			uPrestige := 0
			uXP := 0
			uGold := int64(0)
			
			for c := 0; c < cycles; c++ {
				// 1. Simulate a "Poke" (Combat)
				// Basic difficulty scaling
				diff := 1.0 + float64(uLvl)*0.001
				
				// Simulate victory/defeat chance (simplified from xp.go)
				// #nosec G404
				winChance := 0.7 + (float64(uPrestige) * 0.05) - (diff * 0.1)
				if winChance > 0.95 { winChance = 0.95 }
				if winChance < 0.2 { winChance = 0.2 }
				
				// #nosec G404
				if rand.Float64() < winChance {
					atomic.AddInt64(&totalWins, 1)
					
					// Reward
					// #nosec G404
					rewardXP := int(float64(20+rand.IntN(30)) * diff)
					rewardGold := int64(rewardXP * 5)
					
					// Inflation check (Improvement 44)
					currentSystemGold := atomic.LoadInt64(&totalGold)
					if currentSystemGold > 10000000 {
						mult := 1.0 / (1.0 + float64(currentSystemGold-10000000)/5000000.0)
						rewardGold = int64(float64(rewardGold) * mult)
					}
					
					uXP += rewardXP
					uGold += rewardGold
					atomic.AddInt64(&totalGold, rewardGold)
					atomic.AddInt64(&totalXP, int64(rewardXP))
				} else {
					atomic.AddInt64(&totalLosses, 1)
					penalty := int(float64(uXP) * 0.05)
					if penalty < 10 { penalty = 10 }
					uXP -= penalty
					if uXP < 0 { uXP = 0 }
				}
				
				// 2. Level Up & Prestige
				newLvl := leveling.LevelForXP(uXP)
				if newLvl >= 10000 {
					uPrestige++
					uXP = 0
					uLvl = 1
					atomic.AddInt64(&totalPrestiges, 1)
				} else {
					uLvl = newLvl
				}
				
				// Yield
				if c % 10 == 0 {
					time.Sleep(time.Microsecond)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Printf("CHAOS SIMULATION RESULTS (%d users, %d cycles)\n", userCount, cycles)
	fmt.Println("================================================================================")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Total Wins: %d\n", totalWins)
	fmt.Printf("Total Losses: %d\n", totalLosses)
	fmt.Printf("Win Rate: %.2f%%\n", float64(totalWins)/float64(totalWins+totalLosses)*100)
	fmt.Printf("Total Prestiges: %d\n", totalPrestiges)
	fmt.Printf("Final System Gold: %s\n", bot.FormatGold(totalGold))
	fmt.Printf("Final Avg XP per cycle: %.2f\n", float64(totalXP)/float64(userCount*cycles))
	fmt.Println("================================================================================")
}

func main() {
	// Rarity distribution simulation
	runLootSim(1000000)
	
	// Concurrent chaos simulation
	runChaosSim(100, 1000)
}

func runLootSim(rolls int) {
	fmt.Printf("Starting Loot Distribution Simulation: %d rolls\n", rolls)
	counts := make(map[content.Rarity]int)
	
	for i := 0; i < rolls; i++ {
		// #nosec G404
		r := rand.Float64()
		var rarity content.Rarity
		switch {
		case r < 0.45: rarity = content.RarityCommon
		case r < 0.75: rarity = content.RarityUncommon
		case r < 0.90: rarity = content.RarityRare
		case r < 0.97: rarity = content.RarityEpic
		case r < 0.995: rarity = content.RarityLegendary
		case r < 0.999: rarity = content.RarityMythic
		default: rarity = content.RarityDivine
		}
		counts[rarity]++
	}

	fmt.Println("\nLOOT DISTRIBUTION (1M ROLLS)")
	fmt.Println("================================================================================")
	for i := 0; i <= int(content.RarityDivine); i++ {
		rar := content.Rarity(i)
		fmt.Printf("%-10s: %d (%6.2f%%)\n", rar.String(), counts[rar], float64(counts[rar])/float64(rolls)*100)
	}
	fmt.Println("================================================================================")
}
