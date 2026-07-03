// Command balance auto-tunes mob stat multipliers to hit a target player win
// rate, and provides the shared simulation types used by other sim tools.
package main

import (
	"fmt"
	"math/rand"
)

// AutoBalancer finds the mob multiplier values that produce ~50% win rate.
// Uses binary search on MobSTRMult and MobHPMult.
// Tests at multiple level brackets to find multipliers that work across the progression.
type AutoBalancer struct {
	TargetWinRate    float64 // 0.50
	Tolerance        float64 // 0.02 (48-52%)
	MaxIterations    int
	GroupSize        int
	FightsPerBracket int // fights per level bracket
	PlayerCount      int
	Seed             int64
	Verbose          bool
}

// BalanceResult holds the output of the balancing process.
type BalanceResult struct {
	GroupSize      int
	Iterations     int
	FinalWinRate   float64
	FinalParams    SimParams
	Converged      bool
	History        []BalanceIteration
	BracketResults []BracketResult
}

// BalanceIteration records one iteration of the balancer.
type BalanceIteration struct {
	Iteration  int
	MobHPMult  float64
	MobSTRMult float64
	WinRate    float64
	Adjustment string
}

// BracketResult holds win rate for a specific level bracket.
type BracketResult struct {
	Level   int
	WinRate float64
	Wins    int
	Fights  int
}

// Level brackets to test — represents typical player progression
var levelBrackets = []int{1, 5, 10, 25, 50, 100, 200, 500}

// NewAutoBalancer creates a balancer with default settings.
func NewAutoBalancer(groupSize, fightsPerRun int, targetWinRate, tolerance float64, seed int64, verbose bool) *AutoBalancer {
	return &AutoBalancer{
		TargetWinRate:    targetWinRate,
		Tolerance:        tolerance,
		MaxIterations:    25,
		GroupSize:        groupSize,
		FightsPerBracket: fightsPerRun / len(levelBrackets),
		PlayerCount:      15,
		Seed:             seed,
		Verbose:          verbose,
	}
}

// Run executes the binary search to find balanced parameters.
func (ab *AutoBalancer) Run() BalanceResult {
	params := DefaultParams()

	// Search bounds for mob multipliers
	loSTR, hiSTR := 0.1, 50.0
	loHP, hiHP := 0.1, 50.0

	params.MobSTRMult = (loSTR + hiSTR) / 2.0
	params.MobHPMult = (loHP + hiHP) / 2.0

	result := BalanceResult{
		GroupSize:   ab.GroupSize,
		FinalParams: params,
	}

	if ab.Verbose {
		fmt.Printf("\n🔍 Auto-Balancer: GroupSize=%d, Target=%.1f%%±%.1f%%\n",
			ab.GroupSize, ab.TargetWinRate*100, ab.Tolerance*100)
		fmt.Printf(" Starting: MobHPMult=%.3f, MobSTRMult=%.3f\n", params.MobHPMult, params.MobSTRMult)
		fmt.Printf(" Level brackets: %v\n", levelBrackets)
	}

	for iter := 0; iter < ab.MaxIterations; iter++ {
		// Run simulation with current params across all level brackets
		winRate, bracketResults := ab.runSimulation(params)

		adjustment := ""
		if winRate > ab.TargetWinRate+ab.Tolerance {
			// Players winning too much → mobs too weak → increase mob power
			loSTR = params.MobSTRMult
			loHP = params.MobHPMult
			params.MobSTRMult = (loSTR + hiSTR) / 2.0
			params.MobHPMult = (loHP + hiHP) / 2.0
			adjustment = fmt.Sprintf("↑ STR=%.3f HP=%.3f", params.MobSTRMult, params.MobHPMult)
		} else if winRate < ab.TargetWinRate-ab.Tolerance {
			// Players losing too much → mobs too strong → decrease mob power
			hiSTR = params.MobSTRMult
			hiHP = params.MobHPMult
			params.MobSTRMult = (loSTR + hiSTR) / 2.0
			params.MobHPMult = (loHP + hiHP) / 2.0
			adjustment = fmt.Sprintf("↓ STR=%.3f HP=%.3f", params.MobSTRMult, params.MobHPMult)
		} else {
			// Within tolerance — converged!
			result.Converged = true
			result.FinalWinRate = winRate
			result.FinalParams = params
			result.Iterations = iter + 1
			result.BracketResults = bracketResults

			if ab.Verbose {
				fmt.Printf(" Iter %2d: WinRate=%5.1f%% ✅ CONVERGED! MobHP=%.3f MobSTR=%.3f\n",
					iter+1, winRate*100, params.MobHPMult, params.MobSTRMult)
			}
			break
		}

		bi := BalanceIteration{
			Iteration:  iter + 1,
			MobHPMult:  params.MobHPMult,
			MobSTRMult: params.MobSTRMult,
			WinRate:    winRate,
			Adjustment: adjustment,
		}
		result.History = append(result.History, bi)

		if ab.Verbose {
			fmt.Printf(" Iter %2d: WinRate=%5.1f%% → %s\n",
				iter+1, winRate*100, adjustment)
		}

		result.FinalWinRate = winRate
		result.FinalParams = params
		result.Iterations = iter + 1
		result.BracketResults = bracketResults

		// Check if bounds have converged (step size < 0.05)
		if hiSTR-loSTR < 0.1 && hiHP-loHP < 0.1 {
			result.Converged = true
			break
		}
	}

	if !result.Converged && ab.Verbose {
		fmt.Printf(" ⚠️ Did not converge after %d iterations (final: %.1f%%)\n",
			ab.MaxIterations, result.FinalWinRate*100)
	}

	return result
}

