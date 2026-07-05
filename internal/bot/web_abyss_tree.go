package bot

// The Abyss Skill Web (PoE-style passive tree, 1000 nodes — see
// internal/content/abysstree.go for the generator). Points are earned by
// playing: 1 per character level, 1 per best depth reached, 1 per 10 lifetime
// floors and 25 per Abyss prestige. Allocation must extend a connected path
// from the web's root; a full respec costs tokens.

import (
	"fmt"
	"log"
	"net/http"

	"ts3news/internal/content"
)

const abyssTreeRespecTokens = 50

// abyssTreeRefundGoldPerPoint is the gold charged per skill point when refunding
// nodes individually (single or bulk "refund until this node"). Unlike the full
// respec (tokens), single refunds are a gold sink so players can retune freely.
const abyssTreeRefundGoldPerPoint = 100

// treeRefundSet returns the set of allocated node IDs that must be refunded when
// the player refunds target: target itself plus every other allocated node that
// would be left disconnected from the root once target is removed (its
// dependents further out the branch). The returned slice always contains target.
func treeRefundSet(tree *content.AbyssTreeData, alloc []int, target int) []int {
	keep := make(map[int]bool)
	for _, id := range alloc {
		if id != target {
			keep[id] = true
		}
	}
	// BFS from root through the still-allocated nodes; anything unreached is a
	// dependent of target.
	reached := map[int]bool{0: true}
	queue := []int{0}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, nb := range tree.Adj[curr] {
			if (keep[nb] || nb == 0) && !reached[nb] {
				reached[nb] = true
				queue = append(queue, nb)
			}
		}
	}
	set := []int{target}
	for _, id := range alloc {
		if id != target && !reached[id] {
			set = append(set, id)
		}
	}
	return set
}

// resetAbyssTreeOnLayoutChange wipes every player's allocated web nodes — a
// free full respec — when the generated layout no longer matches the layout
// hash the allocations were made under, then records the current hash. Points
// are derived (level/depth/floors/prestige), so nothing is lost but the
// allocation choices, which may be invalid on the new layout anyway.
func (b *Bot) resetAbyssTreeOnLayoutChange() {
	hash := content.AbyssTree().LayoutHash()
	var stored string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key='abyss_tree_layout'").Scan(&stored)
	if stored == hash {
		return
	}
	res, err := b.DB.Exec("DELETE FROM user_abyss_tree")
	if err != nil {
		log.Printf("Abyss skill web reset failed (layout %q -> %q): %v", stored, hash, err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("Abyss skill web layout changed (%q -> %q): wiped %d allocated nodes — free respec for all players", stored, hash, n)
	}
	if _, err := b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ('abyss_tree_layout', $1)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, hash); err != nil {
		log.Printf("Abyss skill web layout hash save failed: %v", err)
	}
}

