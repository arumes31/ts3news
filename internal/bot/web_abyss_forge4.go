package bot

// Forge round 5 (AB-101..125, docs/ABYSS_IMPROVEMENTS_300.md group E): batch
// temper with an insurance guard, the forge queue, bulk gem upgrade, rune
// scraping, un-attune, masterwork transfer, reforge lock (with the Eternal
// double-reforge privilege), bulk rebalance, brand/special/imbue removal and
// reroll, guided awaken, polish-all, Repair Kit II with crafting crit, socket
// relocation, fusion preview, blessed celestial fusion, recipe favorites,
// material conversion, the second daily undo purchase and forge mastery.
// Same forge shape as rounds 3-4 (web_abyss_forge2.go / web_abyss_forge3.go):
// per-uid lock, {inv_id|slot} body, one transaction, undo snapshot, guarded
// cost deduction, item_data rewrite, forge history, refreshed balances.
//
// Deliberate deviations, forced by file ownership:
//   - AB-108 (perfect corruption) is NOT here: the corrupt handler lives in
//     web_abyss_forge3.go (handleAbyssCorrupt) and needs a hidden 5% roll there.
//   - AB-117 (forge mastery) discounts only the handlers in this file via
//     forge4GoldCost; the forge2/forge3 handlers can't see it.
//   - AB-125 sells the flag here; handleAbyssForgeUndo honors it and records
//     the second use independently from the base daily undo date.

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ts3news/internal/content"
)

// ---- Forge mastery (AB-117) ---------------------------------------------------

// forge4MasteryKey is the app_meta key counting a player's forge4 actions.
func forge4MasteryKey(uid string) string { return "abyss_forge_mastery_" + uid }

// forge4MasteryCount reads the lifetime forge4 action count.
func (b *Bot) forge4MasteryCount(uid string) int {
	var v string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", forge4MasteryKey(uid)).Scan(&v)
	n, _ := strconv.Atoi(v)
	return n
}

// forge4MasteryDiscount is +1% per 50 recorded actions, capped at 5%.
func (b *Bot) forge4MasteryDiscount(uid string) int {
	d := b.forge4MasteryCount(uid) / 50
	if d > 5 {
		d = 5
	}
	return d
}

// forge4MasteryBump records one forge4 action inside the caller's transaction.
func forge4MasteryBump(tx *sql.Tx, uid string) {
	_, _ = tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, '1')
	                ON CONFLICT (key) DO UPDATE SET value = (COALESCE(NULLIF(app_meta.value, ''), '0')::int + 1)::text`,
		forge4MasteryKey(uid))
}

// forge4GoldCost wraps forgeGoldCost with the forge-mastery discount (AB-117).
// Only forge4 handlers see this discount; forge2/forge3 price via forgeGoldCost.
func (s *WebServer) forge4GoldCost(uid string, base int64, r content.Rarity) int64 {
	c := s.bot.forgeGoldCost(uid, base, r)
	c -= c * int64(s.bot.forge4MasteryDiscount(uid)) / 100
	if c < 1 {
		c = 1
	}
	return c
}

// forge4MasteryInfo is the response fragment describing the player's mastery.
func (b *Bot) forge4MasteryInfo(uid string) map[string]any {
	return map[string]any{"actions": b.forge4MasteryCount(uid), "discount_pct": b.forge4MasteryDiscount(uid)}
}

// forge4ItemKey identifies an item specifier as a stable string (used by the
// temper guard and the guided-awaken pending roll).
func forge4ItemKey(invID int64, slot string) string {
	if invID > 0 {
		return fmt.Sprintf("inv:%d", invID)
	}
	return "slot:" + slot
}

// ---- Batch temper (AB-101) ------------------------------------------------------

// handleAbyssBatchTemper tempers toward a target level in a single request: all
// attempts resolve server-side in one transaction, stopping at the target, after
// 2 failures, or when the gold runs out. One undo snapshot covers the whole batch.
func (s *WebServer) handleAbyssBatchTemper(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		Target         int  `json:"target"`
		AutoProtection bool `json:"auto_protection"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if g.Unidentified {
		writeJSON(w, map[string]any{"ok": false, "error": "identify the item first"})
		return
	}
	target := req.Target
	if target > temperMax {
		target = temperMax
	}
	if target <= g.Temper {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("target must be above the current temper (+%d)", g.Temper)})
		return
	}

	var failStacks int
	_ = tx.QueryRow("SELECT temper_fail_stacks FROM users WHERE client_uid=$1", uid).Scan(&failStacks)
	var gold int64
	_ = tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)

	// An active insurance stone (AB-102) on this item turns one fail into a success.
	var guard string
	_ = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1", forge4TemperGuardKey(uid)).Scan(&guard)
	guardActive := guard != "" && guard == forge4ItemKey(req.InvID, req.Slot)
	autoGuard := false
	if req.AutoProtection && !guardActive {
		guardActive, autoGuard = true, true
	}

	var spent int64
	attempts, successes, fails := 0, 0, 0
	guardUsed := false
	stopped := "target reached"
	for g.Temper < target && fails < 2 {
		cost := s.forge4GoldCost(uid, int64(400*(g.Temper+1)), g.Rarity)
		if gold < cost {
			stopped = "out of gold"
			break
		}
		gold -= cost
		spent += cost
		attempts++
		chance := temperChance(g.Temper, failStacks)
		success := rand.Float64() < chance // #nosec G404 -- non-cryptographic forge roll
		if !success && guardActive {
			if autoGuard && !spendMaterials(tx, uid, map[string]int{"core": 2}) {
				writeJSON(w, map[string]any{"ok": false, "error": "automatic temper protection needs 2 Umbral Cores"})
				return
			}
			success, guardActive, guardUsed = true, false, true
		}
		if success {
			g.Temper++
			g.Stats = g.Stats.Scaled(1.02)
			failStacks = 0
			successes++
		} else {
			failStacks++
			fails++
		}
	}
	if fails >= 2 {
		stopped = "stopped after 2 failures"
	}
	if attempts == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold for a single attempt"})
		return
	}

	if successes > 0 {
		s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "batch temper")
		if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
			return
		}
	}
	res, err := tx.Exec("UPDATE users SET gold = gold - $1, temper_fail_stacks = $3 WHERE client_uid=$2 AND gold >= $1", spent, uid, failStacks)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	if guardUsed {
		if _, err := tx.Exec("DELETE FROM app_meta WHERE key=$1", forge4TemperGuardKey(uid)); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "batch temper", fmt.Sprintf("%s → +%d (%d attempts)", g.Name, g.Temper, attempts), fmt.Sprintf("%dg", spent)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "temper": g.Temper, "attempts": attempts, "successes": successes,
		"fails": fails, "spent": spent, "stopped": stopped, "guard_used": guardUsed, "auto_guard": autoGuard,
		"gold": s.bot.abyssGold(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("⚒️ Batch temper on %s: %d/%d succeeded → +%d (%dg spent, %s).", g.Name, successes, attempts, g.Temper, spent, stopped)})
}

