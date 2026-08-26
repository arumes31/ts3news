package bot

// Forge round 3: ten gear-improvement mechanics — polish, reinforce, sharpen,
// awaken, imbue, punch socket, attune, reforge, embrace corruption and
// masterwork. All follow the established forge shape: per-uid lock, optional
// {inv_id|slot} body, one transaction, undo snapshot, guarded cost deduction,
// item_data rewrite, forge history, and a response with refreshed balances.

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"

	"ts3news/internal/content"
)

// forgeItemReq is the shared {inv_id | slot} item specifier used by all forge2
// endpoints (same as the earlier forge handlers).
type forgeItemReq struct {
	InvID int64  `json:"inv_id"`
	Slot  string `json:"slot"`
}

// isWeaponSlot mirrors the weapon check from handleAbyssEtchRune.
func isWeaponSlot(slot content.GearSlot) bool {
	return slot == content.SlotMainHand || slot == content.SlotOffHand || slot == content.SlotRanged
}

// beginForgeTx opens a transaction and loads the target item. On failure it
// writes the error response, rolls back, and returns ok=false; on success the
// caller owns tx (defer Rollback) and gets the parsed gear plus its raw stored
// JSON for the undo snapshot.
func (s *WebServer) beginForgeTx(w http.ResponseWriter, uid string, invID int64, slot string) (*sql.Tx, content.Gear, string, bool) {
	tx, err := s.beginForgeRequestTx(w)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return nil, content.Gear{}, "", false
	}
	g, rawData, ok := loadForgeItem(tx, s.bot, uid, invID, slot)
	if !ok {
		_ = tx.Rollback()
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return nil, content.Gear{}, "", false
	}
	return tx, g, rawData, true
}

// finishForge commits the forge transaction and records history/artisan rep.
func (s *WebServer) finishForge(w http.ResponseWriter, tx *sql.Tx, uid, action, detail, cost string) bool {
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return false
	}
	s.bot.recordForge(uid, action, detail, cost)
	return true
}

// saveForgeItem marshals g and persists it to the right table.
func saveForgeItem(w http.ResponseWriter, tx *sql.Tx, uid string, invID int64, slot string, g content.Gear) bool {
	dataBytes, _ := json.Marshal(g)
	return writeGearItemData(w, tx, uid, invID, slot, string(dataBytes))
}

// readForgeItemReq decodes the item specifier, tolerating an empty body.
func readForgeItemReq(w http.ResponseWriter, r *http.Request, req *forgeItemReq) bool {
	if err := readJSON(r, req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return false
	}
	return true
}

// abyssGold re-reads the player's gold balance for the response.
func (b *Bot) abyssGold(uid string) int64 {
	var gold int64
	_ = b.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	return gold
}

// ---- Polish -----------------------------------------------------------------

