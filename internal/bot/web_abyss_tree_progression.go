package bot

import (
	"sort"

	"ts3news/internal/content"
)

type abyssTreePointSource struct {
	Key        string `json:"key"`
	Label      string `json:"label"`
	Earned     int    `json:"earned"`
	Progress   int    `json:"progress"`
	Target     int    `json:"target"`
	NextReward int    `json:"next_reward"`
	NextLabel  string `json:"next_label"`
}

type abyssTreeSectorMastery struct {
	Sector        int    `json:"sector"`
	Name          string `json:"name"`
	Allocated     int    `json:"allocated"`
	Spent         int    `json:"spent"`
	Level         int    `json:"level"`
	NextMilestone int    `json:"next_milestone"`
	Cosmetic      string `json:"cosmetic"`
}

type abyssTreeArchetypeScore struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Score       int    `json:"score"`
	Share       int    `json:"share"`
	Detected    bool   `json:"detected"`
	Description string `json:"description"`
}

type abyssTreeAchievement struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Progress int    `json:"progress"`
	Target   int    `json:"target"`
}

type abyssTreeProgression struct {
	PointSources []abyssTreePointSource    `json:"point_sources"`
	BasePointCap int                       `json:"base_point_cap"`
	GrossBase    int                       `json:"gross_base"`
	TotalPoints  int                       `json:"total_points"`
	Sectors      []abyssTreeSectorMastery  `json:"sectors"`
	Archetypes   []abyssTreeArchetypeScore `json:"archetypes"`
	Dominant     string                    `json:"dominant"`
	Achievements []abyssTreeAchievement    `json:"achievements"`
}

var abyssTreeSectorNames = [...]string{"War", "Vitality", "Shadow", "Arcane", "Fortune", "Void"}

func (b *Bot) abyssTreeProgressionFor(uid string, allocated []int) abyssTreeProgression {
	var level, bestDepth, prestige int
	var lifetimeFloors int64
	_ = b.DB.QueryRow(
		"SELECT level, abyss_best_depth, abyss_prestige, abyss_lifetime_floors FROM users WHERE client_uid=$1",
		uid,
	).Scan(&level, &bestDepth, &prestige, &lifetimeFloors)
	deepPoints := b.deepDelverPointBonus(uid)
	progression := buildAbyssTreeProgression(
		content.AbyssTree(), allocated, level, bestDepth, prestige, lifetimeFloors, deepPoints,
	)
	return progression
}

func buildAbyssTreeProgression(
	tree *content.AbyssTreeData,
	allocated []int,
	level int,
	bestDepth int,
	prestige int,
	lifetimeFloors int64,
	deepPoints int,
) abyssTreeProgression {
	grossBase := level*2 + bestDepth*3 + int(lifetimeFloors/4) + prestige*60
	progression := abyssTreeProgression{
		BasePointCap: 1000,
		GrossBase:    grossBase,
		TotalPoints:  min(grossBase, 1000) + deepPoints,
		PointSources: []abyssTreePointSource{
			{Key: "level", Label: "Character levels", Earned: level * 2, Progress: level, Target: level + 1, NextReward: 2, NextLabel: "Reach the next character level"},
			{Key: "depth", Label: "Best Abyss depth", Earned: bestDepth * 3, Progress: bestDepth, Target: bestDepth + 1, NextReward: 3, NextLabel: "Set a new best depth"},
			{Key: "floors", Label: "Lifetime floors", Earned: int(lifetimeFloors / 4), Progress: int(lifetimeFloors % 4), Target: 4, NextReward: 1, NextLabel: "Clear more lifetime floors"},
			{Key: "prestige", Label: "Abyss prestige", Earned: prestige * 60, Progress: prestige, Target: prestige + 1, NextReward: 60, NextLabel: "Complete the next Abyss prestige"},
			{Key: "deep_delver", Label: "Deep Delver mastery", Earned: deepPoints, Progress: deepPoints, Target: 500, NextReward: 1, NextLabel: "Upgrade another Deep Delver talent"},
		},
	}
	progression.Sectors = calculateAbyssSectorMastery(tree, allocated)
	progression.Archetypes, progression.Dominant = calculateAbyssArchetypes(tree, allocated)
	progression.Achievements = calculateAbyssTreeAchievements(tree, allocated, progression.Sectors)
	return progression
}

func calculateAbyssSectorMastery(tree *content.AbyssTreeData, allocated []int) []abyssTreeSectorMastery {
	sectors := make([]abyssTreeSectorMastery, len(abyssTreeSectorNames))
	for index, name := range abyssTreeSectorNames {
		sectors[index] = abyssTreeSectorMastery{Sector: index, Name: name}
	}
	for _, id := range allocated {
		node := tree.Node(id)
		if node == nil || node.Sector < 0 || node.Sector >= len(sectors) {
			continue
		}
		sectors[node.Sector].Allocated++
		sectors[node.Sector].Spent += node.Cost()
	}
	for index := range sectors {
		sector := &sectors[index]
		sector.Level = sector.Allocated / 10
		sector.NextMilestone = nextTreeMasteryMilestone(sector.Allocated)
		sector.Cosmetic = treeMasteryCosmetic(sector.Allocated)
	}
	return sectors
}