// ---- Temper insurance stone (AB-102) --------------------------------------------

// forge4TemperGuardKey is the app_meta key holding the guarded item's key.
func forge4TemperGuardKey(uid string) string { return "abyss_temper_guard_" + uid }

// handleAbyssTemperGuard sets a one-shot temper insurance stone (2 Umbral Cores):
// the next failed temper on the chosen item succeeds instead (honored by the
// batch temper above; the single temper handler lives in web_abyss_features.go).
// The guard lives in app_meta because Gear fields require content/ changes.
func (s *WebServer) handleAbyssTemperGuard(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req forgeItemReq
	if !readForgeItemReq(w, r, &req) {
		return
	}

	tx, g, _, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	key := forge4ItemKey(req.InvID, req.Slot)
	var existing string
	_ = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1", forge4TemperGuardKey(uid)).Scan(&existing)
	if existing == key {
		writeJSON(w, map[string]any{"ok": false, "error": "an insurance stone is already active (" + existing + ") — it lasts until it absorbs a fail"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"core": 2}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Umbral Cores (need 2)"})
		return
	}
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
	                      ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, forge4TemperGuardKey(uid), key); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "temper guard", g.Name, "2🟣") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "guard": key, "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("🛡️ An insurance stone now guards %s — its next failed temper succeeds instead.", g.Name)})
}

// ---- Forge queue (AB-103) ---------------------------------------------------------

