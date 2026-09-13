package bot

import (
	"fmt"
	"math/rand/v2"
	"testing"

	"ts3news/internal/content"
)

// This is a deliberately limited combat-only run experiment. It carries exact
// remaining HP through consecutive floors using the production combat engine.
// Shadow combat suppresses database writes, loot, durability, kill-chain grants,
// and economy effects. No rooms, healing between floors, pets, consumables,
// classes, tree bonuses, or loadout changes are modeled. Mana and cooldowns reset
// per fight as in the engine. This is evidence about attrition, not a claim about
// complete persisted runs or the optimal builds available to real characters.
// Observed rounds are approximate: the timeline coalesces frames at equal log
// offsets, so cosmetic log emission can change how many rounds remain visible.
func TestAbyssBalanceCarryHPPlaytest(t *testing.T) {
	const runs, floors = 64, 20
	profiles := []struct {
		name   string
		level  int
		rarity content.Rarity
	}{
		{"new", 10, content.RarityCommon},
		{"mid", 100, content.RarityRare},
		{"veteran", 300, content.RarityLegendary},
	}
	for _, profile := range profiles {
		for _, tierKey := range []string{"normal", "nightmare", "hell", "insanity"} {
			for _, build := range []string{"basic", "novice_skills"} {
				t.Run(fmt.Sprintf("%s/%s/%s", profile.name, tierKey, build), func(t *testing.T) {
					base := abyssBalancePlaytestUser(profile.level, profile.rarity)
					if build == "novice_skills" {
						for _, id := range []string{"S0_1", "S0_2"} {
							skill, ok := content.GetSkillByID(id)
							if !ok {
								t.Fatalf("missing catalog skill %s", id)
							}
							base.Skills = append(base.Skills, skill)
						}
					}
					tier, ok := abyssTierByKey(tierKey)
					if !ok {
						t.Fatal(tierKey)
					}
					attempts, wins, completed, rounds, hpLoss := 0, 0, 0, 0, 0
					bossAttempts, bossWins, woundedStarts, terminalWins := 0, 0, 0, 0
					var baseGold int64
					for run := range runs {
						party := cloneAbyssShadowUsers([]UserInCombat{base})
						for depth := 1; depth <= floors; depth++ {
							beforeHP := party[0].CurrentHP
							if beforeHP < 1 {
								t.Fatal("attempted to continue a dead character")
							}
							if beforeHP < base.Stats.HP {
								woundedStarts++
							}
							seed := uint64(20260913 + run*100 + depth)
							mobs, zone, difficulty := abyssBalancePlaytestEncounter(depth, base.Level, tier, seed)
							random := rand.New(rand.NewPCG(seed, seed^0xbf58476d1ce4e5b9))
							_, _, victory, _, timeline, _ := (&Bot{}).resolveChannelCombatDetailedWithRandom(party, mobs, base.Level, difficulty, zone, random)
							attempts++
							_, boss := abyssDifficulty(depth)
							if boss {
								bossAttempts++
							}
							if len(timeline) == 0 || timeline[0].HP != beforeHP {
								t.Fatalf("floor %d did not start from carried HP %d", depth, beforeHP)
							}
							// Round zero starts each wave; count distinct positive round
							// transitions, not individual player/enemy exchanges.
							previousRound := 0
							for _, frame := range timeline {
								if frame.Round > 0 && frame.Round != previousRound {
									rounds++
								}
								previousRound = frame.Round
							}
							// Raw engine HP may be negative after a lethal overkill;
							// measure remaining health with the timeline's zero clamp.
							afterHP := max(0, party[0].CurrentHP)
							if afterHP > base.Stats.HP {
								t.Fatalf("floor %d invalid remaining HP %d", depth, afterHP)
							}
							hpLoss += max(0, beforeHP-afterHP)
							if !victory {
								break
							}
							wins++
							baseGold += abyssFloorBonus(depth, base.Level)
							if boss {
								bossWins++
							}
							// The engine can report victory on a mutual kill. Preserve
							// that outcome, but never start another floor while dead.
							if afterHP == 0 {
								terminalWins++
								break
							}
							if depth == floors {
								completed++
							}
						}
					}
					if attempts < runs || rounds < attempts {
						t.Fatalf("experiment failed to exercise attrition: attempts=%d wounded=%d rounds=%d", attempts, woundedStarts, rounds)
					}
					if profile.name == "new" && tierKey == "normal" && build == "basic" && woundedStarts == 0 {
						t.Fatal("baseline did not exercise a wounded character on a later floor")
					}
					t.Logf("runs=%d complete_20=%d fights=%d wins=%d boss_wins=%d/%d wounded_starts=%d terminal_wins=%d mean_observed_rounds=%.2f mean_HP_loss_per_fight=%.2f mean_floors_won_per_run=%.2f projected_base_gold_per_run=%.2f consumables=0 persisted_rewards=unmeasured", runs, completed, attempts, wins, bossWins, bossAttempts, woundedStarts, terminalWins, float64(rounds)/float64(attempts), float64(hpLoss)/float64(attempts), float64(wins)/runs, float64(baseGold)/runs)
				})
			}
		}
	}
}
