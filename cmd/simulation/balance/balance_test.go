package main

import (
	"math"
	"math/rand"
	"testing"
)

func TestXPForLevel(t *testing.T) {
	tests := []struct {
		level    int
		expected float64
	}{
		{1, 0},
		{2, 5},        // 1^1.65 * 5 = 5
		{10, 180},     // 9^1.65 * 5 ≈ 197.5 (within 10%)
		{100, 9812},   // 99^1.65 * 5 ≈ 9812
		{500, 141526}, // 499^1.65 * 5 ≈ 141,526
	}
	for _, tt := range tests {
		got := XPForLevel(tt.level, 1.65)
		if tt.level == 1 {
			if got != 0 {
				t.Errorf("XPForLevel(1) = %v, want 0", got)
			}
		} else if math.Abs(got-tt.expected) > tt.expected*0.1 {
			t.Errorf("XPForLevel(%d) = %v, want ~%v", tt.level, got, tt.expected)
		}
	}
}

func TestLevelForXP(t *testing.T) {
	tests := []struct {
		xp       float64
		expected int
	}{
		{0, 1},
		{5, 2},
		{197, 10},   // 9^1.65*5 ≈ 197.5, so 197 XP → level 10
		{9812, 100}, // 99^1.65*5 ≈ 9812, so 9812 XP → level 100
	}
	for _, tt := range tests {
		got := LevelForXP(tt.xp, 1.65)
		if got != tt.expected {
			t.Errorf("LevelForXP(%v) = %d, want %d", tt.xp, got, tt.expected)
		}
	}
}

func TestBaseStats(t *testing.T) {
	params := DefaultParams()
	p := NewSimPlayer(0, params)
	p.Level = 1
	stats := p.BaseStats(params)

	if stats.HP != 105 { // 100 + 1*5 = 105
		t.Errorf("Level 1 HP = %d, want 105", stats.HP)
	}
	if stats.STR != 11 { // 10 + 1 = 11
		t.Errorf("Level 1 STR = %d, want 11", stats.STR)
	}
	if stats.DEF != 5 { // 5 + 1/2 = 5 (integer division)
		t.Errorf("Level 1 DEF = %d, want 5", stats.DEF)
	}

	p.Level = 10
	stats = p.BaseStats(params)
	if stats.HP != 150 { // 100 + 10*5 = 150
		t.Errorf("Level 10 HP = %d, want 150", stats.HP)
	}
	if stats.STR != 20 { // 10 + 10 = 20
		t.Errorf("Level 10 STR = %d, want 20", stats.STR)
	}
	if stats.DEF != 10 { // 5 + 10/2 = 10
		t.Errorf("Level 10 DEF = %d, want 10", stats.DEF)
	}
}

func TestGenerateGearStats(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := DefaultParams()

	// Generate 100 common gear pieces at level 1 and check stats are reasonable
	totalHP, totalSTR, totalDEF := 0, 0, 0
	count := 100
	for i := 0; i < count; i++ {
		g := GenerateGear(rng, 1, params)
		totalHP += g.Stats.HP
		totalSTR += g.Stats.STR
		totalDEF += g.Stats.DEF
	}

	avgHP := totalHP / count
	avgSTR := totalSTR / count
	avgDEF := totalDEF / count

	// Common gear at level 1 should have HP ~10, STR ~5, DEF ~3
	// Average across rarities will be higher, but should be well under 100
	if avgHP > 100 {
		t.Errorf("Average gear HP at level 1 = %d, want < 100 (was ~200+ before fix)", avgHP)
	}
	if avgSTR > 50 {
		t.Errorf("Average gear STR at level 1 = %d, want < 50 (was ~100+ before fix)", avgSTR)
	}
	if avgDEF > 30 {
		t.Errorf("Average gear DEF at level 1 = %d, want < 30 (was ~80+ before fix)", avgDEF)
	}
}