// handleAbyssForgeQueue executes up to 3 forge actions on ONE item with a single
// confirm. All steps are simulated first, then charged and saved atomically; any
// failing step aborts the whole queue (nothing is charged or changed).
// Supported actions: polish, reinforce, sharpen, masterwork, temper.
func (s *WebServer) handleAbyssForgeQueue(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		Actions         []string         `json:"actions"`
		FailurePolicies []string         `json:"failure_policies"`
		GoldCap         int64            `json:"gold_cap"`
		OperationCaps   map[string]int64 `json:"operation_caps"`
		DryRun          bool             `json:"dry_run"`
		Paused          bool             `json:"paused"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if len(req.Actions) == 0 || len(req.Actions) > 3 {
		writeJSON(w, map[string]any{"ok": false, "error": "queue 1 to 3 actions"})
		return
	}
	if req.GoldCap < 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "gold cap cannot be negative"})
		return
	}
	if req.Paused {
		writeJSON(w, map[string]any{"ok": true, "paused": true, "steps": []string{}, "spent": 0,
			"msg": "📋 Forge queue remains paused; no costs or item changes were applied."})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	var failStacks int
	_ = tx.QueryRow("SELECT temper_fail_stacks FROM users WHERE client_uid=$1", uid).Scan(&failStacks)

	var totalGold int64
	mats := map[string]int{}
	polishes := 0
	temperUsed := false
	var steps []string
queueLoop:
	for i, action := range req.Actions {
		goldBefore := totalGold
		stepErr := func(msg string) bool {
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("step %d (%s): %s", i+1, action, msg)})
			return false
		}
		switch action {
		case "polish":
			baseMax := g.MaxDurability
			if cat, found := content.GetGearByID(g.ID); found {
				baseMax = cat.MaxDurability
			}
			if g.MaxDurability >= baseMax+100 {
				_ = stepErr("already polished to the limit")
				return
			}
			totalGold += s.forge4GoldCost(uid, 150, g.Rarity)
			g.MaxDurability += 10
			if g.MaxDurability > baseMax+100 {
				g.MaxDurability = baseMax + 100
			}
			polishes++
			steps = append(steps, "polish")
		case "reinforce":
			if isWeaponSlot(g.Slot) {
				_ = stepErr("weapons can't be reinforced")
				return
			}
			if g.Reinforced >= reinforceMax {
				_ = stepErr("already reinforced to the maximum (10)")
				return
			}
			totalGold += s.forge4GoldCost(uid, 100, g.Rarity)
			mats["dust"] += 2
			inc := g.Stats.DEF * 2 / 100
			if inc < 1 {
				inc = 1
			}
			g.Stats.DEF += inc
			g.Reinforced++
			steps = append(steps, fmt.Sprintf("reinforce %d/10", g.Reinforced))
		case "sharpen":
			if !isWeaponSlot(g.Slot) {
				_ = stepErr("only weapons can be sharpened")
				return
			}
			if g.Sharpened >= reinforceMax {
				_ = stepErr("already sharpened to the maximum (10)")
				return
			}
			totalGold += s.forge4GoldCost(uid, 100, g.Rarity)
			mats["dust"] += 2
			inc := g.Stats.STR * 2 / 100
			if inc < 1 {
				inc = 1
			}
			g.Stats.STR += inc
			g.Sharpened++
			steps = append(steps, fmt.Sprintf("sharpen %d/10", g.Sharpened))
		case "masterwork":
			if g.Quality >= masterworkMax {
				_ = stepErr("already a masterwork (quality 5)")
				return
			}
			mats["dust"] += (g.Quality + 1) * 5
			mats["shard"] += (g.Quality + 1) * 2
			if g.Quality >= 2 {
				mats["core"] += g.Quality - 1
			}
			g.Stats = g.Stats.Scaled(1.03)
			g.Quality++
			steps = append(steps, "masterwork "+qualityNames[g.Quality])
		case "temper":
			if g.Unidentified {
				_ = stepErr("identify the item first")
				return
			}
			if g.Temper >= temperMax {
				_ = stepErr("already tempered to the maximum (+15)")
				return
			}
			totalGold += s.forge4GoldCost(uid, int64(400*(g.Temper+1)), g.Rarity)
			temperUsed = true
			if rand.Float64() < temperChance(g.Temper, failStacks) { // #nosec G404 -- non-cryptographic forge roll
				g.Temper++
				g.Stats = g.Stats.Scaled(1.02)
				failStacks = 0
				steps = append(steps, fmt.Sprintf("temper +%d (success)", g.Temper))
			} else {
				failStacks++
				steps = append(steps, "temper (failed)")
				if i < len(req.FailurePolicies) && req.FailurePolicies[i] == "stop on failure" {
					break queueLoop
				}
			}
		default:
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("step %d: unknown action %q (polish, reinforce, sharpen, masterwork, temper)", i+1, action)})
			return
		}
		if cap := req.OperationCaps[action]; cap > 0 && totalGold-goldBefore > cap {
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("step %d (%s) exceeds its %dg spending limit", i+1, action, cap)})
			return
		}
		if req.GoldCap > 0 && totalGold > req.GoldCap {
			writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("queue exceeds its %dg spending cap", req.GoldCap)})
			return
		}
	}
	if req.DryRun {
		writeJSON(w, map[string]any{"ok": true, "dry_run": true, "steps": steps, "spent": totalGold,
			"materials": mats, "msg": fmt.Sprintf("📋 Dry run complete: %d steps, %dg, no changes applied.", len(steps), totalGold)})
		return
	}

	if totalGold > 0 && !deductGold(w, tx, uid, totalGold) {
		return
	}
	if !spendMaterials(tx, uid, mats) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough materials"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "forge queue")
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if req.Slot != "" && polishes > 0 {
		if _, err := tx.Exec("UPDATE user_gear SET durability = LEAST(durability + $1, $2) WHERE slot=$3 AND client_uid=$4", 10*polishes, g.MaxDurability, req.Slot, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if temperUsed {
		if _, err := tx.Exec("UPDATE users SET temper_fail_stacks=$2 WHERE client_uid=$1", uid, failStacks); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	for range req.Actions {
		forge4MasteryBump(tx, uid)
	}
	if !s.finishForge(w, tx, uid, "forge queue", g.Name+": "+strings.Join(steps, ", "), fmt.Sprintf("%dg", totalGold)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "steps": steps, "spent": totalGold,
		"gold": s.bot.abyssGold(uid), "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("📋 Forge queue complete on %s: %s.", g.Name, strings.Join(steps, " → "))})
}

// ---- Bulk gem upgrade (AB-104) ----------------------------------------------------

// handleAbyssGemUpgradeAll upgrades every tier-I gem socketed in the item to
// tier II in one action, charging the summed single-upgrade cost (gold + 5
// Void Shards each, matching handleAbyssUpgradeGem).
func (s *WebServer) handleAbyssGemUpgradeAll(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		StopAtTier int `json:"stop_at_tier"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if req.StopAtTier == 0 {
		req.StopAtTier = 2
	}
	if req.StopAtTier < 2 || req.StopAtTier > 3 {
		writeJSON(w, map[string]any{"ok": false, "error": "stop_at_tier must be 2 or 3"})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	var upgradeIndexes []int
	steps := 0
	for i, gem := range g.Gemstones {
		base, tier := parseGem(gem)
		if _, valid := gemBaseStats(base); valid && tier < req.StopAtTier {
			upgradeIndexes = append(upgradeIndexes, i)
			steps += req.StopAtTier - tier
		}
	}
	if len(upgradeIndexes) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "no gems below the requested stop tier"})
		return
	}
	goldCost := int64(0)
	materialCost := map[string]int{}
	for _, index := range upgradeIndexes {
		_, tier := parseGem(g.Gemstones[index])
		for tier < req.StopAtTier {
			if tier == 1 {
				goldCost += s.forge4GoldCost(uid, 200, g.Rarity)
				materialCost["shard"] += 5
			} else {
				goldCost += s.forge4GoldCost(uid, 500, g.Rarity)
				materialCost["core"] += 2
			}
			tier++
		}
	}
	if !deductGold(w, tx, uid, goldCost) {
		return
	}
	if !spendMaterials(tx, uid, materialCost) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough materials for every planned gem upgrade"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "bulk gem upgrade")
	var names []string
	for _, i := range upgradeIndexes {
		base, _ := parseGem(g.Gemstones[i])
		baseStats, _ := gemBaseStats(base)
		_, tier := parseGem(g.Gemstones[i])
		for tier < req.StopAtTier {
			g.Stats = g.Stats.Add(baseStats.Scaled(float64(tier)))
			tier++
		}
		g.Gemstones[i] = gemName(base, req.StopAtTier)
		names = append(names, g.Gemstones[i])
	}
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "bulk gem upgrade", fmt.Sprintf("%s ×%d steps", g.Name, steps), fmt.Sprintf("%dg", goldCost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "upgraded": len(upgradeIndexes), "steps": steps, "stop_at_tier": req.StopAtTier, "gems": names,
		"gold": s.bot.abyssGold(uid), "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("💎 Upgraded %d gems on %s to tier %d (%s).", len(upgradeIndexes), g.Name, req.StopAtTier, strings.Join(names, ", "))})
}

// ---- Rune scraping (AB-105) --------------------------------------------------------

// handleAbyssScrapeRune removes an etched rune, recovering half its etch price
// as Abyssal Dust (75 dust for a first-time 150g rune, 25 for a known 50g one).
// A prismatic rune's baked +5% stays baked — only the flag is stripped.
func (s *WebServer) handleAbyssScrapeRune(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req forgeItemReq
	if !readForgeItemReq(w, r, &req) {
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if g.Rune == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "no rune etched on this item"})
		return
	}
	var known bool
	_ = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM user_runes WHERE client_uid=$1 AND rune=$2)", uid, g.Rune).Scan(&known)
	dust := 75 // half of the 150g first-etch price
	if known {
		dust = 25 // half of the 50g re-etch price
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "rune scrape")
	runeName := g.Rune
	g.Rune = ""
	g.Element = ""
	prismaticNote := ""
	if g.Prismatic {
		g.Prismatic = false
		prismaticNote = " The prismatic +5% stays baked into the stats."
	}
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if err := grantMaterialQ(tx, uid, "dust", dust); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "rune scrape", fmt.Sprintf("%s — %s rune", g.Name, runeName), fmt.Sprintf("+%d🌫️", dust)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "dust": dust, "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("🧽 Scraped the %s rune off %s, recovering %d Abyssal Dust.%s", runeName, g.Name, dust, prismaticNote)})
}

