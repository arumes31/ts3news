package bot

// Abyss Skill Web — progression & talent features (AB-151 … AB-175, group G of
// docs/ABYSS_IMPROVEMENTS_300.md). All new state lives in app_meta JSON (no DB
// migrations): loadouts, loose jewels, weekly free-respec marker, last
// allocation (undo), prestige memory, mastery shards.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"ts3news/internal/content"
)

// ---------------------------------------------------------------------------
// app_meta keys
// ---------------------------------------------------------------------------

func abyssTreeLoadoutsKey(uid string) string   { return "abyss_tree_loadouts_" + uid }
func abyssJewelsKey(uid string) string         { return "abyss_jewels_" + uid }
func abyssFreeRespecKey(uid string) string     { return "abyss_free_respec_" + uid }
func abyssLastAllocKey(uid string) string      { return "abyss_last_alloc_" + uid }
func abyssPrestigeMemoryKey(uid string) string { return "abyss_prestige_memory_" + uid }
func abyssMasteryShardKey(uid string) string   { return "abyss_mastery_shard_" + uid }

// ---------------------------------------------------------------------------
// AB-157: first respec each week is free
// ---------------------------------------------------------------------------

// abyssCurrentWeek returns the ISO week marker ("2026-W34") used for the weekly
// free respec.
func abyssCurrentWeek(now time.Time) string {
	y, w := now.UTC().ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

// abyssFreeRespecAvailable reports whether the player's free weekly respec is
// still unused this ISO week (read-only, for page rendering).
func (b *Bot) abyssFreeRespecAvailable(uid string) bool {
	var stored string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssFreeRespecKey(uid)).Scan(&stored)
	return stored != abyssCurrentWeek(time.Now())
}

// chargeTreeRespec takes the respec fee from inside tx: the first respec of
// each ISO week is free (AB-157), later ones cost abyssTreeRespecTokens tokens.
// Returns (wasFree, ok). On failure it has already written the error response.
func chargeTreeRespec(w http.ResponseWriter, tx *sql.Tx, uid string) (bool, bool) {
	week := abyssCurrentWeek(time.Now())
	key := abyssFreeRespecKey(uid)
	var stored string
	_ = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", key).Scan(&stored)
	if stored != week {
		if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, week); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return false, false
		}
		return true, true
	}
	if !deductTokens(w, tx, uid, abyssTreeRespecTokens) {
		return false, false
	}
	return false, true
}

// ---------------------------------------------------------------------------
// AB-167: node of the day (half price, deterministic from the UTC date)
// ---------------------------------------------------------------------------

// abyssNodeOfTheDay returns the deterministic node-of-the-day ID: a hash of the
// UTC date mapped onto the layout, skipping auras (halving a 50-point aura
// would be a balance lever, not a daily nudge). Returns 0 when there are no
// eligible nodes.
func abyssNodeOfTheDay(t time.Time) int {
	tree := content.AbyssTree()
	if len(tree.Nodes) == 0 {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte("abyss-node-of-the-day:" + t.UTC().Format("2006-01-02")))
	start := int(h.Sum32() % uint32(len(tree.Nodes)))
	for i := 0; i < len(tree.Nodes); i++ {
		n := tree.Nodes[(start+i)%len(tree.Nodes)]
		if n.Type != "aura" {
			return n.ID
		}
	}
	return 0
}

// treeNodeCostFor mirrors TreeNode.Cost but applies the node-of-the-day half
// price (rounded up). Client-side effCost() in abysstree.html mirrors this.
func treeNodeCostFor(n *content.TreeNode, dayID int) int {
	c := n.Cost()
	if n.ID == dayID && c > 1 {
		c = (c + 1) / 2
	}
	return c
}

// ---------------------------------------------------------------------------
// AB-168: prestige memory — one chosen node stays allocated for free
// ---------------------------------------------------------------------------

// abyssPrestigeMemoryNode returns the player's chosen prestige-memory node ID
// (0 when none / invalid).
func (b *Bot) abyssPrestigeMemoryNode(uid string) int {
	var v string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssPrestigeMemoryKey(uid)).Scan(&v)
	id, err := strconv.Atoi(v)
	if err != nil || id <= 0 || content.AbyssTree().Node(id) == nil {
		return 0
	}
	return id
}