// handleAbyssPolish raises an item's max durability by 10 (150g via forgeCost),
// capped at the catalog base +100. Equipped items also gain 10 current
// durability, up to the new max.
func (s *WebServer) handleAbyssPolish(w http.ResponseWriter, r *http.Request, uid string) {
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

	baseMax := g.MaxDurability
	if cat, found := content.GetGearByID(g.ID); found {
		baseMax = cat.MaxDurability
	}
	cap := baseMax + 100
	if g.MaxDurability >= cap {
		writeJSON(w, map[string]any{"ok": false, "error": "already polished to the limit"})
		return
	}
	cost := s.bot.forgeGoldCost(uid, 150, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "polish")
	oldMax := g.MaxDurability
	g.MaxDurability += 10
	if g.MaxDurability > cap {
		g.MaxDurability = cap
	}
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if req.Slot != "" {
		if _, err := tx.Exec("UPDATE user_gear SET durability = LEAST(durability + 10, $1) WHERE slot=$2 AND client_uid=$3", g.MaxDurability, req.Slot, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if !s.finishForge(w, tx, uid, "polish", g.Name, fmt.Sprintf("%dg", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "gold": s.bot.abyssGold(uid),
		"msg": fmt.Sprintf("🔧 Polished %s! Max durability %d → %d.", g.Name, oldMax, g.MaxDurability)})
}

// ---- Reinforce & Sharpen ------------------------------------------------------

const reinforceMax = 10

// handleAbyssReinforce bakes +2% DEF (min +1) into armor/jewelry for 100g +
// 2 Abyssal Dust per level, capped at level 10. Weapons are sharpened instead.
func (s *WebServer) handleAbyssReinforce(w http.ResponseWriter, r *http.Request, uid string) {
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

	if isWeaponSlot(g.Slot) {
		writeJSON(w, map[string]any{"ok": false, "error": "weapons can't be reinforced — sharpen them instead"})
		return
	}
	if g.Reinforced >= reinforceMax {
		writeJSON(w, map[string]any{"ok": false, "error": "already reinforced to the maximum (10)"})
		return
	}
	cost := s.bot.forgeGoldCost(uid, 100, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"dust": 2}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Abyssal Dust (need 2)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "reinforce")
	inc := g.Stats.DEF * 2 / 100
	if inc < 1 {
		inc = 1
	}
	g.Stats.DEF += inc
	g.Reinforced++
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "reinforce", fmt.Sprintf("%s → %d/10", g.Name, g.Reinforced), fmt.Sprintf("%dg", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "gold": s.bot.abyssGold(uid), "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("🛡️ Reinforced %s (+%d DEF)! Reinforcement %d/10.", g.Name, inc, g.Reinforced)})
}

// handleAbyssSharpen bakes +2% STR (min +1) into a weapon for 100g + 2 Abyssal
// Dust per level, capped at level 10.
func (s *WebServer) handleAbyssSharpen(w http.ResponseWriter, r *http.Request, uid string) {
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

	if !isWeaponSlot(g.Slot) {
		writeJSON(w, map[string]any{"ok": false, "error": "only weapons can be sharpened"})
		return
	}
	if g.Sharpened >= reinforceMax {
		writeJSON(w, map[string]any{"ok": false, "error": "already sharpened to the maximum (10)"})
		return
	}
	cost := s.bot.forgeGoldCost(uid, 100, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"dust": 2}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Abyssal Dust (need 2)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "sharpen")
	inc := g.Stats.STR * 2 / 100
	if inc < 1 {
		inc = 1
	}
	g.Stats.STR += inc
	g.Sharpened++
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "sharpen", fmt.Sprintf("%s → %d/10", g.Name, g.Sharpened), fmt.Sprintf("%dg", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "gold": s.bot.abyssGold(uid), "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("⚔️ Sharpened %s (+%d STR)! Sharpness %d/10.", g.Name, inc, g.Sharpened)})
}

// ---- Awaken -------------------------------------------------------------------

// awakenPool is the set of dormant Specials an item can awaken with.
var awakenPool = []content.ItemEffect{
	content.EffectVampiric, content.EffectThorns, content.EffectBerserk, content.EffectParry,
	content.EffectLucky, content.EffectQuick, content.EffectBulwark, content.EffectRadiant,
	content.EffectExecutioner, content.EffectFocused,
}

// handleAbyssAwaken rolls a random Special onto an item that has none (3 Umbral
// Cores, once per item).
func (s *WebServer) handleAbyssAwaken(w http.ResponseWriter, r *http.Request, uid string) {
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

	if g.Special != content.EffectNone || g.Awakened {
		writeJSON(w, map[string]any{"ok": false, "error": "this item has no dormant power to awaken"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"core": 3}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Umbral Cores (need 3)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "awaken")
	pick := awakenPool[rand.IntN(len(awakenPool))] // #nosec G404 -- non-cryptographic forge roll
	g.Special = pick
	g.Awakened = true
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "awaken", fmt.Sprintf("%s → %s", g.Name, pick), "3🟣") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("🌟 %s awakens — it is now %s!", g.Name, pick)})
}

// ---- Imbue --------------------------------------------------------------------

// imbueEffects whitelists the effects that can be imbued (body param → enum).
var imbueEffects = map[string]content.ItemEffect{
	"vampiric":    content.EffectVampiric,
	"thorns":      content.EffectThorns,
	"lucky":       content.EffectLucky,
	"quick":       content.EffectQuick,
	"bulwark":     content.EffectBulwark,
	"radiant":     content.EffectRadiant,
	"executioner": content.EffectExecutioner,
	"focused":     content.EffectFocused,
}