// runSimulation runs fight simulations across all level brackets and returns the aggregate win rate.
// Each bracket creates fresh players at that level with appropriate gear.
func (ab *AutoBalancer) runSimulation(params SimParams) (float64, []BracketResult) {
	totalWins := 0
	totalFights := 0
	var bracketResults []BracketResult

	for _, bracketLevel := range levelBrackets {
		rng := rand.New(rand.NewSource(ab.Seed + int64(bracketLevel*1000))) // #nosec G404 - simulation only, non-cryptographic

		// Create players at this level bracket with appropriate gear
		players := make([]*SimPlayer, ab.PlayerCount)
		for i := 0; i < ab.PlayerCount; i++ {
			players[i] = NewSimPlayer(i, params)
			// Set level directly
			players[i].Level = bracketLevel
			players[i].XP = 0 // XP doesn't matter for fixed-level testing

			// Equip gear appropriate for this level bracket.
			// Mirrors real bot progression: new players get 2-3 starter pieces,
			// gradually filling slots as they level. At level 100+ most slots are filled.
			gearSlots := []string{"MainHand", "Chest", "Legs", "Head", "Hands", "Feet",
				"Shoulders", "Waist", "Back", "Ring1", "Ring2", "Neck",
				"Wrist1", "Wrist2", "Accessory1", "Accessory2"}

			// Number of gear slots scales with level (2 at level 1, all 16 at level 200+)
			slotsToEquip := 2 + (bracketLevel-1)*14/199
			if slotsToEquip > len(gearSlots) {
				slotsToEquip = len(gearSlots)
			}
			if slotsToEquip < 2 {
				slotsToEquip = 2
			}

			for si := 0; si < slotsToEquip; si++ {
				s := gearSlots[si]
				g := GenerateGear(rng, bracketLevel, params)
				g.Slot = s
				// Gear rarity scales with level
				if bracketLevel >= 100 && rng.Float64() < 0.3 {
					g.Rarity = RarityEpic
				} else if bracketLevel >= 25 && rng.Float64() < 0.4 {
					g.Rarity = RarityRare
				} else if bracketLevel >= 5 {
					g.Rarity = RarityUncommon
				}
				players[i].EquipGear(g)
			}

			// Add skills appropriate for level
			if bracketLevel >= 5 {
				players[i].Skills = append(players[i].Skills,
					SimSkill{ID: "S1", Name: "Fireball", Type: SkillMagic, Rarity: RarityUncommon, Power: 1.3, IgnoreDef: 0.2},
				)
			}
			if bracketLevel >= 25 {
				players[i].Skills = append(players[i].Skills,
					SimSkill{ID: "S2", Name: "Frost Strike", Type: SkillPhysical, Rarity: RarityRare, Power: 1.5, StunChance: 0.15},
				)
			}
			if bracketLevel >= 100 {
				players[i].Skills = append(players[i].Skills,
					SimSkill{ID: "S3", Name: "Shadow Bolt", Type: SkillMagic, Rarity: RarityEpic, Power: 1.8, IgnoreDef: 0.4},
				)
			}
			if bracketLevel >= 200 {
				players[i].UltimateSkill = &SimUltimate{
					Name:            "Apocalypse",
					Power:           3.0,
					CooldownRounds:  5,
					CurrentCooldown: 0,
					Rarity:          RarityEpic,
				}
			}

			// Add some consumables for higher levels
			if bracketLevel >= 10 {
				players[i].Consumables = append(players[i].Consumables,
					SimConsumable{ID: "C1", Name: "Health Potion", Type: ConsumableHealing, EffectValue: 0.3, Remaining: 3},
				)
			}
			if bracketLevel >= 50 {
				players[i].Consumables = append(players[i].Consumables,
					SimConsumable{ID: "C2", Name: "Revive Scroll", Type: ConsumableRevive, EffectValue: 0.5, Remaining: 1},
				)
			}

			players[i].RecalculateStats(params)
			players[i].CurrentHP = players[i].MaxHP
		}

		// Save original consumable state for reset between fights
		originalConsumables := make([][]SimConsumable, len(players))
		for pi, p := range players {
			snap := make([]SimConsumable, len(p.Consumables))
			copy(snap, p.Consumables)
			originalConsumables[pi] = snap
		}

		// Run fights at this bracket
		bracketWins := 0
		bracketFights := 0
		fightsThisBracket := ab.FightsPerBracket
		if fightsThisBracket < 10 {
			fightsThisBracket = 10
		}

		for fight := 0; fight < fightsThisBracket; fight++ {
			// Group players by party size
			for i := 0; i < len(players); i += ab.GroupSize {
				end := i + ab.GroupSize
				if end > len(players) {
					end = len(players)
				}
				party := players[i:end]
				if len(party) == 0 {
					continue
				}

				// Reset HP, consumables, and cooldowns for fight
				for pi, p := range party {
					p.CurrentHP = p.MaxHP
					// Restore consumables to original state (preserves per-consumable limits)
					absIdx := i + pi
					p.Consumables = make([]SimConsumable, len(originalConsumables[absIdx]))
					copy(p.Consumables, originalConsumables[absIdx])
					// Reset ultimate cooldown
					if p.UltimateSkill != nil {
						p.UltimateSkill.CurrentCooldown = 0
					}
				}

				// Select zone difficulty
				difficulty := params.ZoneDiffMin + rng.Float64()*(params.ZoneDiffMax-params.ZoneDiffMin)

				// Run combat
				var logs []string
				result := ResolveCombat(rng, party, bracketLevel, difficulty, params, &logs)

				if result.Victory {
					bracketWins += len(party)
				}
				bracketFights += len(party)
			}
		}

		bracketWR := 0.0
		if bracketFights > 0 {
			bracketWR = float64(bracketWins) / float64(bracketFights)
		}
		bracketResults = append(bracketResults, BracketResult{
			Level:   bracketLevel,
			WinRate: bracketWR,
			Wins:    bracketWins,
			Fights:  bracketFights,
		})
		totalWins += bracketWins
		totalFights += bracketFights
	}

	if totalFights == 0 {
		return 0, bracketResults
	}
	return float64(totalWins) / float64(totalFights), bracketResults
}
