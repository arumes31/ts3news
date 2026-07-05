package bot

// Abyss expansion 2 (docs/ABYSS_IDEAS.md): crafting materials & recipes, the
// token⇄gold exchange, forge systems (temper, fusions, gem tiers, enchant
// transfer, extraction, deterministic crafting, history/undo/reputation),
// Last Stand revives, specializations, sanctuary upgrades and the new
// Deep-Delver talent nodes.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"ts3news/internal/content"
)

// ---- Crafting materials (#101-#103, #155) ---------------------------------

// abyssMaterial is a stackable crafting currency salvaged from gear.
type abyssMaterial struct {
	ID   string
	Name string
	Icon string
}

var abyssMaterials = []abyssMaterial{
	{"dust", "Abyssal Dust", "🌫️"},
	{"shard", "Void Shard", "🔷"},
	{"core", "Umbral Core", "🟣"},
	{"prism", "Eldritch Prism", "💠"},
}

func abyssMaterialName(id string) string {
	for _, m := range abyssMaterials {
		if m.ID == id {
			return m.Icon + " " + m.Name
		}
	}
	return id
}

// materialYieldForRarity maps a salvaged/dismantled item's rarity to material
// drops. Scavenger (#155) multiplies the count.
func materialYieldForRarity(r content.Rarity) (string, int) {
	switch {
	case r >= content.RarityDivine:
		return "prism", 1
	case r >= content.RarityMythic:
		return "core", 3
	case r >= content.RarityLegendary:
		return "core", 1
	case r >= content.RarityEpic:
		return "shard", 3
	case r >= content.RarityRare:
		return "shard", 1
	default:
		return "dust", 2
	}
}

func (b *Bot) grantMaterial(uid, mat string, n int) error {
	return grantMaterialQ(b.DB, uid, mat, n)
}

// grantMaterialQ is grantMaterial running through any querier, so callers with
// an open transaction can make the grant atomic with their other writes.
func grantMaterialQ(q dbExecQuerier, uid, mat string, n int) error {
	if n <= 0 {
		return nil
	}
	_, err := q.Exec(`INSERT INTO user_materials (client_uid, mat_id, count) VALUES ($1,$2,$3)
	                  ON CONFLICT (client_uid, mat_id) DO UPDATE SET count = user_materials.count + $3`, uid, mat, n)
	return err
}

// scavengerYield applies the Scavenger talent (+20% materials per level, floored).
func scavengerYield(n, level int) int {
	return n + n*level*20/100
}

func (b *Bot) loadMaterials(uid string) map[string]int64 {
	out := map[string]int64{}
	rows, err := b.DB.Query("SELECT mat_id, count FROM user_materials WHERE client_uid=$1", uid)
	if err != nil {
		return out
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		var n int64
		if err := rows.Scan(&id, &n); err == nil {
			out[id] = n
		}
	}
	return out
}

// spendMaterials debits a material cost map inside tx, all-or-nothing.
func spendMaterials(tx *sql.Tx, uid string, cost map[string]int) bool {
	for mat, n := range cost {
		if n <= 0 {
			continue
		}
		res, err := tx.Exec("UPDATE user_materials SET count = count - $1 WHERE client_uid=$2 AND mat_id=$3 AND count >= $1", n, uid, mat)
		if err != nil {
			return false
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			return false
		}
	}
	return true
}

// ---- Recipes & weekly crafting quest (#103-#105) ---------------------------

type craftRecipe struct {
	ID     string
	Name   string
	Desc   string
	ConsID string
	Cost   map[string]int
	Secret bool // discovered via lore fragments (#104)
}

var craftRecipes = []craftRecipe{
	{ID: "brew_small", Name: "Small Health Potion", Desc: "A simple restorative brew.", ConsID: "small_health_potion", Cost: map[string]int{"dust": 4}},
	{ID: "brew_repair", Name: "Repair Kit", Desc: "Patch kit hammered from dust.", ConsID: "repair_kit", Cost: map[string]int{"dust": 6}},
	{ID: "brew_great", Name: "Great Health Potion", Desc: "A potent restorative.", ConsID: "great_health_potion", Cost: map[string]int{"dust": 8, "shard": 1}},
	{ID: "brew_lucky", Name: "Lucky Draught", Desc: "Bottled fortune.", ConsID: "lucky_draught", Cost: map[string]int{"shard": 3}},
	{ID: "brew_master_repair", Name: "Master Repair Kit", Desc: "Restores gear completely.", ConsID: "master_repair_kit", Cost: map[string]int{"shard": 4}, Secret: true},
	{ID: "brew_rejuv", Name: "Rejuvenation Potion", Desc: "Restores 60% of max HP.", ConsID: "rejuvenation_potion", Cost: map[string]int{"shard": 3, "core": 1}, Secret: true},
	{ID: "brew_phoenix", Name: "Phoenix Feather", Desc: "One free return from death.", ConsID: "phoenix_feather", Cost: map[string]int{"core": 3}, Secret: true},
	{ID: "brew_life", Name: "Elixir of Life", Desc: "Full heal in a bottle.", ConsID: "elixir_of_life", Cost: map[string]int{"core": 2, "shard": 4}, Secret: true},
}

func craftRecipeByID(id string) (craftRecipe, bool) {
	for _, r := range craftRecipes {
		if r.ID == id {
			return r, true
		}
	}
	return craftRecipe{}, false
}

func (b *Bot) knownRecipes(uid string) map[string]bool {
	out := map[string]bool{}
	rows, err := b.DB.Query("SELECT recipe_id FROM user_recipes WHERE client_uid=$1", uid)
	if err != nil {
		return out
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			out[id] = true
		}
	}
	return out
}

