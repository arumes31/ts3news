package main

// SimParams holds all tunable knobs for the balance simulation.
// Default values mirror the real bot engine in internal/bot/xp.go
// and internal/content/mobs.go as closely as possible.
type SimParams struct {
	// === Combat Multipliers — PRIMARY BALANCE KNOBS ===
	MobHPMult  float64 // Mob HP scaling multiplier
	MobSTRMult float64 // Mob STR/DMG scaling multiplier
	MobDEFMult float64 // Mob DEF scaling multiplier

	PlayerHPMult  float64 // Player HP scaling multiplier
	PlayerSTRMult float64 // Player STR scaling multiplier
	PlayerDEFMult float64 // Player DEF scaling multiplier

	// === Mob Spawn Config ===
	BaseMobsMin  int     // Min mobs per wave (2)
	BaseMobsMax  int     // Max mobs per wave (4)
	HordeChance  float64 // 15% horde spawn
	BossChance   float64 // 10% boss spawn
	Wave2Chance  float64 // 20% 2nd wave
	Wave3Chance  float64 // 5% 3rd wave
	AmbushChance float64 // 50% mobs go first
	MaxRounds    int     // 20 rounds per wave

	// === Scaling Curves ===
	MobLevelScale    float64 // 0.05 per level
	DifficultyDampen float64 // 0.3 — only 30% of difficulty applies
	GroupSizeScale   float64 // 0.1 per extra player on mob stats
	GroupSizeCap     float64 // 2.5 max group scaling

	// === Player Progression ===
	// Stats use integer division like the real engine:
	//   HP: BaseHP + level*5, STR: BaseSTR + level, DEF: BaseDEF + level/2
	//   SPD: 10+level, LCK: level/5, INT: level/10, STA: level/10
	//   CRT: 5+level/50, DGE: 5+level/50
	BaseHP  int // 100
	BaseSTR int // 10
	BaseDEF int // 5

	PrestigeStatBonus float64 // 0.15 per prestige

	// === Loot Chances (per mob killed) ===
	GearDropChance      float64 // 0.10
	SkillDropChance     float64 // 0.05
	ArtifactChance      float64 // 0.01
	ConsumableChance    float64 // 0.10
	EnchantmentChance   float64 // 0.02
	UniqueItemChance    float64 // 0.01
	UltimateSkillChance float64 // 0.005

	// === Skill Config ===
	SkillProcChance   float64 // 0.30 per attack
	ComboBonus        float64 // 1.25x for same skill twice
	ChainAttackChance float64 // 0.30 for 3+ player groups
	ChainDamagePct    float64 // 0.20 of STR

	// === Ultimate Config ===
	UltimateCooldown  int
	UltimatePowerBase float64

	// === Defense Mechanics ===
	DodgeCap      int     // 25% max
	ParryChance   float64 // 10%
	StealthRound1 bool    // Skip first round mob attacks

	// === Durability ===
	DuraLossChance       float64 // 0.20 per fight
	DuraLossMin          int     // 3
	DuraLossMax          int     // 8
	DefeatDuraLossChance float64 // 0.35
	DefeatDuraLossMin    int     // 5
	DefeatDuraLossMax    int     // 15
	RepairCostPerPoint   int64   // 1 gold per point

	// === Economy ===
	GoldPerMobXP       float64 // Gold = mobXP * factor
	InflationThreshold int64   // 10M gold
	InflationRate      float64 // Decay denominator (5M)

	// === Pity ===
	PityPerLoss float64 // +0.5
	PityCap     float64 // 3.0

	// === Zone ===
	ZoneDiffMin float64 // 0.8
	ZoneDiffMax float64 // 1.5

	// === XP ===
	XPMin          int
	XPMax          int
	ExponentCap    float64
	PrestigeLevel  int
	DeathXPPenalty float64 // 0.05

	// === Intensify / Fatigue ===
	IntensifyPerRound float64 // 0.15 per round
	FatigueStartRound int     // Round 11
	FatiguePerRound   float64 // 0.10 per round past start
	FatigueFloor      float64 // 0.1 minimum

	// === Mob Effect Chances ===
	MobEffectChance    float64 // 0.30
	MobSpellChance     float64 // 0.20
	MobSpellCount      int     // 1 for common, 2 for boss+
	MobEquipItemCount  int     // 1-2 items per mob
	MobEquipItemChance float64 // 0.30 for 2nd item

	// === Death Effect ===
	DeathEffectChance       float64 // 0.10 common, 0.20 trash
	DeathEffectSummonWeight float64 // 0.40
	DeathEffectExploWeight  float64 // 0.20 (0.40-0.60)
	DeathEffectCurseWeight  float64 // 0.20 (0.60-0.80)
	DeathEffectXPWeight     float64 // 0.10 (0.80-0.90)
	DeathEffectLootWeight   float64 // 0.10 (0.90-1.00)

	// === Elemental ===
	ElementalAdvantage float64 // 2.0x
	ElementalDisadv    float64 // 0.5x
	MobElementChance   float64 // 0.40

	// === Misc ===
	MomentumChance              float64 // 0.10 for 10% STR boost
	LifestealHealPenaltyAfter10 float64 // healPenalty for lifesteal after round 10
}

