package bot

// Forge round 4: ten more gear-improvement mechanics — corrupt, curse and
// eldritch infusion, stat rebalance, gem transmute, set branding, temper surge,
// special swap, gear-XP infusion and prismatic runes. Same forge shape as
// round 3 (web_abyss_forge2.go): per-uid lock, {inv_id|slot} body, one
// transaction, undo snapshot, guarded cost deduction, item_data rewrite, forge
// history, refreshed balances in the response.

import (
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strings"

	"ts3news/internal/content"
)

// ---- Corrupt ------------------------------------------------------------------

// handleAbyssCorrupt deliberately corrupts an item (5 Umbral Cores): +50%
// offensive stats, but an HP malus equal to its score — the same trade-off as
// corrupted drops (#83), so it can still be cleansed or embraced afterwards.
func (s *WebServer) handleAbyssCorrupt(w http.ResponseWriter, r *http.Request, uid string) {
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

	if g.Corrupted {
		writeJSON(w, map[string]any{"ok": false, "error": "item is already corrupted"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"core": 5}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Umbral Cores (need 5)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "corrupt")
	g.Stats.STR = g.Stats.STR * 3 / 2
	g.Stats.DEF = g.Stats.DEF * 3 / 2
	g.Stats.SPD = g.Stats.SPD * 3 / 2
	g.CorruptHP = g.Stats.Score()
	g.Stats.HP -= g.CorruptHP
	g.Corrupted = true
	if !strings.HasPrefix(g.Name, "🩸 Corrupted ") {
		g.Name = "🩸 Corrupted " + g.Name
	}
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "corrupt", g.Name, "5🟣") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("😈 %s swells with power (+50%% offensive stats, −%d max HP). Cleanse or embrace it later.", g.Name, g.CorruptHP)})
}

// ---- Curse & Eldritch infusion --------------------------------------------------

// handleAbyssInfuseCurse makes a weapon Cursed (3 cores + 500g): +25% stats,
// but the cursed HP drain applies in combat (same flag as cursed drops).
func (s *WebServer) handleAbyssInfuseCurse(w http.ResponseWriter, r *http.Request, uid string) {
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
		writeJSON(w, map[string]any{"ok": false, "error": "only weapons can be cursed"})
		return
	}
	if g.Cursed {
		writeJSON(w, map[string]any{"ok": false, "error": "item is already cursed"})
		return
	}
	cost := s.bot.forgeGoldCost(uid, 500, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"core": 3}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Umbral Cores (need 3)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "infuse curse")
	g.Stats = g.Stats.Scaled(1.25)
	g.Cursed = true
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "infuse curse", g.Name, fmt.Sprintf("%dg 3🟣", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "gold": s.bot.abyssGold(uid), "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("💀 %s is cursed (+25%% stats, but it drains your HP in combat).", g.Name)})
}

// handleAbyssInfuseEldritch makes an item Eldritch (3 Eldritch Prisms): +25%
// stats and the cosmic-horror affix flag, mirroring eldritch drops.
func (s *WebServer) handleAbyssInfuseEldritch(w http.ResponseWriter, r *http.Request, uid string) {
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

	if g.Eldritch {
		writeJSON(w, map[string]any{"ok": false, "error": "item is already eldritch"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"prism": 3}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Eldritch Prisms (need 3)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "infuse eldritch")
	g.Stats = g.Stats.Scaled(1.25)
	g.Eldritch = true
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "infuse eldritch", g.Name, "3💠") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("🌌 %s is infused with eldritch power (+25%% stats).", g.Name)})
}

// ---- Rebalance ------------------------------------------------------------------

// rebalanceStats maps the selectable stat codes to their Stats fields.
var rebalanceStats = map[string]bool{
	"HP": true, "STR": true, "DEF": true, "SPD": true, "LCK": true,
	"INT": true, "STA": true, "CRT": true, "DGE": true, "MNA": true,
}

