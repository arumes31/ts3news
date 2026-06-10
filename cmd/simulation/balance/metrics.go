package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// FightRecord stores per-fight data for analysis.
type FightRecord struct {
	FightNum       int
	GroupSize      int
	Victory        bool
	Waves          int
	Rounds         int
	MobsKilled     int
	XPGained       float64
	XPPenalty      float64
	GoldGained     int64
	AvgPlayerHP    float64
	DamageDealt    int64
	DamageReceived int64
	AvgLevel       float64
	AvgGearScore   float64
}

// AggregateMetrics stores summary statistics after a full simulation run.
type AggregateMetrics struct {
	GroupSize         int
	TotalFights       int
	TotalWins         int
	TotalLosses       int
	WinRate           float64
	AvgLevel          float64
	MaxLevel          int
	AvgPrestige       float64
	AvgGold           int64
	AvgGearScore      float64
	AvgGearSlots      float64
	AvgItemLevel      float64
	AvgMobsKilled     float64
	AvgRounds         float64
	AvgXPGained       float64
	AvgGoldGained     int64
	AvgDamageDealt    int64
	AvgDamageReceived int64
	AHListed          int
	AHSold            int
	AHSellRate        float64
	AHGoldTraded      int64
	TotalSystemGold   int64
	InflationIndex    float64
	PrestigeCount     int
	LootByRarity      map[string]int
	LootByType        map[string]int
	FinalParams       SimParams
}

// MetricCollector gathers data during a simulation run.
type MetricCollector struct {
	FightRecords []FightRecord
	Players      []*SimPlayer
	AH           *SimAuctionHouse
	Economy      *SimGoldEconomy
	Params       SimParams
	GroupSize    int
}

// NewMetricCollector creates a new collector.
func NewMetricCollector(groupSize int, params SimParams) *MetricCollector {
	return &MetricCollector{
		GroupSize: groupSize,
		Params:    params,
		AH:        &SimAuctionHouse{},
		Economy:   &SimGoldEconomy{},
	}
}

// RecordFight adds a fight record.
func (mc *MetricCollector) RecordFight(fightNum int, result SimCombatResult, players []*SimPlayer) {
	avgHP := 0.0
	avgLvl := 0.0
	avgGS := 0.0
	for _, p := range players {
		avgHP += float64(p.CurrentHP)
		avgLvl += float64(p.Level)
		gs := 0.0
		for _, g := range p.Gear {
			if g.Durability > 0 {
				gs += float64(g.Stats.Score())
			}
		}
		avgGS += gs / 30.0
	}
	n := float64(len(players))
	if n > 0 {
		avgHP /= n
		avgLvl /= n
		avgGS /= n
	}

	mc.FightRecords = append(mc.FightRecords, FightRecord{
		FightNum:       fightNum,
		GroupSize:      mc.GroupSize,
		Victory:        result.Victory,
		Waves:          result.Waves,
		Rounds:         result.TotalRounds,
		MobsKilled:     result.MobsKilled,
		XPGained:       result.XPGained,
		XPPenalty:      result.XPPenalty,
		GoldGained:     result.GoldGained,
		AvgPlayerHP:    avgHP,
		DamageDealt:    result.DamageDealt,
		DamageReceived: result.DamageReceived,
		AvgLevel:       avgLvl,
		AvgGearScore:   avgGS,
	})
}

