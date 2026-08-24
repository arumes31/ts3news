package content

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
)

const (
	TreeCatalogSchemaVersion = 2

	TreeNodeSmall     = "small"
	TreeNodeNotable   = "notable"
	TreeNodeKeystone  = "keystone"
	TreeNodeBridge    = "bridge"
	TreeNodeAura      = "aura"
	TreeNodeSocket    = "socket"
	TreeNodeOffshoot  = "offshoot"
)

// TreeEffect describes one percentage modifier understood by the server. The
// catalog is also sent to the browser, keeping labels and explanations aligned
// with combat rather than duplicating game rules in JavaScript.
type TreeEffect struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Category    string `json:"category"`
	IsPenalty   bool   `json:"is_penalty,omitempty"`
}

var treeEffects = map[string]TreeEffect{
	"alternating_element_power": {Key: "alternating_element_power", Label: "Alternating elements", Description: "Amplifies a skill when its element differs from the previous non-physical cast.", Category: "skill"},
	"buff_duration":              {Key: "buff_duration", Label: "Buff duration", Description: "Extends beneficial skill effects.", Category: "skill"},
	"companion_skill_power":      {Key: "companion_skill_power", Label: "Companion command power", Description: "Improves companion attacks and active companion commands.", Category: "companion"},
	"consumable_save":     {Key: "consumable_save", Label: "Consumable conservation", Description: "Chance that an Abyss consumable keeps its charge.", Category: "utility"},
	"crt_pct":             {Key: "crt_pct", Label: "Critical rating", Description: "Multiplies critical rating during Abyss combat.", Category: "offense"},
	"def_pct":             {Key: "def_pct", Label: "Defense", Description: "Multiplies defense during Abyss combat.", Category: "defense"},
	"def_to_lifesteal":    {Key: "def_to_lifesteal", Label: "Defense conversion", Description: "Converts a share of defense into life steal.", Category: "conversion"},
	"debuff_duration":      {Key: "debuff_duration", Label: "Debuff duration", Description: "Extends hostile control and debuff effects.", Category: "skill"},
	"defense_penetration":  {Key: "defense_penetration", Label: "Defense penetration", Description: "Adds defense penetration to damaging skills.", Category: "skill"},
	"dge_pct":             {Key: "dge_pct", Label: "Dodge rating", Description: "Multiplies dodge rating during Abyss combat.", Category: "defense"},
	"escrow_bonus":        {Key: "escrow_bonus", Label: "Escrow reward", Description: "Multiplies the cache added after a victorious floor.", Category: "economy"},
	"gold_find":           {Key: "gold_find", Label: "Gold find", Description: "Increases gold found in Abyss encounters.", Category: "economy"},
	"hp_pct":              {Key: "hp_pct", Label: "Maximum health", Description: "Multiplies maximum health during Abyss combat.", Category: "defense"},
	"hp_regen":            {Key: "hp_regen", Label: "Health regeneration", Description: "Restores health over real time in an active Abyss run.", Category: "defense"},
	"hp_to_def":           {Key: "hp_to_def", Label: "Health conversion", Description: "Converts a share of maximum health into defense.", Category: "conversion"},
	"int_pct":             {Key: "int_pct", Label: "Intellect", Description: "Multiplies intellect during Abyss combat.", Category: "offense"},
	"int_to_mna":          {Key: "int_to_mna", Label: "Intellect conversion", Description: "Converts a share of intellect into mana.", Category: "conversion"},
	"item_skill_power":     {Key: "item_skill_power", Label: "Combat item power", Description: "Amplifies healing and stat gains from live combat items.", Category: "skill"},
	"lck_pct":             {Key: "lck_pct", Label: "Luck", Description: "Multiplies luck during Abyss combat.", Category: "utility"},
	"limit_break":         {Key: "limit_break", Label: "Limit Break", Description: "Multiplies the five primary combat attributes.", Category: "keystone"},
	"loot_find":           {Key: "loot_find", Label: "Loot find", Description: "Improves Abyss loot rarity rolls.", Category: "economy"},
	"material_yield":      {Key: "material_yield", Label: "Material yield", Description: "Increases crafting materials earned in the Abyss.", Category: "economy"},
	"magic_skill_power":    {Key: "magic_skill_power", Label: "Magic skill power", Description: "Multiplies damage dealt by magic skills.", Category: "skill"},
	"low_health_skill_power": {Key: "low_health_skill_power", Label: "Low-health skill power", Description: "Amplifies skills while the caster is below 35% health.", Category: "skill"},
	"healing_skill_power":  {Key: "healing_skill_power", Label: "Healing skill power", Description: "Multiplies healing performed by skills.", Category: "skill"},
	"elemental_skill_power": {Key: "elemental_skill_power", Label: "Elemental skill power", Description: "Multiplies non-physical skill damage.", Category: "skill"},
	"opener_skill_power":   {Key: "opener_skill_power", Label: "Opening skill power", Description: "Amplifies skills on round one while the caster is at full health.", Category: "skill"},
	"pet_betrayal_reduce": {Key: "pet_betrayal_reduce", Label: "Companion loyalty", Description: "Reduces the chance of hostile companion behavior.", Category: "companion"},
	"pet_damage_pct":      {Key: "pet_damage_pct", Label: "Companion damage", Description: "Multiplies companion damage.", Category: "companion"},
	"physical_skill_power": {Key: "physical_skill_power", Label: "Physical skill power", Description: "Multiplies damage dealt by physical skills.", Category: "skill"},
	"relic_skill_power":    {Key: "relic_skill_power", Label: "Active relic power", Description: "Amplifies the healing and guard supplied by active relics.", Category: "skill"},
	"repeated_skill_retention": {Key: "repeated_skill_retention", Label: "Repeated-skill retention", Description: "Reduces the diminishing return from repeatedly casting the same skill.", Category: "skill"},
	"skill_damage":        {Key: "skill_damage", Label: "Skill damage", Description: "Multiplies damage dealt by non-ultimate skills.", Category: "skill"},
	"skill_cooldown_recovery": {Key: "skill_cooldown_recovery", Label: "Skill recovery", Description: "Reduces regular skill cooldown duration.", Category: "skill"},
	"skill_mana_cost":     {Key: "skill_mana_cost", Label: "Skill mana efficiency", Description: "Reduces mana spent when casting skills.", Category: "skill"},
	"stun_effectiveness":  {Key: "stun_effectiveness", Label: "Stun effectiveness", Description: "Multiplies the chance for skills to stun.", Category: "skill"},
	"support_party_power": {Key: "support_party_power", Label: "Party support power", Description: "Amplifies healing and support skills in parties.", Category: "skill"},
	"spd_pct":             {Key: "spd_pct", Label: "Speed", Description: "Multiplies speed during Abyss combat.", Category: "offense"},
	"spd_to_dge":          {Key: "spd_to_dge", Label: "Speed conversion", Description: "Converts a share of speed into dodge rating.", Category: "conversion"},
	"str_pct":             {Key: "str_pct", Label: "Strength", Description: "Multiplies strength during Abyss combat.", Category: "offense"},
	"str_to_spd":          {Key: "str_to_spd", Label: "Strength conversion", Description: "Converts a share of strength into speed.", Category: "conversion"},
	"stun_immunity":       {Key: "stun_immunity", Label: "Stun immunity", Description: "Prevents combat turns from being lost to stuns.", Category: "defense"},
	"token_gain":          {Key: "token_gain", Label: "Token gain", Description: "Increases Abyss tokens received while banking.", Category: "economy"},
	"ult_cooldown":        {Key: "ult_cooldown", Label: "Ultimate recovery", Description: "Reduces ultimate cooldown duration.", Category: "skill"},
	"ult_damage":          {Key: "ult_damage", Label: "Ultimate damage", Description: "Multiplies ultimate damage.", Category: "skill"},
	"ultimate_charge":     {Key: "ultimate_charge", Label: "Ultimate charge", Description: "Accelerates ultimate cooldown recovery after use.", Category: "skill"},
	"xp_gain":             {Key: "xp_gain", Label: "Experience gain", Description: "Increases experience earned from Abyss floors.", Category: "progression"},
	"xp_to_gold":          {Key: "xp_to_gold", Label: "Experience conversion", Description: "Converts part of floor experience into gold.", Category: "conversion"},
}