// loadTreeAllocated returns the player's allocated node IDs. It fails closed:
// callers that gate spending (allocate, respec) must treat an error as "state
// unknown" and refuse, rather than proceeding as if the tree were empty.
func (b *Bot) loadTreeAllocated(uid string) ([]int, error) {
	rows, err := b.DB.Query("SELECT node_id FROM user_abyss_tree WHERE client_uid=$1", uid)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
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
// Combat/loot paths call this best-effort: a read failure yields no bonus
// (never a crash), which only ever under-rewards.
func (b *Bot) treeBonusFor(uid string) content.TreeBonus {
	alloc, err := b.loadTreeAllocated(uid)
	if err != nil {
		return content.TreeBonus{}
	}
	tb := content.AbyssTree().BonusFor(alloc)

	// Apply Prestige Reset Bonus Multipliers (Item 61): +1% flat stats per prestige
	var prestige int
	_ = b.DB.QueryRow("SELECT prestige FROM users WHERE client_uid=$1", uid).Scan(&prestige)
	if prestige > 0 {
		multiplier := 1.0 + 0.01*float64(prestige)
		tb.Stats.HP = int(float64(tb.Stats.HP) * multiplier)
		tb.Stats.MNA = int(float64(tb.Stats.MNA) * multiplier)
		tb.Stats.STR = int(float64(tb.Stats.STR) * multiplier)
		tb.Stats.DEF = int(float64(tb.Stats.DEF) * multiplier)
		tb.Stats.SPD = int(float64(tb.Stats.SPD) * multiplier)
		tb.Stats.LCK = int(float64(tb.Stats.LCK) * multiplier)
		tb.Stats.INT = int(float64(tb.Stats.INT) * multiplier)
		tb.Stats.STA = int(float64(tb.Stats.STA) * multiplier)
		tb.Stats.CRT = int(float64(tb.Stats.CRT) * multiplier)
		tb.Stats.DGE = int(float64(tb.Stats.DGE) * multiplier)
		tb.Stats.CHA = int(float64(tb.Stats.CHA) * multiplier)
		tb.Stats.STN = int(float64(tb.Stats.STN) * multiplier)
		tb.Stats.SHN = int(float64(tb.Stats.SHN) * multiplier)
		tb.Stats.HGR = int(float64(tb.Stats.HGR) * multiplier)
	}

	return tb
}

// handleAbyssTreePage renders the skill web.
func (s *WebServer) handleAbyssTreePage(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}
	tree := content.AbyssTree()
	alloc, err := s.bot.loadTreeAllocated(uid)
	if err != nil {
		http.Error(w, "failed to load skill web state", http.StatusInternalServerError)
		return
	}
	total := s.bot.treePointsTotal(uid)
	// Use the Bot-level bonus (not tree.BonusFor) so the page shows the same
	// totals combat applies — including the prestige multiplier.
	tb := s.bot.treeBonusFor(uid)

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

	used := tree.SpentPoints(alloc)
	avail := total - used
	if avail < 0 {
		avail = 0
	}

	s.render(w, "abysstree", map[string]any{
		"Title":     "Abyss Skill Web",
		"Nav":       "abyss",
		"U":         u,
		"Nodes":     tree.Nodes,
		"Edges":     edges,
		"Allocated": alloc,
		"Points":    total,
		"Used":      used,
		"Avail":     avail,
		"BonusPct":  pctView,
		// Raw maps seed the client summary so it always mirrors the
		// server-computed totals (socket adjacency, Temporal Shift, prestige).
		"BonusPctRaw": tb.Pct,
		"Bonus":       tb.Stats,
		// Best depth drives the client-side mirror of the allocation floor
		// gates (Item 62), so gated nodes are shown locked instead of
		// advertising an allocate the server would refuse.
		"BestDepth": s.bot.loadAbyssStats(uid).BestDepth,
		"RespecTk":  abyssTreeRespecTokens,
		// Talent/Spec updates
		"Stats":     s.bot.loadAbyssStats(uid),
		"Spec":      s.bot.abyssSpec(uid),
		"SpecDefs":  abyssSpecs,
		"Tokens":    s.bot.abyssTokens(uid),
		"NodeGates": abyssUpgradeMinDepth,
	})
}

// treePctLabelPublic re-exports the content label helper for the summary box.
func treePctLabelPublic(k string) string {
	labels := map[string]string{
		"str_pct": "STR (Abyss)", "hp_pct": "max HP (Abyss)", "spd_pct": "SPD (Abyss)",
		"int_pct": "INT (Abyss)", "gold_find": "gold from drops", "loot_find": "loot find",
		"escrow_bonus": "escrow floor bonus", "xp_gain": "floor XP",
		"token_gain": "tokens on bank", "material_yield": "crafting materials",
		"skill_damage": "skill damage", "skill_mana_cost": "skill mana cost reduction",
		"consumable_save": "chance consumables keep their charge",
	}
	if l, ok := labels[k]; ok {
		return l
	}
	return k
}

