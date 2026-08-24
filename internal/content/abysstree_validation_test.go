package content

import (
	"math"
	"strings"
	"testing"
)

func TestValidateAbyssTreeCatalog(t *testing.T) {
	if err := ValidateAbyssTree(AbyssTree()); err != nil {
		t.Fatalf("production catalog is invalid: %v", err)
	}
}

func TestValidateAbyssTreeRejectsInvalidCatalogs(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(*AbyssTreeData)
		errorMatch string
	}{
		{
			name: "duplicate id",
			mutate: func(tree *AbyssTreeData) {
				tree.Nodes[1].ID = tree.Nodes[0].ID
			},
			errorMatch: "duplicated",
		},
		{
			name: "unknown effect",
			mutate: func(tree *AbyssTreeData) {
				tree.Nodes[0].Pct = map[string]float64{"not_a_real_effect": 0.1}
			},
			errorMatch: "unknown effect",
		},
		{
			name: "non finite effect",
			mutate: func(tree *AbyssTreeData) {
				tree.Nodes[0].Pct = map[string]float64{"str_pct": math.Inf(1)}
			},
			errorMatch: "non-finite",
		},
		{
			name: "self edge",
			mutate: func(tree *AbyssTreeData) {
				id := tree.Nodes[0].ID
				tree.Adj[id] = append(tree.Adj[id], id)
			},
			errorMatch: "self edge",
		},
		{
			name: "one way edge",
			mutate: func(tree *AbyssTreeData) {
				id := tree.Nodes[0].ID
				neighbor := tree.Adj[id][0]
				tree.Adj[neighbor] = removeTreeTestNeighbor(tree.Adj[neighbor], id)
			},
			errorMatch: "not bidirectional",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tree := cloneTreeForTest(AbyssTree())
			tt.mutate(tree)
			err := ValidateAbyssTree(tree)
			if err == nil || !strings.Contains(err.Error(), tt.errorMatch) {
				t.Fatalf("ValidateAbyssTree() error = %v, want containing %q", err, tt.errorMatch)
			}
		})
	}
}

func TestAbyssTreeTopologyHashIgnoresMetadataAndBalance(t *testing.T) {
	tree := cloneTreeForTest(AbyssTree())
	want := tree.TopologyHash()
	tree.Nodes[0].Name = "Renamed without moving"
	tree.Nodes[0].Desc = "New copy"
	tree.Nodes[0].Stats.STR++
	tree.Nodes[0].Pct = map[string]float64{"str_pct": 0.99}
	if got := tree.TopologyHash(); got != want {
		t.Fatalf("TopologyHash changed for metadata/balance edit: %s -> %s", want, got)
	}
	if tree.LayoutHash() == AbyssTree().LayoutHash() {
		t.Fatal("balance hash did not change for a balance edit")
	}
}

func TestAbyssTreeCatalogSummary(t *testing.T) {
	tree := AbyssTree()
	summary := tree.CatalogSummary()
	if summary.SchemaVersion != TreeCatalogSchemaVersion || summary.Nodes != len(tree.Nodes) {
		t.Fatalf("summary = %+v", summary)
	}
	if summary.Edges <= 0 || summary.ByKind[TreeNodeKeystone] < 40 || summary.LayoutHash == "" || summary.BalanceHash == "" {
		t.Fatalf("incomplete summary = %+v", summary)
	}
}

func TestAbyssTreeIncludesSkillSpecialisationEffects(t *testing.T) {
	wanted := map[string]bool{
		"physical_skill_power": false, "magic_skill_power": false,
		"healing_skill_power": false, "buff_duration": false,
		"debuff_duration": false, "skill_cooldown_recovery": false,
		"stun_effectiveness": false, "defense_penetration": false,
		"elemental_skill_power": false, "low_health_skill_power": false,
		"opener_skill_power": false, "alternating_element_power": false,
		"repeated_skill_retention": false, "support_party_power": false,
		"ultimate_charge": false, "item_skill_power": false,
		"companion_skill_power": false, "relic_skill_power": false,
	}
	for _, node := range AbyssTree().Nodes {
		for key := range node.Pct {
			if _, ok := wanted[key]; ok {
				wanted[key] = true
			}
		}
	}
	for key, found := range wanted {
		if !found {
			t.Errorf("skill specialisation effect %q has no node", key)
		}
		if _, ok := TreeEffectByKey(key); !ok {
			t.Errorf("skill specialisation effect %q is not registered", key)
		}
	}
}

func TestTreeEffectsAreStableAndDocumented(t *testing.T) {
	effects := TreeEffects()
	if len(effects) != len(treeEffects) {
		t.Fatalf("TreeEffects() returned %d effects, want %d", len(effects), len(treeEffects))
	}
	for i, effect := range effects {
		if effect.Key == "" || effect.Label == "" || effect.Description == "" || effect.Category == "" {
			t.Errorf("effect %d is incomplete: %+v", i, effect)
		}
		if i > 0 && effects[i-1].Key >= effect.Key {
			t.Errorf("effects are not deterministically ordered at %q", effect.Key)
		}
	}
}

func cloneTreeForTest(source *AbyssTreeData) *AbyssTreeData {
	tree := &AbyssTreeData{
		Nodes:   append([]TreeNode{}, source.Nodes...),
		Adj:     make(map[int][]int, len(source.Adj)),
		Portals: append([][2]int{}, source.Portals...),
		byID:    map[int]*TreeNode{},
	}
	for id, neighbors := range source.Adj {
		tree.Adj[id] = append([]int{}, neighbors...)
	}
	for i := range tree.Nodes {
		tree.byID[tree.Nodes[i].ID] = &tree.Nodes[i]
	}
	return tree
}

func removeTreeTestNeighbor(neighbors []int, target int) []int {
	out := make([]int, 0, len(neighbors))
	for _, neighbor := range neighbors {
		if neighbor != target {
			out = append(out, neighbor)
		}
	}
	return out
}