// DefaultParams returns parameters that mirror the real bot engine.
func DefaultParams() SimParams {
	return SimParams{
		// Primary balance knobs — will be auto-tuned
		MobHPMult:  1.0,
		MobSTRMult: 1.0,
		MobDEFMult: 1.0,

		PlayerHPMult:  1.0,
		PlayerSTRMult: 1.0,
		PlayerDEFMult: 1.0,

		// Mob spawn
		BaseMobsMin:  2,
		BaseMobsMax:  4,
		HordeChance:  0.15,
		BossChance:   0.10,
		Wave2Chance:  0.20,
		Wave3Chance:  0.05,
		AmbushChance: 0.50,
		MaxRounds:    20,

		// Scaling
		MobLevelScale:    0.05,
		DifficultyDampen: 0.30,
		GroupSizeScale:   0.10,
		GroupSizeCap:     2.50,

		// Player progression (integer division like real engine)
		BaseHP:            100,
		BaseSTR:           10,
		BaseDEF:           5,
		PrestigeStatBonus: 0.15,

		// Loot
		GearDropChance:      0.10,
		SkillDropChance:     0.05,
		ArtifactChance:      0.01,
		ConsumableChance:    0.10,
		EnchantmentChance:   0.02,
		UniqueItemChance:    0.01,
		UltimateSkillChance: 0.005,

		// Skills
		SkillProcChance:   0.30,
		ComboBonus:        1.25,
		ChainAttackChance: 0.30,
		ChainDamagePct:    0.20,

		// Ultimate
		UltimateCooldown:  5,
		UltimatePowerBase: 5.0,

		// Defense
		DodgeCap:      25,
		ParryChance:   0.10,
		StealthRound1: true,

		// Durability
		DuraLossChance:       0.20,
		DuraLossMin:          3,
		DuraLossMax:          8,
		DefeatDuraLossChance: 0.35,
		DefeatDuraLossMin:    5,
		DefeatDuraLossMax:    15,
		RepairCostPerPoint:   1,

		// Economy
		GoldPerMobXP:       0.75,
		InflationThreshold: 10_000_000,
		InflationRate:      5_000_000,

		// Pity
		PityPerLoss: 0.50,
		PityCap:     3.0,

		// Zone
		ZoneDiffMin: 0.8,
		ZoneDiffMax: 1.5,

		// XP
		XPMin:          30,
		XPMax:          65,
		ExponentCap:    1.65, // Match real bot's XP curve exponent
		PrestigeLevel:  5000,
		DeathXPPenalty: 0.05,

		// Intensify / Fatigue
		IntensifyPerRound: 0.15,
		FatigueStartRound: 11,
		FatiguePerRound:   0.10,
		FatigueFloor:      0.1,

		// Mob effects
		MobEffectChance:    0.30,
		MobSpellChance:     0.20,
		MobSpellCount:      1,
		MobEquipItemCount:  1,
		MobEquipItemChance: 0.30,

		// Death effects
		DeathEffectChance:       0.10,
		DeathEffectSummonWeight: 0.40,
		DeathEffectExploWeight:  0.20,
		DeathEffectCurseWeight:  0.20,
		DeathEffectXPWeight:     0.10,
		DeathEffectLootWeight:   0.10,

		// Elemental
		ElementalAdvantage: 2.0,
		ElementalDisadv:    0.5,
		MobElementChance:   0.40,

		// Misc
		MomentumChance:              0.10,
		LifestealHealPenaltyAfter10: 0.8,
	}
}

// Clone returns a deep copy of the params.
func (p SimParams) Clone() SimParams {
	return p // value type, all fields are value types
}
