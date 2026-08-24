package bot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"ts3news/internal/content"
)

const abyssForgeCatalogSchemaVersion = 1

type abyssForgeDiscipline string

const (
	forgeDisciplineSmithing      abyssForgeDiscipline = "smithing"
	forgeDisciplineEnchanting    abyssForgeDiscipline = "enchanting"
	forgeDisciplineGemcraft      abyssForgeDiscipline = "gemcraft"
	forgeDisciplineTransmutation abyssForgeDiscipline = "transmutation"
	forgeDisciplineAscension     abyssForgeDiscipline = "ascension"
	forgeDisciplineMaintenance   abyssForgeDiscipline = "maintenance"
)

type abyssForgeCostMode string

const (
	forgeCostNone    abyssForgeCostMode = "none"
	forgeCostDynamic abyssForgeCostMode = "authoritative_dynamic"
)

type abyssForgeCostModel struct {
	Mode        abyssForgeCostMode `json:"mode"`
	Gold        bool               `json:"gold"`
	Tokens      bool               `json:"tokens"`
	MaterialIDs []string           `json:"material_ids"`
	Formula     string             `json:"formula"`
}

type abyssForgeSuccessModel struct {
	Kind string  `json:"kind"`
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
}

type abyssForgeOperation struct {
	ID                string                 `json:"id"`
	Label             string                 `json:"label"`
	Discipline        abyssForgeDiscipline   `json:"discipline"`
	RequiresGear      bool                   `json:"requires_gear"`
	CompatibleSlots   []content.GearSlot     `json:"compatible_slots"`
	MinimumRarity     content.Rarity         `json:"minimum_rarity"`
	Cost              abyssForgeCostModel    `json:"cost"`
	Success           abyssForgeSuccessModel `json:"success"`
	Failure           string                 `json:"failure"`
	Reversible        bool                   `json:"reversible"`
	UndoPolicy        string                 `json:"undo_policy"`
	UndoWindowSeconds int64                  `json:"undo_window_seconds"`
	Prerequisites     []string               `json:"prerequisites"`
	Excludes          []string               `json:"excludes"`
	ItemEffects       []content.ItemEffect   `json:"item_effects"`
}

type abyssForgeCatalogSummary struct {
	SchemaVersion  int    `json:"schema_version"`
	OperationCount int    `json:"operation_count"`
	CatalogHash    string `json:"catalog_hash"`
}

var abyssForgeCatalog = buildAbyssForgeCatalog()

