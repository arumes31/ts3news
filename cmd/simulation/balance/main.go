package main

import (
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"
)

func main() {
	// CLI flags
	fights := flag.Int("fights", 1000, "Number of fights per simulation run")
	players := flag.Int("players", 15, "Number of simulated players")
	groupSizes := flag.String("groups", "1,4,5,6", "Comma-separated group sizes to test")
	balance := flag.Bool("balance", true, "Enable auto-balancer to find 50% win rate")
	targetRate := flag.Float64("target", 0.50, "Target win rate (0.0-1.0)")
	tolerance := flag.Float64("tolerance", 0.02, "Win rate tolerance (0.02 = ±2%)")
	maxIter := flag.Int("max-iter", 20, "Max balancer iterations")
	seed := flag.Int64("seed", 0, "Random seed (0 = time-based)")
	output := flag.String("output", "console", "Output format: console, csv, json, all")
	outdir := flag.String("outdir", ".", "Output directory for csv/json files")
	verbose := flag.Bool("verbose", false, "Print per-fight details")
	quick := flag.Bool("quick", false, "Quick mode: 100 fights, 5 players, less verbose")

	flag.Parse()

	// Quick mode overrides
	if *quick {
		*fights = 100
		*players = 5
		*groupSizes = "1,4"
		*maxIter = 10
	}

	// Seed
	if *seed == 0 {
		*seed = time.Now().UnixNano()
	}

	// Parse group sizes
	sizes := ParseGroupSizes(*groupSizes)

	fmt.Println("╔══════════════════════════════════════════════════════════════╗")
	fmt.Println("║  TS3NEWS RPG BALANCE SIMULATION                             ║")
	fmt.Println("║  Target: ~50% Win Rate across group sizes                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Printf("  Fights: %d | Players: %d | Seed: %d\n", *fights, *players, *seed)
	fmt.Printf("  Group Sizes: %v | Auto-Balance: %v\n", sizes, *balance)
	fmt.Printf("  Target Win Rate: %.1f%% ± %.1f%%\n", *targetRate*100, *tolerance*100)
	fmt.Println()

	// Run simulation for each group size
	var allResults []BalanceResult

	for _, gs := range sizes {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("  GROUP SIZE: %d\n", gs)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

		if *balance {
			ab := NewAutoBalancer(gs, *fights, *targetRate, *tolerance, *seed, *verbose || *quick)
			ab.PlayerCount = *players
			ab.MaxIterations = *maxIter
			result := ab.Run()
			allResults = append(allResults, result)

			// Print per-bracket win rates
			if len(result.BracketResults) > 0 {
				fmt.Println()
				fmt.Println(" ┌─────────────────────────────────────────────┐")
				fmt.Println(" │ WIN RATE BY LEVEL BRACKET                   │")
				fmt.Println(" ├──────────┬──────────┬──────────┬────────────┤")
				fmt.Println(" │ Level    │ Wins     │ Fights   │ Win Rate   │")
				fmt.Println(" ├──────────┼──────────┼──────────┼────────────┤")
				for _, br := range result.BracketResults {
					fmt.Printf(" │ %-8d │ %-8d │ %-8d │ %5.1f%%     │\n",
						br.Level, br.Wins, br.Fights, br.WinRate*100)
				}
				fmt.Println(" └──────────┴──────────┴──────────┴────────────┘")
			}

			// Run final simulation with balanced params for detailed metrics
			mc := RunFullSimulation(*seed, result.FinalParams, *players, gs, *fights, *verbose)
			am := mc.ComputeAggregate()
			am.FinalParams = result.FinalParams
			PrintSummary(am)

			// Output files
			if *output == "csv" || *output == "all" {
				filename := fmt.Sprintf("%s/balance_group%d.csv", *outdir, gs)
				if err := WriteCSV(mc.FightRecords, filename); err == nil {
					fmt.Printf("  📊 CSV written to %s\n", filename)
				}
			}
			if *output == "json" || *output == "all" {
				filename := fmt.Sprintf("%s/balance_group%d.json", *outdir, gs)
				if err := WriteJSON(am, filename); err == nil {
					fmt.Printf("  📋 JSON written to %s\n", filename)
				}
			}
		} else {
			// Run with default params (no balancing)
			params := DefaultParams()
			mc := RunFullSimulation(*seed, params, *players, gs, *fights, *verbose)
			am := mc.ComputeAggregate()
			am.FinalParams = params
			PrintSummary(am)
		}
	}

	// Print comparison table
	if len(allResults) > 1 {
		fmt.Println()
		fmt.Println("╔══════════════════════════════════════════════════════════════╗")
		fmt.Println("║  COMPARISON TABLE — Balanced Parameters per Group Size      ║")
		fmt.Println("╠══════════════════════════════════════════════════════════════╣")
		fmt.Printf("║  %-6s │ %-8s │ %-8s │ %-8s │ %-8s │ %-8s ║\n",
			"Group", "WinRate", "MobHP", "MobSTR", "MobDEF", "Iters")
		fmt.Println("╠══════════════════════════════════════════════════════════════╣")
		for _, r := range allResults {
			converged := "✅"
			if !r.Converged {
				converged = "❌"
			}
			fmt.Printf("║  %-6d │ %5.1f%%%s │ %6.3f  │ %6.3f  │ %6.3f  │ %4d   ║\n",
				r.GroupSize, r.FinalWinRate*100, converged,
				r.FinalParams.MobHPMult, r.FinalParams.MobSTRMult,
				r.FinalParams.MobDEFMult, r.Iterations)
		}
		fmt.Println("╚══════════════════════════════════════════════════════════════╝")
	}

	// Print tuning recommendations
	if len(allResults) > 0 {
		fmt.Println()
		fmt.Println("══════════════════════════════════════════════════════════════")
		fmt.Println("  TUNING RECOMMENDATIONS")
		fmt.Println("══════════════════════════════════════════════════════════════")

		// Find the average balanced multipliers
		avgHP := 0.0
		avgSTR := 0.0
		convergedCount := 0
		for _, r := range allResults {
			if r.Converged {
				avgHP += r.FinalParams.MobHPMult
				avgSTR += r.FinalParams.MobSTRMult
				convergedCount++
			}
		}
		if convergedCount > 0 {
			avgHP /= float64(convergedCount)
			avgSTR /= float64(convergedCount)
			fmt.Printf("  Average balanced multipliers (across converged group sizes):\n")
			fmt.Printf("    MobHPMult  = %.3f\n", avgHP)
			fmt.Printf("    MobSTRMult = %.3f\n", avgSTR)
			fmt.Println()
			fmt.Println("  To apply these to the real bot, update the mob scaling in:")
			fmt.Println("    internal/content/mobs.go — SpawnMob() scaling factors")
			fmt.Println("    internal/bot/xp.go — resolveChannelCombat() difficulty")
		}

		// Group-size scaling recommendation
		if len(allResults) > 1 {
			fmt.Println()
			fmt.Println("  Group-size scaling factors (relative to solo):")
			soloHP := 0.0
			soloSTR := 0.0
			for _, r := range allResults {
				if r.GroupSize == 1 && r.Converged {
					soloHP = r.FinalParams.MobHPMult
					soloSTR = r.FinalParams.MobSTRMult
				}
			}
			if soloHP > 0 && soloSTR > 0 {
				for _, r := range allResults {
					if r.GroupSize > 1 && r.Converged {
						hpScale := r.FinalParams.MobHPMult / soloHP
						strScale := r.FinalParams.MobSTRMult / soloSTR
						fmt.Printf("    Group %d: HP %.2fx, STR %.2fx\n",
							r.GroupSize, hpScale, strScale)
					}
				}
			}
		}
	}
}

// RunFullSimulation runs a complete fight simulation with the given parameters.
func RunFullSimulation(seed int64, params SimParams, playerCount, groupSize, totalFights int, verbose bool) *MetricCollector {
	rng := rand.New(rand.NewSource(seed))

	// Create players
	players := make([]*SimPlayer, playerCount)
	for i := 0; i < playerCount; i++ {
		players[i] = NewSimPlayer(i, params)
		// Starter gear — mirrors real bot: 2 Novice pieces (Common) + 1 Uncommon
		for si, s := range []string{"MainHand", "Chest", "Legs"} {
			g := GenerateGear(rng, 1, params)
			g.Slot = s
			if si < 2 {
				g.Rarity = RarityCommon // Novice gear
			} else {
				g.Rarity = RarityUncommon // Trusty Longsword equivalent
			}
			players[i].EquipGear(g)
		}
		// Starter skills
		players[i].Skills = append(players[i].Skills,
			SimSkill{ID: "S0_1", Name: "Novice Spark", Type: SkillMagic, Rarity: RarityCommon, Power: 1.1},
			SimSkill{ID: "S0_2", Name: "Novice Punch", Type: SkillPhysical, Rarity: RarityCommon, Power: 1.1},
		)
		players[i].RecalculateStats(params)
		players[i].CurrentHP = players[i].MaxHP
	}

	mc := NewMetricCollector(groupSize, params)
	mc.Players = players
	mc.AH = &SimAuctionHouse{}
	mc.Economy = &SimGoldEconomy{}

	lootSystem := &SimLootSystem{AH: mc.AH}

	for fight := 1; fight <= totalFights; fight++ {
		// Group players by party size
		for i := 0; i < len(players); i += groupSize {
			end := i + groupSize
			if end > len(players) {
				end = len(players)
			}
			party := players[i:end]
			if len(party) == 0 {
				continue
			}

			// Compute average level
			avgLvl := avgLevel(party)

			// Reset HP for fight
			for _, p := range party {
				p.CurrentHP = p.MaxHP
			}

			// Select zone difficulty
			difficulty := params.ZoneDiffMin + rng.Float64()*(params.ZoneDiffMax-params.ZoneDiffMin)

			// Run combat
			var logs []string
			result := ResolveCombat(rng, party, avgLvl, difficulty, params, &logs)

			// Record metrics
			mc.RecordFight(fight, result, party)

			// Post-combat: loot, durability, XP, gold
			if result.Victory {
				// Loot
				for _, p := range party {
					lootSystem.RollLootForPlayer(rng, p, result.MobsKilled, fight, params)
				}

				// Gold distribution
				DistributeGold(rng, party, result.MobsKilled, result.GoldGained*int64(len(party)), mc.Economy, params)
			}

			// Durability loss
			for _, p := range party {
				ApplyDurabilityLoss(rng, p, !result.Victory, params)
			}

			// XP — combat XP only (no base activity XP; that's a separate event in the real bot)
			for _, p := range party {
				if result.Victory {
					xp := result.XPGained
					// Apply gear XP multiplier: average of all gear XPMult values
					gearCount := 0
					gearMultSum := 0.0
					for _, g := range p.Gear {
						if g.Durability > 0 {
							gearMultSum += g.XPMult
							gearCount++
						}
					}
					if gearCount > 0 {
						xp *= gearMultSum / float64(gearCount)
					}
					AwardXP(p, xp, params)
				} else {
					ApplyDeathPenalty(p, params)
				}
			}

			if verbose && fight <= 5 {
				fmt.Printf("  Fight %d (Group %d): %v | Rounds: %d | Mobs: %d | XP: %.0f\n",
					fight, groupSize, result.Victory, result.TotalRounds, result.MobsKilled, result.XPGained)
			}
		}

		// Auction house processing every 10 fights
		if fight%10 == 0 {
			for _, p := range players {
				mc.AH.AutoPurchaseUpgrades(p, fight)
			}
			mc.AH.RemoveExpired(fight)
		}
	}

	return mc
}

// Silence unused import warning
var _ = math.Pi
var _ = os.Stdout
