package bot

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"
)

type abyssForgeRecipeGuide struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Category      string         `json:"category"`
	Description   string         `json:"description"`
	Materials     map[string]int `json:"materials"`
	Sources       []string       `json:"material_sources"`
	Substitutions []string       `json:"substitutions"`
	Starter       bool           `json:"starter"`
	Event         bool           `json:"event"`
	SlotTarget    string         `json:"slot_target,omitempty"`
	RarityTarget  string         `json:"rarity_target,omitempty"`
	SetTarget     string         `json:"set_target,omitempty"`
	DiscoveryHint string         `json:"discovery_hint"`
	Discovered    bool           `json:"discovered"`
	CraftCount    int64          `json:"craft_count"`
}

type abyssForgeOperationGuide struct {
	Operation       string             `json:"operation"`
	Summary         string             `json:"summary"`
	OutcomeFamilies []string           `json:"outcome_families"`
	Safeguards      []string           `json:"safeguards"`
	HardCaps        map[string]float64 `json:"hard_caps"`
	SoftCaps        map[string]float64 `json:"soft_caps"`
	Filters         []string           `json:"filters"`
	Glossary        map[string]string  `json:"glossary"`
	Stages          []string           `json:"stages"`
	RiskTiers       []string           `json:"risk_tiers"`
	SupportsBatch   bool               `json:"supports_batch"`
	SupportsUndo    bool               `json:"supports_undo"`
	SupportsPreset  bool               `json:"supports_preset"`
	Policies        []string           `json:"failure_policies"`
}

type abyssForgeMasteryTrack struct {
	Discipline string   `json:"discipline"`
	XP         int64    `json:"xp"`
	Level      int      `json:"level"`
	NextXP     int64    `json:"next_xp"`
	Cosmetics  []string `json:"cosmetics"`
	Unlocks    []string `json:"convenience_unlocks"`
}

type abyssForgeWorkbenchData struct {
	SchemaVersion int                        `json:"schema_version"`
	CatalogHash   string                     `json:"catalog_hash"`
	Recipes       []abyssForgeRecipeGuide    `json:"recipes"`
	Guides        []abyssForgeOperationGuide `json:"operation_guides"`
	StatPresets   map[string]map[string]int  `json:"stat_weight_presets"`
	Mastery       []abyssForgeMasteryTrack   `json:"mastery"`
	Contracts     []abyssForgeObjective      `json:"daily_contracts"`
	Projects      []abyssForgeObjective      `json:"weekly_projects"`
	QueuePolicies []string                   `json:"queue_policies"`
	MaterialFlow  map[string][]int64         `json:"material_flow_7d"`
	GeneratedAt   time.Time                  `json:"generated_at"`
	ActiveEvent   string                     `json:"active_event_recipe"`
	CraftCap      int                        `json:"craft_cap"`
	PresetSlots   int                        `json:"queue_preset_slots"`
	CosmeticTheme string                     `json:"cosmetic_theme"`
	Milestones    map[string]int64           `json:"milestones"`
}

type abyssForgeObjective struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Current   int    `json:"current"`
	Target    int    `json:"target"`
	Completed bool   `json:"completed"`
	Reward    string `json:"reward"`
}

type abyssForgeReceipt struct {
	ID        int64           `json:"id"`
	Operation string          `json:"operation"`
	ItemName  string          `json:"item_name"`
	Result    json.RawMessage `json:"result"`
	CreatedAt time.Time       `json:"created_at"`
}