// discoverRandomRecipe unlocks one still-secret recipe, returning its name.
// Called when a lore fragment is found (#104).
func (b *Bot) discoverRandomRecipe(uid string) string {
	known := b.knownRecipes(uid)
	var locked []craftRecipe
	for _, r := range craftRecipes {
		if r.Secret && !known[r.ID] {
			locked = append(locked, r)
		}
	}
	if len(locked) == 0 {
		return ""
	}
	pick := locked[rand.IntN(len(locked))] // #nosec G404 -- non-cryptographic
	if _, err := b.DB.Exec("INSERT INTO user_recipes (client_uid, recipe_id) VALUES ($1,$2) ON CONFLICT DO NOTHING", uid, pick.ID); err != nil {
		return ""
	}
	return pick.Name
}

const craftQuestTarget = 3

// craftQuestWeek is the ISO-week key for the weekly crafting quest (#105).
func craftQuestWeek() string {
	y, w := time.Now().UTC().ISOWeek()
	return fmt.Sprintf("%d-W%02d", y, w)
}

// handleAbyssCraft crafts one consumable from materials (#103) and advances the
// weekly crafting quest (#105): 3 crafts in a week pay 🜲15 and a Master Repair Kit.
func (s *WebServer) handleAbyssCraft(w http.ResponseWriter, r *http.Request, uid string) {
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
	rec, ok := craftRecipeByID(req.RecipeID)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown recipe"})
		return
	}
	if rec.Secret && !s.bot.knownRecipes(uid)[rec.ID] {
		writeJSON(w, map[string]any{"ok": false, "error": "recipe not yet discovered — find more lore fragments"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !spendMaterials(tx, uid, rec.Cost) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough materials"})
		return
	}
	// Weekly quest progress, reset when the ISO week rolls over.
	week := craftQuestWeek()
	if _, err := tx.Exec(`UPDATE users SET craft_quest_done = CASE WHEN craft_quest_week = $2 THEN craft_quest_done + 1 ELSE 1 END,
	                                       craft_quest_week = $2 WHERE client_uid=$1`, uid, week); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var done int
	_ = tx.QueryRow("SELECT craft_quest_done FROM users WHERE client_uid=$1", uid).Scan(&done)

	// Grant the crafted item (and any weekly-quest reward) inside the same
	// transaction as the material spend, so a failure can't leave materials
	// debited without the rewards landing.
	grantCons := func(consID string) bool {
		if _, err := tx.Exec(
			`INSERT INTO user_consumables (client_uid, cons_id, remaining_fights)
			 VALUES ($1, $2, 1)
			 ON CONFLICT (client_uid, cons_id)
			 DO UPDATE SET remaining_fights = user_consumables.remaining_fights + EXCLUDED.remaining_fights`,
			uid, consID); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return false
		}
		return true
	}
	if !grantCons(rec.ConsID) {
		return
	}
	msg := "Crafted " + rec.Name + "!"
	questDone := done == craftQuestTarget
	if questDone {
		if _, err := tx.Exec("UPDATE users SET abyss_tokens = abyss_tokens + 15 WHERE client_uid=$1", uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if !grantCons("master_repair_kit") {
			return
		}
		msg += " 🏆 Weekly crafting quest complete: +🜲15 and a Master Repair Kit!"
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	// Best-effort passive stack combining, same as grantConsumable would do.
	s.bot.autoCombineConsumable(uid, rec.ConsID)
	if questDone {
		s.bot.autoCombineConsumable(uid, "master_repair_kit")
	}
	writeJSON(w, map[string]any{
		"ok": true, "msg": msg, "quest_done": done, "quest_target": craftQuestTarget,
		"materials": s.bot.loadMaterials(uid), "tokens": s.bot.abyssTokens(uid),
		"consumables": s.bot.getConsumables(uid),
	})
}

// ---- Token ⇄ gold exchange (#126, 1 token = 100K gold) ---------------------

const (
	abyssTokenBuyGold  = 100_000 // gold cost to buy 1 token
	abyssTokenSellGold = 90_000  // gold received selling 1 token (spread is the sink)
)

func (s *WebServer) handleAbyssExchange(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Dir    string `json:"dir"` // "buy" tokens with gold | "sell" tokens for gold
		Amount int64  `json:"amount"`
	}
	if err := readJSON(r, &req); err != nil || req.Amount <= 0 || req.Amount > 1000 {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	switch req.Dir {
	case "buy":
		cost := req.Amount * abyssTokenBuyGold
		if !deductGold(w, tx, uid, cost) {
			return
		}
		if _, err := tx.Exec("UPDATE users SET abyss_tokens = abyss_tokens + $1 WHERE client_uid=$2", req.Amount, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	case "sell":
		if !deductTokens(w, tx, uid, req.Amount) {
			return
		}
		if _, err := tx.Exec("UPDATE users SET gold = gold + $1 WHERE client_uid=$2", req.Amount*abyssTokenSellGold, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	default:
		writeJSON(w, map[string]any{"ok": false, "error": "dir must be buy or sell"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{"ok": true, "gold": gold, "tokens": s.bot.abyssTokens(uid)})
}

// ---- Forge shared: reputation, happy hour, history, undo (#114/#121/#123/#116)

// forgeDiscountPct is the artisan-reputation discount: 5% per 25 actions, max 20%.
func forgeDiscountPct(rep int) int {
	d := rep / 25 * 5
	if d > 20 {
		d = 20
	}
	return d
}

// forgeHappyHour (#121): a fixed daily discount hour (18:00–18:59 UTC), +20% off.
func forgeHappyHour() bool {
	return time.Now().UTC().Hour() == 18
}

// forgeCost applies reputation and happy-hour discounts to a base gold cost.
func (b *Bot) forgeCost(uid string, base int64) int64 {
	var rep int
	_ = b.DB.QueryRow("SELECT forge_rep FROM users WHERE client_uid=$1", uid).Scan(&rep)
	pct := forgeDiscountPct(rep)
	if forgeHappyHour() {
		pct += 20
	}
	if pct > 40 {
		pct = 40
	}
	c := base - base*int64(pct)/100
	if c < 1 {
		c = 1
	}
	return c
}

// recordForge appends to the forge history (#123) and grows artisan rep (#114).
func (b *Bot) recordForge(uid, action, detail, cost string) {
	_, _ = b.DB.Exec("INSERT INTO forge_history (client_uid, action, detail, cost) VALUES ($1,$2,$3,$4)", uid, action, detail, cost)
	_, _ = b.DB.Exec("UPDATE users SET forge_rep = forge_rep + 1 WHERE client_uid=$1", uid)
}

// forgeUndoSnapshot stores the pre-action item state of the most recent forge
// action (#116); performing the undo is what's limited to once per day.
type forgeUndoSnapshot struct {
	InvID    int64  `json:"inv_id,omitempty"`
	Slot     string `json:"slot,omitempty"`
	ItemData string `json:"item_data"`
	Action   string `json:"action"`
}

// snapshotForgeUndo runs inside the caller's forge transaction so the snapshot
// only persists if the action it belongs to actually commits, and always
// overwrites so the stored undo reflects the latest action.
func (b *Bot) snapshotForgeUndo(tx *sql.Tx, uid string, invID int64, slot, itemData, action string) {
	snap, _ := json.Marshal(forgeUndoSnapshot{InvID: invID, Slot: slot, ItemData: itemData, Action: action})
	_, _ = tx.Exec(`UPDATE users SET forge_undo=$2 WHERE client_uid=$1`, uid, string(snap))
}

// handleAbyssForgeUndo restores the last snapshotted forge action (#116). The
// undo itself is limited to once per day (forge_undo_date records the day it
// was last used).
func (s *WebServer) handleAbyssForgeUndo(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var raw sql.NullString
	var usedToday bool
	_ = s.bot.DB.QueryRow("SELECT forge_undo, COALESCE(forge_undo_date = CURRENT_DATE, FALSE) FROM users WHERE client_uid=$1", uid).Scan(&raw, &usedToday)
	if usedToday {
		writeJSON(w, map[string]any{"ok": false, "error": "undo already used today"})
		return
	}
	if !raw.Valid || raw.String == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "nothing to undo"})
		return
	}
	var snap forgeUndoSnapshot
	if err := json.Unmarshal([]byte(raw.String), &snap); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "corrupt snapshot"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !writeGearItemData(w, tx, uid, snap.InvID, snap.Slot, snap.ItemData) {
		return
	}
	if _, err := tx.Exec("UPDATE users SET forge_undo=NULL, forge_undo_date=CURRENT_DATE WHERE client_uid=$1", uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	s.bot.recordForge(uid, "undo", "reverted "+snap.Action, "free")
	writeJSON(w, map[string]any{"ok": true, "msg": "Last forge action (" + snap.Action + ") reverted."})
}

type forgeHistoryRow struct {
	Action string
	Detail string
	Cost   string
	When   string
}

func (b *Bot) loadForgeHistory(uid string, limit int) []forgeHistoryRow {
	rows, err := b.DB.Query("SELECT action, detail, cost, created_at FROM forge_history WHERE client_uid=$1 ORDER BY id DESC LIMIT $2", uid, limit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []forgeHistoryRow
	for rows.Next() {
		var v forgeHistoryRow
		var t time.Time
		if err := rows.Scan(&v.Action, &v.Detail, &v.Cost, &t); err == nil {
			v.When = t.Format("Jan 02 15:04")
			out = append(out, v)
		}
	}
	return out
}

// loadForgeItem resolves an {inv_id | slot} item specifier used by all forge
// endpoints, returning the parsed gear and its raw stored JSON.
func loadForgeItem(tx *sql.Tx, b *Bot, uid string, invID int64, slot string) (content.Gear, string, bool) {
	var gearID string
	var itemData sql.NullString
	var err error
	if invID > 0 {
		err = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2", invID, uid).Scan(&gearID, &itemData)
	} else if slot != "" {
		err = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2", slot, uid).Scan(&gearID, &itemData)
	} else {
		return content.Gear{}, "", false
	}
	if err != nil {
		return content.Gear{}, "", false
	}
	g, ok := b.makeGear(gearID, itemData)
	return g, itemData.String, ok
}

// ---- Temper (#106) + fail-stack pity (#107) --------------------------------

const temperMax = 15

// temperChance is the success odds for the next temper attempt.
func temperChance(level, failStacks int) float64 {
	c := 0.95 - 0.06*float64(level) + 0.05*float64(failStacks)
	if c > 0.95 {
		c = 0.95
	}
	if c < 0.20 {
		c = 0.20
	}
	return c
}

func (s *WebServer) handleAbyssTemper(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		InvID int64  `json:"inv_id"`
		Slot  string `json:"slot"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	g, rawData, ok := loadForgeItem(tx, s.bot, uid, req.InvID, req.Slot)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	if g.Unidentified {
		writeJSON(w, map[string]any{"ok": false, "error": "identify the item first"})
		return
	}
	if g.Temper >= temperMax {
		writeJSON(w, map[string]any{"ok": false, "error": "already tempered to the maximum (+15)"})
		return
	}

	var failStacks int
	_ = tx.QueryRow("SELECT temper_fail_stacks FROM users WHERE client_uid=$1", uid).Scan(&failStacks)
	cost := s.bot.forgeCost(uid, int64(400*(g.Temper+1)))
	if !deductGold(w, tx, uid, cost) {
		return
	}

	chance := temperChance(g.Temper, failStacks)
	// #nosec G404 -- non-cryptographic forge roll
	success := rand.Float64() < chance

	if success {
		s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "temper")
		g.Temper++
		g.Stats = g.Stats.Scaled(1.02)
		dataBytes, _ := json.Marshal(g)
		if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
			return
		}
		if _, err := tx.Exec("UPDATE users SET temper_fail_stacks = 0 WHERE client_uid=$1", uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	} else {
		if _, err := tx.Exec("UPDATE users SET temper_fail_stacks = temper_fail_stacks + 1 WHERE client_uid=$1", uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	if success {
		s.bot.recordForge(uid, "temper", fmt.Sprintf("%s → +%d", g.Name, g.Temper), fmt.Sprintf("%dg", cost))
		writeJSON(w, map[string]any{"ok": true, "success": true, "temper": g.Temper, "gold": gold,
			"msg": fmt.Sprintf("⚒️ Temper succeeded! %s is now +%d (+2%% stats).", g.Name, g.Temper)})
		return
	}
	s.bot.recordForge(uid, "temper", g.Name+" failed", fmt.Sprintf("%dg", cost))
	writeJSON(w, map[string]any{"ok": true, "success": false, "temper": g.Temper, "gold": gold,
		"fail_stacks": failStacks + 1,
		"msg":         fmt.Sprintf("💥 Temper failed (%.0f%% chance). Pity grows: +5%% on the next attempt.", chance*100)})
}

// ---- Gem tiers & upgrades (#84) --------------------------------------------

// gemBaseStats is the tier-I stat block of each gem type (matches socketing).
func gemBaseStats(name string) (content.Stats, bool) {
	switch name {
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

// parseGem splits "Ruby II" into base name and tier (1-3).
func parseGem(s string) (string, int) {
	base := s
	tier := 1
	if strings.HasSuffix(s, " III") {
		base, tier = strings.TrimSuffix(s, " III"), 3
	} else if strings.HasSuffix(s, " II") {
		base, tier = strings.TrimSuffix(s, " II"), 2
	}
	return base, tier
}

func gemName(base string, tier int) string {
	switch tier {
	case 2:
		return base + " II"
	case 3:
		return base + " III"
	}
	return base
}

// handleAbyssUpgradeGem merges a socketed gem up a tier (#84): I→II doubles its
// stats, II→III doubles them again. Costs gold + materials.
func (s *WebServer) handleAbyssUpgradeGem(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		InvID    int64  `json:"inv_id"`
		Slot     string `json:"slot"`
		GemIndex int    `json:"gem_index"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	g, rawData, ok := loadForgeItem(tx, s.bot, uid, req.InvID, req.Slot)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	if req.GemIndex < 0 || req.GemIndex >= len(g.Gemstones) {
		writeJSON(w, map[string]any{"ok": false, "error": "no gem in that socket"})
		return
	}
	base, tier := parseGem(g.Gemstones[req.GemIndex])
	baseStats, valid := gemBaseStats(base)
	if !valid {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown gem type"})
		return
	}
	if tier >= 3 {
		writeJSON(w, map[string]any{"ok": false, "error": "gem is already tier III"})
		return
	}

	var goldCost int64
	var mats map[string]int
	if tier == 1 {
		goldCost, mats = s.bot.forgeCost(uid, 200), map[string]int{"shard": 5}
	} else {
		goldCost, mats = s.bot.forgeCost(uid, 500), map[string]int{"core": 2}
	}
	if !deductGold(w, tx, uid, goldCost) {
		return
	}
	if !spendMaterials(tx, uid, mats) {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough materials"})
		return
	}

	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "gem upgrade")
	// Delta added on upgrade equals the current total (I→II adds 1×base, II→III adds 2×base).
	delta := baseStats.Scaled(float64(tier))
	g.Stats = g.Stats.Add(delta)
	g.Gemstones[req.GemIndex] = gemName(base, tier+1)
	dataBytes, _ := json.Marshal(g)
	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	s.bot.recordForge(uid, "gem", fmt.Sprintf("%s → %s", g.Name, gemName(base, tier+1)), fmt.Sprintf("%dg", goldCost))
	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("💎 %s upgraded to %s!", base, gemName(base, tier+1)),
		"materials": s.bot.loadMaterials(uid)})
}

// ---- Socket extraction (#117) ----------------------------------------------

func (s *WebServer) handleAbyssExtractGem(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		InvID    int64  `json:"inv_id"`
		Slot     string `json:"slot"`
		GemIndex int    `json:"gem_index"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	g, rawData, ok := loadForgeItem(tx, s.bot, uid, req.InvID, req.Slot)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	if req.GemIndex < 0 || req.GemIndex >= len(g.Gemstones) {
		writeJSON(w, map[string]any{"ok": false, "error": "no gem in that socket"})
		return
	}
	cost := s.bot.forgeCost(uid, 100)
	if !deductGold(w, tx, uid, cost) {
		return
	}

	base, tier := parseGem(g.Gemstones[req.GemIndex])
	baseStats, valid := gemBaseStats(base)
	if !valid {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown gem type"})
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "gem extraction")
	// Remove the gem's full contribution: tier I=1×, II=2×, III=4× base.
	totalMult := []float64{0, 1, 2, 4}[tier]
	g.Stats = g.Stats.Add(baseStats.Scaled(-totalMult))
	g.Gemstones = append(g.Gemstones[:req.GemIndex], g.Gemstones[req.GemIndex+1:]...)
	dataBytes, _ := json.Marshal(g)
	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	// The pried-out gem crumbles into shards (2 per tier) — credited in the same
	// transaction as the removal so the payout can't be lost after the commit.
	if err := grantMaterialQ(tx, uid, "shard", 2*tier); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	s.bot.recordForge(uid, "extract", fmt.Sprintf("%s from %s", gemName(base, tier), g.Name), fmt.Sprintf("%dg", cost))
	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("🔧 Extracted %s — it crumbled into %d Void Shards. Socket freed.", gemName(base, tier), 2*tier),
		"materials": s.bot.loadMaterials(uid)})
}

// ---- Enchant transfer (#111) -----------------------------------------------

func (s *WebServer) handleAbyssTransferEnchant(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		FromSlot string `json:"from_slot"`
		ToSlot   string `json:"to_slot"`
	}
	if err := readJSON(r, &req); err != nil || req.FromSlot == "" || req.ToSlot == "" || req.FromSlot == req.ToSlot {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var ench sql.NullString
	if err := tx.QueryRow("SELECT enchantment_id FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, req.FromSlot).Scan(&ench); err != nil || !ench.Valid || ench.String == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "source item has no enchantment"})
		return
	}
	var toEnch sql.NullString
	if err := tx.QueryRow("SELECT enchantment_id FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, req.ToSlot).Scan(&toEnch); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "no gear equipped in the target slot"})
		return
	}
	// Never silently destroy an enchantment already sitting on the target.
	if toEnch.Valid && toEnch.String != "" {
		writeJSON(w, map[string]any{"ok": false, "error": "target item already has an enchantment — move that one away first"})
		return
	}
	if !deductTokens(w, tx, uid, 5) {
		return
	}
	if _, err := tx.Exec("UPDATE user_gear SET enchantment_id=NULL WHERE client_uid=$1 AND slot=$2", uid, req.FromSlot); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("UPDATE user_gear SET enchantment_id=$3 WHERE client_uid=$1 AND slot=$2", uid, req.ToSlot, ench.String); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	s.bot.recordForge(uid, "enchant transfer", req.FromSlot+" → "+req.ToSlot, "🜲5")
	writeJSON(w, map[string]any{"ok": true, "msg": "🔰 Enchantment moved from " + req.FromSlot + " to " + req.ToSlot + ".", "tokens": s.bot.abyssTokens(uid)})
}

// ---- Fusions: Ancient (#92) and Mythic (#112) -------------------------------

// handleAbyssFuse consumes 3 same-slot Legendary inventory pieces and forges the
// strongest into an "Ancient" version with +30% stats (#92).
func (s *WebServer) handleAbyssFuse(w http.ResponseWriter, r *http.Request, uid string) {
	s.fuseCommon(w, r, uid, "ancient")
}

// handleAbyssMythicFuse consumes 2 same-slot Mythic pieces: 25% chance of a
// Divine ascension, otherwise the survivor keeps +10% stats (#112).
func (s *WebServer) handleAbyssMythicFuse(w http.ResponseWriter, r *http.Request, uid string) {
	s.fuseCommon(w, r, uid, "mythic")
}

func (s *WebServer) fuseCommon(w http.ResponseWriter, r *http.Request, uid, mode string) {
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
	need := 3
	minRarity := content.RarityLegendary
	if mode == "mythic" {
		need = 2
		minRarity = content.RarityMythic
	}
	if len(req.InvIDs) != need {
		writeJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("select exactly %d items", need)})
		return
	}

	tx, err := s.bot.DB.Begin()
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
		if g.Rarity != minRarity {
			writeJSON(w, map[string]any{"ok": false, "error": "all items must be " + minRarity.String()})
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

	cost := s.bot.forgeCost(uid, 2000)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	// Consume every input piece.
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

	// The best CR piece survives, boosted.
	best := items[0]
	for _, g := range items[1:] {
		if g.CombatRating() > best.CombatRating() {
			best = g
		}
	}
	var msg string
	if mode == "ancient" {
		best.Stats = best.Stats.Scaled(1.30)
		best.Name = "Ancient " + best.Name
		msg = fmt.Sprintf("🏺 The three relics collapse into one: %s (+30%% stats)!", best.Name)
	} else {
		// #nosec G404 -- non-cryptographic fusion roll
		if rand.Float64() < 0.25 {
			best.Rarity = content.RarityDivine
			best.Stats = best.Stats.Scaled(1.25)
			best.Name = "Divine " + best.Name
			msg = fmt.Sprintf("🌟 DIVINE ASCENSION! %s emerges (+25%% stats)!", best.Name)
		} else {
			best.Stats = best.Stats.Scaled(1.10)
			msg = fmt.Sprintf("⚗️ The fusion holds: %s keeps +10%% stats (no ascension this time).", best.Name)
		}
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
	s.bot.recordForge(uid, mode+" fusion", best.Name, fmt.Sprintf("%dg", cost))
	writeJSON(w, map[string]any{"ok": true, "msg": msg})
}

// ---- Deterministic legendary crafting (#120) --------------------------------

var craftLegendaryCost = map[string]int{"dust": 500, "shard": 50, "core": 10}

func (s *WebServer) handleAbyssCraftLegendary(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		GearID string `json:"gear_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	g, ok := content.GetGearByID(req.GearID)
	if !ok || g.Rarity != content.RarityLegendary {
		writeJSON(w, map[string]any{"ok": false, "error": "pick a Legendary catalog item"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !spendMaterials(tx, uid, craftLegendaryCost) {
		writeJSON(w, map[string]any{"ok": false, "error": "needs 500 Dust, 50 Shards and 10 Cores"})
		return
	}
	dataBytes, _ := json.Marshal(g)
	if _, err := tx.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability, item_data) VALUES ($1,$2,$3,$4)",
		uid, g.ID, g.MaxDurability, string(dataBytes)); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	s.bot.recordForge(uid, "legendary craft", g.Name, "500🌫️ 50🔷 10🟣")
	writeJSON(w, map[string]any{"ok": true, "msg": "⭐ Forged " + g.Name + " — delivered to your backpack.",
		"materials": s.bot.loadMaterials(uid)})
}

// ---- Corruption cleansing (#83) ---------------------------------------------

func (s *WebServer) handleAbyssCleanse(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		InvID int64  `json:"inv_id"`
		Slot  string `json:"slot"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	g, rawData, ok := loadForgeItem(tx, s.bot, uid, req.InvID, req.Slot)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	if !g.Corrupted {
		writeJSON(w, map[string]any{"ok": false, "error": "item is not corrupted"})
		return
	}
	cost := s.bot.forgeCost(uid, 800)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	s.bot.snapshotForgeUndo(tx, uid, req.InvID, req.Slot, rawData, "cleanse")
	g.Stats.HP += g.CorruptHP
	g.Corrupted = false
	g.CorruptHP = 0
	g.Name = strings.TrimPrefix(g.Name, "🩸 Corrupted ")
	dataBytes, _ := json.Marshal(g)
	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	s.bot.recordForge(uid, "cleanse", g.Name, fmt.Sprintf("%dg", cost))
	writeJSON(w, map[string]any{"ok": true, "msg": "✨ Corruption purged — the HP malus is lifted, the power remains."})
}

// ---- Repair all (#124) & auto-repair (#125) ---------------------------------

// abyssRepairAllCost prices a full repair: 2 gold per missing durability point.
func (b *Bot) abyssRepairAllCost(uid string) int64 {
	b.ensureGearMaxDurability(uid)
	var missing int64
	_ = b.DB.QueryRow("SELECT COALESCE(SUM(GREATEST("+gearMaxDurExpr+" - durability, 0)),0) FROM user_gear WHERE client_uid=$1", uid).Scan(&missing)
	return missing * 2
}

func (s *WebServer) handleAbyssRepairAll(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Preview bool `json:"preview"`
	}
	_ = readJSON(r, &req)

	cost := s.bot.abyssRepairAllCost(uid)
	if req.Preview {
		writeJSON(w, map[string]any{"ok": true, "cost": cost})
		return
	}
	if cost <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "nothing to repair"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !deductGold(w, tx, uid, cost) {
		return
	}
	if _, err := tx.Exec("UPDATE user_gear SET durability = "+gearMaxDurExpr+" WHERE client_uid=$1", uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("🛠️ All gear repaired for %dg.", cost), "gold": gold})
}

func (s *WebServer) handleAbyssAutoRepair(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		On bool `json:"on"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if _, err := s.bot.DB.Exec("UPDATE users SET abyss_auto_repair=$1 WHERE client_uid=$2", req.On, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	msg := "Auto-repair disabled."
	if req.On {
		msg = "Auto-repair enabled — gear is repaired (and charged) before every descent."
	}
	writeJSON(w, map[string]any{"ok": true, "msg": msg, "on": req.On})
}

// ---- Identify all (#99) ------------------------------------------------------

func (s *WebServer) handleAbyssIdentifyAll(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	type unidentified struct {
		invID int64
		slot  string
		g     content.Gear
	}
	var items []unidentified

	collect := func(query string, isInv bool) {
		rows, err := s.bot.DB.Query(query, uid)
		if err != nil {
			return
		}
		defer func() { _ = rows.Close() }()
		for rows.Next() {
			var invID int64
			var slot, gearID string
			var itemData sql.NullString
			var scanErr error
			if isInv {
				scanErr = rows.Scan(&invID, &gearID, &itemData)
			} else {
				scanErr = rows.Scan(&slot, &gearID, &itemData)
			}
			if scanErr != nil {
				continue
			}
			if g, ok := s.bot.makeGear(gearID, itemData); ok && g.Unidentified {
				items = append(items, unidentified{invID: invID, slot: slot, g: g})
			}
		}
	}
	collect("SELECT id, gear_id, item_data FROM user_inventory WHERE client_uid=$1", true)
	collect("SELECT slot, gear_id, item_data FROM user_gear WHERE client_uid=$1", false)

	if len(items) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "nothing to identify"})
		return
	}
	cost := int64(100 * len(items))
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !deductGold(w, tx, uid, cost) {
		return
	}
	for _, it := range items {
		it.g.Unidentified = false
		dataBytes, _ := json.Marshal(it.g)
		if !writeGearItemData(w, tx, uid, it.invID, it.slot, string(dataBytes)) {
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("🔍 Identified %d item(s) for %dg.", len(items), cost), "gold": gold})
}

// ---- Last Stand (#15) ---------------------------------------------------------

// abyssLastStandCost scales with depth so deep saves cost real tokens.
func abyssLastStandCost(depth int) int64 {
	return int64(5 + depth/5)
}

// handleAbyssLastStand: a downed player pays tokens to stand back up at 25% HP
// (+5% per Mercy level) on the same floor. Unlike the double-or-nothing revive
// it never risks the cache — but banking locks for the next 2 floors.
func (s *WebServer) handleAbyssLastStand(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || !run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "not downed"})
		return
	}
	if run.LastStandUsed {
		writeJSON(w, map[string]any{"ok": false, "error": "Last Stand already spent this run"})
		return
	}

	st := s.bot.loadAbyssStats(uid)
	cost := abyssLastStandCost(run.Depth)
	// Revive against the true combat max (base+gear+skill-web), matching handleAbyssRevive
	// and every other Abyss HP surface — calculateTotalStats alone omits the tree bonus.
	stats := s.bot.abyssCombatStats(uid)
	revivePct := 25 + 5*st.UpMercy
	reviveHP := stats.HP * revivePct / 100
	if reviveHP < 1 {
		reviveHP = 1
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !deductTokens(w, tx, uid, cost) {
		return
	}
	if _, err := tx.Exec("UPDATE users SET current_hp=$1 WHERE client_uid=$2", reviveHP, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("UPDATE abyss_active SET last_stand_used=TRUE, bank_locked_floors=2, floor_type='combat', last_action_at=NOW() WHERE client_uid=$1", uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "hp": reviveHP, "max_hp": stats.HP, "tokens": s.bot.abyssTokens(uid),
		"msg": fmt.Sprintf("🛡️ LAST STAND! You rise at %d%% HP — but the exit is sealed for the next 2 floors.", revivePct),
	})
}