// gearStatRef returns a pointer to a Stats field by its code.
func gearStatRef(s *content.Stats, code string) *int {
	switch code {
	case "HP":
		return &s.HP
	case "STR":
		return &s.STR
	case "DEF":
		return &s.DEF
	case "SPD":
		return &s.SPD
	case "LCK":
		return &s.LCK
	case "INT":
		return &s.INT
	case "STA":
		return &s.STA
	case "CRT":
		return &s.CRT
	case "DGE":
		return &s.DGE
	case "MNA":
		return &s.MNA
	}
	return nil
}

// handleAbyssRebalance moves 25% of one combat stat into another (200g) — turn
// a dump stat into your build's main stat.
func (s *WebServer) handleAbyssRebalance(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	from, to := strings.ToUpper(req.From), strings.ToUpper(req.To)
	if !rebalanceStats[from] || !rebalanceStats[to] || from == to {
		writeJSON(w, map[string]any{"ok": false, "error": "pick two different stats"})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	src := gearStatRef(&g.Stats, from)
	moved := *src / 4
	if moved < 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough " + from + " to rebalance"})
		return
	}
	cost := s.bot.forgeGoldCost(uid, 200, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "rebalance")
	*src -= moved
	*gearStatRef(&g.Stats, to) += moved
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "rebalance", fmt.Sprintf("%s %s→%s", g.Name, from, to), fmt.Sprintf("%dg", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "gold": s.bot.abyssGold(uid),
		"msg": fmt.Sprintf("⚖️ Rebalanced %s: %d %s → %s.", g.Name, moved, from, to)})
}

// ---- Gem Transmute ----------------------------------------------------------------

// abyssGemBaseStats returns the base (tier I) stat line of a gem type.
func abyssGemBaseStats(base string) (content.Stats, bool) {
	switch base {
	case "Ruby":
		return content.Stats{HP: 100}, true
	case "Sapphire":
		return content.Stats{MNA: 50}, true
	case "Emerald":
		return content.Stats{STR: 15}, true
	case "Diamond":
		return content.Stats{DEF: 15}, true
	case "Topaz":
		return content.Stats{CRT: 5}, true
	}
	return content.Stats{}, false
}

// abyssGemTierMultiplier maps a gem tier suffix to its stat multiplier.
func abyssGemTierMultiplier(suffix string) int {
	switch suffix {
	case "III":
		return 4
	case "II":
		return 2
	}
	return 1
}

// handleAbyssGemTransmute changes the first socketed gem into a chosen type
// (150g), swapping its baked stat line while keeping the tier.
func (s *WebServer) handleAbyssGemTransmute(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		Gem string `json:"gem"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	newBase := strings.TrimSpace(req.Gem)
	newStats, valid := abyssGemBaseStats(newBase)
	if !valid {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid gemstone type"})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if len(g.Gemstones) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "no socketed gem to transmute"})
		return
	}
	old := g.Gemstones[0]
	oldBase, oldSuffix, _ := strings.Cut(old, " ")
	oldStats, okOld := abyssGemBaseStats(oldBase)
	if !okOld {
		writeJSON(w, map[string]any{"ok": false, "error": "unrecognised socketed gem"})
		return
	}
	if oldBase == newBase {
		writeJSON(w, map[string]any{"ok": false, "error": "that gem is already socketed here"})
		return
	}
	tierMult := abyssGemTierMultiplier(oldSuffix)

	cost := s.bot.forgeGoldCost(uid, 150, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "gem transmute")
	// Swap the baked stats: remove the old line, add the new one at the same tier.
	g.Stats = g.Stats.Add(oldStats.Scaled(float64(-tierMult)))
	g.Stats = g.Stats.Add(newStats.Scaled(float64(tierMult)))
	newName := newBase
	if oldSuffix != "" {
		newName += " " + oldSuffix
	}
	g.Gemstones[0] = newName
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "gem transmute", fmt.Sprintf("%s %s→%s", g.Name, old, newName), fmt.Sprintf("%dg", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "gold": s.bot.abyssGold(uid),
		"msg": fmt.Sprintf("💎 Transmuted %s into %s on %s.", old, newName, g.Name)})
}

// ---- Brand (set membership) --------------------------------------------------------

// brandableSets are the named Abyss sets an item can be branded into.
var brandableSets = map[string]bool{"predator": true, "warden": true, "harvester": true}

// handleAbyssBrand stamps an item into a named Abyss set (10 Umbral Cores) so it
// counts toward that set's bonus tiers.
func (s *WebServer) handleAbyssBrand(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		Set string `json:"set"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	set := strings.ToLower(strings.TrimSpace(req.Set))
	if !brandableSets[set] {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown set — pick predator, warden or harvester"})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if g.SetID == set {
		writeJSON(w, map[string]any{"ok": false, "error": "item already belongs to that set"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"core": 10}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Umbral Cores (need 10)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "brand")
	g.SetID = set
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "brand", fmt.Sprintf("%s → %s", g.Name, set), "10🟣") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("🏷️ %s is now branded into the %s set.", g.Name, set)})
}