var treeNodeKinds = map[string]bool{
	TreeNodeSmall: true, TreeNodeNotable: true, TreeNodeKeystone: true,
	TreeNodeBridge: true, TreeNodeAura: true, TreeNodeSocket: true,
	TreeNodeOffshoot: true,
}

func TreeEffects() []TreeEffect {
	keys := make([]string, 0, len(treeEffects))
	for key := range treeEffects {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]TreeEffect, 0, len(keys))
	for _, key := range keys {
		out = append(out, treeEffects[key])
	}
	return out
}

func TreeEffectByKey(key string) (TreeEffect, bool) {
	effect, ok := treeEffects[key]
	return effect, ok
}

type TreeCatalogSummary struct {
	SchemaVersion int            `json:"schema_version"`
	LayoutHash    string         `json:"layout_hash"`
	BalanceHash   string         `json:"balance_hash"`
	Nodes         int            `json:"nodes"`
	Edges         int            `json:"edges"`
	Portals       int            `json:"portals"`
	ByKind        map[string]int `json:"by_kind"`
	BySector      map[int]int    `json:"by_sector"`
}

func (t *AbyssTreeData) CatalogSummary() TreeCatalogSummary {
	summary := TreeCatalogSummary{
		SchemaVersion: TreeCatalogSchemaVersion,
		LayoutHash:    t.TopologyHash(),
		BalanceHash:   t.LayoutHash(),
		Nodes:         len(t.Nodes),
		Portals:       len(t.Portals),
		ByKind:        map[string]int{},
		BySector:      map[int]int{},
	}
	for _, node := range t.Nodes {
		summary.ByKind[node.Type]++
		summary.BySector[node.Sector]++
	}
	for id, neighbors := range t.Adj {
		for _, neighbor := range neighbors {
			if id < neighbor {
				summary.Edges++
			}
		}
	}
	return summary
}