// treeSpentEx is SpentPoints with the prestige-memory node exempted: the chosen
// node is allocated for free (AB-168), so its cost doesn't count against the
// player's point budget.
func (b *Bot) treeSpentEx(uid string, alloc []int) int {
	spent := content.AbyssTree().SpentPoints(alloc)
	if mem := b.abyssPrestigeMemoryNode(uid); mem > 0 {
		for _, id := range alloc {
			if id == mem {
				if n := content.AbyssTree().Node(mem); n != nil {
					spent -= n.Cost()
				}
				break
			}
		}
	}
	if spent < 0 {
		spent = 0
	}
	return spent
}

// ApplyPrestigeMemory re-allocates the player's chosen prestige-memory node
// after a prestige (AB-168): even if the prestige flow (or a layout reset)
// wiped the web, this one node comes back for free. Idempotent and best-effort.
//
// WIRING NOTE: the prestige handler lives in web_abyss.go (handleAbyssPrestige)
// which is owned elsewhere — the parent must add one line there after the
// prestige UPDATE commits:  ApplyPrestigeMemory(s.bot, uid)
func ApplyPrestigeMemory(b *Bot, uid string) {
	id := b.abyssPrestigeMemoryNode(uid)
	if id <= 0 {
		return
	}
	if _, err := b.DB.Exec(
		"INSERT INTO user_abyss_tree (client_uid, node_id) VALUES ($1,$2) ON CONFLICT DO NOTHING", uid, id); err != nil {
		log.Printf("abyss prestige memory apply failed for %s (node %d): %v", uid, id, err)
	}
}

// handleAbyssTreePrestigeMemorySet chooses (or clears, node_id=0) the node kept
// for free across prestiges. The node must currently be allocated.
func (s *WebServer) handleAbyssTreePrestigeMemorySet(w http.ResponseWriter, r *http.Request, uid string) {
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

	if req.NodeID == 0 {
		if _, err := s.bot.DB.Exec("DELETE FROM app_meta WHERE key=$1", abyssPrestigeMemoryKey(uid)); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "node_id": 0, "msg": "★ Prestige memory cleared."})
		return
	}

	node := content.AbyssTree().Node(req.NodeID)
	if node == nil {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown node"})
		return
	}
	alloc, err := s.bot.loadTreeAllocatedContext(r.Context(), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
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
		writeJSON(w, map[string]any{"ok": false, "error": "prestige memory node must be allocated"})
		return
	}

	if _, err := s.bot.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		abyssPrestigeMemoryKey(uid), strconv.Itoa(req.NodeID)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "node_id": req.NodeID,
		"msg": "★ Prestige memory set: " + node.Name + " stays allocated for free after prestiging."})
}

// ---------------------------------------------------------------------------
// AB-151: three saved loadouts + AB-170: build-code import (shared apply core)
// ---------------------------------------------------------------------------

// treeConnectedSet reports whether every ID in ids is reachable from the root
// walking only nodes inside ids (plus the always-allocated root).
func treeConnectedSet(tree *content.AbyssTreeData, ids map[int]bool) bool {
	reached := map[int]bool{0: true}
	queue := []int{0}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, nb := range tree.Adj[curr] {
			if (ids[nb] || nb == 0) && !reached[nb] {
				reached[nb] = true
				queue = append(queue, nb)
			}
		}
	}
	for id := range ids {
		if !reached[id] {
			return false
		}
	}
	return true
}