// ---- Specializations (#161) ----------------------------------------------------

var abyssSpecs = map[string]string{
	"delver":    "Delver — +10% floor reward XP",
	"plunderer": "Plunderer — +10% escrow floor bonus",
	"warden":    "Warden — +5% all stats inside the Abyss",
}

func (b *Bot) abyssSpec(uid string) string {
	var spec string
	_ = b.DB.QueryRow("SELECT abyss_spec FROM users WHERE client_uid=$1", uid).Scan(&spec)
	return spec
}

func (s *WebServer) handleAbyssSetSpec(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Spec string `json:"spec"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if _, ok := abyssSpecs[req.Spec]; !ok && req.Spec != "" {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown specialization"})
		return
	}

	cur := s.bot.abyssSpec(uid)
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	// First pick is free and clearing is free; only swapping one spec for
	// another costs tokens.
	if cur != "" && req.Spec != "" && req.Spec != cur {
		if !deductTokens(w, tx, uid, 25) {
			return
		}
	}
	if _, err := tx.Exec("UPDATE users SET abyss_spec=$1 WHERE client_uid=$2", req.Spec, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	msg := "Specialization cleared."
	if req.Spec != "" {
		msg = "Specialization set: " + abyssSpecs[req.Spec]
	}
	writeJSON(w, map[string]any{"ok": true, "msg": msg, "spec": req.Spec, "tokens": s.bot.abyssTokens(uid)})
}

// ---- Sanctuary upgrades (#38, #113) ---------------------------------------------

type sanctuaryUpgrade struct {
	Key    string
	Name   string
	Desc   string
	Max    int
	CostTk int64 // per level
}

var sanctuaryUpgrades = []sanctuaryUpgrade{
	{"heal", "Warm Hearth", "Rest-floor healing costs 25% less per level", 3, 15},
	{"repair", "Anvil Blessing", "Rest-floor repairs cost 25% less per level", 3, 15},
	{"forge", "Crafting Station", "Unlocks a free full repair once per rest floor", 1, 40},
}

func (b *Bot) loadSanctuary(uid string) map[string]int {
	out := map[string]int{}
	var raw sql.NullString
	_ = b.DB.QueryRow("SELECT abyss_sanctuary::text FROM users WHERE client_uid=$1", uid).Scan(&raw)
	if raw.Valid && raw.String != "" {
		_ = json.Unmarshal([]byte(raw.String), &out)
	}
	return out
}

func (s *WebServer) handleAbyssSanctuaryBuy(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Key string `json:"key"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	var up *sanctuaryUpgrade
	for i := range sanctuaryUpgrades {
		if sanctuaryUpgrades[i].Key == req.Key {
			up = &sanctuaryUpgrades[i]
		}
	}
	if up == nil {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown upgrade"})
		return
	}
	sanct := s.bot.loadSanctuary(uid)
	if sanct[up.Key] >= up.Max {
		writeJSON(w, map[string]any{"ok": false, "error": "already at max level"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !deductTokens(w, tx, uid, up.CostTk) {
		return
	}
	sanct[up.Key]++
	buf, _ := json.Marshal(sanct)
	if _, err := tx.Exec("UPDATE users SET abyss_sanctuary=$1::jsonb WHERE client_uid=$2", string(buf), uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("🕊️ %s upgraded to level %d!", up.Name, sanct[up.Key]),
		"tokens": s.bot.abyssTokens(uid), "sanctuary": sanct})
}

// ---- Forge history endpoint (#123) ------------------------------------------------

func (s *WebServer) handleAbyssForgeHistory(w http.ResponseWriter, r *http.Request, uid string) {
	rows := s.bot.loadForgeHistory(uid, 20)
	out := make([]map[string]string, 0, len(rows))
	for _, v := range rows {
		out = append(out, map[string]string{"action": v.Action, "detail": v.Detail, "cost": v.Cost, "when": v.When})
	}
	writeJSON(w, map[string]any{"ok": true, "history": out})
}

// ---- Unequip / auto-equip undo (UX-53) ----------------------------------------------

// handleAbyssUnequip moves an equipped item back to the backpack. Used by the
// 10-second "undo" link on auto-equip loot notices: auto-equip only fires on a
// previously-empty slot, so moving the piece to the backpack is a true undo.
func (s *WebServer) handleAbyssUnequip(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Slot   string `json:"slot"`
		GearID string `json:"gear_id"` // optional: bind the undo to the exact auto-equipped item
	}
	if err := readJSON(r, &req); err != nil || req.Slot == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var dura int
	var itemData sql.NullString
	if err := tx.QueryRow("SELECT gear_id, durability, item_data FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, req.Slot).Scan(&gearID, &dura, &itemData); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "nothing equipped in that slot"})
		return
	}
	// When the caller pinned a specific item (the auto-equip undo link), refuse
	// if the slot's occupant changed since — the undo must never strip a
	// different piece the player equipped in the meantime.
	if req.GearID != "" && req.GearID != gearID {
		writeJSON(w, map[string]any{"ok": false, "error": "that slot holds a different item now — nothing was moved"})
		return
	}
	if _, err := tx.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability, item_data) VALUES ($1,$2,$3,$4)", uid, gearID, dura, itemData); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("DELETE FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, req.Slot); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": "↩️ Moved the " + req.Slot + " piece to your backpack."})
}