// TopologyHash is the allocation-compatibility fingerprint. Unlike LayoutHash,
// it excludes effects and costs, so balance or copy changes never wipe a valid
// player's path through an unchanged graph.
func (t *AbyssTreeData) TopologyHash() string {
	h := fnv.New64a()
	for i := range t.Nodes {
		node := &t.Nodes[i]
		_, _ = fmt.Fprintf(h, "n:%d:%s:%d:%d:%d;", node.ID, node.Type, node.Ring, node.Slot, node.Sector)
	}
	ids := make([]int, 0, len(t.Adj))
	for id := range t.Adj {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		neighbors := append([]int{}, t.Adj[id]...)
		sort.Ints(neighbors)
		for _, neighbor := range neighbors {
			if id < neighbor {
				_, _ = fmt.Fprintf(h, "e:%d-%d;", id, neighbor)
			}
		}
	}
	return fmt.Sprintf("%016x", h.Sum64())
}

func ValidateAbyssTree(tree *AbyssTreeData) error {
	if tree == nil {
		return errors.New("abyss tree is nil")
	}
	errs := []error{}
	seen := make(map[int]bool, len(tree.Nodes))
	for i := range tree.Nodes {
		node := &tree.Nodes[i]
		switch {
		case node.ID <= 0:
			errs = append(errs, fmt.Errorf("node %d has invalid id", node.ID))
		case seen[node.ID]:
			errs = append(errs, fmt.Errorf("node %d is duplicated", node.ID))
		default:
			seen[node.ID] = true
		}
		if !treeNodeKinds[node.Type] {
			errs = append(errs, fmt.Errorf("node %d has unknown kind %q", node.ID, node.Type))
		}
		if node.Name == "" || node.Desc == "" {
			errs = append(errs, fmt.Errorf("node %d has empty name or description", node.ID))
		}
		if node.Cost() <= 0 || node.Cost() > 100 {
			errs = append(errs, fmt.Errorf("node %d has invalid cost %d", node.ID, node.Cost()))
		}
		if node.Sector < 0 || node.Sector >= treeSectors || node.Ring < 0 || node.Slot < 0 || node.Slot >= treeSlots {
			errs = append(errs, fmt.Errorf("node %d has invalid position metadata", node.ID))
		}
		if math.IsNaN(node.X) || math.IsInf(node.X, 0) || math.IsNaN(node.Y) || math.IsInf(node.Y, 0) {
			errs = append(errs, fmt.Errorf("node %d has non-finite coordinates", node.ID))
		}
		for key, value := range node.Pct {
			if _, ok := treeEffects[key]; !ok {
				errs = append(errs, fmt.Errorf("node %d has unknown effect %q", node.ID, key))
			}
			if math.IsNaN(value) || math.IsInf(value, 0) {
				errs = append(errs, fmt.Errorf("node %d effect %q is non-finite", node.ID, key))
			}
		}
	}

	for id, neighbors := range tree.Adj {
		if id != 0 && !seen[id] {
			errs = append(errs, fmt.Errorf("adjacency references unknown node %d", id))
		}
		neighborSeen := map[int]bool{}
		for _, neighbor := range neighbors {
			if neighbor == id {
				errs = append(errs, fmt.Errorf("node %d has a self edge", id))
			}
			if neighbor != 0 && !seen[neighbor] {
				errs = append(errs, fmt.Errorf("node %d references unknown neighbor %d", id, neighbor))
			}
			if neighborSeen[neighbor] {
				errs = append(errs, fmt.Errorf("node %d repeats edge to %d", id, neighbor))
			}
			neighborSeen[neighbor] = true
			if !containsTreeNeighbor(tree.Adj[neighbor], id) {
				errs = append(errs, fmt.Errorf("edge %d-%d is not bidirectional", id, neighbor))
			}
		}
	}

	reached := map[int]bool{0: true}
	queue := []int{0}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, neighbor := range tree.Adj[current] {
			if !reached[neighbor] {
				reached[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}
	for id := range seen {
		if !reached[id] {
			errs = append(errs, fmt.Errorf("node %d is unreachable from root", id))
		}
	}
	for _, portal := range tree.Portals {
		if portal[0] == portal[1] || !containsTreeNeighbor(tree.Adj[portal[0]], portal[1]) {
			errs = append(errs, fmt.Errorf("portal %d-%d is not a valid edge", portal[0], portal[1]))
		}
	}
	return errors.Join(errs...)
}

func containsTreeNeighbor(neighbors []int, expected int) bool {
	for _, neighbor := range neighbors {
		if neighbor == expected {
			return true
		}
	}
	return false
}

func normalizeTreeAdjacency(adjacency map[int][]int) {
	for id, neighbors := range adjacency {
		sort.Ints(neighbors)
		write := 0
		for _, neighbor := range neighbors {
			if write > 0 && neighbors[write-1] == neighbor {
				continue
			}
			neighbors[write] = neighbor
			write++
		}
		adjacency[id] = neighbors[:write]
	}
}
