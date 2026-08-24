package content

import (
	"math/rand/v2"
	"sort"
	"testing"
	"time"
)

const abyssTreeTopologyGolden = "dc6c6882c73e1af8"

func TestAbyssTreeLayoutGolden(t *testing.T) {
	if got := AbyssTree().TopologyHash(); got != abyssTreeTopologyGolden {
		t.Fatalf("topology golden changed: got %q, want %q", got, abyssTreeTopologyGolden)
	}
}

func TestAbyssTreeGraphConnectivityProperties(t *testing.T) {
	tree := AbyssTree()
	reached := map[int]bool{0: true}
	queue := []int{0}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		neighbors := tree.Adj[current]
		if !sort.IntsAreSorted(neighbors) {
			t.Fatalf("neighbors for %d are not canonical: %v", current, neighbors)
		}
		for index, next := range neighbors {
			if index > 0 && next == neighbors[index-1] {
				t.Fatalf("duplicate edge %d-%d", current, next)
			}
			if !reached[next] {
				reached[next] = true
				queue = append(queue, next)
			}
		}
	}
	if len(reached) != len(tree.Nodes)+1 {
		t.Fatalf("reachable vertices = %d, want %d", len(reached), len(tree.Nodes)+1)
	}
}

func TestAbyssTreePathfindingProperties(t *testing.T) {
	tree := AbyssTree()
	random := rand.New(rand.NewPCG(20260824, 184))
	for sample := 0; sample < 250; sample++ {
		target := tree.Nodes[random.IntN(len(tree.Nodes))].ID
		path := testAbyssTreeShortestPath(tree, target)
		if len(path) < 2 || path[0] != 0 || path[len(path)-1] != target {
			t.Fatalf("invalid root path to %d: %v", target, path)
		}
		for index := 1; index < len(path); index++ {
			if !containsTreeNeighbor(tree.Adj[path[index-1]], path[index]) {
				t.Fatalf("path uses absent edge %d-%d", path[index-1], path[index])
			}
		}
	}
}

func TestAbyssTreePointAccountingProperties(t *testing.T) {
	tree := AbyssTree()
	random := rand.New(rand.NewPCG(20260824, 185))
	for sample := 0; sample < 100; sample++ {
		count := 1 + random.IntN(80)
		ids := make([]int, 0, count)
		seen := map[int]bool{}
		manualCost := 0
		manualBonus := TreeBonus{Pct: map[string]float64{}}
		for len(ids) < count {
			node := &tree.Nodes[random.IntN(len(tree.Nodes))]
			if seen[node.ID] {
				continue
			}
			seen[node.ID] = true
			ids = append(ids, node.ID)
			manualCost += node.Cost()
			manualBonus.Stats = manualBonus.Stats.Add(node.Stats)
			for key, value := range node.Pct {
				manualBonus.Pct[key] += value
			}
		}
		for _, id := range ids {
			node := tree.Node(id)
			if node.Type == TreeNodeSocket {
				for _, neighbor := range tree.Adj[id] {
					if seen[neighbor] {
						manualBonus.Stats.HP += 15
						manualBonus.Stats.STR += 5
						manualBonus.Stats.INT += 5
					}
				}
			}
			if node.Name == "⏳ Temporal Shift" {
				weekday := time.Now().Weekday()
				if weekday == time.Saturday || weekday == time.Sunday {
					manualBonus.Pct["gold_find"] += 0.15
				} else {
					manualBonus.Pct["xp_gain"] += 0.10
				}
			}
		}
		if got := tree.SpentPoints(ids); got != manualCost {
			t.Fatalf("spent points = %d, want %d", got, manualCost)
		}
		gotBonus := tree.BonusFor(ids)
		if gotBonus.Stats != manualBonus.Stats || !equalTreePct(gotBonus.Pct, manualBonus.Pct) {
			t.Fatalf("bonus accounting mismatch for sample %d", sample)
		}
	}
}

func testAbyssTreeShortestPath(tree *AbyssTreeData, target int) []int {
	queue := []int{0}
	previous := map[int]int{}
	reached := map[int]bool{0: true}
	for len(queue) > 0 && !reached[target] {
		current := queue[0]
		queue = queue[1:]
		for _, next := range tree.Adj[current] {
			if !reached[next] {
				reached[next] = true
				previous[next] = current
				queue = append(queue, next)
			}
		}
	}
	if !reached[target] {
		return nil
	}
	path := []int{target}
	for path[len(path)-1] != 0 {
		path = append(path, previous[path[len(path)-1]])
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func equalTreePct(left, right map[string]float64) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