// handleAbyssTreeAllocate spends the node's point cost (1/2/3 by size) on a
// node. The node must extend a connected path: adjacent to an already-allocated
// node or to the root.
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

	// Fail closed: with the allocation state unknown we cannot validate spent
	// points or path connectivity, so refuse instead of treating it as empty.
	alloc, err := s.bot.loadTreeAllocated(uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	have := map[int]bool{0: true} // the root is always allocated
	for _, id := range alloc {
		have[id] = true
	}
	if have[req.NodeID] {
		writeJSON(w, map[string]any{"ok": false, "error": "already allocated"})
		return
	}
	spent := tree.SpentPoints(alloc)
	cost := node.Cost()
	if spent+cost > s.bot.treePointsTotal(uid) {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("not enough skill points — this node costs %d (spent %d of %d)", cost, spent, s.bot.treePointsTotal(uid))})
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

	// Abyss Floor Unlocks check (Item 62)
	if node.Type == "keystone" || node.Ring > 10 {
		var maxFloor int
		_ = s.bot.DB.QueryRow("SELECT COALESCE(abyss_best_depth, 0) FROM users WHERE client_uid=$1", uid).Scan(&maxFloor)
		requiredFloor := 10
		if node.Type == "keystone" {
			requiredFloor = 30
		} else if node.Ring > 20 {
			requiredFloor = 20
		}
		if maxFloor < requiredFloor {
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("Requires clearing Abyss Floor %d (your record is %d)", requiredFloor, maxFloor)})
			return
		}
	}

	if _, err := s.bot.DB.Exec(
		"INSERT INTO user_abyss_tree (client_uid, node_id) VALUES ($1,$2) ON CONFLICT DO NOTHING", uid, req.NodeID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	tb := s.bot.treeBonusFor(uid)
	writeJSON(w, map[string]any{
		"ok": true, "node_id": req.NodeID, "used": spent + cost,
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

	alloc, err := s.bot.loadTreeAllocated(uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if len(alloc) == 0 {
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

// handleAbyssTreeRefund refunds a node for gold (Item 48). Refunding a node that
// other allocated nodes branch off of would disconnect them, so the refund
// always cascades "until this node": the target plus every dependent further out
// the branch is refunded together. Cost is gold per skill point refunded.
func (s *WebServer) handleAbyssTreeRefund(w http.ResponseWriter, r *http.Request, uid string) {
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

	alloc, err := s.bot.loadTreeAllocated(uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	found := false
	for _, id := range alloc {
		if id == req.NodeID {
			found = true
			break
		}
	}
	if !found {
		writeJSON(w, map[string]any{"ok": false, "error": "node is not allocated"})
		return
	}

	// The target plus every dependent that would be orphaned once it is removed.
	refundIDs := treeRefundSet(tree, alloc, req.NodeID)
	refundCost := 0
	for _, id := range refundIDs {
		if n := tree.Node(id); n != nil {
			refundCost += n.Cost()
		} else {
			refundCost++
		}
	}
	goldCost := int64(refundCost) * abyssTreeRefundGoldPerPoint

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	for _, id := range refundIDs {
		if _, err := tx.Exec("DELETE FROM user_abyss_tree WHERE client_uid=$1 AND node_id=$2", uid, id); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if !deductGold(w, tx, uid, goldCost) {
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	// Recalculate stats for response
	refunded := make(map[int]bool, len(refundIDs))
	for _, id := range refundIDs {
		refunded[id] = true
	}
	var remainingAlloc []int
	for _, id := range alloc {
		if !refunded[id] {
			remainingAlloc = append(remainingAlloc, id)
		}
	}

	msg := "🌳 Refunded: " + node.Name
	if len(refundIDs) > 1 {
		msg = fmt.Sprintf("🌳 Refunded %d nodes (down to %s) for 🜲 %d gold.", len(refundIDs), node.Name, goldCost)
	} else {
		msg = fmt.Sprintf("🌳 Refunded: %s for 🜲 %d gold.", node.Name, goldCost)
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)

	tb := s.bot.treeBonusFor(uid)
	writeJSON(w, map[string]any{
		"ok": true, "node_id": req.NodeID, "used": tree.SpentPoints(remainingAlloc),
		"points":   s.bot.treePointsTotal(uid),
		"refunded": refundIDs,
		"msg":      msg,
		"stats":    tb.Stats, "pct": tb.Pct,
		"gold": gold,
	})
}
