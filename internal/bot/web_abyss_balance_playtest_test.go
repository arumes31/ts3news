package bot

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"testing"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/leveling"
)

// These isolated full-health snapshots use the production engine and catalog.
// Persistent bonuses, skills, pets, consumables, events and run attrition are
// absent. Adjacent ordinary floors prevent a boss-only matrix hiding hordes.
func TestAbyssBalanceCombatPlaytest(t *testing.T) {
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
		user := abyssBalancePlaytestUser(profile.level, profile.rarity)
		t.Logf("profile=%s level=%d slots=%d HP=%d STR=%d DEF=%d", profile.name, user.Level, len(user.Equipped), user.Stats.HP, user.Stats.STR, user.Stats.DEF)
		for _, key := range []string{"normal", "nightmare", "hell", "insanity"} {
			tier, ok := abyssTierByKey(key)
			if !ok {
				t.Fatal(key)
			}
			for _, depth := range []int{1, 19, 20, 49, 50, 99, 100, 199, 200, 399, 400} {
				t.Run(fmt.Sprintf("%s/%s/floor_%d", profile.name, key, depth), func(t *testing.T) {
					wins, trials, hpSum, enemyHPSum := 0, 0, 0, 0
					for encounter := range 8 {
						mobs, zone, difficulty := abyssBalancePlaytestEncounter(depth, user.Level, tier, uint64(encounter+1))
						for _, mob := range mobs {
							if mob.MaxHP < 1 || mob.Stats.STR < 1 {
								t.Fatalf("invalid mob: %+v", mob)
							}
							enemyHPSum += mob.MaxHP
						}
						result, err := (&Bot{}).simulatePreparedAbyssCombat(context.Background(), []UserInCombat{user}, mobs, user.Level, difficulty, zone, 8, [2]uint64{uint64(depth), uint64(encounter + 50)})
						if err != nil {
							t.Fatal(err)
						}
						wins += result.Wins
						trials += result.Trials
						hpSum += result.MedianWinHPPct
					}
					t.Logf("wins=%d/%d mean_encounter_HP=%d mean_median_winner_HP=%d%% base_gold=%d", wins, trials, enemyHPSum/8, hpSum/8, abyssFloorBonus(depth, user.Level))
					if depth == 1 && key == "normal" && wins < trials/2 {
						t.Errorf("normal entrance must be approachable with catalog gear: %d/%d wins", wins, trials)
					}
				})
			}
		}
	}
}

func abyssBalancePlaytestUser(level int, rarity content.Rarity) UserInCombat {
	stats := content.Stats{HP: 100 + level*5, STR: 10 + level, DEF: 5 + level/2, SPD: 10 + level, LCK: level / 5, INT: level / 10, STA: level / 10, CRT: 5 + level/50, DGE: 5 + level/50}
	equipped := map[content.GearSlot]content.Gear{}
	catalog := content.GearAppearanceCatalog()
	for _, slot := range content.AllSlots {
		if content.IsPetGearSlot(slot) {
			continue
		}
		var candidates []content.Gear
		for _, gear := range catalog {
			if gear.Slot == slot && gear.Rarity == rarity && !content.IsAbyssGearID(gear.ID) && !content.IsInsanityGearID(gear.ID) {
				candidates = append(candidates, gear)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].CombatRating() == candidates[j].CombatRating() {
				return candidates[i].ID < candidates[j].ID
			}
			return candidates[i].CombatRating() < candidates[j].CombatRating()
		})
		gear := candidates[len(candidates)/2]
		equipped[slot] = gear
		stats = stats.Add(gear.Stats)
	}
	return UserInCombat{UID: "balance", Nickname: "Balance", Level: level, Stats: stats, CurrentHP: stats.HP, Equipped: equipped, EscrowLoot: true, IsClone: true, shadow: true}
}

func abyssBalancePlaytestEncounter(depth, level int, tier abyssTier, seed uint64) ([]*content.Mob, content.Zone, float64) {
	random := rand.New(rand.NewPCG(seed, seed+100))
	difficulty, boss := abyssDifficulty(depth)
	difficulty *= tier.DiffMult
	biome := content.AbyssBiomeForAffinity(depth, "", int(seed))
	difficulty *= biome.DiffMod
	mobLevel := abyssMobLevel(depth, level)
	zone := content.GetRandomZoneWithRandom(level, 0, random)
	var mobs []content.Mob
	if boss {
		mobs = abyssBossEncounterAt(depth, mobLevel, difficulty, time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC))
	} else {
		mobs = content.SpawnMobGroupWithRandom(mobLevel, zone, difficulty*zone.Difficulty, 1, false, random)
	}
	escalateMobsWithRandom(mobs, depth, boss && depth%10 == 0, random)
	out := make([]*content.Mob, len(mobs))
	for i := range mobs {
		mobs[i].Stats.STR = max(1, int(float64(mobs[i].Stats.STR)*abyssMobDamageMult))
		mobs[i].MaxHP = mobs[i].Stats.HP
		mobs[i].CurrentHP = mobs[i].MaxHP
		out[i] = &mobs[i]
	}
	return out, zone, difficulty
}

