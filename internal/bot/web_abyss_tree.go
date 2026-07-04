package bot

// The Abyss Skill Web (PoE-style passive tree, 1000 nodes — see
// internal/content/abysstree.go for the generator). Points are earned by
// playing: 1 per character level, 1 per best depth reached, 1 per 10 lifetime
// floors and 25 per Abyss prestige. Allocation must extend a connected path
// from the web's root; a full respec costs tokens.

import (
	"fmt"
	"net/http"

	"ts3news/internal/content"
)

const abyssTreeRespecTokens = 50

// loadTreeAllocated returns the player's allocated node IDs.
func (b *Bot) loadTreeAllocated(uid string) []int {
	rows, err := b.DB.Query("SELECT node_id FROM user_abyss_tree WHERE client_uid=$1", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			out = append(out, id)
		}
	}
	return out
}

// treePointsTotal is how many skill points the player has earned so far.
func (b *Bot) treePointsTotal(uid string) int {
	var level, bestDepth, prestige int
	var lifetimeFloors int64
	_ = b.DB.QueryRow(
		"SELECT level, abyss_best_depth, abyss_prestige, abyss_lifetime_floors FROM users WHERE client_uid=$1", uid,
	).Scan(&level, &bestDepth, &prestige, &lifetimeFloors)
	pts := level + bestDepth + int(lifetimeFloors/10) + prestige*25
	if pts > 1000 {
		pts = 1000
	}
	return pts
}

// treeBonusFor sums the player's allocated web nodes into one bonus block.
func (b *Bot) treeBonusFor(uid string) content.TreeBonus {
	return content.AbyssTree().BonusFor(b.loadTreeAllocated(uid))
}

// handleAbyssTreePage renders the skill web.
func (s *WebServer) handleAbyssTreePage(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}
	tree := content.AbyssTree()
	alloc := s.bot.loadTreeAllocated(uid)
	total := s.bot.treePointsTotal(uid)
	tb := tree.BonusFor(alloc)

	// Flatten adjacency into edge pairs once for the client renderer.
	type edge [2]int
	seen := map[edge]bool{}
	var edges []edge
	for a, ns := range tree.Adj {
		for _, b := range ns {
			e := edge{a, b}
			if a > b {
				e = edge{b, a}
			}
			if !seen[e] {
				seen[e] = true
				edges = append(edges, e)
			}
		}
	}

	pctView := map[string]string{}
	for k, v := range tb.Pct {
		pctView[treePctLabelPublic(k)] = fmt.Sprintf("%+.1f%%", v*100)
	}

	s.render(w, "abysstree", map[string]any{
		"Title":     "Abyss Skill Web",
		"Nav":       "abyss",
		"U":         u,
		"Nodes":     tree.Nodes,
		"Edges":     edges,
		"Allocated": alloc,
		"Points":    total,
		"Used":      len(alloc),
		"BonusPct":  pctView,
		"Bonus":     tb.Stats,
		"RespecTk":  abyssTreeRespecTokens,
	})
}

// treePctLabelPublic re-exports the content label helper for the summary box.
func treePctLabelPublic(k string) string {
	labels := map[string]string{
		"str_pct": "STR (Abyss)", "hp_pct": "max HP (Abyss)", "spd_pct": "SPD (Abyss)",
		"int_pct": "INT (Abyss)", "gold_find": "gold from drops", "loot_find": "loot find",
		"escrow_bonus": "escrow floor bonus", "xp_gain": "floor XP",
		"token_gain": "tokens on bank", "material_yield": "crafting materials",
	}
	if l, ok := labels[k]; ok {
		return l
	}
	return k
}

// handleAbyssTreeAllocate spends one point on a node. The node must extend a
// connected path: adjacent to an already-allocated node or to the root.
func (s *WebServer) handleAbyssTreeAllocate(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		NodeID int `json:"node_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	tree := content.AbyssTree()
	node := tree.Node(req.NodeID)
	if node == nil {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown node"})
		return
	}

	alloc := s.bot.loadTreeAllocated(uid)
	have := map[int]bool{0: true} // the root is always allocated
	for _, id := range alloc {
		have[id] = true
	}
	if have[req.NodeID] {
		writeJSON(w, map[string]any{"ok": false, "error": "already allocated"})
		return
	}
	if len(alloc) >= s.bot.treePointsTotal(uid) {
		writeJSON(w, map[string]any{"ok": false, "error": "no skill points available — descend deeper, level up or prestige"})
		return
	}
	connected := false
	for _, nb := range tree.Adj[req.NodeID] {
		if have[nb] {
			connected = true
			break
		}
	}
	if !connected {
		writeJSON(w, map[string]any{"ok": false, "error": "node is not connected to your path"})
		return
	}

	if _, err := s.bot.DB.Exec(
		"INSERT INTO user_abyss_tree (client_uid, node_id) VALUES ($1,$2) ON CONFLICT DO NOTHING", uid, req.NodeID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	tb := tree.BonusFor(append(alloc, req.NodeID))
	writeJSON(w, map[string]any{
		"ok": true, "node_id": req.NodeID, "used": len(alloc) + 1,
		"points": s.bot.treePointsTotal(uid),
		"msg":    "🌳 Allocated: " + node.Name,
		"stats":  tb.Stats, "pct": tb.Pct,
	})
}

// handleAbyssTreeRespec clears every allocated node for a token fee.
func (s *WebServer) handleAbyssTreeRespec(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	if len(s.bot.loadTreeAllocated(uid)) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "nothing to respec"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !deductTokens(w, tx, uid, abyssTreeRespecTokens) {
		return
	}
	if _, err := tx.Exec("DELETE FROM user_abyss_tree WHERE client_uid=$1", uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("🌳 Skill web reset for 🜲%d — every point refunded.", abyssTreeRespecTokens),
		"tokens": s.bot.abyssTokens(uid)})
}