func abyssForgeRecipeGuides() []abyssForgeRecipeGuide {
	guides := make([]abyssForgeRecipeGuide, 0, len(craftRecipes)+6)
	for index, recipe := range craftRecipes {
		category := "consumables"
		if recipe.Secret {
			category = "discoveries"
		}
		guides = append(guides, abyssForgeRecipeGuide{
			ID: recipe.ID, Name: recipe.Name, Category: category, Description: recipe.Desc,
			Materials: recipe.Cost, Sources: []string{"Abyss dismantling", "floor rewards", "forge contracts"},
			Substitutions: []string{"10 dust → 1 shard", "10 shards → 1 core", "5 cores → 1 prism"},
			Starter:       index < 2, Discovered: !recipe.Secret,
			DiscoveryHint: map[bool]string{true: "Recover lore fragments to reveal this recipe.", false: "Known from the start."}[recipe.Secret],
		})
	}
	targets := []abyssForgeRecipeGuide{
		{ID: "target_weapon", Name: "Targeted Weapon", Category: "gear", SlotTarget: "weapon", RarityTarget: "Rare+", DiscoveryHint: "Unlocked by Smithing mastery 3."},
		{ID: "target_armor", Name: "Targeted Armor", Category: "gear", SlotTarget: "armor", RarityTarget: "Rare+", DiscoveryHint: "Unlocked by Smithing mastery 3."},
		{ID: "target_set", Name: "Set Crucible", Category: "sets", SetTarget: "selected set", RarityTarget: "Legendary+", DiscoveryHint: "Unlocked by the weekly forge project."},
		{ID: "event_solar", Name: "Solar Crucible", Category: "events", Event: true, RarityTarget: "Mythic", DiscoveryHint: "Deterministic during the active solar event."},
	}
	return append(guides, targets...)
}

func abyssForgeOperationGuides() []abyssForgeOperationGuide {
	guides := []abyssForgeOperationGuide{
		{Operation: "target_craft", Summary: "Choose an exact slot, rarity, set, event recipe, and duplicate-protection policy before crafting.", OutcomeFamilies: []string{"slot target", "rarity target", "set target", "event deterministic"}, Safeguards: []string{"avoid duplicate", "upgrade duplicate", "recycle duplicate", "signed cost preview"}},
		{Operation: "temper", Summary: "Raise temper with visible odds, pity, protection, and batch stop rules.", Stages: []string{"+0–5 Stable", "+6–10 Tempered", "+11–15 Volatile", "+16–20 Surge"}, Safeguards: []string{"insurance guard", "stop on failure", "stop at target", "session budget"}, HardCaps: map[string]float64{"temper": 20}, SupportsBatch: true, SupportsUndo: true, SupportsPreset: true, Policies: []string{"stop", "skip", "continue protected"}},
		{Operation: "reforge", Summary: "Shape stats with lock-cost scaling and accept/reject thresholds.", OutcomeFamilies: []string{"offense", "defense", "speed", "utility", "balanced"}, Safeguards: []string{"lock up to two stats", "minimum quality", "compare before accept", "one-step rollback", "recalibration cap"}, SoftCaps: map[string]float64{"CRT": 50, "DGE": 40, "SPD": 250}, SupportsUndo: true, SupportsPreset: true},
		{Operation: "socket_gem", Summary: "Plan gem placement, resonance, relocation, extraction, and bulk upgrades.", OutcomeFamilies: []string{"ruby offense", "sapphire defense", "topaz utility", "prismatic resonance"}, Stages: []string{"I", "II", "III"}, Safeguards: []string{"safe extraction quote", "stop on rarity", "atomic relocation"}, SupportsBatch: true, SupportsUndo: true, SupportsPreset: true},
		{Operation: "etch_rune", Summary: "Filter rune families and preview exact offensive or defensive impact.", Filters: []string{"offensive", "defensive", "utility", "elemental"}, Glossary: map[string]string{"power": "increases direct damage", "ward": "adds resistance", "haste": "improves action speed", "leech": "restores health from damage"}, Safeguards: []string{"conflict explanation", "exact scrape recovery", "atomic replacement"}, SupportsUndo: true, SupportsPreset: true},
		{Operation: "transfer_enchant", Summary: "Preview enchant transfer, conflicts, binding, and donor loss.", OutcomeFamilies: []string{"offense", "defense", "resource", "utility"}, Safeguards: []string{"donor confirmation", "conflict check", "before/after comparison"}, SupportsUndo: true},
		{Operation: "awaken", Summary: "Choose a guided awakening family while keeping published pity visible.", OutcomeFamilies: []string{"guardian", "reaper", "arcanist", "trickster"}, Stages: []string{"Dormant", "Awakened", "Guided"}, Safeguards: []string{"visible pity", "family preview", "duplicate protection"}, SupportsUndo: true},
		{Operation: "corrupt", Summary: "Choose a risk tier and inspect the complete outcome distribution.", RiskTiers: []string{"Measured", "Dangerous", "Cataclysmic"}, OutcomeFamilies: []string{"empowered", "warped", "perfect", "fractured"}, Safeguards: []string{"protection consumable", "cleanse recovery quote", "irreversible embrace warning"}, SupportsUndo: true},
		{Operation: "fuse", Summary: "Validate ingredient compatibility, rarity floors, duplicate protection, and boosted contributions.", Stages: []string{"Legendary", "Mythic", "Celestial"}, Safeguards: []string{"rarity floor", "duplicate choice", "compatibility explanation", "milestone tracking"}, SupportsBatch: true},
		{Operation: "masterwork", Summary: "Track quality stages and compare transfer losses before committing.", Stages: []string{"Base", "Fine", "Superior", "Exquisite", "Flawless", "Masterwork"}, Safeguards: []string{"transfer comparison", "loss warning", "undo snapshot"}, HardCaps: map[string]float64{"quality": 5}, SupportsUndo: true},
		{Operation: "special_reroll", Summary: "Select effect families and exclusions before rerolling.", OutcomeFamilies: []string{"damage", "defense", "recovery", "mobility", "loot"}, Safeguards: []string{"family selection", "exclusion list", "compare before accept"}, SupportsUndo: true, SupportsPreset: true},
		{Operation: "brand", Summary: "Progress an item brand, inspect provenance, and preview safe removal.", Stages: []string{"Unbranded", "Marked", "Established", "Exalted"}, Safeguards: []string{"unbranding preview", "set conflict warning", "provenance receipt"}, SupportsUndo: true},
	}
	return guides
}