// ---- Temper Surge ------------------------------------------------------------------

const temperSurgeMax = 20

// handleAbyssTemperSurge pushes a +15 temper into the dangerous +16..+20 range:
// 2000g per attempt, a flat 50% success chance, no pity — but a failed surge
// never breaks or downgrades the item.
func (s *WebServer) handleAbyssTemperSurge(w http.ResponseWriter, r *http.Request, uid string) {
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

	if g.Temper < temperMax {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("temper the item to +%d first", temperMax)})
		return
	}
	if g.Temper >= temperSurgeMax {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("already surged to the maximum (+%d)", temperSurgeMax)})
		return
	}
	cost := s.bot.forgeGoldCost(uid, 2000, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "temper surge")
	success := rand.Float64() < 0.5 // #nosec G404 -- non-cryptographic forge roll
	msg := fmt.Sprintf("🔥 The surge fails — %s holds at +%d (gold spent).", g.Name, g.Temper)
	if success {
		g.Temper++
		g.Stats = g.Stats.Scaled(1.02)
		msg = fmt.Sprintf("🔥 SURGE! %s reaches temper +%d (+2%% stats).", g.Name, g.Temper)
	}
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "temper surge", fmt.Sprintf("%s → +%d", g.Name, g.Temper), fmt.Sprintf("%dg", cost)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "success": success, "gold": s.bot.abyssGold(uid), "msg": msg})
}

// ---- Special Swap ------------------------------------------------------------------

// handleAbyssSwapSpecial exchanges the Special effects of two items (8 Umbral
// Cores); both must carry a Special.
func (s *WebServer) handleAbyssSwapSpecial(w http.ResponseWriter, r *http.Request, uid string) {
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
		writeJSON(w, map[string]any{"ok": false, "error": "pick a second item"})
		return
	}
	if req.InvID > 0 && req.InvID == req.InvID2 {
		writeJSON(w, map[string]any{"ok": false, "error": "pick two different items"})
		return
	}
	if req.Slot != "" && req.Slot == req.Slot2 {
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
		writeJSON(w, map[string]any{"ok": false, "error": "second item not found"})
		return
	}
	if g1.Special == content.EffectNone || g2.Special == content.EffectNone {
		writeJSON(w, map[string]any{"ok": false, "error": "both items must carry a Special"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"core": 8}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Umbral Cores (need 8)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, raw1, "special swap")
	g1.Special, g2.Special = g2.Special, g1.Special
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g1) {
		return
	}
	if !saveForgeItem(w, tx, uid, req.InvID2, req.Slot2, g2) {
		return
	}
	_ = raw2 // snapshot covers the primary item; the swap is symmetric anyway
	if !s.finishForge(w, tx, uid, "special swap", fmt.Sprintf("%s ↔ %s", g1.Name, g2.Name), "8🟣") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("🔀 Swapped Specials: %s now has %s, %s now has %s.", g1.Name, g1.Special, g2.Name, g2.Special)})
}