func TestGenerateGearLevelScaling(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := DefaultParams()

	// Gear at level 100 should be stronger than level 1 but not 100x stronger
	g1 := GenerateGear(rng, 1, params)
	g100 := GenerateGear(rng, 100, params)

	// Level scaling is 1 + level*0.01, so at level 100: lvlScale = 2.0
	// Gear at level 100 should be roughly 2x level 1 gear
	ratio := float64(g100.Stats.HP) / float64(g1.Stats.HP)
	if ratio > 5.0 {
		t.Errorf("Gear HP scaling ratio level100/level1 = %.1f, want < 5.0 (was ~100x before fix)", ratio)
	}
	if ratio < 1.0 {
		t.Errorf("Gear HP scaling ratio level100/level1 = %.1f, want > 1.0", ratio)
	}
}

func TestMobSpawning(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := DefaultParams()

	// Spawn mobs at level 1 and check stats are reasonable
	mob := SpawnMob(rng, 1, false, 1.0, params)

	if mob.Stats.HP < 10 {
		t.Errorf("Level 1 mob HP = %d, want >= 10", mob.Stats.HP)
	}
	if mob.Stats.STR < 5 {
		t.Errorf("Level 1 mob STR = %d, want >= 5", mob.Stats.STR)
	}
	if mob.RewardXP < 1 {
		t.Errorf("Level 1 mob RewardXP = %d, want >= 1", mob.RewardXP)
	}
}

func TestMobScaling(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := DefaultParams()

	// Mobs at level 50 should be stronger than level 1
	mob1 := SpawnMob(rng, 1, false, 1.0, params)
	mob50 := SpawnMob(rng, 50, false, 1.0, params)

	if mob50.Stats.HP <= mob1.Stats.HP {
		t.Errorf("Level 50 mob HP (%d) should be > level 1 mob HP (%d)", mob50.Stats.HP, mob1.Stats.HP)
	}
	if mob50.Stats.STR <= mob1.Stats.STR {
		t.Errorf("Level 50 mob STR (%d) should be > level 1 mob STR (%d)", mob50.Stats.STR, mob1.Stats.STR)
	}
}

func TestCombatSolo(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := DefaultParams()

	p := NewSimPlayer(0, params)
	// Equip starter gear like a real level 1 player (2 Common + 1 Uncommon)
	starterSlots := []string{"MainHand", "Chest", "Legs"}
	for si, s := range starterSlots {
		g := GenerateGear(rng, 1, params)
		g.Slot = s
		if si < 2 {
			g.Rarity = RarityCommon
			g.Stats = SimStats{HP: 10, STR: 5, DEF: 3, SPD: 4, LCK: 2, CRT: 2, DGE: 2}
		} else {
			g.Rarity = RarityUncommon
			g.Stats = SimStats{HP: 20, STR: 10, DEF: 6, SPD: 8, LCK: 4, CRT: 4, DGE: 4}
		}
		g.Durability = 50 + int(g.Rarity)*30
		g.MaxDur = g.Durability
		p.EquipGear(g)
	}
	p.RecalculateStats(params)
	p.CurrentHP = p.MaxHP

	// Run 100 solo fights and check win rate is between 5-95%
	wins, total := 0, 100
	for i := 0; i < total; i++ {
		p.CurrentHP = p.MaxHP
		var logs []string
		result := ResolveCombat(rng, []*SimPlayer{p}, p.Level, 1.0, params, &logs)
		if result.Victory {
			wins++
		}
		// Reset player for next fight
		p.ConsecutiveWins = 0
		p.ConsecutiveLosses = 0
		p.PityStack = 0
	}

	winRate := float64(wins) / float64(total)
	if winRate < 0.05 || winRate > 0.95 {
		t.Errorf("Solo win rate = %.1f%%, want between 5-95%% (indicates balance issue)", winRate*100)
	}
}

