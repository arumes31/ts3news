package bot

// The Abyss Skill Web (PoE-style passive tree, 1000 base nodes + the 100-node
// outer Ascendant Rim = 1100 — see
// internal/content/abysstree.go for the generator). Points are earned by
// playing: 1 per character level, 1 per best depth reached, 1 per 10 lifetime
// floors and 25 per Abyss prestige. Allocation must extend a connected path
// from the web's root; a full respec costs tokens.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

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

	tx, err := b.DB.Begin()
	if err != nil {
		log.Printf("Abyss skill web reset transaction failed: %v", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	var stored string
	err = tx.QueryRow("SELECT value FROM app_meta WHERE key='abyss_tree_layout' FOR UPDATE").Scan(&stored)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Abyss skill web reset failed to check stored layout hash: %v", err)
		return
	}

	if stored == hash {
		return
	}

	res, err := tx.Exec("DELETE FROM user_abyss_tree")
	if err != nil {
		log.Printf("Abyss skill web reset failed (layout %q -> %q): %v", stored, hash, err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("Abyss skill web layout changed (%q -> %q): wiped %d allocated nodes — free respec for all players", stored, hash, n)
	}

	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ('abyss_tree_layout', $1)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, hash); err != nil {
		log.Printf("Abyss skill web layout hash save failed: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Abyss skill web reset transaction commit failed: %v", err)
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
	// The web grew from ~1000 to ~5100 nodes, so points scale up to match: the old
	// formula is roughly tripled and the cap raised to 5000 (aura mega-nodes cost
	// 50 each, so the outermost rim is still a serious long-term investment).
	pts := level*2 + bestDepth*3 + int(lifetimeFloors/4) + prestige*60
	if pts > 5000 {
		pts = 5000
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

	// Build map for quick O(1) allocation lookup
	allocatedMap := make(map[int]bool, len(alloc))
	for _, nid := range alloc {
		allocatedMap[nid] = true
	}

	// 1. Jewel Sockets adjacent modifiers & Timeless Jewels area buffs
	var socketMap map[int]string
	var storedJson string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_sockets_"+uid).Scan(&storedJson)
	if storedJson != "" {
		_ = json.Unmarshal([]byte(storedJson), &socketMap)
	}
	for nid := range allocatedMap {
		n := content.AbyssTree().Node(nid)
		if n != nil && n.Type == "socket" && socketMap != nil {
			if jewel, ok := socketMap[nid]; ok && jewel != "" {
				if strings.HasPrefix(jewel, "timeless_") {
					// Timeless Jewel: timeless_<seed>_<size>_<stat>
					parts := strings.Split(jewel, "_")
					if len(parts) == 4 {
						var seed int
						_, _ = fmt.Sscanf(parts[1], "%d", &seed)
						size := parts[2]
						statType := parts[3]

						radius := 120.0
						switch size {
						case "medium":
							radius = 220.0
						case "large":
							radius = 350.0
						}

						// Buff all allocated nodes in radius
						for neighborID := range allocatedMap {
							neighbor := content.AbyssTree().Node(neighborID)
							if neighbor != nil {
								dx := n.X - neighbor.X
								dy := n.Y - neighbor.Y
								dist := math.Sqrt(dx*dx + dy*dy)
								if dist <= radius {
									// Apply stat
									switch statType {
									case "str":
										tb.Stats.STR += 5
									case "int":
										tb.Stats.INT += 5
									case "spd":
										tb.Stats.SPD += 5
									case "hp":
										tb.Stats.HP += 15
									}
									// Seed-based flavor
									switch seed % 3 {
									case 0:
										tb.Stats.DEF += 1
									case 1:
										tb.Stats.LCK += 1
									case 2:
										tb.Stats.CRT += 1
									}
								}
							}
						}
					}
				} else {
					// Normal Jewel
					for _, neighborID := range content.AbyssTree().Adj[nid] {
						if allocatedMap[neighborID] {
							switch jewel {
							case "ruby":
								tb.Stats.STR += 15
							case "sapphire":
								tb.Stats.INT += 15
							case "topaz":
								tb.Stats.SPD += 15
							}
						}
					}
				}
			}
		}
	}

	// 2. Timed Keystone Perks (Chronobreaker/Limit Break active buff)
	var expStr string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", fmt.Sprintf("abyss_keystone_active_%s_%d", uid, content.NodeLimitBreak)).Scan(&expStr)
	if expStr != "" {
		if expTime, err := time.Parse(time.RFC3339, expStr); err == nil && time.Now().Before(expTime) {
			tb.Pct["xp_gain"] += 0.50
		}
	}

	// 3. Synergy Nodes
	if allocatedMap[617] { // Arcane Warrior notable
		arcaneCount := 0
		for nid := range allocatedMap {
			n := content.AbyssTree().Node(nid)
			if n != nil && n.Sector == 3 { // Arcane Sector
				arcaneCount++
			}
		}
		tb.Stats.STR += 2 * arcaneCount
	}
	if allocatedMap[478] { // Beastmaster's Command / Void Vitality
		voidCount := 0
		for nid := range allocatedMap {
			n := content.AbyssTree().Node(nid)
			if n != nil && n.Sector == 5 { // Void Sector
				voidCount++
			}
		}
		tb.Stats.HP += 8 * voidCount
	}

	// 4. Specialization Bonuses (Specialist's Harmony node 771)
	if allocatedMap[771] {
		var spec string
		_ = b.DB.QueryRow("SELECT active_specialization FROM users WHERE client_uid=$1", uid).Scan(&spec)
		switch spec {
		case "warden":
			tb.Pct["hp_pct"] += 0.15
		case "delver":
			tb.Pct["str_pct"] += 0.15
		case "plunderer":
			tb.Pct["gold_find"] += 0.25
		}
	}

	// 5. Abyss Depth-Scaling (Abyssal Attunement node 818)
	if allocatedMap[818] {
		var bestDepth int
		_ = b.DB.QueryRow("SELECT abyss_best_depth FROM users WHERE client_uid=$1", uid).Scan(&bestDepth)
		tb.Pct["str_pct"] += 0.005 * float64(bestDepth)
		tb.Pct["hp_pct"] += 0.005 * float64(bestDepth)
	}

	// 6. Prestige Multipliers (Prestige Focus node 834)
	if allocatedMap[834] {
		var prestige int
		_ = b.DB.QueryRow("SELECT abyss_prestige FROM users WHERE client_uid=$1", uid).Scan(&prestige)
		if prestige > 0 {
			mult := 0.10 * float64(prestige)
			for nid := range allocatedMap {
				if nid != 834 {
					n := content.AbyssTree().Node(nid)
					if n != nil && n.Sector == 0 { // War Sector
						tb.Stats.HP += int(float64(n.Stats.HP) * mult)
						tb.Stats.MNA += int(float64(n.Stats.MNA) * mult)
						tb.Stats.STR += int(float64(n.Stats.STR) * mult)
						tb.Stats.DEF += int(float64(n.Stats.DEF) * mult)
						tb.Stats.SPD += int(float64(n.Stats.SPD) * mult)
						tb.Stats.INT += int(float64(n.Stats.INT) * mult)
						for k, v := range n.Pct {
							tb.Pct[k] += v * mult
						}
					}
				}
			}
		}
	}

	// 7. Set Resonance (Set Resonance node 883): +5% str/hp/int/spd per full set tier
	// (every 2 equipped pieces sharing a set). Counts equipped gear by EffectiveSetID,
	// mirroring activeLootMult's set counting — the old query hit a non-existent
	// `user_gears` table (with no such columns) and silently granted nothing.
	if allocatedMap[883] {
		setCount := map[string]int{}
		for _, g := range b.getEquippedItems(uid) {
			if sid := g.EffectiveSetID(); sid != "" {
				setCount[sid]++
			}
		}
		var setBonusTiers int
		for _, cnt := range setCount {
			if cnt >= 2 {
				setBonusTiers += cnt / 2
			}
		}
		if setBonusTiers > 0 {
			mult := 0.05 * float64(setBonusTiers)
			tb.Pct["str_pct"] += mult
			tb.Pct["hp_pct"] += mult
			tb.Pct["int_pct"] += mult
			tb.Pct["spd_pct"] += mult
		}
	}

	// 8. Skill Transformations (Elemental Transmutation node 932)
	if allocatedMap[932] {
		if strPct := tb.Pct["str_pct"]; strPct != 0 {
			tb.Pct["int_pct"] += strPct * 0.5
			tb.Pct["str_pct"] -= strPct * 0.5
		}
	}

	// 9. Generic Abyss talents (Deep-Delver extension + the active spec's sub-tree).
	// Stored key→level in app_meta; each level's Stats/Pct rides this same bonus so
	// every generic talent shares the live combat/economy pipeline. Added before the
	// prestige multiplier so talent stats scale with prestige like the rest.
	talentBonus := b.abyssTalentBonus(uid)
	tb.Stats = tb.Stats.Add(talentBonus.Stats)
	for k, v := range talentBonus.Pct {
		tb.Pct[k] += v
	}

	// Apply Prestige Reset Bonus Multipliers (Item 61): +1% flat tree stats per Abyss
	// prestige. Keyed off abyss_prestige (like treePointsTotal and applyAbyssRegen),
	// not the main-game prestige column, and folded through the canonical Stats.Scaled
	// helper so it can't drift from every other stat scaler.
	var prestige int
	_ = b.DB.QueryRow("SELECT abyss_prestige FROM users WHERE client_uid=$1", uid).Scan(&prestige)
	if prestige > 0 {
		tb.Stats = tb.Stats.Scaled(1.0 + 0.01*float64(prestige))
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

	// Load sockets
	var socketsJson string
	_ = s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_sockets_"+uid).Scan(&socketsJson)
	if socketsJson == "" {
		socketsJson = "{}"
	}

	// Load keystone status
	var activeUntil string
	_ = s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", fmt.Sprintf("abyss_keystone_active_%s_%d", uid, content.NodeLimitBreak)).Scan(&activeUntil)
	var cooldownUntil string
	_ = s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", fmt.Sprintf("abyss_keystone_cooldown_%s_%d", uid, content.NodeLimitBreak)).Scan(&cooldownUntil)

	st := s.bot.loadAbyssStats(uid) // one ~19-column read, reused for BestDepth + Stats below
	s.render(w, "abysstree", map[string]any{
		"Title":     "Abyss Skill Web",
		"Nav":       "abyss",
		"U":         u,
		"Nodes":     tree.Nodes,
		"Edges":     edges,
		// Only the handful of real cross-sector portals render with the distinct
		// animated style; every other long edge stays a plain skill path.
		"Portals": tree.Portals,
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
		"BestDepth": st.BestDepth,
		"RespecTk":  abyssTreeRespecTokens,
		// Talent/Spec updates
		"Stats":     st,
		"Spec":      s.bot.abyssSpec(uid),
		"SpecDefs":  abyssSpecs,
		// Deep-Delver extension: 50 generic talent nodes + their allocated levels,
		// concatenated into the client TALENTS array (single source of truth in Go).
		"DelverTalentDefs":   content.DeepDelverTalents,
		"DelverTalentLevels": s.bot.loadAbyssTalentLevels(uid),
		// Per-spec allocatable sub-trees (50 nodes each); the active spec's tree is
		// drawn in the Specializations tab. Levels reuse DelverTalentLevels above
		// (loadAbyssTalentLevels returns every generic key, spec nodes included).
		"SpecTalentDefs": content.SpecTalents,
		"Tokens":    s.bot.abyssTokens(uid),
		"NodeGates": abyssUpgradeMinDepth,
		// Layout-derived special node IDs, injected so the client special-cases the
		// right nodes instead of the old 1000-node literals (974/999) that drift when
		// the ring count changes.
		"LimitBreakID": content.NodeLimitBreak,
		"SanctuaryID":  content.NodeSecretSanctuary,
		"Sockets":   socketsJson,
		"ActiveKeystoneExpiry": activeUntil,
		"ActiveKeystoneCooldown": cooldownUntil,
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
		"hp_regen":        "HP regen / sec",
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

	// Achievement Gate Nodes check (Item 60)
	if req.NodeID == 697 {
		var maxFloor int
		_ = s.bot.DB.QueryRow("SELECT COALESCE(abyss_best_depth, 0) FROM users WHERE client_uid=$1", uid).Scan(&maxFloor)
		if maxFloor < 25 {
			writeJSON(w, map[string]any{"ok": false, "error": "Requires clearing Abyss Floor 25 to allocate this node (Victor's Trophy)"})
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

	var msg string
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

type socketRequest struct {
	NodeID int    `json:"node_id"`
	Jewel  string `json:"jewel"` // ruby | sapphire | topaz | ""
}

func (s *WebServer) handleAbyssTreeSocket(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req socketRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid request"})
		return
	}

	node := content.AbyssTree().Node(req.NodeID)
	if node == nil || node.Type != "socket" {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid socket node"})
		return
	}

	alloc, err := s.bot.loadTreeAllocated(uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}
	allocated := false
	for _, id := range alloc {
		if id == req.NodeID {
			allocated = true
			break
		}
	}
	if !allocated {
		writeJSON(w, map[string]any{"ok": false, "error": "socket node is not allocated"})
		return
	}

	if req.Jewel != "" && req.Jewel != "ruby" && req.Jewel != "sapphire" && req.Jewel != "topaz" {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid jewel type"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	if req.Jewel != "" {
		if !deductGold(w, tx, uid, 100) {
			return
		}
	}

	// Load existing socket mapping
	var storedJson string
	_ = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_sockets_"+uid).Scan(&storedJson)
	socketMap := map[int]string{}
	if storedJson != "" {
		_ = json.Unmarshal([]byte(storedJson), &socketMap)
	}

	if req.Jewel == "" {
		delete(socketMap, req.NodeID)
	} else {
		socketMap[req.NodeID] = req.Jewel
	}

	newJson, _ := json.Marshal(socketMap)
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, "abyss_sockets_"+uid, string(newJson)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)

	tb := s.bot.treeBonusFor(uid)
	writeJSON(w, map[string]any{
		"ok": true,
		"msg": fmt.Sprintf("💎 Jewel Socket updated: %s", req.Jewel),
		"gold": gold,
		"stats": tb.Stats,
		"pct": tb.Pct,
		"sockets": socketMap,
	})
}

type keystoneRequest struct {
	NodeID int `json:"node_id"`
}

func (s *WebServer) handleAbyssTreeActivateKeystone(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req keystoneRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid request"})
		return
	}

	if req.NodeID != content.NodeLimitBreak {
		writeJSON(w, map[string]any{"ok": false, "error": "node is not a timed buff keystone"})
		return
	}

	alloc, err := s.bot.loadTreeAllocated(uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}
	allocated := false
	for _, id := range alloc {
		if id == req.NodeID {
			allocated = true
			break
		}
	}
	if !allocated {
		writeJSON(w, map[string]any{"ok": false, "error": "keystone node is not allocated"})
		return
	}

	// Check Cooldown: key `abyss_keystone_cooldown_<uid>_<nodeID>`
	var cdStr string
	_ = s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", fmt.Sprintf("abyss_keystone_cooldown_%s_%d", uid, req.NodeID)).Scan(&cdStr)
	if cdStr != "" {
		if cdTime, err := time.Parse(time.RFC3339, cdStr); err == nil && time.Now().Before(cdTime) {
			diff := time.Until(cdTime)
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("Keystone is on cooldown for another %s", diff.Round(time.Second))})
			return
		}
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Activate buff: expires in 1 hour
	expiry := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, fmt.Sprintf("abyss_keystone_active_%s_%d", uid, req.NodeID), expiry); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}

	// Set Cooldown: expires in 24 hours
	cooldown := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, fmt.Sprintf("abyss_keystone_cooldown_%s_%d", uid, req.NodeID), cooldown); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}

	tb := s.bot.treeBonusFor(uid)
	writeJSON(w, map[string]any{
		"ok": true,
		"msg": "⏳ Chronobreak activated! +50% XP Gain for the next 1 hour.",
		"stats": tb.Stats,
		"pct": tb.Pct,
		"active_until": expiry,
		"cooldown_until": cooldown,
	})
}