// validateTreeLoadout dedupes and validates a saved/imported node set against
// the current layout, depth gates and the player's point budget. Returns the
// canonical cleaned ID list using the same daily discount and prestige-memory
// rules shown by the planner.
func (s *WebServer) validateTreeLoadout(ctx context.Context, uid string, ids []int) ([]int, string) {
	if len(ids) == 0 {
		return nil, "empty build"
	}
	if len(ids) > abyssTreePlanMaxNodes {
		return nil, "build is too large"
	}
	analysis, err := s.currentAbyssTreePlan(ctx, uid, ids)
	if err != nil {
		return nil, "failed to verify build"
	}
	if commitError := abyssTreePlanCommitError(analysis); commitError != "" {
		return nil, commitError
	}
	for _, id := range analysis.IDs {
		if id != treeNodeVictorsTrophy {
			continue
		}
		var earned bool
		if err := s.bot.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM abyss_achievements
			WHERE client_uid=$1 AND code='depth_25')`, uid).Scan(&earned); err != nil {
			return nil, "failed to verify achievements"
		}
		if !earned {
			return nil, "Victor's Trophy requires the Depth 25 achievement"
		}
		break
	}
	return analysis.IDs, ""
}

// applyTreeLoadout atomically refunds everything and allocates the given set,
// charging the normal respec cost (first of the week free, AB-157). Callers
// must already hold the abyss lock.
func (s *WebServer) applyTreeLoadout(ctx context.Context, w http.ResponseWriter, uid string, ids []int) {
	clean, verr := s.validateTreeLoadout(ctx, uid, ids)
	if verr != "" {
		writeJSON(w, map[string]any{"ok": false, "error": verr})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	free, ok := chargeTreeRespec(w, tx, uid)
	if !ok {
		return
	}
	if err := commitAbyssTreeReplacement(ctx, tx, uid, clean); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	msg := fmt.Sprintf("🌳 Build applied — %d nodes allocated.", len(clean))
	if free {
		msg += " (free weekly respec)"
	} else {
		msg += fmt.Sprintf(" (🜲 %d respec)", abyssTreeRespecTokens)
	}
	tb := s.bot.treeBonusFor(uid)
	writeJSON(w, map[string]any{
		"ok": true, "msg": msg,
		"used":   s.bot.treeSpentEx(uid, clean),
		"points": s.bot.treePointsTotal(uid),
		"tokens": s.bot.abyssTokens(uid),
		"stats":  tb.Stats, "pct": tb.Pct,
	})
}

// loadTreeLoadouts reads the 3 saved loadout slots: slot ("1".."3") → node IDs.
func loadTreeLoadouts(stored string) map[string][]int {
	out := map[string][]int{}
	if stored == "" {
		return out
	}
	if err := json.Unmarshal([]byte(stored), &out); err != nil {
		log.Printf("abyss tree loadouts corrupt: %v", err)
		return map[string][]int{}
	}
	return out
}

// handleAbyssTreeLoadoutSave saves the current allocation into slot 1-3 (AB-151).
func (s *WebServer) handleAbyssTreeLoadoutSave(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Slot int    `json:"slot"`
		Name string `json:"name"`
	}
	if err := readJSON(r, &req); err != nil || req.Slot < 1 || req.Slot > 3 {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request (slot 1-3)"})
		return
	}
	alloc, err := s.bot.loadTreeAllocatedContext(r.Context(), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if len(alloc) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "nothing allocated to save"})
		return
	}
	sort.Ints(alloc)

	var stored string
	_ = s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssTreeLoadoutsKey(uid)).Scan(&stored)
	loadouts := loadTreeLoadouts(stored)
	loadouts[strconv.Itoa(req.Slot)] = alloc
	var storedNames string
	_ = s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssTreeLoadoutNamesKey(uid)).Scan(&storedNames)
	names := loadTreeLoadoutNames(storedNames)
	names[strconv.Itoa(req.Slot)] = normalizeAbyssPresetName(req.Name, req.Slot)
	newJson, _ := json.Marshal(loadouts)
	nameJSON, _ := json.Marshal(names)
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(r.Context(), `INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssTreeLoadoutsKey(uid), string(newJson)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.ExecContext(r.Context(), `INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssTreeLoadoutNamesKey(uid), string(nameJSON)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "count": len(alloc),
		"name": names[strconv.Itoa(req.Slot)],
		"msg":  fmt.Sprintf("💾 %s saved (%d nodes).", names[strconv.Itoa(req.Slot)], len(alloc))})
}

// handleAbyssTreeLoadoutApply respecs and re-allocates a saved slot in one
// transaction, charging the normal respec cost (AB-151).
func (s *WebServer) handleAbyssTreeLoadoutApply(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Slot int `json:"slot"`
	}
	if err := readJSON(r, &req); err != nil || req.Slot < 1 || req.Slot > 3 {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request (slot 1-3)"})
		return
	}
	var stored string
	_ = s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssTreeLoadoutsKey(uid)).Scan(&stored)
	ids := loadTreeLoadouts(stored)[strconv.Itoa(req.Slot)]
	if len(ids) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("loadout slot %d is empty", req.Slot)})
		return
	}
	s.applyTreeLoadout(r.Context(), w, uid, ids)
}

// handleAbyssTreeBuildImport applies a decoded build code (AB-170): the client
// decodes the shareable string into a plain node-ID list; the server validates
// and applies it exactly like a loadout.
func (s *WebServer) handleAbyssTreeBuildImport(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		IDs  []int  `json:"ids"`
		Code string `json:"code"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if req.Code != "" {
		code, err := decodeAbyssTreeBuildCode(req.Code)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if code.Layout != content.AbyssTree().TopologyHash() {
			writeJSON(w, map[string]any{"ok": false, "error": "build code targets another tree layout; preview it before committing"})
			return
		}
		req.IDs = code.IDs
	}
	s.applyTreeLoadout(r.Context(), w, uid, req.IDs)
}

// ---------------------------------------------------------------------------
// AB-152: keystone swap without refunding the path
// ---------------------------------------------------------------------------

// handleAbyssTreeSwapKeystone swaps an allocated keystone for another keystone
// in one step — the rest of the path stays allocated (AB-152). The swap is
// refused when it would disconnect the remaining allocation, and honored
// against the timed-keystone cooldown: you cannot swap IN a keystone whose
// activation cooldown is still running.
func (s *WebServer) handleAbyssTreeSwapKeystone(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		FromID int `json:"from_id"`
		ToID   int `json:"to_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	tree := content.AbyssTree()
	from, to := tree.Node(req.FromID), tree.Node(req.ToID)
	if from == nil || to == nil || from.Type != "keystone" || to.Type != "keystone" {
		writeJSON(w, map[string]any{"ok": false, "error": "both nodes must be keystones"})
		return
	}
	if req.FromID == req.ToID {
		writeJSON(w, map[string]any{"ok": false, "error": "already have that keystone"})
		return
	}

	alloc, err := s.bot.loadTreeAllocatedContext(r.Context(), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	newSet := map[int]bool{}
	fromAllocated := false
	for _, id := range alloc {
		if id == req.FromID {
			fromAllocated = true
			continue
		}
		if id == req.ToID {
			writeJSON(w, map[string]any{"ok": false, "error": "target keystone is already allocated"})
			return
		}
		newSet[id] = true
	}
	if !fromAllocated {
		writeJSON(w, map[string]any{"ok": false, "error": "source keystone is not allocated"})
		return
	}
	newSet[req.ToID] = true

	// The point budget must still cover the swap (keystones all cost the same,
	// but the prestige-memory exemption could make sets differ in principle).
	total := s.bot.treePointsTotal(uid)
	resultingAlloc := make([]int, 0, len(newSet))
	for id := range newSet {
		resultingAlloc = append(resultingAlloc, id)
	}
	if spent := s.bot.treeSpentEx(uid, resultingAlloc); spent > total {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("not enough skill points (would need %d of %d)", spent, total)})
		return
	}

	// Depth gate for the incoming keystone mirrors the allocate handler.
	var maxFloor int
	_ = s.bot.DB.QueryRow("SELECT COALESCE(abyss_best_depth, 0) FROM users WHERE client_uid=$1", uid).Scan(&maxFloor)
	if maxFloor < 30 {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("Requires clearing Abyss Floor 30 (your record is %d)", maxFloor)})
		return
	}

	if !treeConnectedSet(tree, newSet) {
		writeJSON(w, map[string]any{"ok": false, "error": "swap would disconnect your path — refund manually instead"})
		return
	}

	// Honor the timed-keystone cooldown on the incoming keystone.
	var cdStr string
	_ = s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssKeystoneCooldownKey(uid, req.ToID)).Scan(&cdStr)
	if cdStr != "" {
		if cdTime, err := time.Parse(time.RFC3339, cdStr); err == nil && time.Now().Before(cdTime) {
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("%s is on cooldown for another %s", to.Name, time.Until(cdTime).Round(time.Second))})
			return
		}
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec("DELETE FROM user_abyss_tree WHERE client_uid=$1 AND node_id=$2", uid, req.FromID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec(
		"INSERT INTO user_abyss_tree (client_uid, node_id) VALUES ($1,$2) ON CONFLICT DO NOTHING", uid, req.ToID); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	tb := s.bot.treeBonusFor(uid)
	writeJSON(w, map[string]any{
		"ok": true, "from_id": req.FromID, "to_id": req.ToID,
		"msg":   fmt.Sprintf("👑 Keystone swapped: %s → %s (path kept).", from.Name, to.Name),
		"stats": tb.Stats, "pct": tb.Pct,
	})
}

// ---------------------------------------------------------------------------
// AB-155: jewel crafting (+ loose jewel storage)
// ---------------------------------------------------------------------------

// abyssJewelMaxTier caps jewel fusion so tiers stay a finite chase.
const abyssJewelMaxTier = 5

// parseJewelCode splits a normal jewel code into base type and tier:
// "ruby" → ("ruby", 1), "ruby_3" → ("ruby", 3). Timeless codes return ok=false
// (they carry their own seed/size/stat format and are never fused).
func parseJewelCode(code string) (string, int, string, bool) {
	if code == "" || strings.HasPrefix(code, "timeless_") {
		return "", 0, "", false
	}
	base, tier := code, 1
	if i := strings.LastIndex(code, "_"); i >= 0 {
		t, err := strconv.Atoi(code[i+1:])
		if err != nil {
			return "", 0, "", false
		}
		base, tier = code[:i], t
	}
	switch base {
	case "ruby", "sapphire", "topaz":
	default:
		return "", 0, "", false
	}
	if tier < 1 || tier > abyssJewelMaxTier {
		return "", 0, "", false
	}
	return base, tier, jewelTierCode(base, tier), true
}

// jewelTierCode builds the code for base at tier ("ruby", "ruby_2", …).
func jewelTierCode(base string, tier int) string {
	if tier <= 1 {
		return base
	}
	return fmt.Sprintf("%s_%d", base, tier)
}

// loadTreeJewels reads the loose-jewel pouch (code → count). It fails closed:
// a corrupt pouch is an error, so callers never silently drop stored jewels.
func loadTreeJewels(stored string) (map[string]int, error) {
	out := map[string]int{}
	if stored == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(stored), &out); err != nil {
		return nil, err
	}
	canonical := make(map[string]int, len(out))
	for code, count := range out {
		if _, _, key, ok := parseJewelCode(code); ok {
			canonical[key] += count
		} else {
			canonical[code] += count
		}
	}
	return canonical, nil
}

// abyssJewelsMap loads the loose-jewel pouch for rendering/bonus paths; a
// corrupt pouch degrades to empty with a log line (read-only callers).
func (b *Bot) abyssJewelsMap(uid string) map[string]int {
	var stored string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssJewelsKey(uid)).Scan(&stored)
	m, err := loadTreeJewels(stored)
	if err != nil {
		log.Printf("abyss jewel pouch corrupt for %s: %v", uid, err)
		return map[string]int{}
	}
	return m
}

// handleAbyssTreeJewelFuse fuses 3 identical jewels (loose and/or socketed)
// into one jewel of the next tier (AB-155). Loose copies are consumed first,
// then socketed ones (cleared from their sockets in ascending node order).
func (s *WebServer) handleAbyssTreeJewelFuse(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Jewel string `json:"jewel"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	base, tier, canonical, ok := parseJewelCode(req.Jewel)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid or non-fusable jewel (timeless jewels cannot be fused)"})
		return
	}
	req.Jewel = canonical
	if tier >= abyssJewelMaxTier {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("tier %d is the maximum", abyssJewelMaxTier)})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Loose pouch.
	var jewelsJson string
	_ = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", abyssJewelsKey(uid)).Scan(&jewelsJson)
	loose, err := loadTreeJewels(jewelsJson)
	if err != nil {
		log.Printf("abyss jewel pouch corrupt for %s: %v", uid, err)
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	// Socketed jewels.
	var socketsJson string
	_ = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", abyssSocketKey(uid)).Scan(&socketsJson)
	socketMap := map[int]string{}
	if socketsJson != "" {
		if err := json.Unmarshal([]byte(socketsJson), &socketMap); err != nil {
			log.Printf("abyss socket map corrupt for %s: %v", uid, err)
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	for nodeID, code := range socketMap {
		if _, _, key, ok := parseJewelCode(code); ok {
			socketMap[nodeID] = key
		}
	}

	avail := loose[req.Jewel]
	var socketedAt []int
	for nid, code := range socketMap {
		if code == req.Jewel {
			avail++
			socketedAt = append(socketedAt, nid)
		}
	}
	if avail < 3 {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("need 3× %s to fuse (you have %d)", req.Jewel, avail)})
		return
	}

	need := 3
	if loose[req.Jewel] > 0 {
		take := loose[req.Jewel]
		if take > need {
			take = need
		}
		loose[req.Jewel] -= take
		if loose[req.Jewel] == 0 {
			delete(loose, req.Jewel)
		}
		need -= take
	}
	if need > 0 {
		sort.Ints(socketedAt)
		for _, nid := range socketedAt {
			if need == 0 {
				break
			}
			delete(socketMap, nid)
			need--
		}
	}

	result := jewelTierCode(base, tier+1)
	loose[result]++

	newJewels, _ := json.Marshal(loose)
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssJewelsKey(uid), string(newJewels)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	newSockets, _ := json.Marshal(socketMap)
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssSocketKey(uid), string(newSockets)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	tb := s.bot.treeBonusFor(uid)
	writeJSON(w, map[string]any{
		"ok":      true,
		"msg":     fmt.Sprintf("💎 Fused 3× %s → 1× %s (tier %d)! It is in your jewel pouch.", req.Jewel, result, tier+1),
		"sockets": socketMap,
		"jewels":  loose,
		"stats":   tb.Stats, "pct": tb.Pct,
	})
}