// ComputeAggregate calculates summary metrics from collected data.
func (mc *MetricCollector) ComputeAggregate() AggregateMetrics {
	am := AggregateMetrics{
		GroupSize:    mc.GroupSize,
		FinalParams:  mc.Params,
		LootByRarity: make(map[string]int),
		LootByType:   make(map[string]int),
	}

	if len(mc.FightRecords) == 0 {
		return am
	}

	totalWins := 0
	totalMobs := 0
	totalRounds := 0
	totalXP := 0.0
	totalGold := int64(0)
	totalDmgDealt := int64(0)
	totalDmgRecv := int64(0)

	for _, fr := range mc.FightRecords {
		am.TotalFights++
		if fr.Victory {
			totalWins++
		}
		totalMobs += fr.MobsKilled
		totalRounds += fr.Rounds
		totalXP += fr.XPGained
		totalGold += fr.GoldGained
		totalDmgDealt += fr.DamageDealt
		totalDmgRecv += fr.DamageReceived
	}

	am.TotalWins = totalWins
	am.TotalLosses = am.TotalFights - totalWins
	if am.TotalFights > 0 {
		am.WinRate = float64(totalWins) / float64(am.TotalFights) * 100.0
		am.AvgMobsKilled = float64(totalMobs) / float64(am.TotalFights)
		am.AvgRounds = float64(totalRounds) / float64(am.TotalFights)
		am.AvgXPGained = totalXP / float64(am.TotalFights)
		am.AvgGoldGained = totalGold / int64(am.TotalFights)
		am.AvgDamageDealt = totalDmgDealt / int64(am.TotalFights)
		am.AvgDamageReceived = totalDmgRecv / int64(am.TotalFights)
	}

	// Player stats
	if len(mc.Players) > 0 {
		totalLvl := 0
		maxLvl := 0
		totalPrestige := 0
		totalGold := int64(0)
		totalGS := 0.0
		totalSlots := 0.0
		totalILvl := 0.0
		n := float64(len(mc.Players))

		for _, p := range mc.Players {
			totalLvl += p.Level
			if p.Level > maxLvl {
				maxLvl = p.Level
			}
			totalPrestige += p.Prestige
			totalGold += p.Gold
			gs := 0.0
			ilvl := 0.0
			gearCount := 0
			for _, g := range p.Gear {
				if g.Durability > 0 {
					gs += float64(g.Stats.Score())
					ilvl += g.ItemLevel
					gearCount++
				}
			}
			totalGS += gs / 30.0
			totalSlots += float64(gearCount)
			if gearCount > 0 {
				totalILvl += ilvl / float64(gearCount)
			}
		}

		am.AvgLevel = float64(totalLvl) / n
		am.MaxLevel = maxLvl
		am.AvgPrestige = float64(totalPrestige) / n
		am.AvgGold = totalGold / int64(n)
		am.AvgGearScore = totalGS / n
		am.AvgGearSlots = totalSlots / n
		am.AvgItemLevel = totalILvl / n
		am.PrestigeCount = totalPrestige
	}

	// Auction house stats
	am.AHListed = mc.AH.TotalListed
	am.AHSold = mc.AH.TotalSold
	if am.AHListed > 0 {
		am.AHSellRate = float64(am.AHSold) / float64(am.AHListed) * 100.0
	}
	am.AHGoldTraded = mc.AH.TotalGoldTraded
	am.TotalSystemGold = mc.Economy.TotalSystemGold
	if am.TotalSystemGold > 0 {
		am.InflationIndex = float64(am.TotalSystemGold) / float64(mc.Params.InflationThreshold)
	}

	return am
}

// PrintSummary outputs aggregate metrics to the console.
func PrintSummary(am AggregateMetrics) {
	fmt.Printf("\n╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║  BALANCE SIMULATION RESULTS — Group Size: %d                  ║\n", am.GroupSize)
	fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║  COMBAT                                                      ║\n")
	fmt.Printf("║  Win Rate:        %6.1f%%  (%dW / %dL / %d total)            \n",
		am.WinRate, am.TotalWins, am.TotalLosses, am.TotalFights)
	fmt.Printf("║  Avg Mobs Killed: %6.1f                                     \n", am.AvgMobsKilled)
	fmt.Printf("║  Avg Rounds:      %6.1f                                     \n", am.AvgRounds)
	fmt.Printf("║  Avg Dmg Dealt:   %6d                                       \n", am.AvgDamageDealt)
	fmt.Printf("║  Avg Dmg Recv:    %6d                                       \n", am.AvgDamageReceived)
	fmt.Printf("║                                                              ║\n")
	fmt.Printf("║  PROGRESSION                                                 ║\n")
	fmt.Printf("║  Avg Level:       %6.1f  (Max: %d)                          \n", am.AvgLevel, am.MaxLevel)
	fmt.Printf("║  Avg Prestige:    %6.1f  (Total: %d)                        \n", am.AvgPrestige, am.PrestigeCount)
	fmt.Printf("║  Avg XP/Fight:    %6.1f                                     \n", am.AvgXPGained)
	fmt.Printf("║                                                              ║\n")
	fmt.Printf("║  GEAR                                                         ║\n")
	fmt.Printf("║  Avg Gear Score:  %6.1f                                     \n", am.AvgGearScore)
	fmt.Printf("║  Avg Gear Slots:  %6.1f / 30                                \n", am.AvgGearSlots)
	fmt.Printf("║  Avg Item Level:  %6.1f                                     \n", am.AvgItemLevel)
	fmt.Printf("║                                                              ║\n")
	fmt.Printf("║  ECONOMY                                                      ║\n")
	fmt.Printf("║  Avg Gold:        %6d                                       \n", am.AvgGold)
	fmt.Printf("║  Avg Gold/Fight:  %6d                                       \n", am.AvgGoldGained)
	fmt.Printf("║  System Gold:     %6d  (Inflation: %.2fx)                   \n", am.TotalSystemGold, am.InflationIndex)
	fmt.Printf("║  AH Listed:       %6d  Sold: %d (%.1f%% sell rate)          \n", am.AHListed, am.AHSold, am.AHSellRate)
	fmt.Printf("║  AH Gold Traded:  %6d                                       \n", am.AHGoldTraded)
	fmt.Printf("║                                                              ║\n")
	fmt.Printf("║  TUNED PARAMS                                                ║\n")
	fmt.Printf("║  MobHPMult:       %6.3f                                     \n", am.FinalParams.MobHPMult)
	fmt.Printf("║  MobSTRMult:      %6.3f                                     \n", am.FinalParams.MobSTRMult)
	fmt.Printf("║  MobDEFMult:      %6.3f                                     \n", am.FinalParams.MobDEFMult)
	fmt.Printf("║  PlayerHPMult:    %6.3f                                     \n", am.FinalParams.PlayerHPMult)
	fmt.Printf("║  PlayerSTRMult:   %6.3f                                     \n", am.FinalParams.PlayerSTRMult)
	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
}

