package content

import "testing"

// TestAbyssTreeShape locks in the web's contract: exactly 1000 unique nodes,
// every one of them reachable from the root, and effects that sum cleanly.
func TestAbyssTreeShape(t *testing.T) {
	tree := AbyssTree()

	// The deterministic grid+keystones+bridges still form the 1000-node base; the
	// outer Ascendant Rim adds more on top. No exact-count hardlock (the web is
	// allowed to grow), just a floor so the base can never silently shrink.
	if len(tree.Nodes) < 1000 {
		t.Fatalf("expected at least the 1000 base nodes, got %d", len(tree.Nodes))
	}

	seen := map[int]bool{}
	names := map[string]int{}
	for _, n := range tree.Nodes {
		if n.ID <= 0 {
			t.Fatalf("node has non-positive ID %d (0 is reserved for the root)", n.ID)
		}
		if seen[n.ID] {
			t.Fatalf("duplicate node ID %d", n.ID)
		}
		seen[n.ID] = true
		if n.Name == "" || n.Desc == "" {
			t.Errorf("node %d has empty name/desc", n.ID)
		}
		// Every node name must be unique across the entire web.
		if prev, dup := names[n.Name]; dup {
			t.Errorf("duplicate node name %q (nodes %d and %d)", n.Name, prev, n.ID)
		}
		names[n.Name] = n.ID
	}

	// BFS from the root: every node must be reachable (multiple paths are a
	// bonus; zero paths would strand content).
	visited := map[int]bool{0: true}
	queue := []int{0}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, nb := range tree.Adj[cur] {
			if !visited[nb] {
				visited[nb] = true
				queue = append(queue, nb)
			}
		}
	}
	for _, n := range tree.Nodes {
		if !visited[n.ID] {
			t.Errorf("node %d (%s) is unreachable from the root", n.ID, n.Name)
		}
	}

	// Determinism: a second build must produce identical IDs and effects.
	again := buildAbyssTree()
	for i := range tree.Nodes {
		a, b := tree.Nodes[i], again.Nodes[i]
		if a.ID != b.ID || a.Name != b.Name || a.Stats != b.Stats {
			t.Fatalf("tree generation is not deterministic at index %d: %+v vs %+v", i, a, b)
		}
	}

	// At least the 40 base keystones (36 rim + 4 crown); outer-rim tradeoff
	// keystones add more on top.
	keystones := 0
	juggernaut := 0
	for _, n := range tree.Nodes {
		if n.Type == "keystone" {
			keystones++
			if n.Name == "⚔️ Juggernaut" {
				juggernaut = n.ID
			}
		}
	}
	if keystones < 40 {
		t.Errorf("expected at least the 40 base keystones, got %d", keystones)
	}
	if juggernaut == 0 {
		t.Fatal("Juggernaut keystone not found")
	}

	// Bonus summation: allocating a keystone yields its pct effect.
	tb := tree.BonusFor([]int{juggernaut})
	if tb.Pct["str_pct"] != 0.25 {
		t.Errorf("Juggernaut keystone: expected str_pct 0.25, got %v", tb.Pct["str_pct"])
	}
}