// ---------------------------------------------------------------------------
// AB-159: mastery shard — single-branch refund consuming a boss-drop counter
// ---------------------------------------------------------------------------

// abyssMasteryShards reads the mastery-shard counter (boss drop hook lives in
// the loot pipeline — outside these files; see the note to the parent).
func (b *Bot) abyssMasteryShards(uid string) int {
	var v string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssMasteryShardKey(uid)).Scan(&v)
	n, _ := strconv.Atoi(v)
	if n < 0 {
		n = 0
	}
	return n
}

func (b *Bot) grantAbyssMasteryShard(uid string) bool {
	_, err := b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, '1')
		ON CONFLICT (key) DO UPDATE SET value=(COALESCE(NULLIF(app_meta.value, ''), '0')::int + 1)::text`,
		abyssMasteryShardKey(uid))
	if err != nil {
		log.Printf("abyss mastery shard grant failed for %s: %v", uid, err)
		return false
	}
	return true
}

// handleAbyssTreeBranchRefund refunds a node plus its dependents (same cascade
// as the gold refund) for free, consuming one mastery shard (AB-159).
func (s *WebServer) handleAbyssTreeBranchRefund(w http.ResponseWriter, r *http.Request, uid string) {
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
	alloc, err := s.bot.loadTreeAllocatedContext(r.Context(), uid)
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
	refundIDs := treeRefundSet(tree, alloc, req.NodeID)

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	key := abyssMasteryShardKey(uid)
	var shardStr string
	_ = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", key).Scan(&shardStr)
	shards, _ := strconv.Atoi(shardStr)
	if shards < 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "no mastery shards — bosses can drop them"})
		return
	}

	for _, id := range refundIDs {
		if _, err := tx.Exec("DELETE FROM user_abyss_tree WHERE client_uid=$1 AND node_id=$2", uid, id); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, strconv.Itoa(shards-1)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	refunded := make(map[int]bool, len(refundIDs))
	for _, id := range refundIDs {
		refunded[id] = true
	}
	var remaining []int
	for _, id := range alloc {
		if !refunded[id] {
			remaining = append(remaining, id)
		}
	}

	tb := s.bot.treeBonusFor(uid)
	writeJSON(w, map[string]any{
		"ok": true, "node_id": req.NodeID,
		"used":     s.bot.treeSpentEx(uid, remaining),
		"points":   s.bot.treePointsTotal(uid),
		"refunded": refundIDs,
		"shards":   shards - 1,
		"msg":      fmt.Sprintf("🔮 Mastery shard consumed — refunded %d node(s) down to %s for free.", len(refundIDs), node.Name),
		"stats":    tb.Stats, "pct": tb.Pct,
	})
}

// ---------------------------------------------------------------------------
// AB-172: free undo of the last allocation within 60 seconds
// ---------------------------------------------------------------------------

// abyssUndoWindow is how long after allocating a node it can be undone for free.
const abyssUndoWindow = 60 * time.Second

type abyssLastAllocRecord struct {
	Node int    `json:"node"`
	At   string `json:"at"`
}

// recordTreeAllocationContext remembers the player's latest allocation for the
// free 60-second undo (AB-172). Best-effort: a failure only loses the undo option.
func (b *Bot) recordTreeAllocationContext(ctx context.Context, uid string, nodeID int) time.Time {
	now := time.Now().UTC()
	rec, _ := json.Marshal(abyssLastAllocRecord{Node: nodeID, At: now.Format(time.RFC3339)})
	if _, err := b.DB.ExecContext(ctx, `INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssLastAllocKey(uid), string(rec)); err != nil {
		log.Printf("abyss undo marker write failed for %s: %v", uid, err)
	}
	return now.Add(abyssUndoWindow)
}

