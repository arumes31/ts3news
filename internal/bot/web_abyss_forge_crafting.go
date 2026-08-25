package bot

// Forge crafting and account handlers: mass polishing, consumable crafting,
// sockets, fusion, recipe favorites, material conversion, and undo capacity.

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
			_ = rows.Close()
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		g, ok := s.bot.makeGear(gearID, itemData)
		if !ok {
			continue
		}
		pieces = append(pieces, equippedPiece{slot, g, itemData.String})
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := rows.Close(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

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
	if !s.finishForge(w, tx, uid, "polish all", fmt.Sprintf("%d pieces", len(polishable)), fmt.Sprintf("%dg", total)) {
		return
	}
	writeJSON(w, map[string]any{"ok": true, "polished": len(polishable), "items": names, "spent": total, "undo": len(polishable) == 1,
		"gold": s.bot.abyssGold(uid), "mastery": s.bot.forge4MasteryInfo(uid),
		"msg": fmt.Sprintf("🔧 Polished %d equipped pieces for %dg: %s.", len(polishable), total, strings.Join(names, ", "))})
}

// ---- Repair Kit II (AB-116) + crafting crit (AB-122) --------------------------------------------------------

func forge4CraftOutput(roll float64) (count int, critical bool) {
	if roll < 0.05 {
		return 2, true
	}
	return 1, false
}

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
	count, crit := forge4CraftOutput(rand.Float64()) // #nosec G404 -- non-cryptographic crafting roll
	if _, err := tx.Exec(
		`INSERT INTO user_consumables (client_uid, cons_id, remaining_fights)
		 VALUES ($1, 'repair_kit_ii', $2)
		 ON CONFLICT (client_uid, cons_id)
		 DO UPDATE SET remaining_fights = user_consumables.remaining_fights + EXCLUDED.remaining_fights`,
		uid, count); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
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

func forge4RelocateGem(gems []string, from, to int) ([]string, bool) {
	if from < 0 || from >= len(gems) || to < 0 || to >= len(gems) || from == to {
		return nil, false
	}
	out := append([]string(nil), gems...)
	gem := out[from]
	if from < to {
		copy(out[from:to], out[from+1:to+1])
	} else {
		copy(out[to+1:from+1], out[to:from])
	}
	out[to] = gem
	return out, true
}

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
	relocated, valid := forge4RelocateGem(g.Gemstones, req.From, req.To)
	if !valid {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("pick two different socket indexes (0-%d)", n-1)})
		return
	}
	cost := s.forge4GoldCost(uid, 50, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "socket relocation")
	gem := g.Gemstones[req.From]
	g.Gemstones = relocated
	if !saveForgeItem(w, tx, uid, req.InvID, req.Slot, g) {
		return
	}
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
		InvIDs  []int64 `json:"inv_ids"`
		Boosted bool    `json:"boosted"`
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
		ascendChance := 0.25
		if mode == "celestial" && req.Boosted {
			ascendChance = 0.50
		}
		outcomes = append(outcomes, map[string]any{"chance": ascendChance, "ascended": true,
			"name": prefix + best.Name, "rarity": ascend.String(), "stats": up.Stats, "cr": up.CombatRating()})
		flat := best
		flat.Stats = flat.Stats.Scaled(1.10)
		outcomes = append(outcomes, map[string]any{"chance": 1 - ascendChance, "ascended": false,
			"name": best.Name, "stats": flat.Stats, "cr": flat.CombatRating()})
	}
	boosted := req.Boosted && mode == "celestial"
	prismCost := 0
	if boosted {
		prismCost = 10
	}
	writeJSON(w, map[string]any{"ok": true, "mode": mode, "cost": cost, "survivor": survivor, "outcomes": outcomes,
		"boosted": boosted, "prism_cost": prismCost,
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

func (b *Bot) forge4RecipeFavorites(uid string) ([]string, map[string]bool) {
	var favorites []string
	var raw string
	if err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", forge4RecipeFavKey(uid)).Scan(&raw); err == nil {
		_ = json.Unmarshal([]byte(raw), &favorites)
	}
	if len(favorites) > 8 {
		favorites = favorites[:8]
	}
	set := make(map[string]bool, len(favorites))
	normalized := make([]string, 0, len(favorites))
	for _, recipeID := range favorites {
		if _, ok := craftRecipeByID(recipeID); ok && !set[recipeID] {
			set[recipeID] = true
			normalized = append(normalized, recipeID)
		}
	}
	return normalized, set
}

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
	recipe, ok := craftRecipeByID(req.RecipeID)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown recipe"})
		return
	}
	if recipe.Secret && !s.bot.knownRecipes(uid)[recipe.ID] {
		writeJSON(w, map[string]any{"ok": false, "error": "discover this recipe before favoriting it"})
		return
	}

	favs, _ := s.bot.forge4RecipeFavorites(uid)
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

	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, '1')
	                      ON CONFLICT (key) DO NOTHING`, forge4Undo2Key(uid))
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n != 1 {
		writeJSON(w, map[string]any{"ok": false, "error": "you already own the second daily forge undo"})
		return
	}
	if !deductTokens(w, tx, uid, 50) {
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "tokens": s.bot.abyssTokens(uid),
		"msg": "🕊️ Sanctuary expanded — you may now undo two forge actions per day."})
}