// ---- Gear-XP Infusion ----------------------------------------------------------------

// handleAbyssInfuseXP sacrifices another backpack item to feed a weapon's gear
// XP: the victim's combat rating (÷10) becomes KillCount on the target, driving
// it toward its gear-XP milestones. Free — the cost is the sacrificed item.
func (s *WebServer) handleAbyssInfuseXP(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		forgeItemReq
		SacrificeInvID int64 `json:"sacrifice_inv_id"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if req.SacrificeInvID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "pick a backpack item to sacrifice"})
		return
	}
	if req.InvID > 0 && req.InvID == req.SacrificeInvID {
		writeJSON(w, map[string]any{"ok": false, "error": "an item cannot consume itself"})
		return
	}

	tx, g, rawData, ok := s.beginForgeTx(w, uid, req.InvID, req.Slot)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback() }()

	if !isWeaponSlot(g.Slot) {
		writeJSON(w, map[string]any{"ok": false, "error": "only weapons gain gear XP"})
		return
	}
	victim, _, ok := loadForgeItem(tx, s.bot, uid, req.SacrificeInvID, "")
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "sacrifice not found in backpack"})
		return
	}
	if victim.Attuned {
		writeJSON(w, map[string]any{"ok": false, "error": victim.Name + " is attuned to you"})
		return
	}
	xp := int(victim.CombatRating()) / 10
	if xp < 1 {
		xp = 1
	}
	res, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2", req.SacrificeInvID, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "sacrifice vanished mid-infusion"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "infuse gear xp")
	g.KillCount += xp
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "infuse gear xp", fmt.Sprintf("%s +%d XP", g.Name, xp), victim.Name) {
		return
	}
	writeJSON(w, map[string]any{"ok": true,
		"msg": fmt.Sprintf("🩸 %s devours %s — +%d gear XP (%d total).", g.Name, victim.Name, xp, g.KillCount)})
}

// ---- Prismatic Rune ------------------------------------------------------------------

// handleAbyssPrismaticRune elevates an etched rune (2 Eldritch Prisms, once per
// item): +5% is baked into the item's highest combat stat.
func (s *WebServer) handleAbyssPrismaticRune(w http.ResponseWriter, r *http.Request, uid string) {
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
		writeJSON(w, map[string]any{"ok": false, "error": "etch a rune first"})
		return
	}
	if g.Prismatic {
		writeJSON(w, map[string]any{"ok": false, "error": "rune is already prismatic"})
		return
	}
	if !spendMaterials(tx, uid, map[string]int{"prism": 2}) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough Eldritch Prisms (need 2)"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "prismatic rune")
	// Find the highest combat stat and bake +5% into it.
	bestCode, bestVal := "STR", g.Stats.STR
	for _, code := range []string{"HP", "DEF", "SPD", "LCK", "INT", "STA", "CRT", "DGE", "MNA"} {
		if v := *gearStatRef(&g.Stats, code); v > bestVal {
			bestCode, bestVal = code, v
		}
	}
	inc := bestVal * 5 / 100
	if inc < 1 {
		inc = 1
	}
	*gearStatRef(&g.Stats, bestCode) += inc
	g.Prismatic = true
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
	if !s.finishForge(w, tx, uid, "prismatic rune", fmt.Sprintf("%s +%d %s", g.Name, inc, bestCode), "2💠") {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "materials": s.bot.loadMaterials(uid),
		"msg": fmt.Sprintf("🌈 The %s rune on %s turns prismatic (+%d %s).", g.Rune, g.Name, inc, bestCode)})
}