// WriteCSV writes fight records to a CSV file.
func WriteCSV(records []FightRecord, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Header
	header := "fight_num,group_size,victory,waves,rounds,mobs_killed,xp_gained,xp_penalty,gold_gained,avg_player_hp,damage_dealt,damage_received,avg_level,avg_gear_score\n"
	if _, err := f.WriteString(header); err != nil {
		return err
	}

	// Data
	for _, r := range records {
		victory := "0"
		if r.Victory {
			victory = "1"
		}
		line := fmt.Sprintf("%d,%d,%s,%d,%d,%d,%.1f,%.1f,%d,%.1f,%d,%d,%.1f,%.1f\n",
			r.FightNum, r.GroupSize, victory, r.Waves, r.Rounds, r.MobsKilled,
			r.XPGained, r.XPPenalty, r.GoldGained, r.AvgPlayerHP,
			r.DamageDealt, r.DamageReceived, r.AvgLevel, r.AvgGearScore)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}

// WriteJSON writes aggregate metrics to a JSON file.
func WriteJSON(am AggregateMetrics, filename string) error {
	// Convert to a JSON-friendly format
	data := map[string]interface{}{
		"group_size":          am.GroupSize,
		"total_fights":        am.TotalFights,
		"total_wins":          am.TotalWins,
		"total_losses":        am.TotalLosses,
		"win_rate":            am.WinRate,
		"avg_level":           am.AvgLevel,
		"max_level":           am.MaxLevel,
		"avg_prestige":        am.AvgPrestige,
		"avg_gold":            am.AvgGold,
		"avg_gear_score":      am.AvgGearScore,
		"avg_gear_slots":      am.AvgGearSlots,
		"avg_item_level":      am.AvgItemLevel,
		"avg_mobs_killed":     am.AvgMobsKilled,
		"avg_rounds":          am.AvgRounds,
		"avg_xp_gained":       am.AvgXPGained,
		"avg_gold_gained":     am.AvgGoldGained,
		"avg_damage_dealt":    am.AvgDamageDealt,
		"avg_damage_received": am.AvgDamageReceived,
		"ah_listed":           am.AHListed,
		"ah_sold":             am.AHSold,
		"ah_sell_rate":        am.AHSellRate,
		"ah_gold_traded":      am.AHGoldTraded,
		"total_system_gold":   am.TotalSystemGold,
		"inflation_index":     am.InflationIndex,
		"prestige_count":      am.PrestigeCount,
		"params": map[string]interface{}{
			"mob_hp_mult":     am.FinalParams.MobHPMult,
			"mob_str_mult":    am.FinalParams.MobSTRMult,
			"mob_def_mult":    am.FinalParams.MobDEFMult,
			"player_hp_mult":  am.FinalParams.PlayerHPMult,
			"player_str_mult": am.FinalParams.PlayerSTRMult,
		},
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, jsonData, 0644)
}

// ParseGroupSizes parses a comma-separated list of group sizes.
func ParseGroupSizes(s string) []int {
	if s == "" {
		return []int{1, 4, 5, 6}
	}
	var sizes []int
	for _, part := range splitByComma(s) {
		n, err := strconv.Atoi(part)
		if err == nil && n > 0 && n <= 20 {
			sizes = append(sizes, n)
		}
	}
	if len(sizes) == 0 {
		return []int{1, 4, 5, 6}
	}
	return sizes
}

func splitByComma(s string) []string {
	var parts []string
	current := ""
	for _, c := range s {
		if c == ',' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