// ---- Bestiary mastery (#168) --------------------------------------------------------

// bestiaryMasteryFamilies counts mob families with 100+ recorded kills; each
// grants +1% STR inside the Abyss (capped at +10%).
func (b *Bot) bestiaryMasteryFamilies(uid string) int {
	var n int
	_ = b.DB.QueryRow("SELECT COUNT(*) FROM abyss_bestiary WHERE client_uid=$1 AND kills >= 100", uid).Scan(&n)
	return n
}

// ---- Prestige tiers (#19) -----------------------------------------------------------

// abyssPrestigeTier maps prestige count to a display tier name + aura.
func abyssPrestigeTier(p int) (string, string) {
	switch {
	case p >= 5:
		return "Void Sovereign", "🌌"
	case p >= 4:
		return "Umbral Lord", "🟣"
	case p >= 3:
		return "Nadir Walker", "🔮"
	case p >= 2:
		return "Deep Warden", "💠"
	case p >= 1:
		return "Abyss Touched", "✨"
	}
	return "", ""
}

// ---- Rift peek (#35) + floor queue -------------------------------------------------

// handleAbyssRiftPeek pre-rolls the next floors so the player sees what's
// coming. Cartographer (#157) reveals one extra floor per level and cheapens it.
func (s *WebServer) handleAbyssRiftPeek(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.FloorType != "event" || !strings.Contains(run.EventState, `"rift"`) {
		writeJSON(w, map[string]any{"ok": false, "error": "no rift on this floor"})
		return
	}
	st := s.bot.loadAbyssStats(uid)
	n := 3 + st.UpCartographer
	cost := int64(50 * (run.Depth + 1))
	cost -= cost * int64(st.UpCartographer) / 10
	if cost < 10 {
		cost = 10
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if !deductGold(w, tx, uid, cost) {
		return
	}
	queue := make([]string, 0, n)
	for i := 0; i < n; i++ {
		// Boss floors are fixed; everything else uses the standard odds.
		if (run.Depth+1+i)%abyssBossEvery == 0 {
			queue = append(queue, "combat")
			continue
		}
		c := rollFloorCandidates(1)
		queue = append(queue, c[0].Type)
	}
	buf, _ := json.Marshal(queue)
	if _, err := tx.Exec("UPDATE abyss_active SET floor_queue=$1 WHERE client_uid=$2", string(buf), uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	labels := make([]string, len(queue))
	for i, t := range queue {
		info := floorCandidateInfo[t]
		labels[i] = fmt.Sprintf("Floor %d: %s %s", run.Depth+1+i, info.Icon, info.Label)
	}
	writeJSON(w, map[string]any{"ok": true, "gold": gold, "peek": labels,
		"msg": "👁️ The rift shows what waits below. These floors are now sealed to your fate."})
}

// ---- Gear XP (#108) ------------------------------------------------------------

// gearXPMilestones bake +5% stats into the weapon at these kill counts.
var gearXPMilestones = []int{100, 250, 500}

// tickGearXP advances the equipped MainHand weapon's kill count after a floor
// victory and bakes a +5% stat milestone when one is crossed. Returns a
// player-facing milestone message, or "".
func (b *Bot) tickGearXP(uid string) string {
	var gearID string
	var itemData sql.NullString
	if err := b.DB.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE client_uid=$1 AND slot='MainHand'", uid).Scan(&gearID, &itemData); err != nil {
		return ""
	}
	g, ok := b.makeGear(gearID, itemData)
	if !ok {
		return ""
	}
	g.KillCount++
	msg := ""
	if g.MilestoneTier < len(gearXPMilestones) && g.KillCount >= gearXPMilestones[g.MilestoneTier] {
		g.MilestoneTier++
		g.Stats = g.Stats.Scaled(1.05)
		msg = fmt.Sprintf("⚔️ %s reached %d kills — it grows stronger (+5%% stats, tier %d/%d)!",
			g.Name, g.KillCount, g.MilestoneTier, len(gearXPMilestones))
	}
	dataBytes, _ := json.Marshal(g)
	_, _ = b.DB.Exec("UPDATE user_gear SET item_data=$1 WHERE client_uid=$2 AND slot='MainHand'", string(dataBytes), uid)
	return msg
}

// popFloorQueue consumes the head of the pre-rolled floor queue, if any.
func (b *Bot) popFloorQueue(uid string) (string, bool) {
	var raw sql.NullString
	_ = b.DB.QueryRow("SELECT floor_queue::text FROM abyss_active WHERE client_uid=$1", uid).Scan(&raw)
	if !raw.Valid || raw.String == "" {
		return "", false
	}
	var queue []string
	if err := json.Unmarshal([]byte(raw.String), &queue); err != nil || len(queue) == 0 {
		return "", false
	}
	head := queue[0]
	rest := queue[1:]
	if len(rest) == 0 {
		_, _ = b.DB.Exec("UPDATE abyss_active SET floor_queue=NULL WHERE client_uid=$1", uid)
	} else {
		buf, _ := json.Marshal(rest)
		_, _ = b.DB.Exec("UPDATE abyss_active SET floor_queue=$1 WHERE client_uid=$2", string(buf), uid)
	}
	return head, true
}
