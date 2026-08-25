package bot

// Forge automation handlers: batch temper, insurance, queued actions, and
// bulk gemstone upgrades. Split from web_abyss_forge4.go to keep each Forge
// subsystem independently reviewable.

import (
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"

	"ts3news/internal/content"
)

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
	if !s.finishForge(w, tx, uid, "forge queue", g.Name+": "+strings.Join(steps, ", "), fmt.Sprintf("%dg", totalGold)) {
		return
	}
	s.bot.forge4MasteryAdd(uid, len(steps)-1)
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
	if !s.finishForge(w, tx, uid, "bulk gem upgrade", fmt.Sprintf("%s ×%d steps", g.Name, steps), fmt.Sprintf("%dg", goldCost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "upgraded": len(upgradeIndexes), "steps": steps, "stop_at_tier": req.StopAtTier, "gems": names,
		"gold": s.bot.abyssGold(uid), "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("💎 Upgraded %d gems on %s to tier %d (%s).", len(upgradeIndexes), g.Name, req.StopAtTier, strings.Join(names, ", "))})
}