// ---- Un-attune (AB-106) --------------------------------------------------------------

// handleAbyssUnattune strips the Attuned flag for 50 tokens so the item can be
// fused, salvaged, dismantled or auctioned again. The attunement's +5% stat
// multiplier is reversed as closely as integer stat rounding allows.
func (s *WebServer) handleAbyssUnattune(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req forgeItemReq
	if !readForgeItemReq(w, r, &req) {
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if !g.Attuned {
		writeJSON(w, map[string]any{"ok": false, "error": "item is not attuned"})
		return
	}
	if !deductTokens(w, tx, uid, 50) {
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "un-attune")
	g.Attuned = false
	g.Stats = g.Stats.Scaled(1 / 1.05)
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "un-attune", g.Name, "🜲50") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "tokens": s.bot.abyssTokens(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("⛓️‍💥 %s is no longer attuned and loses its +5%% attunement stats. It can be fused, salvaged, dismantled and auctioned again.", g.Name)})
}

// ---- Masterwork transfer (AB-107) ------------------------------------------------------

// handleAbyssMasterworkTransfer moves an item's masterwork Quality onto another
// same-slot item at 80% efficiency (floor), zeroing the source's Quality. The
// source keeps its already-baked stats (they can't be unbaked); the target gets
// x1.03 per transferred level baked in. Costs 2 Umbral Cores + gold.
func (s *WebServer) handleAbyssMasterworkTransfer(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		InvID2 int64  `json:"inv_id2"`
		Slot2  string `json:"slot2"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if req.InvID2 <= 0 && req.Slot2 == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "pick a target item"})
		return
	}
	if (req.InvID > 0 && req.InvID == req.InvID2) || (req.Slot != "" && req.Slot == req.Slot2) {
		writeJSON(w, map[string]any{"ok": false, "error": "pick two different items"})
		return
	}

	tx, g1, raw1, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	g2, raw2, ok := loadForgeItem(tx, s.bot, uid, req.InvID2, req.Slot2)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "target item not found"})
		return
	}
	if g1.Slot != g2.Slot {
		writeJSON(w, map[string]any{"ok": false, "error": "both items must share the same slot"})
		return
	}
	if g1.Quality <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "the source item has no masterwork quality to transfer"})
		return
	}
	moved := g1.Quality * 4 / 5
	if room := masterworkMax - g2.Quality; moved > room {
		moved = room
	}
	if moved < 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "nothing would transfer (80% efficiency, target nearly maxed)"})
		return
	}
	cost := s.forge4GoldCost(uid, 300, g2.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"core": 2}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Umbral Cores (need 2)"})
		return
	}
	s.bot.snapshotForgeUndoPair(tx, uid, req.InvID, req.Slot, raw1, req.InvID2, req.Slot2, raw2, "masterwork transfer")
	fromQ := g1.Quality
	g1.Quality = 0 // source stats stay baked; only the Quality rating moves
	for i := 0; i < moved; i++ {
		g2.Stats = g2.Stats.Scaled(1.03)
	}
	g2.Quality += moved
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g1) {
		return
	}
	if !saveForgeItem(w, tx, uid, req.InvID2, req.Slot2, g2) {
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "masterwork transfer", fmt.Sprintf("%s → %s (%s)", g1.Name, g2.Name, qualityNames[g2.Quality]), fmt.Sprintf("%dg 2🟣", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "moved": moved, "quality": g2.Quality,
		"gold": s.bot.abyssGold(uid), "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("🏅 Transferred masterwork: %s quality %d → 0, %s is now %s (+%d levels at 80%% efficiency).", g1.Name, fromQ, g2.Name, qualityNames[g2.Quality], moved)})
}

// ---- Reforge lock (AB-109) + Eternal double-reforge (AB-121) ------------------------------

// forge4ReforgeUsesKey is the app_meta key tracking locked-reforge uses as
// "YYYY-MM-DD:count" (UTC day).
func forge4ReforgeUsesKey(uid string) string { return "abyss_reforge_uses_" + uid }

// handleAbyssReforgeLock is reforge with one stat line excluded from the ±10%
// reroll, at double price (600g base). Limited to once per day — Eternal items
// may be locked-reforged twice per day (AB-121). Note: the plain reforge in
// web_abyss_forge2.go has no daily counter; this counter only gates this endpoint.
func (s *WebServer) handleAbyssReforgeLock(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		LockStat string `json:"lock_stat"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	lockStat := strings.ToUpper(strings.TrimSpace(req.LockStat))
	if !rebalanceStats[lockStat] {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid lock_stat — pick HP, STR, DEF, SPD, LCK, INT, STA, CRT, DGE or MNA"})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if g.Rarity < content.RarityRare {
		writeJSON(w, map[string]any{"ok": false, "error": "only Rare or better gear can be reforged"})
		return
	}
	maxUses := 1
	if g.Rarity >= content.RarityEternal {
		maxUses = 2
	}
	today := time.Now().UTC().Format("2006-01-02")
	uses := 0
	var v string
	if err := tx.QueryRow("SELECT value FROM app_meta WHERE key=$1", forge4ReforgeUsesKey(uid)).Scan(&v); err == nil {
		if d, c, found := strings.Cut(v, ":"); found && d == today {
			uses, _ = strconv.Atoi(c)
		}
	}
	if uses >= maxUses {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("locked reforge limit reached for today (%d/%d)", uses, maxUses)})
		return
	}
	cost := s.forge4GoldCost(uid, 600, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "reforge lock")
	crBefore := g.CombatRating()
	// #nosec G404 -- non-cryptographic forge rolls
	reroll := func(v int) int {
		if v == 0 {
			return 0
		}
		return int(float64(v) * (0.90 + rand.Float64()*0.20))
	}
	for _, code := range []string{"HP", "STR", "DEF", "SPD", "LCK", "INT", "STA", "CRT", "DGE", "MNA"} {
		if code == lockStat {
			continue
		}
		p := gearStatRef(&g.Stats, code)
		*p = reroll(*p)
	}
	crAfter := g.CombatRating()
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	uses++
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
	                      ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`,
		forge4ReforgeUsesKey(uid), fmt.Sprintf("%s:%d", today, uses)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "reforge lock", fmt.Sprintf("%s (locked %s)", g.Name, lockStat), fmt.Sprintf("%dg", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "uses_left": maxUses - uses,
		"gold": s.bot.abyssGold(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("🎲 Reforged %s with %s locked — CR %.0f → %.0f.", g.Name, lockStat, crBefore, crAfter)})
}

// ---- Bulk rebalance (AB-110) -----------------------------------------------------------

// handleAbyssRebalanceAll shifts 10% of every (positive) combat stat into one
// chosen stat at once (500g base). Only positive stats contribute.
func (s *WebServer) handleAbyssRebalanceAll(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		To string `json:"to"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	to := strings.ToUpper(strings.TrimSpace(req.To))
	if !rebalanceStats[to] {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid target stat — pick HP, STR, DEF, SPD, LCK, INT, STA, CRT, DGE or MNA"})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	total := 0
	for _, code := range []string{"HP", "STR", "DEF", "SPD", "LCK", "INT", "STA", "CRT", "DGE", "MNA"} {
		if code == to {
			continue
		}
		if v := *gearStatRef(&g.Stats, code); v > 0 {
			total += v / 10
		}
	}
	if total < 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough stats to rebalance"})
		return
	}
	cost := s.forge4GoldCost(uid, 500, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "bulk rebalance")
	for _, code := range []string{"HP", "STR", "DEF", "SPD", "LCK", "INT", "STA", "CRT", "DGE", "MNA"} {
		if code == to {
			continue
		}
		p := gearStatRef(&g.Stats, code)
		if *p > 0 {
			*p -= *p / 10
		}
	}
	*gearStatRef(&g.Stats, to) += total
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "bulk rebalance", fmt.Sprintf("%s → %s", g.Name, to), fmt.Sprintf("%dg", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "moved": total, "gold": s.bot.abyssGold(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("⚖️ Rebalanced %s: 10%% of every stat (+%d total) flows into %s.", g.Name, total, to)})
}