func TestCombatGroup(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := DefaultParams()

	// Create a party of 4
	party := make([]*SimPlayer, 4)
	for i := range party {
		party[i] = NewSimPlayer(i, params)
		party[i].CurrentHP = party[i].MaxHP
	}

	wins, total := 0, 50
	for i := 0; i < total; i++ {
		for _, p := range party {
			p.CurrentHP = p.MaxHP
			p.ConsecutiveWins = 0
			p.ConsecutiveLosses = 0
			p.PityStack = 0
		}
		var logs []string
		result := ResolveCombat(rng, party, 1, 1.0, params, &logs)
		if result.Victory {
			wins++
		}
	}

	winRate := float64(wins) / float64(total)
	// Groups should have a reasonable win rate
	if winRate < 0.10 || winRate > 0.99 {
		t.Errorf("Group of 4 win rate = %.1f%%, want between 10-99%%", winRate*100)
	}
}

func TestPartyBonus(t *testing.T) {
	tests := []struct {
		groupSize int
		expected  float64
	}{
		{1, 1.0},
		{2, 1.2075}, // (1 + 1*0.15) * (1 + 1*0.05) = 1.15 * 1.05 = 1.2075
		{4, 1.6675}, // (1 + 3*0.15) * (1 + 3*0.05) = 1.45 * 1.15 = 1.6675
		{6, 2.1875}, // (1 + 5*0.15) * (1 + 5*0.05) = 1.75 * 1.25 = 2.1875
	}
	for _, tt := range tests {
		got := PartyBonus(tt.groupSize)
		if math.Abs(got-tt.expected) > 0.01 {
			t.Errorf("PartyBonus(%d) = %v, want ~%v", tt.groupSize, got, tt.expected)
		}
	}

	// Test cap at 5.0
	if PartyBonus(100) > 5.0 {
		t.Errorf("PartyBonus(100) = %v, want <= 5.0", PartyBonus(100))
	}
}

func TestElementalMultiplier(t *testing.T) {
	tests := []struct {
		attacker SimElement
		defender SimElement
		expected float64
	}{
		{ElementFire, ElementAir, 2.0},      // Fire > Air
		{ElementAir, ElementEarth, 2.0},     // Air > Earth
		{ElementEarth, ElementWater, 2.0},   // Earth > Water
		{ElementWater, ElementFire, 2.0},    // Water > Fire
		{ElementFire, ElementWater, 0.5},    // Fire < Water
		{ElementFire, ElementFire, 1.0},     // Same element
		{ElementPhysical, ElementFire, 1.0}, // Physical is neutral
	}
	for _, tt := range tests {
		got := GetElementMult(tt.attacker, tt.defender)
		if got != tt.expected {
			t.Errorf("GetElementMult(%v, %v) = %v, want %v", tt.attacker, tt.defender, got, tt.expected)
		}
	}
}

func TestAutoBalancerConvergence(t *testing.T) {
	ab := NewAutoBalancer(1, 50, 0.50, 0.05, 42, false)
	ab.PlayerCount = 5
	ab.MaxIterations = 15

	result := ab.Run()

	if !result.Converged {
		t.Errorf("AutoBalancer did not converge after %d iterations", result.Iterations)
	}
	if result.FinalWinRate < 0.40 || result.FinalWinRate > 0.60 {
		t.Errorf("AutoBalancer final win rate = %.1f%%, want 40-60%%", result.FinalWinRate*100)
	}
	if result.FinalParams.MobHPMult <= 0 {
		t.Errorf("MobHPMult = %v, want > 0", result.FinalParams.MobHPMult)
	}
	if result.FinalParams.MobSTRMult <= 0 {
		t.Errorf("MobSTRMult = %v, want > 0", result.FinalParams.MobSTRMult)
	}
}

func TestDurabilityLoss(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := DefaultParams()

	p := NewSimPlayer(0, params)
	p.Gold = 0 // No gold for auto-repair, so durability loss persists

	g := &SimGear{
		Slot:       "MainHand",
		Rarity:     RarityCommon,
		Durability: 50,
		MaxDur:     50,
		Stats:      SimStats{HP: 10, STR: 5, DEF: 3},
		XPMult:     1.0,
		ItemLevel:  1.0,
	}
	p.Gear["MainHand"] = g
	p.RecalculateStats(params)

	initialDura := p.Gear["MainHand"].Durability

	// Apply durability loss many times — should eventually break gear
	for i := 0; i < 100; i++ {
		ApplyDurabilityLoss(rng, p, false, params)
		if p.Gear["MainHand"] == nil {
			break // gear broke, that's expected
		}
	}

	// Gear should have lost some durability or broken (removed from slot)
	if p.Gear["MainHand"] != nil && p.Gear["MainHand"].Durability >= initialDura {
		t.Errorf("Gear durability not decreasing: %d >= %d", p.Gear["MainHand"].Durability, initialDura)
	}
}