func abyssForgeMasteryLevel(xp int64) (int, int64) {
	level := int(xp/100) + 1
	if level > 20 {
		level = 20
	}
	return level, int64(level) * 100
}

func abyssForgeMasteryRewards(level int) ([]string, []string) {
	cosmetics, unlocks := []string{}, []string{"favorite recipes"}
	if level >= 3 {
		unlocks = append(unlocks, "queue preset slot")
	}
	if level >= 5 {
		cosmetics = append(cosmetics, "bronze artisan frame")
		unlocks = append(unlocks, "expanded craft cap")
	}
	if level >= 10 {
		cosmetics = append(cosmetics, "silver anvil flourish")
		unlocks = append(unlocks, "extra queue preset slots")
	}
	if level >= 20 {
		cosmetics = append(cosmetics, "master artisan title")
		unlocks = append(unlocks, "maximum craft cap")
	}
	return cosmetics, unlocks
}

func abyssForgeDisciplineForAction(action string) string {
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(action)), " ", "_")
	if operation, ok := abyssForgeOperationByID(normalized); ok {
		return string(operation.Discipline)
	}
	for _, operation := range abyssForgeCatalog {
		if strings.Contains(normalized, operation.ID) || strings.Contains(operation.ID, normalized) {
			return string(operation.Discipline)
		}
	}
	return string(forgeDisciplineMaintenance)
}

