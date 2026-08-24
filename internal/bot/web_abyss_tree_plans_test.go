package bot

import (
	"testing"

	"ts3news/internal/content"
)

func TestAnalyzeAbyssTreePlanCanonicalizesAndComputesDelta(t *testing.T) {
	t.Parallel()

	tree := content.AbyssTree()
	first := tree.Adj[0][0]
	second := 0
	for _, neighbor := range tree.Adj[first] {
		if neighbor != 0 {
			second = neighbor
			break
		}
	}
	if second == 0 {
		t.Fatal("production tree has no two-node path from root")
	}

	analysis := analyzeAbyssTreePlan(tree, []int{first}, []int{first, second, second}, 100, 100, 0, -1)
	if !analysis.Valid {
		t.Fatalf("analysis should be valid: %+v", analysis)
	}
	if len(analysis.IDs) != 2 || len(analysis.Added) != 1 || analysis.Added[0] != second || len(analysis.Removed) != 0 {
		t.Fatalf("canonical plan sets are wrong: ids=%v added=%v removed=%v", analysis.IDs, analysis.Added, analysis.Removed)
	}
	wantStats := tree.Node(second).Stats
	if analysis.StatDelta != wantStats {
		t.Fatalf("stat delta = %+v, want %+v", analysis.StatDelta, wantStats)
	}
	for key, value := range tree.Node(second).Pct {
		if analysis.PctDelta[key] != value {
			t.Fatalf("pct delta %q = %v, want %v", key, analysis.PctDelta[key], value)
		}
	}
}

func TestAnalyzeAbyssTreePlanWarnings(t *testing.T) {
	t.Parallel()

	tree := content.AbyssTree()
	far := tree.Nodes[len(tree.Nodes)-1].ID
	analysis := analyzeAbyssTreePlan(tree, nil, []int{far, 999999999}, 0, 0, 0, -1)
	if analysis.Valid {
		t.Fatal("invalid disconnected, missing, over-budget plan reported valid")
	}
	if len(analysis.Missing) != 1 || analysis.Missing[0] != 999999999 {
		t.Fatalf("missing = %v", analysis.Missing)
	}
	if analysis.Connected {
		t.Fatal("far node without its path reported connected")
	}
	if len(analysis.Warnings) < 3 {
		t.Fatalf("warnings = %v, want missing/connectivity/budget warnings", analysis.Warnings)
	}
}

func TestAnalyzeAbyssTreePlanAppliesDiscountAndMemory(t *testing.T) {
	t.Parallel()

	tree := content.AbyssTree()
	first := tree.Adj[0][0]
	node := tree.Node(first)
	analysis := analyzeAbyssTreePlan(tree, nil, []int{first}, 100, 100, first, first)
	if analysis.PlannedCost != 0 {
		t.Fatalf("prestige-memory node cost = %d, want 0", analysis.PlannedCost)
	}
	withoutMemory := analyzeAbyssTreePlan(tree, nil, []int{first}, 100, 100, 0, first)
	want := (node.Cost() + 1) / 2
	if withoutMemory.PlannedCost != want {
		t.Fatalf("discounted cost = %d, want %d", withoutMemory.PlannedCost, want)
	}
}

func TestAbyssTreeQuoteCommitParity(t *testing.T) {
	t.Parallel()
	tree := content.AbyssTree()
	first := tree.Adj[0][0]
	tests := []struct {
		name   string
		ids    []int
		points int
		depth  int
	}{
		{name: "valid", ids: []int{first}, points: 10, depth: 100},
		{name: "missing", ids: []int{first, 99999999}, points: 10, depth: 100},
		{name: "over budget", ids: []int{first}, points: 0, depth: 100},
		{name: "empty", ids: nil, points: 10, depth: 100},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			analysis := analyzeAbyssTreePlan(tree, nil, test.ids, test.points, test.depth, 0, -1)
			if accepted := abyssTreePlanCommitError(analysis) == ""; accepted != analysis.Valid {
				t.Fatalf("quote valid=%v, commit accepts=%v, error=%q", analysis.Valid, accepted, abyssTreePlanCommitError(analysis))
			}
		})
	}
}

func TestBuildAbyssTreeRefundQuoteMatchesCosts(t *testing.T) {
	t.Parallel()
	tree := content.AbyssTree()
	first := tree.Adj[0][0]
	quote := buildAbyssTreeRefundQuote(tree, []int{first}, first, first)
	wantPoints := (tree.Node(first).Cost() + 1) / 2
	if quote.AffectedCount != 1 || quote.PointTotal != wantPoints ||
		quote.GoldTotal != int64(wantPoints)*abyssTreeRefundGoldPerPoint {
		t.Fatalf("refund quote = %+v", quote)
	}
	if len(quote.NodeCosts) != 1 || quote.NodeCosts[0].ID != first || quote.NodeCosts[0].Points != wantPoints {
		t.Fatalf("refund node costs = %+v", quote.NodeCosts)
	}
}