// ---- Brand removal (AB-111) ---------------------------------------------------------------

// handleAbyssUnbrand strips a set brand (2 Umbral Cores); the item returns to
// the legacy pool (EffectiveSetID falls back for old ABYSS_ gear).
func (s *WebServer) handleAbyssUnbrand(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req forgeItemReq
	if !readForgeItemReq(w, r, &req) {
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if g.SetID == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "item carries no set brand"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"core": 2}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Umbral Cores (need 2)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "brand removal")
	oldSet := g.SetID
	g.SetID = ""
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "brand removal", fmt.Sprintf("%s — %s", g.Name, oldSet), "2🟣") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("🏷️ The %s brand is stripped from %s — it returns to the legacy pool.", oldSet, g.Name)})
}

// ---- Special reroll (AB-112) ------------------------------------------------------------------

// handleAbyssSpecialReroll rerolls an item's Special outright (6 Umbral Cores),
// drawing from the awaken pool excluding the current Special.
func (s *WebServer) handleAbyssSpecialReroll(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		ExcludedEffects []string `json:"excluded_effects"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if len(req.ExcludedEffects) > 8 {
		writeJSON(w, map[string]any{"ok": false, "error": "exclude at most eight Specials"})
		return
	}
	excluded := make(map[string]bool, len(req.ExcludedEffects))
	for _, effect := range req.ExcludedEffects {
		excluded[strings.ToLower(strings.TrimSpace(effect))] = true
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if g.Special == content.EffectNone {
		writeJSON(w, map[string]any{"ok": false, "error": "item has no Special to reroll"})
		return
	}
	var pool []content.ItemEffect
	for _, e := range awakenPool {
		if e != g.Special && !excluded[strings.ToLower(string(e))] {
			pool = append(pool, e)
		}
	}
	if len(pool) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "no alternative Specials available"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"core": 6}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Umbral Cores (need 6)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "special reroll")
	old := g.Special
	g.Special = pool[rand.IntN(len(pool))] // #nosec G404 -- non-cryptographic forge roll
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "special reroll", fmt.Sprintf("%s %s→%s", g.Name, old, g.Special), "6🟣") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "special": string(g.Special), "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("🎲 %s's Special is rerolled: %s → %s!", g.Name, old, g.Special)})
}

// ---- Guided awaken (AB-113) ----------------------------------------------------------------------

// forge4GuidedAwakenKey is the app_meta key storing a pending guided-awaken roll.
func forge4GuidedAwakenKey(uid string) string { return "abyss_guided_awaken_" + uid }

type forge4GuidedAwakenPending struct {
	ItemKey string   `json:"item_key"`
	Options []string `json:"options"`
}

// handleAbyssGuidedAwaken is the two-step guided awaken (6 Umbral Cores = double
// the plain awaken): without `choice` it rolls 3 Specials from the awaken pool,
// stores them and returns them for free (re-calling returns the same roll — no
// re-rolling until committed); with `choice` (0-2) it charges and applies the
// picked Special.
func (s *WebServer) handleAbyssGuidedAwaken(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		Choice *int `json:"choice"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if g.Special != content.EffectNone || g.Awakened {
		writeJSON(w, map[string]any{"ok": false, "error": "this item has no dormant power to awaken"})
		return
	}
	key := forge4ItemKey(req.InvID, req.Slot)
	var pending forge4GuidedAwakenPending
	var raw string
	if err := tx.QueryRow("SELECT value FROM app_meta WHERE key=$1", forge4GuidedAwakenKey(uid)).Scan(&raw); err == nil {
		_ = json.Unmarshal([]byte(raw), &pending)
	}
	if len(pending.Options) == 3 && pending.ItemKey == key {
		if req.Choice == nil {
			writeJSON(w, map[string]any{"ok": true, "options": pending.Options, "item": key, "msg": "Pick one of your rolled Specials (choice 0-2) — 6 Umbral Cores on commit."})
			return
		}
	} else {
		if req.Choice != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "no pending roll for this item — call without choice first"})
			return
		}
		// #nosec G404 -- non-cryptographic forge rolls
		pending = forge4GuidedAwakenPending{ItemKey: key, Options: []string{
			string(awakenPool[rand.IntN(len(awakenPool))]),
			string(awakenPool[rand.IntN(len(awakenPool))]),
			string(awakenPool[rand.IntN(len(awakenPool))]),
		}}
		buf, _ := json.Marshal(pending)
		if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		                      ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, forge4GuidedAwakenKey(uid), string(buf)); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if err := tx.Commit(); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "options": pending.Options, "item": key,
			"msg": "🔮 The forge reveals three dormant powers. Commit with choice 0-2 (6 Umbral Cores) — the roll is locked in until then."})
		return
	}

	choice := *req.Choice
	if choice < 0 || choice > 2 {
		writeJSON(w, map[string]any{"ok": false, "error": "choice must be 0, 1 or 2"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"core": 6}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Umbral Cores (need 6)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "guided awaken")
	pick := content.ItemEffect(pending.Options[choice])
	g.Special = pick
	g.Awakened = true
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if _, err := tx.Exec("DELETE FROM app_meta WHERE key=$1", forge4GuidedAwakenKey(uid)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "guided awaken", fmt.Sprintf("%s → %s", g.Name, pick), "6🟣") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "special": string(pick), "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("🌟 %s awakens with your chosen power — it is now %s!", g.Name, pick)})
}