func (s *WebServer) loadAbyssForgeMastery(uid string) []abyssForgeMasteryTrack {
	xpByDiscipline := map[string]int64{}
	rows, err := s.bot.DB.Query("SELECT discipline, mastery_xp FROM abyss_forge_progression WHERE client_uid=$1", uid)
	if err == nil {
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var discipline string
			var xp int64
			if rows.Scan(&discipline, &xp) == nil {
				xpByDiscipline[discipline] = xp
			}
		}
	}
	disciplines := []string{"smithing", "enchanting", "gemcraft", "transmutation", "ascension", "maintenance"}
	tracks := make([]abyssForgeMasteryTrack, 0, len(disciplines))
	for _, discipline := range disciplines {
		xp := xpByDiscipline[discipline]
		level, next := abyssForgeMasteryLevel(xp)
		cosmetics, unlocks := abyssForgeMasteryRewards(level)
		tracks = append(tracks, abyssForgeMasteryTrack{
			Discipline: discipline, XP: xp, Level: level, NextXP: next,
			Cosmetics: cosmetics, Unlocks: unlocks,
		})
	}
	return tracks
}

func (s *WebServer) abyssForgeWorkbench(uid string) abyssForgeWorkbenchData {
	summary := currentAbyssForgeCatalogSummary()
	recipes := abyssForgeRecipeGuides()
	known := s.bot.knownRecipes(uid)
	craftCounts := map[string]int64{}
	rows, err := s.bot.DB.Query("SELECT recipe_id, craft_count FROM abyss_forge_recipe_crafts WHERE client_uid=$1", uid)
	if err == nil {
		for rows.Next() {
			var recipeID string
			var count int64
			if rows.Scan(&recipeID, &count) == nil {
				craftCounts[recipeID] = count
			}
		}
		_ = rows.Close()
	}
	for index := range recipes {
		if known[recipes[index].ID] {
			recipes[index].Discovered = true
		}
		recipes[index].CraftCount = craftCounts[recipes[index].ID]
	}
	dailyActions, dailyCrafts, weeklyActions, weeklyMasterworks, weeklyCelestial := 0, 0, 0, 0, 0
	rows, err = s.bot.DB.Query(`SELECT action, created_at FROM forge_history
		WHERE client_uid=$1 AND created_at >= CURRENT_DATE - INTERVAL '6 days'`, uid)
	if err == nil {
		today := time.Now().UTC().Format("2006-01-02")
		for rows.Next() {
			var action string
			var created time.Time
			if rows.Scan(&action, &created) != nil {
				continue
			}
			weeklyActions++
			normalized := strings.ToLower(action)
			if strings.Contains(normalized, "masterwork") {
				weeklyMasterworks++
			}
			if strings.Contains(normalized, "celestial fusion") {
				weeklyCelestial++
			}
			if created.UTC().Format("2006-01-02") == today {
				dailyActions++
				if strings.Contains(normalized, "craft") {
					dailyCrafts++
				}
			}
		}
		_ = rows.Close()
	}
	objective := func(id, label string, current, target int, reward string) abyssForgeObjective {
		return abyssForgeObjective{ID: id, Label: label, Current: min(current, target), Target: target, Completed: current >= target, Reward: reward}
	}
	mastery := s.loadAbyssForgeMastery(uid)
	maxLevel := 1
	for _, track := range mastery {
		maxLevel = max(maxLevel, track.Level)
	}
	craftCap := min(99, 10+maxLevel*3)
	presetSlots := 1 + maxLevel/3
	theme := "apprentice"
	if maxLevel >= 20 {
		theme = "master"
	} else if maxLevel >= 10 {
		theme = "silver"
	} else if maxLevel >= 5 {
		theme = "bronze"
	}
	return abyssForgeWorkbenchData{
		SchemaVersion: summary.SchemaVersion, CatalogHash: summary.CatalogHash,
		Recipes: recipes, Guides: abyssForgeOperationGuides(), Mastery: mastery,
		StatPresets: map[string]map[string]int{
			"damage": {"STR": 100, "CRT": 80, "SPD": 60}, "tank": {"HP": 80, "DEF": 100, "DGE": 60},
			"caster": {"INT": 100, "MNA": 70, "SPD": 60}, "balanced": {"STR": 60, "DEF": 60, "HP": 60, "SPD": 60},
		},
		Contracts: []abyssForgeObjective{
			objective("daily_actions", "Complete three forge operations", dailyActions, 3, "+15 mastery XP"),
			objective("daily_craft", "Craft one discovered recipe", dailyCrafts, 1, "first-craft bonus"),
		},
		Projects: []abyssForgeObjective{
			objective("weekly_actions", "Complete twelve forge operations", weeklyActions, 12, "artisan cosmetic progress"),
			objective("weekly_masterwork", "Complete a Masterwork operation", weeklyMasterworks, 1, "+25 mastery XP"),
			objective("weekly_celestial", "Complete a Celestial fusion", weeklyCelestial, 1, "celestial milestone"),
		},
		QueuePolicies: []string{"stop on failure", "skip failed item", "continue within cap", "pause for manual approval"},
		MaterialFlow:  s.bot.loadAbyssForgeMaterialFlow(uid),
		GeneratedAt:   time.Now().UTC(),
		ActiveEvent:   abyssForgeEventRecipe(time.Now()),
		CraftCap:      craftCap,
		PresetSlots:   presetSlots,
		CosmeticTheme: theme,
		Milestones:    s.loadAbyssForgeMilestones(uid),
	}
}