type rollTimelessRequest struct {
	NodeID int `json:"node_id"`
}

func (s *WebServer) handleAbyssTreeRollTimeless(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req rollTimelessRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid request"})
		return
	}

	node := content.AbyssTree().Node(req.NodeID)
	if node == nil || node.Type != "socket" {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid socket node"})
		return
	}

	alloc, err := s.bot.loadTreeAllocated(uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}
	allocated := false
	for _, id := range alloc {
		if id == req.NodeID {
			allocated = true
			break
		}
	}
	if !allocated {
		writeJSON(w, map[string]any{"ok": false, "error": "socket node is not allocated"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Roll Timeless Jewel costs 500 gold
	if !deductGold(w, tx, uid, 500) {
		return
	}

	// Roll timeless attributes: seed (1..8000), size (small/medium/large), stat (str/int/spd/hp)
	seed := rand.IntN(8000) + 1
	roll := rand.Float64()
	size := "small"
	if roll >= 0.98 {
		size = "large"
	} else if roll >= 0.80 {
		size = "medium"
	}

	statRoll := rand.IntN(4)
	statType := "str"
	switch statRoll {
	case 1:
		statType = "int"
	case 2:
		statType = "spd"
	case 3:
		statType = "hp"
	}

	jewelCode := fmt.Sprintf("timeless_%d_%s_%s", seed, size, statType)

	// Load existing socket mapping
	var storedJson string
	_ = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1", "abyss_sockets_"+uid).Scan(&storedJson)
	socketMap := map[int]string{}
	if storedJson != "" {
		_ = json.Unmarshal([]byte(storedJson), &socketMap)
	}

	socketMap[req.NodeID] = jewelCode

	newJson, _ := json.Marshal(socketMap)
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, "abyss_sockets_"+uid, string(newJson)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db error"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)

	// Get a nice player-facing description of the rolled jewel
	jewelDesc := ""
	switch statType {
	case "str":
		jewelDesc = "+5 STR"
	case "int":
		jewelDesc = "+5 INT"
	case "spd":
		jewelDesc = "+5 SPD"
	case "hp":
		jewelDesc = "+15 HP"
	}
	extraLabel := ""
	switch seed % 3 {
	case 0:
		extraLabel = " (+1 DEF)"
	case 1:
		extraLabel = " (+1 LCK)"
	case 2:
		extraLabel = " (+1 CRT)"
	}

	msg := fmt.Sprintf("💎 Timeless Jewel Rolled! Seed: %d, Size: %s, Area Buff: %s%s", seed, size, jewelDesc, extraLabel)

	tb := s.bot.treeBonusFor(uid)
	writeJSON(w, map[string]any{
		"ok": true,
		"msg": msg,
		"gold": gold,
		"stats": tb.Stats,
		"pct": tb.Pct,
		"sockets": socketMap,
	})
}