// ---- Imbue removal (AB-114) -------------------------------------------------------------------------

// handleAbyssImbueRemove strips an imbued effect (1 Eldritch Prism): the Imbued
// flag is cleared and its effect is removed from BonusEffects.
func (s *WebServer) handleAbyssImbueRemove(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req forgeItemReq
	if !readForgeItemReq(w, r, &req) {
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if g.Imbued == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "item is not imbued"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"prism": 1}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Eldritch Prisms (need 1)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "imbue removal")
	removed := g.Imbued
	filtered := g.BonusEffects[:0]
	for _, e := range g.BonusEffects {
		if e != content.ItemEffect(g.Imbued) {
			filtered = append(filtered, e)
		}
	}
	g.BonusEffects = filtered
	g.Imbued = ""
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "imbue removal", fmt.Sprintf("%s — %s", g.Name, removed), "1💠") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "removed": removed, "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("💠 The imbued %s effect is stripped from %s.", removed, g.Name)})
}

// ---- Polish-all equipped (AB-115) ------------------------------------------------------------------------

// handleAbyssPolishAll polishes every equipped piece that isn't at the polish
// cap, in one transaction, charging the summed per-item cost (150g base each).
// The undo snapshot only covers the single-item case — the forge undo framework
// stores at most two items, so polishing 3+ pieces leaves no undo.
func (s *WebServer) handleAbyssPolishAll(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	tx, err := s.beginForgeRequestTx(w)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.Query("SELECT slot, gear_id, item_data FROM user_gear WHERE client_uid=$1 FOR UPDATE", uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	type equippedPiece struct {
		slot string
		g    content.Gear
		raw  string
	}
	var pieces []equippedPiece
	for rows.Next() {
		var slot, gearID string
		var itemData sql.NullString
		if err := rows.Scan(&slot, &gearID, &itemData); err != nil {
			continue
		}
		g, ok := s.bot.makeGear(gearID, itemData)
		if !ok {
			continue
		}
		pieces = append(pieces, equippedPiece{slot, g, itemData.String})
	}
	_ = rows.Close()

	var total int64
	var polishable []int
	for i, p := range pieces {
		baseMax := p.g.MaxDurability
		if cat, found := content.GetGearByID(p.g.ID); found {
			baseMax = cat.MaxDurability
		}
		if p.g.MaxDurability < baseMax+100 {
			polishable = append(polishable, i)
			total += s.forge4GoldCost(uid, 150, p.g.Rarity)
		}
	}
	if len(polishable) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "every equipped piece is already polished to the limit"})
		return
	}
	if !deductGold(w, tx, uid, total) {
		return
	}
	// Undo only when a single item is mutated (see doc comment).
	if len(polishable) == 1 {
		p := pieces[polishable[0]]
		s.bot.snapshotForgeUndo(tx, uid, 0, p.slot, p.raw, "polish all")
	}
	var names []string
	for _, i := range polishable {
		p := &pieces[i]
		baseMax := p.g.MaxDurability
		if cat, found := content.GetGearByID(p.g.ID); found {
			baseMax = cat.MaxDurability
		}
		p.g.MaxDurability += 10
		if p.g.MaxDurability > baseMax+100 {
			p.g.MaxDurability = baseMax + 100
		}
		if !saveForgeItem(w, tx, uid, 0, p.slot, p.g) {
			return
		}
		if _, err := tx.Exec("UPDATE user_gear SET durability = LEAST(durability + 10, $1) WHERE slot=$2 AND client_uid=$3", p.g.MaxDurability, p.slot, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		names = append(names, p.g.Name)
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "polish all", fmt.Sprintf("%d pieces", len(polishable)), fmt.Sprintf("%dg", total)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "polished": len(polishable), "items": names, "spent": total, "undo": len(polishable) == 1,
		"gold": s.bot.abyssGold(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("🔧 Polished %d equipped pieces for %dg: %s.", len(polishable), total, strings.Join(names, ", "))})
}

// ---- Repair Kit II (AB-116) + crafting crit (AB-122) --------------------------------------------------------