func (s *WebServer) loadAbyssForgeMilestones(uid string) map[string]int64 {
	milestones := map[string]int64{}
	rows, err := s.bot.DB.Query("SELECT milestone_id, progress FROM abyss_forge_milestones WHERE client_uid=$1", uid)
	if err != nil {
		return milestones
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		var progress int64
		if rows.Scan(&id, &progress) == nil {
			milestones[id] = progress
		}
	}
	return milestones
}

func (b *Bot) loadAbyssForgeMaterialFlow(uid string) map[string][]int64 {
	flow := map[string][]int64{"dust": make([]int64, 7), "shard": make([]int64, 7), "core": make([]int64, 7), "prism": make([]int64, 7)}
	rows, err := b.DB.Query(`SELECT mat_id, direction, amount, created_at FROM abyss_forge_material_flow
		WHERE client_uid=$1 AND created_at >= CURRENT_DATE - INTERVAL '6 days' ORDER BY created_at`, uid)
	if err != nil {
		return flow
	}
	defer func() { _ = rows.Close() }()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	for rows.Next() {
		var material, direction string
		var amount int64
		var created time.Time
		if rows.Scan(&material, &direction, &amount, &created) != nil || flow[material] == nil {
			continue
		}
		day := 6 - int(today.Sub(created.UTC().Truncate(24*time.Hour))/(24*time.Hour))
		if day < 0 || day >= 7 {
			continue
		}
		if direction == "sink" {
			amount = -amount
		}
		flow[material][day] += amount
	}
	return flow
}

func (s *WebServer) handleAbyssForgeWorkbench(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	if !s.abyssFeatures.enabled("forge", uid) {
		writeJSON(w, map[string]any{"ok": false, "error": "forge workbench is temporarily disabled"})
		return
	}
	data := s.abyssForgeWorkbench(uid)
	sort.Slice(data.Recipes, func(i, j int) bool { return data.Recipes[i].Name < data.Recipes[j].Name })
	writeJSON(w, map[string]any{"ok": true, "workbench": data})
}

func (s *WebServer) handleAbyssForgeReceipts(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	rows, err := s.bot.DB.QueryContext(r.Context(), `SELECT id, operation, item_name, result, created_at
		FROM abyss_forge_receipts WHERE client_uid=$1 ORDER BY id DESC LIMIT 50`, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = rows.Close() }()
	receipts := make([]abyssForgeReceipt, 0, 50)
	for rows.Next() {
		var receipt abyssForgeReceipt
		if rows.Scan(&receipt.ID, &receipt.Operation, &receipt.ItemName, &receipt.Result, &receipt.CreatedAt) == nil {
			receipts = append(receipts, receipt)
		}
	}
	writeJSON(w, map[string]any{"ok": true, "receipts": receipts})
}