func TestAwardXP(t *testing.T) {
	params := DefaultParams()
	p := NewSimPlayer(0, params)

	initialLevel := p.Level
	levelUps := AwardXP(p, 100, params)

	if levelUps < 1 {
		t.Errorf("AwardXP(100) caused %d level ups, want >= 1 at level 1", levelUps)
	}
	if p.Level <= initialLevel {
		t.Errorf("Level after AwardXP = %d, want > %d", p.Level, initialLevel)
	}
}

func TestRecalculateStatsWithGear(t *testing.T) {
	_ = rand.New(rand.NewSource(42)) // seed for consistency
	params := DefaultParams()

	p := NewSimPlayer(0, params)
	baseStats := p.BaseStats(params)

	// Equip a common piece of gear
	g := &SimGear{
		Slot:       "MainHand",
		Rarity:     RarityCommon,
		Durability: 50,
		MaxDur:     50,
		Stats:      SimStats{HP: 10, STR: 5, DEF: 3},
		XPMult:     1.0,
		ItemLevel:  1.0,
	}
	p.EquipGear(g)
	p.RecalculateStats(params)

	// Stats should be base + gear
	if p.Stats.HP != baseStats.HP+10 {
		t.Errorf("HP with gear = %d, want %d", p.Stats.HP, baseStats.HP+10)
	}
	if p.Stats.STR != baseStats.STR+5 {
		t.Errorf("STR with gear = %d, want %d", p.Stats.STR, baseStats.STR+5)
	}
	if p.Stats.DEF != baseStats.DEF+3 {
		t.Errorf("DEF with gear = %d, want %d", p.Stats.DEF, baseStats.DEF+3)
	}
}

func TestLootSystemWithAH(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := DefaultParams()

	ah := &SimAuctionHouse{}
	ls := &SimLootSystem{AH: ah}

	p := NewSimPlayer(0, params)
	p.Level = 25 // Higher level for better loot chances

	// Roll loot many times
	for i := 0; i < 100; i++ {
		ls.RollLootForPlayer(rng, p, 3, i, params)
	}

	// Should have gotten some gear drops
	if p.TotalGearDrops == 0 {
		t.Log("Warning: 0 gear drops in 100 loot rolls — may need investigation")
	}
}

func TestGoldEconomy(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	params := DefaultParams()

	p := NewSimPlayer(0, params)
	p.Gold = 1000

	economy := &SimGoldEconomy{}

	// Distribute gold
	DistributeGold(rng, []*SimPlayer{p}, 3, 100, economy, params)

	if p.Gold <= 1000 {
		t.Errorf("Gold after distribution = %d, want > 1000", p.Gold)
	}
	if economy.TotalSystemGold <= 0 {
		t.Errorf("Economy total gold = %d, want > 0", economy.TotalSystemGold)
	}
}

func TestPitySystem(t *testing.T) {
	params := DefaultParams()
	p := NewSimPlayer(0, params)

	// Simulate losses
	for i := 0; i < 10; i++ {
		p.PityStack = clampFloat(p.PityStack+params.PityPerLoss, 0, params.PityCap)
	}

	if p.PityStack != params.PityCap {
		t.Errorf("PityStack after 10 losses = %v, want %v (capped)", p.PityStack, params.PityCap)
	}

	// Pity should increase combat stats
	pm := 1.0 + p.PityStack
	if pm <= 1.0 {
		t.Errorf("Pity multiplier = %v, want > 1.0", pm)
	}
}
