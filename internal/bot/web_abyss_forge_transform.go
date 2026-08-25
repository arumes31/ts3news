package bot

// Forge item-transformation handlers: rune recovery, attunement, masterwork,
// reforge, rebalance, brands, Specials, guided awakening, and imbues.

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
	if !s.finishForge(w, tx, uid, "un-attune", g.Name, "🜲50") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "tokens": s.bot.abyssTokens(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("⛓️‍💥 %s is no longer attuned and loses its +5%% attunement stats. It can be fused, salvaged, dismantled and auctioned again.", g.Name)})
}

// ---- Masterwork transfer (AB-107) ------------------------------------------------------

func forge4MasterworkTransferLevels(sourceQuality, targetQuality int) int {
	moved := max(0, sourceQuality) * 4 / 5
	return min(moved, max(0, masterworkMax-targetQuality))
}

func forge4RemoveMasterworkStats(stats content.Stats, quality int) content.Stats {
	for range max(0, quality) {
		stats = stats.Scaled(1 / 1.03)
	}
	return stats
}

// handleAbyssMasterworkTransfer moves an item's masterwork Quality onto another
// same-slot item at 80% efficiency (floor), zeroing the source's Quality and
// reversing its baked quality multipliers. The target gets x1.03 per transferred
// level. Integer rounding makes the reversal approximate but prevents repeatedly
// transferring and re-masterworking one source to duplicate stats.
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
	moved := forge4MasterworkTransferLevels(g1.Quality, g2.Quality)
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
	g1.Stats = forge4RemoveMasterworkStats(g1.Stats, fromQ)
	g1.Quality = 0
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

func forge4ReforgeDailyLimit(rarity content.Rarity) int {
	if rarity >= content.RarityEternal {
		return 2
	}
	return 1
}

func forge4ConsumeReforgeUse(tx *sql.Tx, uid string, rarity content.Rarity) (usesLeft int, allowed bool, err error) {
	var lockedUID string
	if err := tx.QueryRow("SELECT client_uid FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&lockedUID); err != nil {
		return 0, false, err
	}
	today := time.Now().UTC().Format("2006-01-02")
	uses := 0
	var value string
	if err := tx.QueryRow("SELECT value FROM app_meta WHERE key=$1", forge4ReforgeUsesKey(uid)).Scan(&value); err == nil {
		if day, count, found := strings.Cut(value, ":"); found && day == today {
			uses, _ = strconv.Atoi(count)
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, err
	}
	limit := forge4ReforgeDailyLimit(rarity)
	if uses >= limit {
		return 0, false, nil
	}
	uses++
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, forge4ReforgeUsesKey(uid), fmt.Sprintf("%s:%d", today, uses)); err != nil {
		return 0, false, err
	}
	return limit - uses, true, nil
}

// handleAbyssReforgeLock is reforge with one stat line excluded from the ±10%
// reroll, at double price (600g base). Limited to once per day — Eternal items
// may be reforged twice per day (AB-121). Plain and locked reforge share the
// same account-level daily allowance, so switching endpoints cannot bypass it.
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
	usesLeft, allowed, err := forge4ConsumeReforgeUse(tx, uid, g.Rarity)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if !allowed {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("reforge limit reached for today (%d/day)", forge4ReforgeDailyLimit(g.Rarity))})
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
	if !s.finishForge(w, tx, uid, "reforge lock", fmt.Sprintf("%s (locked %s)", g.Name, lockStat), fmt.Sprintf("%dg", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "uses_left": usesLeft,
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
	if !s.finishForge(w, tx, uid, "special reroll", fmt.Sprintf("%s %s→%s", g.Name, old, g.Special), "6🟣") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "special": string(g.Special), "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("🎲 %s's Special is rerolled: %s → %s!", g.Name, old, g.Special)})
}

// ---- Guided awaken (AB-113) ----------------------------------------------------------------------

// forge4GuidedAwakenKey is the app_meta key storing a pending guided-awaken roll.
func forge4GuidedAwakenKey(uid, itemKey string) string {
	return "abyss_guided_awaken_" + uid + "_" + itemKey
}

type forge4GuidedAwakenPending struct {
	ItemKey string   `json:"item_key"`
	Options []string `json:"options"`
}

func forge4GuidedAwakenOptions() []string {
	options := make([]string, 0, 3)
	seen := make(map[content.ItemEffect]bool, 3)
	for len(options) < 3 {
		effect := awakenPool[rand.IntN(len(awakenPool))] // #nosec G404 -- non-cryptographic forge roll
		if seen[effect] {
			continue
		}
		seen[effect] = true
		options = append(options, string(effect))
	}
	return options
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
	pendingKey := forge4GuidedAwakenKey(uid, key)
	var pending forge4GuidedAwakenPending
	var raw string
	if err := tx.QueryRow("SELECT value FROM app_meta WHERE key=$1", pendingKey).Scan(&raw); err == nil {
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
		pending = forge4GuidedAwakenPending{ItemKey: key, Options: forge4GuidedAwakenOptions()}
		buf, _ := json.Marshal(pending)
		if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		                      ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, pendingKey, string(buf)); err != nil {
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
	if _, err := tx.Exec("DELETE FROM app_meta WHERE key=$1", pendingKey); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
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
	if !s.finishForge(w, tx, uid, "imbue removal", fmt.Sprintf("%s — %s", g.Name, removed), "1💠") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "removed": removed, "materials": s.bot.loadMaterials(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("💠 The imbued %s effect is stripped from %s.", removed, g.Name)})
}
