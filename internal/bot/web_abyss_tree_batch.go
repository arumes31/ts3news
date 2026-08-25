package bot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ts3news/internal/content"
)

type abyssSeasonalTreeBranch struct {
	Key        string
	Label      string
	Desc       string
	Sector     int
	Ends       string
	ActiveNode int
}

func abyssSeasonalTree(at time.Time) abyssSeasonalTreeBranch {
	utc := at.UTC()
	quarter := (int(utc.Month()) - 1) / 3
	sector := (utc.Year()*4 + quarter) % 6
	labels := []string{"Crimson Campaign", "Hearthwake", "Veiled Hunt", "Astral Study", "Gilded Road", "Void Pilgrimage"}
	startMonth := time.Month(quarter*3 + 1)
	ends := time.Date(utc.Year(), startMonth+3, 1, 0, 0, 0, 0, time.UTC)
	return abyssSeasonalTreeBranch{
		Key:   "season-" + strconv.Itoa(utc.Year()) + "-q" + strconv.Itoa(quarter+1),
		Label: labels[sector], Sector: sector, Ends: ends.Format("2006-01-02"),
		Desc: "Allocate 5 nodes in the highlighted sector to gain +5% material yield this season.",
	}
}

func abyssTreeLoadoutNamesKey(uid string) string { return "abyss_tree_loadout_names_" + uid }

func loadTreeLoadoutNames(stored string) map[string]string {
	out := map[string]string{}
	_ = json.Unmarshal([]byte(stored), &out)
	for slot, name := range out {
		n, err := strconv.Atoi(slot)
		if err != nil || n < 1 || n > 3 {
			delete(out, slot)
			continue
		}
		out[slot] = normalizeAbyssPresetName(name, n)
	}
	return out
}

func (s *WebServer) handleAbyssTreeBatchAllocate(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		IDs []int `json:"ids"`
	}
	if readJSON(r, &req) != nil || len(req.IDs) == 0 || len(req.IDs) > 100 {
		writeJSON(w, map[string]any{"ok": false, "error": "choose 1-100 queued nodes"})
		return
	}
	req.IDs = canonicalAbyssTreeIDs(req.IDs)
	alloc, err := s.bot.loadTreeAllocatedContext(r.Context(), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	tree := content.AbyssTree()
	have := map[int]bool{0: true}
	for _, id := range alloc {
		have[id] = true
	}
	var bestDepth int
	_ = s.bot.DB.QueryRowContext(r.Context(), "SELECT COALESCE(abyss_best_depth, 0) FROM users WHERE client_uid=$1", uid).Scan(&bestDepth)
	var hasDepth25Achievement bool
	_ = s.bot.DB.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM abyss_achievements
		WHERE client_uid=$1 AND code='depth_25')`, uid).Scan(&hasDepth25Achievement)
	spent := s.bot.treeSpentEx(uid, alloc)
	total := s.bot.treePointsTotal(uid)
	dayID := abyssNodeOfTheDay(time.Now())
	clean := make([]int, 0, len(req.IDs))
	nodeCosts := make(map[int]int, len(req.IDs))
	batchCost := 0
	for _, id := range req.IDs {
		node := tree.Node(id)
		if node == nil || have[id] {
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("node %d is unknown or already allocated", id)})
			return
		}
		connected := false
		for _, neighbor := range tree.Adj[id] {
			if have[neighbor] {
				connected = true
				break
			}
		}
		if !connected {
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("node %d is not connected in queue order", id)})
			return
		}
		required := 0
		if node.Type == "keystone" {
			required = 30
		} else if node.Ring > 20 {
			required = 20
		} else if node.Ring > 10 {
			required = 10
		}
		if id == treeNodeVictorsTrophy && !hasDepth25Achievement {
			writeJSON(w, map[string]any{"ok": false, "error": "Victor's Trophy requires the Depth 25 achievement"})
			return
		}
		if bestDepth < required {
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("node %d requires Abyss Floor %d", id, required)})
			return
		}
		cost := treeNodeCostFor(node, dayID)
		nodeCosts[id] = cost
		batchCost += cost
		spent += cost
		if spent > total {
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("queued changes cost more than your %d points", total)})
			return
		}
		have[id] = true
		clean = append(clean, id)
	}

	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	for _, id := range clean {
		if _, err := tx.ExecContext(r.Context(), "INSERT INTO user_abyss_tree (client_uid, node_id) VALUES ($1,$2)", uid, id); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	tb := s.bot.treeBonusFor(uid)
	writeJSON(w, map[string]any{"ok": true, "used": spent, "points": total, "stats": tb.Stats, "pct": tb.Pct,
		"node_costs": nodeCosts, "total_cost": batchCost,
		"msg": fmt.Sprintf("🌳 Applied %d queued tree changes atomically.", len(clean))})
}

func canonicalAbyssTreeIDs(ids []int) []int {
	seen := make(map[int]bool, len(ids))
	canonical := make([]int, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		canonical = append(canonical, id)
	}
	return canonical
}