func buildAbyssForgeCatalog() []abyssForgeOperation {
	groups := []struct {
		discipline abyssForgeDiscipline
		ids        []string
	}{
		{forgeDisciplineSmithing, []string{
			"temper", "batch_temper", "temper_guard", "temper_surge", "polish", "polish_all",
			"reinforce", "sharpen", "reforge", "reforge_lock", "rebalance", "rebalance_all",
			"masterwork", "masterwork_transfer", "upgrade_gear", "forge_queue",
		}},
		{forgeDisciplineEnchanting, []string{
			"etch_rune", "scrape_rune", "prismatic_rune", "transfer_enchant", "imbue", "imbue_remove",
			"attune", "unattune", "brand", "unbrand", "special_reroll", "swap_special", "recalibrate",
		}},
		{forgeDisciplineGemcraft, []string{
			"socket_gem", "punch_socket", "socket_relocate", "upgrade_gem", "gem_upgrade_all",
			"extract_gem", "transmute_gem",
		}},
		{forgeDisciplineTransmutation, []string{
			"craft", "craft_legendary", "target_craft", "transmute", "convert_mats", "craft_repair_kit2",
			"fuse", "mythic_fuse", "celestial_fuse", "celestial_fuse_boosted", "fuse_preview",
		}},
		{forgeDisciplineAscension, []string{
			"awaken", "awaken_guided", "corrupt", "embrace", "infuse_curse", "infuse_eldritch", "infuse_xp",
		}},
		{forgeDisciplineMaintenance, []string{
			"cleanse", "repair_all", "auto_repair", "identify", "identify_all", "insure_item", "dismantle",
		}},
	}

	reversible := map[string]bool{
		"temper": true, "batch_temper": true, "temper_surge": true, "polish": true, "polish_all": true,
		"reinforce": true, "sharpen": true, "reforge": true, "reforge_lock": true, "rebalance": true,
		"rebalance_all": true, "masterwork": true, "masterwork_transfer": true, "forge_queue": true,
		"scrape_rune": true, "prismatic_rune": true, "imbue": true, "imbue_remove": true,
		"attune": true, "unattune": true, "brand": true, "unbrand": true, "special_reroll": true,
		"swap_special": true, "punch_socket": true, "socket_relocate": true, "upgrade_gem": true,
		"gem_upgrade_all": true, "extract_gem": true, "transmute_gem": true, "awaken": true,
		"awaken_guided": true, "corrupt": true, "embrace": true, "infuse_curse": true,
		"infuse_eldritch": true, "infuse_xp": true, "cleanse": true,
	}
	randomized := map[string]bool{
		"temper": true, "batch_temper": true, "temper_surge": true, "reforge": true, "reforge_lock": true,
		"rebalance": true, "rebalance_all": true, "special_reroll": true, "transmute_gem": true,
		"awaken": true, "awaken_guided": true, "corrupt": true, "embrace": true,
	}
	minimumRarity := map[string]content.Rarity{
		"craft_legendary": content.RarityLegendary, "fuse": content.RarityLegendary,
		"mythic_fuse": content.RarityMythic, "celestial_fuse": content.RarityCelestial,
		"celestial_fuse_boosted": content.RarityCelestial, "awaken": content.RarityLegendary,
		"awaken_guided": content.RarityLegendary, "masterwork": content.RarityLegendary,
		"masterwork_transfer": content.RarityLegendary,
	}
	nonGear := map[string]bool{
		"craft": true, "craft_legendary": true, "target_craft": true, "convert_mats": true, "craft_repair_kit2": true,
		"fuse": true, "mythic_fuse": true, "celestial_fuse": true, "celestial_fuse_boosted": true,
		"fuse_preview": true, "repair_all": true, "auto_repair": true, "identify_all": true,
		"transfer_enchant": true, "dismantle": true,
	}

	var operations []abyssForgeOperation
	for _, group := range groups {
		for _, id := range group.ids {
			requiresGear := !nonGear[id]
			op := abyssForgeOperation{
				ID: id, Label: forgeOperationLabel(id), Discipline: group.discipline,
				RequiresGear: requiresGear, MinimumRarity: minimumRarity[id],
				Cost: abyssForgeCostModel{Mode: forgeCostDynamic, Gold: true, Tokens: true,
					MaterialIDs: []string{"dust", "shard", "core", "prism"},
					Formula:     "computed by the authoritative operation handler from current item and player state"},
				Success: abyssForgeSuccessModel{Kind: "deterministic", Min: 1, Max: 1},
				Failure: "none", Reversible: reversible[id], UndoPolicy: "none",
			}
			if requiresGear {
				op.CompatibleSlots = append([]content.GearSlot(nil), content.AllSlots...)
			}
			if randomized[id] {
				op.Success = abyssForgeSuccessModel{Kind: "bounded_chance", Min: 0.01, Max: 1}
				op.Failure = "operation-specific no-change, downgrade, or item-state consequence"
			}
			if op.Reversible {
				op.UndoPolicy = "latest_snapshot_until_replaced"
			}
			operations = append(operations, op)
		}
	}

	effects := []content.ItemEffect{
		content.EffectThorns, content.EffectVampiric, content.EffectBerserk, content.EffectLucky,
		content.EffectTreasureHunter, content.EffectQuick, content.EffectBulwark, content.EffectRadiant,
		content.EffectFragile, content.EffectSteady, content.EffectMindControl, content.EffectRegenStack,
		content.EffectPhoenix, content.EffectStealth, content.EffectParry, content.EffectCleanse,
		content.EffectExecutioner, content.EffectFocused,
	}
	for i := range operations {
		if operations[i].ID == "imbue" || operations[i].ID == "special_reroll" || operations[i].ID == "swap_special" {
			operations[i].ItemEffects = append([]content.ItemEffect(nil), effects...)
		}
	}
	sort.Slice(operations, func(i, j int) bool { return operations[i].ID < operations[j].ID })
	return operations
}

func forgeOperationLabel(id string) string {
	label := strings.ReplaceAll(id, "_", " ")
	return strings.ToUpper(label[:1]) + label[1:]
}

func abyssForgeOperations() []abyssForgeOperation {
	operations := make([]abyssForgeOperation, len(abyssForgeCatalog))
	for i, op := range abyssForgeCatalog {
		operations[i] = op
		operations[i].CompatibleSlots = append([]content.GearSlot(nil), op.CompatibleSlots...)
		operations[i].Cost.MaterialIDs = append([]string(nil), op.Cost.MaterialIDs...)
		operations[i].Prerequisites = append([]string(nil), op.Prerequisites...)
		operations[i].Excludes = append([]string(nil), op.Excludes...)
		operations[i].ItemEffects = append([]content.ItemEffect(nil), op.ItemEffects...)
	}
	return operations
}