// handleAbyssTreeUndo refunds the most recent allocation for free when it
// happened within abyssUndoWindow (AB-172).
func (s *WebServer) handleAbyssTreeUndo(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var stored string
	_ = s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssLastAllocKey(uid)).Scan(&stored)
	if stored == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "nothing to undo"})
		return
	}
	var rec abyssLastAllocRecord
	if err := json.Unmarshal([]byte(stored), &rec); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "nothing to undo"})
		return
	}
	at, err := time.Parse(time.RFC3339, rec.At)
	if err != nil || time.Since(at) > abyssUndoWindow {
		writeJSON(w, map[string]any{"ok": false, "error": "undo window (60s) has passed"})
		return
	}
	tree := content.AbyssTree()
	node := tree.Node(rec.Node)
	if node == nil {
		writeJSON(w, map[string]any{"ok": false, "error": "node no longer exists"})
		return
	}
	alloc, err := s.bot.loadTreeAllocatedContext(r.Context(), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	found := false
	for _, id := range alloc {
		if id == rec.Node {
			found = true
			break
		}
	}
	if !found {
		writeJSON(w, map[string]any{"ok": false, "error": "that node is no longer allocated"})
		return
	}
	refundIDs := treeRefundSet(tree, alloc, rec.Node)

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
	if _, err := tx.Exec("DELETE FROM app_meta WHERE key=$1", abyssLastAllocKey(uid)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	refunded := make(map[int]bool, len(refundIDs))
	for _, id := range refundIDs {
		refunded[id] = true
	}
	var remaining []int
	for _, id := range alloc {
		if !refunded[id] {
			remaining = append(remaining, id)
		}
	}
	tb := s.bot.treeBonusFor(uid)
	writeJSON(w, map[string]any{
		"ok": true, "node_id": rec.Node,
		"used":     s.bot.treeSpentEx(uid, remaining),
		"points":   s.bot.treePointsTotal(uid),
		"refunded": refundIDs,
		"msg":      "↩️ Undid: " + node.Name + " (free)",
		"stats":    tb.Stats, "pct": tb.Pct,
	})
}