func nextTreeMasteryMilestone(allocated int) int {
	for _, milestone := range []int{10, 25, 50, 100} {
		if allocated < milestone {
			return milestone
		}
	}
	return 100
}

func treeMasteryCosmetic(allocated int) string {
	switch {
	case allocated >= 100:
		return "Animated discipline crown"
	case allocated >= 50:
		return "Discipline aura"
	case allocated >= 25:
		return "Discipline sigil"
	case allocated >= 10:
		return "Discipline trail"
	default:
		return "Trail unlocks at 10 nodes"
	}
}

func calculateAbyssArchetypes(tree *content.AbyssTreeData, allocated []int) ([]abyssTreeArchetypeScore, string) {
	scores := map[string]float64{"glass_cannon": 0, "sustain": 0, "control": 0, "support": 0}
	for _, id := range allocated {
		node := tree.Node(id)
		if node == nil {
			continue
		}
		scores["glass_cannon"] += float64(max(0, node.Stats.STR))/10 + float64(max(0, node.Stats.CRT))/5
		scores["sustain"] += float64(max(0, node.Stats.HP))/100 + float64(max(0, node.Stats.DEF))/10 + float64(max(0, node.Stats.STA))/10
		scores["control"] += float64(max(0, node.Stats.SPD))/10 + float64(max(0, node.Stats.INT))/20
		scores["support"] += float64(max(0, node.Stats.INT))/15 + float64(max(0, node.Stats.MNA))/100
		for key, value := range node.Pct {
			amount := value * 100
			scoreAbyssArchetypeEffect(scores, key, amount)
			if amount < 0 && (key == "hp_pct" || key == "def_pct" || key == "dge_pct") {
				scores["glass_cannon"] += -amount
			}
		}
	}
	definitions := []abyssTreeArchetypeScore{
		{Key: "glass_cannon", Label: "Glass cannon", Description: "Burst damage, critical pressure, and accepted defensive trade-offs."},
		{Key: "sustain", Label: "Sustain", Description: "Health, defense, regeneration, healing, and life steal."},
		{Key: "control", Label: "Control", Description: "Stuns, debuffs, cooldowns, speed, and defense penetration."},
		{Key: "support", Label: "Support", Description: "Healing, buffs, party amplification, companions, items, and relics."},
	}
	total := 0
	for _, value := range scores {
		total += int(value)
	}
	dominant := definitions[0].Key
	for index := range definitions {
		entry := &definitions[index]
		entry.Score = int(scores[entry.Key])
		if total > 0 {
			entry.Share = int(float64(entry.Score) / float64(total) * 100)
		}
		entry.Detected = entry.Score >= 10 && entry.Share >= 20
		if scores[entry.Key] > scores[dominant] {
			dominant = entry.Key
		}
	}
	return definitions, dominant
}

func scoreAbyssArchetypeEffect(scores map[string]float64, key string, amount float64) {
	groups := map[string][]string{
		"glass_cannon": {"str_pct", "crt_pct", "skill_damage", "physical_skill_power", "magic_skill_power", "elemental_skill_power", "ult_damage", "opener_skill_power", "low_health_skill_power"},
		"sustain":      {"hp_pct", "def_pct", "hp_regen", "def_to_lifesteal", "healing_skill_power", "stun_immunity"},
		"control":      {"spd_pct", "dge_pct", "stun_effectiveness", "debuff_duration", "defense_penetration", "skill_cooldown_recovery", "ult_cooldown", "ultimate_charge"},
		"support":      {"healing_skill_power", "buff_duration", "support_party_power", "item_skill_power", "companion_skill_power", "relic_skill_power", "pet_damage_pct"},
	}
	for archetype, keys := range groups {
		for _, candidate := range keys {
			if key == candidate && amount > 0 {
				scores[archetype] += amount
			}
		}
	}
}

func calculateAbyssTreeAchievements(
	tree *content.AbyssTreeData,
	allocated []int,
	sectors []abyssTreeSectorMastery,
) []abyssTreeAchievement {
	keystones := 0
	effects := map[string]bool{}
	for _, id := range allocated {
		if node := tree.Node(id); node != nil {
			if node.Type == content.TreeNodeKeystone {
				keystones++
			}
			for key := range node.Pct {
				effects[key] = true
			}
		}
	}
	touchedSectors := 0
	for _, sector := range sectors {
		if sector.Allocated > 0 {
			touchedSectors++
		}
	}
	return []abyssTreeAchievement{
		{Key: "first_path", Label: "First Path", Progress: min(len(allocated), 1), Target: 1},
		{Key: "web_walker", Label: "Web Walker", Progress: min(len(allocated), 100), Target: 100},
		{Key: "six_disciplines", Label: "Six Disciplines", Progress: touchedSectors, Target: len(sectors)},
		{Key: "keystone_scholar", Label: "Keystone Scholar", Progress: min(keystones, 10), Target: 10},
		{Key: "effect_lexicon", Label: "Effect Lexicon", Progress: len(effects), Target: len(content.TreeEffects())},
	}
}

func sortedAbyssArchetypeKeys(scores []abyssTreeArchetypeScore) []string {
	keys := make([]string, 0, len(scores))
	for _, score := range scores {
		keys = append(keys, score.Key)
	}
	sort.Strings(keys)
	return keys
}