func abyssForgeCatalogHash(operations []abyssForgeOperation) string {
	payload, _ := json.Marshal(struct {
		SchemaVersion int                   `json:"schema_version"`
		Operations    []abyssForgeOperation `json:"operations"`
	}{abyssForgeCatalogSchemaVersion, operations})
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func currentAbyssForgeCatalogSummary() abyssForgeCatalogSummary {
	return abyssForgeCatalogSummary{
		SchemaVersion:  abyssForgeCatalogSchemaVersion,
		OperationCount: len(abyssForgeCatalog),
		CatalogHash:    abyssForgeCatalogHash(abyssForgeCatalog),
	}
}

func validateAbyssForgeCatalog(operations []abyssForgeOperation) error {
	validDisciplines := map[abyssForgeDiscipline]bool{
		forgeDisciplineSmithing: true, forgeDisciplineEnchanting: true, forgeDisciplineGemcraft: true,
		forgeDisciplineTransmutation: true, forgeDisciplineAscension: true, forgeDisciplineMaintenance: true,
	}
	validMaterials := make(map[string]bool, len(abyssMaterials))
	for _, material := range abyssMaterials {
		validMaterials[material.ID] = true
	}
	validSlots := make(map[content.GearSlot]bool, len(content.AllSlots))
	for _, slot := range content.AllSlots {
		validSlots[slot] = true
	}
	validEffects := make(map[content.ItemEffect]bool)
	for _, op := range buildAbyssForgeCatalog() {
		for _, effect := range op.ItemEffects {
			validEffects[effect] = true
		}
	}

	byID := make(map[string]abyssForgeOperation, len(operations))
	for _, op := range operations {
		if op.ID == "" || op.Label == "" {
			return fmt.Errorf("forge operation has empty id or label")
		}
		if _, exists := byID[op.ID]; exists {
			return fmt.Errorf("duplicate forge operation id %q", op.ID)
		}
		byID[op.ID] = op
		if !validDisciplines[op.Discipline] {
			return fmt.Errorf("forge operation %q has invalid discipline %q", op.ID, op.Discipline)
		}
		if op.RequiresGear && len(op.CompatibleSlots) == 0 {
			return fmt.Errorf("forge operation %q requires gear but declares no slots", op.ID)
		}
		seenSlots := make(map[content.GearSlot]bool, len(op.CompatibleSlots))
		for _, slot := range op.CompatibleSlots {
			if !validSlots[slot] || seenSlots[slot] {
				return fmt.Errorf("forge operation %q has invalid or duplicate slot %q", op.ID, slot)
			}
			seenSlots[slot] = true
		}
		if op.MinimumRarity < content.RarityCommon || op.MinimumRarity > content.RarityEternal {
			return fmt.Errorf("forge operation %q has invalid minimum rarity", op.ID)
		}
		if op.Cost.Mode != forgeCostNone && op.Cost.Mode != forgeCostDynamic {
			return fmt.Errorf("forge operation %q has invalid cost mode %q", op.ID, op.Cost.Mode)
		}
		if op.Cost.Mode == forgeCostDynamic && op.Cost.Formula == "" {
			return fmt.Errorf("forge operation %q has no cost formula", op.ID)
		}
		for _, materialID := range op.Cost.MaterialIDs {
			if !validMaterials[materialID] {
				return fmt.Errorf("forge operation %q references unknown material %q", op.ID, materialID)
			}
		}
		if op.Success.Min < 0 || op.Success.Max > 1 || op.Success.Min > op.Success.Max {
			return fmt.Errorf("forge operation %q has invalid success range", op.ID)
		}
		if op.Success.Kind != "deterministic" && op.Success.Kind != "bounded_chance" {
			return fmt.Errorf("forge operation %q has invalid success model %q", op.ID, op.Success.Kind)
		}
		if op.Failure == "" {
			return fmt.Errorf("forge operation %q has no failure consequence", op.ID)
		}
		if op.Reversible && op.UndoPolicy == "none" {
			return fmt.Errorf("reversible forge operation %q has no undo policy", op.ID)
		}
		if !op.Reversible && (op.UndoPolicy != "none" || op.UndoWindowSeconds != 0) {
			return fmt.Errorf("irreversible forge operation %q declares an undo window", op.ID)
		}
		for _, effect := range op.ItemEffects {
			if !validEffects[effect] {
				return fmt.Errorf("forge operation %q references unknown item effect %q", op.ID, effect)
			}
		}
	}

	for _, op := range operations {
		for _, ref := range append(append([]string(nil), op.Prerequisites...), op.Excludes...) {
			if _, ok := byID[ref]; !ok {
				return fmt.Errorf("forge operation %q references unknown operation %q", op.ID, ref)
			}
		}
	}
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("forge prerequisite cycle includes %q", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, prerequisite := range byID[id].Prerequisites {
			if err := visit(prerequisite); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for id := range byID {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}