// handleAbyssImbue layers one whitelisted bonus effect onto an item (2 Eldritch
// Prisms, once per item, no duplicates of an effect it already carries).
func (s *WebServer) handleAbyssImbue(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		Effect string `json:"effect"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	eff, valid := imbueEffects[strings.ToLower(strings.TrimSpace(req.Effect))]
	if !valid {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid effect — pick vampiric, thorns, lucky, quick, bulwark, radiant, executioner or focused"})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if g.Imbued != "" {
		writeJSON(w, map[string]any{"ok": false, "error": "this item is already imbued"})
		return
	}
	if g.Special == eff {
		writeJSON(w, map[string]any{"ok": false, "error": "the item already carries that effect"})
		return
	}
	for _, e := range g.BonusEffects {
		if e == eff {
			writeJSON(w, map[string]any{"ok": false, "error": "the item already carries that effect"})
			return
		}
	}
	if !spendMaterials(tx, uid, map[string]int{"prism": 2}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Eldritch Prisms (need 2)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "imbue")
	g.BonusEffects = append(g.BonusEffects, eff)
	g.Imbued = string(eff)
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "imbue", fmt.Sprintf("%s → %s", g.Name, eff), "2💠") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("💠 Imbued %s with %s!", g.Name, eff)})
}

// ---- Punch Socket ---------------------------------------------------------------

const maxPunchedSockets = 4

func abyssPunchSocketResult(current int, roll float64) (sockets int, perfect bool) {
	sockets = current + 1
	if current == maxPunchedSockets-1 && roll < 0.10 {
		return maxPunchedSockets + 1, true
	}
	return sockets, false
}

// handleAbyssPunchSocket adds one gemstone socket to an item (10 Void Shards).
// The normal cap is four; the fourth punch has a ten-percent Perfect result
// that grants a fifth socket.
func (s *WebServer) handleAbyssPunchSocket(w http.ResponseWriter, r *http.Request, uid string) {
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

	if g.Sockets >= maxPunchedSockets {
		writeJSON(w, map[string]any{"ok": false, "error": "already at the socket limit (4)"})
		return
	}
	forgeFloorUsed, err := claimAbyssForgeFloorInTx(tx, uid, "punch_socket")
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if s.forgeQuoteRequiresFloor(r, "punch_socket") && !forgeFloorUsed {
		writeJSON(w, map[string]any{"ok": false, "error": "Silent Anvil changed; refresh the forge quote"})
		return
	}
	if !forgeFloorUsed && !spendMaterials(tx, uid, map[string]int{"shard": 10}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Void Shards (need 10)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "punch socket")
	sockets, perfect := abyssPunchSocketResult(g.Sockets, rand.Float64())
	g.Sockets = sockets
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	costLabel := "10🔷"
	if forgeFloorUsed {
		costLabel = "Silent Anvil"
	}
	if !s.finishForge(w, tx, uid, "punch socket", fmt.Sprintf("%s → %d sockets", g.Name, g.Sockets), costLabel) {
		return
	}
	message := fmt.Sprintf("🔨 Punched a new socket into %s (%d/%d).", g.Name, g.Sockets, maxPunchedSockets)
	if perfect {
		message = fmt.Sprintf("💎 Perfect punch! %s gained two sockets at once (%d total).", g.Name, g.Sockets)
	}
	if forgeFloorUsed {
		message = "⚒️ Silent Anvil: " + message
	}
	writeJSON(w, map[string]any{"ok": true, "materials": s.bot.loadMaterials(uid),
		"perfect": perfect, "sockets": g.Sockets, "forge_floor_used": forgeFloorUsed, "msg": message})
}

// ---- Attune ---------------------------------------------------------------------

// handleAbyssAttune binds an item to its owner for 15 tokens (once): +5% stats
// baked in, and the item can no longer be fused, salvaged, dismantled or
// auctioned (enforced in those handlers).
func (s *WebServer) handleAbyssAttune(w http.ResponseWriter, r *http.Request, uid string) {
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

	if g.Attuned {
		writeJSON(w, map[string]any{"ok": false, "error": "already attuned to you"})
		return
	}
	if !deductTokens(w, tx, uid, 15) {
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "attune")
	g.Stats = g.Stats.Scaled(1.05)
	g.Attuned = true
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "attune", g.Name, "🜲15") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "tokens": s.bot.abyssTokens(uid),
		"msg": fmt.Sprintf("🔗 %s is now attuned to you (+5%% stats). It can no longer be fused, salvaged, dismantled or auctioned.", g.Name)})
}

// ---- Reforge --------------------------------------------------------------------

// handleAbyssReforge rerolls every nonzero combat stat of a Rare-or-better item
// by an independent random factor in [0.90, 1.10] for 300g (flavour stats
// untouched), and reports the net combat-rating delta.
func (s *WebServer) handleAbyssReforge(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		Family      string  `json:"family"`
		MinQuality  float64 `json:"min_quality"`
		RejectBelow bool    `json:"reject_below"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	req.Family = strings.ToLower(strings.TrimSpace(req.Family))
	validFamilies := map[string]bool{"": true, "balanced": true, "offense": true, "defense": true, "speed": true, "utility": true}
	if !validFamilies[req.Family] || req.MinQuality < 0 || req.MinQuality > 200 {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid reforge family or minimum quality"})
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
	cost := s.bot.forgeGoldCost(uid, 300, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	crBefore := g.CombatRating()
	// #nosec G404 -- non-cryptographic forge rolls
	reroll := func(code string, v int) int {
		if v == 0 {
			return 0
		}
		preferred := req.Family == "" || req.Family == "balanced" ||
			(req.Family == "offense" && (code == "STR" || code == "CRT" || code == "INT")) ||
			(req.Family == "defense" && (code == "HP" || code == "DEF" || code == "DGE")) ||
			(req.Family == "speed" && (code == "SPD" || code == "DGE")) ||
			(req.Family == "utility" && (code == "LCK" || code == "STA" || code == "MNA"))
		low, spread := 0.90, 0.15
		if preferred && req.Family != "" && req.Family != "balanced" {
			low, spread = 1.0, 0.12
		}
		return int(float64(v) * (low + rand.Float64()*spread))
	}
	g.Stats.HP = reroll("HP", g.Stats.HP)
	g.Stats.STR = reroll("STR", g.Stats.STR)
	g.Stats.DEF = reroll("DEF", g.Stats.DEF)
	g.Stats.SPD = reroll("SPD", g.Stats.SPD)
	g.Stats.LCK = reroll("LCK", g.Stats.LCK)
	g.Stats.INT = reroll("INT", g.Stats.INT)
	g.Stats.STA = reroll("STA", g.Stats.STA)
	g.Stats.CRT = reroll("CRT", g.Stats.CRT)
	g.Stats.DGE = reroll("DGE", g.Stats.DGE)
	g.Stats.MNA = reroll("MNA", g.Stats.MNA)
	crAfter := g.CombatRating()
	quality := 100.0
	if crBefore > 0 {
		quality = crAfter / crBefore * 100
	}
	if req.RejectBelow && quality < req.MinQuality {
		if !s.finishForge(w, tx, uid, "reforge rejected", g.Name, fmt.Sprintf("%dg", cost)) {
			return
		}
		writeJSON(w, map[string]any{"ok": true, "accepted": false, "quality": quality, "uses_left": usesLeft, "gold": s.bot.abyssGold(uid),
			"msg": fmt.Sprintf("🎲 Reforge rejected automatically at %.1f%% quality; the original item was kept.", quality)})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "reforge")
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "reforge", g.Name, fmt.Sprintf("%dg", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "accepted": true, "quality": quality, "family": req.Family, "uses_left": usesLeft, "gold": s.bot.abyssGold(uid),
		"msg": fmt.Sprintf("🎲 Reforged %s — CR %.0f → %.0f (%.1f%% quality).", g.Name, crBefore, crAfter, quality)})
}

// ---- Embrace Corruption ---------------------------------------------------------

// handleAbyssEmbrace lets the owner of a corrupted item embrace the corruption
// (1 Eldritch Prism, once): the HP malus is halved permanently, but the item
// can never be cleansed (enforced in handleAbyssCleanse).
func (s *WebServer) handleAbyssEmbrace(w http.ResponseWriter, r *http.Request, uid string) {
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

	if !g.Corrupted {
		writeJSON(w, map[string]any{"ok": false, "error": "item is not corrupted"})
		return
	}
	if g.Embraced {
		writeJSON(w, map[string]any{"ok": false, "error": "the corruption is already embraced"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"prism": 1}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Eldritch Prisms (need 1)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "embrace corruption")
	// The malus is baked into Stats.HP, so halving it refunds half the HP.
	g.Stats.HP += g.CorruptHP - g.CorruptHP/2
	g.CorruptHP /= 2
	g.Embraced = true
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "embrace corruption", g.Name, "1💠") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("🩸 You embrace the corruption within %s — the HP malus is halved, but it can never be cleansed.", g.Name)})
}