// handleAbyssCraftRepairKit2 crafts a Repair Kit II for 8 Abyssal Dust, with a
// 5% crafting-crit chance of double output (AB-122). It does not advance the
// weekly crafting quest (that counter lives in handleAbyssCraft).
func (s *WebServer) handleAbyssCraftRepairKit2(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	tx, err := s.beginForgeRequestTx(w)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	if !spendMaterials(tx, uid, map[string]int{"dust": 8}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Abyssal Dust (need 8)"})
		return
	}
	crit := rand.Float64() < 0.05 // #nosec G404 -- non-cryptographic crafting roll
	count := 1
	if crit {
		count = 2
	}
	if _, err := tx.Exec(
		`INSERT INTO user_consumables (client_uid, cons_id, remaining_fights)
		 VALUES ($1, 'repair_kit_ii', $2)
		 ON CONFLICT (client_uid, cons_id)
		 DO UPDATE SET remaining_fights = user_consumables.remaining_fights + EXCLUDED.remaining_fights`,
		uid, count); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	forge4MasteryBump(tx, uid)
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	s.bot.autoCombineConsumable(uid, "repair_kit_ii")
	s.bot.recordForge(uid, "craft", "Repair Kit II", "8🌫️")
	msg := "🧰 Crafted a Repair Kit II!"
	if crit {
		msg = "🧰✨ CRITICAL CRAFT! Double output — 2 Repair Kits II!"
	}
	writeJSON(w, map[string]any{"ok": true, "crit": crit, "count": count,
		"materials": s.bot.loadMaterials(uid), "consumables": s.bot.getConsumables(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": msg})
}

// ---- Socket relocation (AB-118) --------------------------------------------------------------------------------

// handleAbyssSocketRelocate moves a socketed gem to another position in the
// Gemstones array (order matters for gem tooling), for a small gold fee.
func (s *WebServer) handleAbyssSocketRelocate(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		From int `json:"from"`
		To   int `json:"to"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	n := len(g.Gemstones)
	if req.From < 0 || req.From >= n || req.To < 0 || req.To >= n || req.From == req.To {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("pick two different socket indexes (0-%d)", n-1)})
		return
	}
	cost := s.forge4GoldCost(uid, 50, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "socket relocation")
	gem := g.Gemstones[req.From]
	rest := append(g.Gemstones[:req.From], g.Gemstones[req.From+1:]...)
	rest = append(rest, "")
	copy(rest[req.To+1:], rest[req.To:])
	rest[req.To] = gem
	g.Gemstones = rest
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	forge4MasteryBump(tx, uid)
	if !s.finishForge(w, tx, uid, "socket relocation", fmt.Sprintf("%s: %s %d→%d", g.Name, gem, req.From, req.To), fmt.Sprintf("%dg", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "gemstones": g.Gemstones, "gold": s.bot.abyssGold(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("🔨 Moved %s from socket %d to socket %d on %s.", gem, req.From, req.To, g.Name)})
}

// ---- Fusion preview (AB-119) --------------------------------------------------------------------------------------

// handleAbyssFusePreview projects the outcome of a fusion without consuming
// anything: same validation and survivor pick as fuseCommon, returning both
// possible outcomes with their chances and projected stats.
func (s *WebServer) handleAbyssFusePreview(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		InvIDs []int64 `json:"inv_ids"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if len(req.InvIDs) < 2 || len(req.InvIDs) > 3 {
		writeJSON(w, map[string]any{"ok": false, "error": "fusion preview requires 2 or 3 items"})
		return
	}
	seen := make(map[int64]bool, len(req.InvIDs))
	for _, id := range req.InvIDs {
		if seen[id] {
			writeJSON(w, map[string]any{"ok": false, "error": "duplicate inventory item"})
			return
		}
		seen[id] = true
	}

	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var items []content.Gear
	var slot content.GearSlot
	for i, id := range req.InvIDs {
		g, _, ok := loadForgeItem(tx, s.bot, uid, id, "")
		if !ok {
			writeJSON(w, map[string]any{"ok": false, "error": "item not found in backpack"})
			return
		}
		if g.Attuned {
			writeJSON(w, map[string]any{"ok": false, "error": g.Name + " is attuned to you and cannot be fused"})
			return
		}
		if i == 0 {
			slot = g.Slot
		} else if g.Slot != slot {
			writeJSON(w, map[string]any{"ok": false, "error": "all items must share the same slot"})
			return
		}
		items = append(items, g)
	}
	if len(items) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "no items given"})
		return
	}
	rarity := items[0].Rarity
	for _, g := range items[1:] {
		if g.Rarity != rarity {
			writeJSON(w, map[string]any{"ok": false, "error": "all items must share the same rarity"})
			return
		}
	}
	mode := ""
	switch {
	case rarity == content.RarityLegendary && len(items) == 3:
		mode = "ancient"
	case rarity == content.RarityMythic && len(items) == 2:
		mode = "mythic"
	case rarity == content.RarityCelestial && len(items) == 2:
		mode = "celestial"
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "no fusion recipe matches: 3 Legendary, 2 Mythic or 2 Celestial same-slot pieces"})
		return
	}

	best := items[0]
	for _, g := range items[1:] {
		if g.CombatRating() > best.CombatRating() {
			best = g
		}
	}
	cost := s.forge4GoldCost(uid, 2000, rarity)
	survivor := map[string]any{"name": best.Name, "cr": best.CombatRating(), "stats": best.Stats}
	var outcomes []map[string]any
	if mode == "ancient" {
		out := best
		out.Stats = out.Stats.Scaled(1.30)
		outcomes = append(outcomes, map[string]any{"chance": 1.0, "ascended": false,
			"name": "Ancient " + best.Name, "stats": out.Stats, "cr": out.CombatRating()})
	} else {
		ascend := content.RarityDivine
		prefix := "Divine "
		if mode == "celestial" {
			ascend = content.RarityEternal
			prefix = "Eternal "
		}
		up := best
		up.Rarity = ascend
		up.Stats = up.Stats.Scaled(1.25)
		outcomes = append(outcomes, map[string]any{"chance": 0.25, "ascended": true,
			"name": prefix + best.Name, "rarity": ascend.String(), "stats": up.Stats, "cr": up.CombatRating()})
		flat := best
		flat.Stats = flat.Stats.Scaled(1.10)
		outcomes = append(outcomes, map[string]any{"chance": 0.75, "ascended": false,
			"name": best.Name, "stats": flat.Stats, "cr": flat.CombatRating()})
	}
	writeJSON(w, map[string]any{"ok": true, "mode": mode, "cost": cost, "survivor": survivor, "outcomes": outcomes,
		"msg": fmt.Sprintf("🔮 %s fusion preview: %s survives.", mode, best.Name)})
}

// ---- Celestial fuse safety (AB-120) ---------------------------------------------------------------------------------