// Forecasts inspect real per-roll bands, without claiming persisted escrow
// grants. XP uses the actual floor reward, not discarded mob RewardXP.
func TestAbyssBalanceRewardPlaytest(t *testing.T) {
	for _, level := range []int{10, 100, 300, 1000, PrestigeThreshold - 1} {
		step := leveling.XPForLevel(level+1) - leveling.XPForLevel(level)
		t.Logf("level=%d XP_to_next=%d", level, step)
		for _, depth := range []int{1, 20, 50, 100, 200, 400} {
			tier, _ := abyssTierByKey("normal")
			_, zone, _ := abyssBalancePlaytestEncounter(depth, level, tier, 17)
			fc := abyssDropForecastData(max(1, zone.Difficulty), lootRarityScale(abyssMobLevel(depth, level)))
			sum := fc.Ultimate + fc.Title + fc.Unique + fc.Artifact + fc.Enchant + fc.Skill + fc.Consumable + fc.Gear + fc.Common
			if math.Abs(sum-1) > 1e-9 || fc.Gear <= 0 {
				t.Errorf("level=%d depth=%d invalid or unreachable gear bands: %+v", level, depth, fc)
			}
			totalXP := 0
			for roll := 1; roll <= 20; roll++ {
				totalXP += abyssCombatFloorXP(roll, depth, tier, true)
			}
			meanXP := float64(totalXP) / 20
			t.Logf("level=%d depth=%d ordinary_gear_band=%.1f%% common_band=%.1f%% mean_XP=%.1f mean_floors_to_next_level=%.1f", level, depth, fc.Gear*100, fc.Common*100, meanXP, float64(step)/meanXP)
		}
	}
}

func TestAbyssBalanceHighQualityKeepsEveryLootCategory(t *testing.T) {
	for _, quality := range []float64{4, 10.798, 100, 1000} {
		forecast := abyssDropForecastData(quality, 0.8)
		bands := []float64{forecast.Ultimate, forecast.Title, forecast.Unique, forecast.Artifact, forecast.Enchant, forecast.Skill, forecast.Consumable, forecast.Gear, forecast.Common}
		sum := 0.0
		for _, probability := range bands {
			if probability <= 0 || probability >= 1 {
				t.Errorf("quality %.3f starves or monopolizes a loot category: %+v", quality, forecast)
			}
			sum += probability
		}
		if math.Abs(sum-1) > 1e-9 || forecast.Common < 0.1-1e-9 {
			t.Errorf("quality %.3f must preserve a normalized distribution and common drops: %+v", quality, forecast)
		}
		wantRatio := gearChance / (ultimateSkillChance * 0.8)
		if got := forecast.Gear / forecast.Ultimate; math.Abs(got-wantRatio) > 1e-9 {
			t.Errorf("quality %.3f changed relative gear/ultimate weights: %.4f want %.4f", quality, got, wantRatio)
		}
	}
}

func TestAbyssBalanceBaselineLootProbabilitiesRemainStable(t *testing.T) {
	forecast := abyssDropForecastData(1, 1)
	for _, check := range []struct {
		name string
		got  float64
		want float64
	}{
		{"ultimate", forecast.Ultimate, ultimateSkillChance},
		{"title", forecast.Title, titleChance},
		{"unique", forecast.Unique, uniqueItemChance},
		{"artifact", forecast.Artifact, artifactChance},
		{"enchant", forecast.Enchant, enchChance},
		{"skill", forecast.Skill, skillChance},
		{"consumable", forecast.Consumable, consChance},
		{"gear", forecast.Gear, gearChance},
	} {
		if math.Abs(check.got-check.want) > 1e-9 {
			t.Errorf("baseline %s probability %.5f want %.5f", check.name, check.got, check.want)
		}
	}
}

func TestAbyssBalanceFloorXPRewardsRiskWithoutSuicideFarming(t *testing.T) {
	for _, test := range []struct {
		name    string
		roll    int
		depth   int
		tier    string
		victory bool
		want    int
	}{
		{"entrance", 10, 1, "normal", true, 10},
		{"deeper", 10, 50, "normal", true, 19},
		{"floor400", 10, 400, "normal", true, 89},
		{"depth cap", 10, 100000, "normal", true, 90},
		{"hell", 10, 50, "hell", true, 38},
		{"insanity", 10, 400, "insanity", true, 281},
		{"low roll clamped", -2, 1, "normal", true, 1},
		{"high roll clamped", 200, 1, "normal", true, 20},
		{"negative depth", 10, -100, "normal", true, 10},
		{"loss minimum", 1, 400, "insanity", false, 1},
		{"loss rounding", 10, 400, "insanity", false, 3},
		{"loss maximum", 20, 100000, "insanity", false, 5},
	} {
		t.Run(test.name, func(t *testing.T) {
			tier, ok := abyssTierByKey(test.tier)
			if !ok {
				t.Fatal(test.tier)
			}
			if got := abyssCombatFloorXP(test.roll, test.depth, tier, test.victory); got != test.want {
				t.Errorf("floor XP = %d want %d", got, test.want)
			}
		})
	}
}