// ---- Masterwork -----------------------------------------------------------------

const masterworkMax = 5

// qualityNames indexes the display name of each masterwork level.
var qualityNames = []string{"", "Fine", "Superior", "Exquisite", "Flawless", "Masterwork"}

// handleAbyssMasterwork raises an item's Quality 0→5, baking x1.03 combat stats
// per level. Level q (0-based) costs (q+1)*5 dust + (q+1)*2 shards, plus (q-1)
// cores from q>=2.
func (s *WebServer) handleAbyssMasterwork(w http.ResponseWriter, r *http.Request, uid string) {
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

	q := g.Quality
	if q >= masterworkMax {
		writeJSON(w, map[string]any{"ok": false, "error": "already a masterwork (quality 5)"})
		return
	}
	mats := map[string]int{"dust": (q + 1) * 5, "shard": (q + 1) * 2}
	if q >= 2 {
		mats["core"] = q - 1
	}
	if !spendMaterials(tx, uid, mats) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough materials"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "masterwork")
	g.Stats = g.Stats.Scaled(1.03)
	g.Quality++
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	costStr := fmt.Sprintf("%d🌫️ %d🔷", (q+1)*5, (q+1)*2)
	if q >= 2 {
		costStr += fmt.Sprintf(" %d🟣", q-1)
	}
	if !s.finishForge(w, tx, uid, "masterwork", fmt.Sprintf("%s → %s", g.Name, qualityNames[g.Quality]), costStr) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "quality": g.Quality, "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("🏅 %s improved to %s quality (+3%% stats).", g.Name, qualityNames[g.Quality])})
}