// handleAbyssCelestialFuseBoosted is the celestial fusion with the safety
// blessing: 10 Eldritch Prisms raise the Eternal-ascension roll from 25% to 50%.
// Same math as fuseCommon's celestial branch (which lives in
// web_abyss_features.go and takes no boost parameter — hence this copy).
func (s *WebServer) handleAbyssCelestialFuseBoosted(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		InvIDs []int64 `json:"inv_ids"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if len(req.InvIDs) != 2 {
		writeJSON(w, map[string]any{"ok": false, "error": "select exactly 2 items"})
		return
	}

	tx, err := s.beginForgeRequestTx(w)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var items []content.Gear
	var slot content.GearSlot
	for i, id := range req.InvIDs {
		g, _, ok := loadForgeItem(tx, s.bot, uid, id, "")
		if !ok {
			writeJSON(w, map[string]any{"ok": false, "error": "item not found in backpack"})
			return
		}
		if g.Rarity != content.RarityCelestial {
			writeJSON(w, map[string]any{"ok": false, "error": "both items must be Celestial"})
			return
		}
		if g.Attuned {
			writeJSON(w, map[string]any{"ok": false, "error": g.Name + " is attuned to you and cannot be fused"})
			return
		}
		if i == 0 {
			slot = g.Slot
		} else if g.Slot != slot {
			writeJSON(w, map[string]any{"ok": false, "error": "both items must share the same slot"})
			return
		}
		items = append(items, g)
	}

	cost := s.forge4GoldCost(uid, 2000, content.RarityCelestial)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"prism": 10}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Eldritch Prisms (need 10)"})
		return
	}
	for _, id := range req.InvIDs {
		res, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2", id, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "item vanished mid-fuse"})
			return
		}
	}

	best := items[0]
	if items[1].CombatRating() > best.CombatRating() {
		best = items[1]
	}
	ascended := rand.Float64() < 0.5 // #nosec G404 -- non-cryptographic fusion roll
	var msg string
	if ascended {
		best.Rarity = content.RarityEternal
		best.Stats = best.Stats.Scaled(1.25)
		best.Name = "Eternal " + best.Name
		msg = fmt.Sprintf("🌟 ETERNAL ASCENSION! The blessed fusion pays off: %s (+25%% stats)!", best.Name)
	} else {
		best.Stats = best.Stats.Scaled(1.10)
		msg = fmt.Sprintf("⚗️ The blessed fusion holds: %s keeps +10%% stats (no ascension even at 50%%).", best.Name)
	}
	dataBytes, _ := json.Marshal(best)
	if _, err := tx.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability, item_data) VALUES ($1,$2,$3,$4)",
		uid, best.ID, best.MaxDurability, string(dataBytes)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	forge4MasteryBump(tx, uid)
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	s.bot.recordForge(uid, "celestial fusion (blessed)", best.Name, fmt.Sprintf("%dg 10💠", cost))
	if ascended {
		go s.bot.broadcastAbyssEternalDrop(uid, best.Name)
	}
	writeJSON(w, map[string]any{"ok": true, "ascended": ascended, "msg": msg,
		"gold": s.bot.abyssGold(uid), "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid)})
}

// ---- Recipe favorites (AB-123) ---------------------------------------------------------------------------------------

// forge4RecipeFavKey is the app_meta key storing the player's favorite recipes.
func forge4RecipeFavKey(uid string) string { return "abyss_recipe_favs_" + uid }

// handleAbyssRecipeFav toggles a recipe in the player's favorites list (stored
// in app_meta, max 8). The craft UI pins favorites atop the craft list.
func (s *WebServer) handleAbyssRecipeFav(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		RecipeID string `json:"recipe_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if _, ok := craftRecipeByID(req.RecipeID); !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown recipe"})
		return
	}

	var favs []string
	var raw string
	if err := s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", forge4RecipeFavKey(uid)).Scan(&raw); err == nil {
		_ = json.Unmarshal([]byte(raw), &favs)
	}
	faved := true
	out := favs[:0]
	found := false
	for _, f := range favs {
		if f == req.RecipeID {
			found = true
			continue
		}
		out = append(out, f)
	}
	if found {
		favs, faved = out, false
	} else {
		if len(favs) >= 8 {
			writeJSON(w, map[string]any{"ok": false, "error": "favorite list is full (8) — unpin one first"})
			return
		}
		favs = append(favs, req.RecipeID)
	}
	buf, _ := json.Marshal(favs)
	if _, err := s.bot.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
	                            ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, forge4RecipeFavKey(uid), string(buf)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	msg := "📌 Recipe pinned to favorites."
	if !faved {
		msg = "📌 Recipe removed from favorites."
	}
	writeJSON(w, map[string]any{"ok": true, "favorited": faved, "favorites": favs, "msg": msg})
}

// ---- Material conversion (AB-124) --------------------------------------------------------------------------------------

// forge4ConvertRates maps "from:to" to the unit cost in the source material.
var forge4ConvertRates = map[string]struct {
	from, to string
	rate     int
}{
	"dust:shard": {"dust", "shard", 10},
	"shard:core": {"shard", "core", 10},
	"core:prism": {"core", "prism", 5},
}

// handleAbyssConvertMats converts crafting materials upward: 10 dust → 1 shard,
// 10 shard → 1 core, 5 core → 1 prism, `count` times (default 1).
func (s *WebServer) handleAbyssConvertMats(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		From  string `json:"from"`
		To    string `json:"to"`
		Count int    `json:"count"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	conv, ok := forge4ConvertRates[strings.ToLower(req.From)+":"+strings.ToLower(req.To)]
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unsupported conversion — dust→shard, shard→core or core→prism"})
		return
	}
	if req.Count < 1 {
		req.Count = 1
	}
	if req.Count > 1000 {
		req.Count = 1000
	}

	tx, err := s.beginForgeRequestTx(w)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	if !spendMaterials(tx, uid, map[string]int{conv.from: conv.rate * req.Count}) {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("not enough %s (need %d)", abyssMaterialName(conv.from), conv.rate*req.Count)})
		return
	}
	if err := grantMaterialQ(tx, uid, conv.to, req.Count); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("⚗️ Converted %d× %s into %d× %s.", conv.rate*req.Count, abyssMaterialName(conv.from), req.Count, abyssMaterialName(conv.to))})
}

// ---- Second forge undo per day (AB-125) ---------------------------------------------------------------------------------

// Forge4Undo2Key is the app_meta flag marking the purchased second daily undo.
const Forge4Undo2Key = "abyss_forge_undo2"

func forge4Undo2Key(uid string) string { return Forge4Undo2Key + "_" + uid }

// handleAbyssSanctuaryUndo2 sells the sanctuary upgrade that allows a second
// forge undo per day (50 tokens, one-time).
func (s *WebServer) handleAbyssSanctuaryUndo2(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var owned string
	_ = s.bot.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", forge4Undo2Key(uid)).Scan(&owned)
	if owned == "1" {
		writeJSON(w, map[string]any{"ok": false, "error": "you already own the second daily forge undo"})
		return
	}

	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !deductTokens(w, tx, uid, 50) {
		return
	}
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, '1')
	                      ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, forge4Undo2Key(uid)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "tokens": s.bot.abyssTokens(uid),
		"msg": "🕊️ Sanctuary expanded — you may now undo two forge actions per day."})
}
