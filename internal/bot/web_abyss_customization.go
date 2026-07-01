package bot

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

// writeGearItemData persists updated item JSON to the correct table (inventory by
// id, or equipped gear by slot), always scoped to the owning client_uid. On a DB
// error it writes a JSON error response and returns false so the caller aborts the
// transaction before charging the player.
func writeGearItemData(w http.ResponseWriter, tx *sql.Tx, uid string, invID int64, slot, data string) bool {
	var err error
	if invID > 0 {
		_, err = tx.Exec("UPDATE user_inventory SET item_data=$1 WHERE id=$2 AND client_uid=$3", data, invID, uid)
	} else {
		_, err = tx.Exec("UPDATE user_gear SET item_data=$1 WHERE slot=$2 AND client_uid=$3", data, slot, uid)
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return false
	}
	return true
}

// deductGold debits the player's gold with a balance-guarded UPDATE so the charge
// commits atomically with the item change and can never overdraw. It writes an
// error response and returns false if the debit errored or the player can't afford it.
func deductGold(w http.ResponseWriter, tx *sql.Tx, uid string, cost int64) bool {
	res, err := tx.Exec("UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1", cost, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return false
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return false
	}
	return true
}

// deductTokens debits Abyss tokens with the same balance guard as deductGold.
func deductTokens(w http.ResponseWriter, tx *sql.Tx, uid string, cost int64) bool {
	res, err := tx.Exec("UPDATE users SET abyss_tokens = abyss_tokens - $1 WHERE client_uid=$2 AND abyss_tokens >= $1", cost, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return false
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens"})
		return false
	}
	return true
}

// handleAbyssIdentify spends 100 gold to identify an item in inventory or equipped.
func (s *WebServer) handleAbyssIdentify(w http.ResponseWriter, r *http.Request, uid string) {
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

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	if gold < 100 {
		writeJSON(w, map[string]any{"ok": false, "error": "Not enough gold (requires 100 gold)"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var g content.Gear
	var gearID string
	var itemData sql.NullString
	var queryErr error

	if req.InvID > 0 {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2", req.Slot, uid).Scan(&gearID, &itemData)
	} else {
		writeJSON(w, map[string]any{"ok": false, "error": "missing item specifier"})
		return
	}

	if queryErr != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}

	g, ok := s.bot.makeGear(gearID, itemData)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown item"})
		return
	}

	if !g.Unidentified {
		writeJSON(w, map[string]any{"ok": false, "error": "item is already identified"})
		return
	}

	g.Unidentified = false
	dataBytes, _ := json.Marshal(g)

	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if !deductGold(w, tx, uid, 100) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "msg": "Item successfully identified!", "gold": gold - 100})
}

// handleAbyssSocketGem spends 50 gold to socket a gemstone.
func (s *WebServer) handleAbyssSocketGem(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		InvID int64  `json:"inv_id"`
		Slot  string `json:"slot"`
		Gem   string `json:"gem"` // Ruby, Sapphire, Emerald, Diamond, Topaz
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	gem := strings.TrimSpace(req.Gem)
	gemStats := content.Stats{}
	valid := false
	switch gem {
	case "Ruby":
		gemStats.HP = 100
		valid = true
	case "Sapphire":
		gemStats.MNA = 50
		valid = true
	case "Emerald":
		gemStats.STR = 15
		valid = true
	case "Diamond":
		gemStats.DEF = 15
		valid = true
	case "Topaz":
		gemStats.CRT = 5
		valid = true
	}

	if !valid {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid gemstone type"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	if gold < 50 {
		writeJSON(w, map[string]any{"ok": false, "error": "Not enough gold (requires 50 gold)"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	var queryErr error

	if req.InvID > 0 {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2", req.Slot, uid).Scan(&gearID, &itemData)
	} else {
		writeJSON(w, map[string]any{"ok": false, "error": "missing item specifier"})
		return
	}

	if queryErr != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}

	g, ok := s.bot.makeGear(gearID, itemData)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown item"})
		return
	}

	if g.Unidentified {
		writeJSON(w, map[string]any{"ok": false, "error": "identify the item first"})
		return
	}

	if g.Sockets <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "this item has no gemstone sockets"})
		return
	}

	if len(g.Gemstones) >= g.Sockets {
		writeJSON(w, map[string]any{"ok": false, "error": "no empty sockets available"})
		return
	}

	g.Gemstones = append(g.Gemstones, gem)
	g.Stats = g.Stats.Add(gemStats)

	dataBytes, _ := json.Marshal(g)

	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if !deductGold(w, tx, uid, 50) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Successfully socketed %s into your gear!", gem), "gold": gold - 50})
}

// handleAbyssEtchRune spends 150 gold to etch an elemental rune.
func (s *WebServer) handleAbyssEtchRune(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		InvID int64  `json:"inv_id"`
		Slot  string `json:"slot"`
		Rune  string `json:"rune"` // Fire, Water, Earth, Air, Physical
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	runeType := strings.TrimSpace(req.Rune)
	valid := runeType == "Fire" || runeType == "Water" || runeType == "Earth" || runeType == "Air" || runeType == "Physical"
	if !valid {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid rune element"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	if gold < 150 {
		writeJSON(w, map[string]any{"ok": false, "error": "Not enough gold (requires 150 gold)"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	var queryErr error

	if req.InvID > 0 {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2", req.Slot, uid).Scan(&gearID, &itemData)
	} else {
		writeJSON(w, map[string]any{"ok": false, "error": "missing item specifier"})
		return
	}

	if queryErr != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}

	g, ok := s.bot.makeGear(gearID, itemData)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown item"})
		return
	}

	if g.Unidentified {
		writeJSON(w, map[string]any{"ok": false, "error": "identify the item first"})
		return
	}

	isWeapon := g.Slot == content.SlotMainHand || g.Slot == content.SlotOffHand || g.Slot == content.SlotRanged
	if !isWeapon {
		writeJSON(w, map[string]any{"ok": false, "error": "runes can only be etched on weapons or shields"})
		return
	}

	g.Rune = runeType
	g.Element = content.Element(runeType)

	dataBytes, _ := json.Marshal(g)

	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if !deductGold(w, tx, uid, 150) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Etched %s Rune into your weapon!", runeType), "gold": gold - 150})
}

// handleAbyssRecalibrate spends 5 tokens to reroll a single stat on Legendary+ gear.
func (s *WebServer) handleAbyssRecalibrate(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		InvID int64  `json:"inv_id"`
		Slot  string `json:"slot"`
		Stat  string `json:"stat"` // HP, MNA, STR, DEF, SPD, LCK, INT, STA, CRT, DGE
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	stat := req.Stat
	if stat != "HP" && stat != "MNA" && stat != "STR" && stat != "DEF" && stat != "SPD" && stat != "LCK" && stat != "INT" && stat != "STA" && stat != "CRT" && stat != "DGE" {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid stat"})
		return
	}

	var tokens int64
	_ = s.bot.DB.QueryRow("SELECT abyss_tokens FROM users WHERE client_uid=$1", uid).Scan(&tokens)
	if tokens < 5 {
		writeJSON(w, map[string]any{"ok": false, "error": "Not enough tokens (requires 5 tokens)"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	var queryErr error

	if req.InvID > 0 {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2", req.Slot, uid).Scan(&gearID, &itemData)
	} else {
		writeJSON(w, map[string]any{"ok": false, "error": "missing item specifier"})
		return
	}

	if queryErr != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}

	g, ok := s.bot.makeGear(gearID, itemData)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown item"})
		return
	}

	if g.Unidentified {
		writeJSON(w, map[string]any{"ok": false, "error": "identify the item first"})
		return
	}

	if g.Rarity < content.RarityLegendary {
		writeJSON(w, map[string]any{"ok": false, "error": "recalibration requires Legendary-or-better gear"})
		return
	}

	// Recalibrate stat randomly
	switch stat {
	case "HP":
		g.Stats.HP = 200 + rand.IntN(400)
	case "MNA":
		g.Stats.MNA = 40 + rand.IntN(110)
	case "STR":
		g.Stats.STR = 40 + rand.IntN(80)
	case "DEF":
		g.Stats.DEF = 20 + rand.IntN(60)
	case "SPD":
		g.Stats.SPD = 20 + rand.IntN(60)
	case "LCK":
		g.Stats.LCK = 15 + rand.IntN(40)
	case "INT":
		g.Stats.INT = 15 + rand.IntN(45)
	case "STA":
		g.Stats.STA = 10 + rand.IntN(30)
	case "CRT":
		g.Stats.CRT = 5 + rand.IntN(15)
	case "DGE":
		g.Stats.DGE = 5 + rand.IntN(15)
	}

	dataBytes, _ := json.Marshal(g)

	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if !deductTokens(w, tx, uid, 5) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Successfully recalibrated %s stat!", stat), "tokens": tokens - 5})
}

// handleAbyssUpgradeGear spends 10 tokens to upgrade weapon tier.
func (s *WebServer) handleAbyssUpgradeGear(w http.ResponseWriter, r *http.Request, uid string) {
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

	var tokens int64
	_ = s.bot.DB.QueryRow("SELECT abyss_tokens FROM users WHERE client_uid=$1", uid).Scan(&tokens)
	if tokens < 10 {
		writeJSON(w, map[string]any{"ok": false, "error": "Not enough tokens (requires 10 tokens)"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	var queryErr error

	if req.InvID > 0 {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2", req.Slot, uid).Scan(&gearID, &itemData)
	} else {
		writeJSON(w, map[string]any{"ok": false, "error": "missing item specifier"})
		return
	}

	if queryErr != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}

	g, ok := s.bot.makeGear(gearID, itemData)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown item"})
		return
	}

	if g.Unidentified {
		writeJSON(w, map[string]any{"ok": false, "error": "identify the item first"})
		return
	}

	if g.Rarity >= content.RarityDivine {
		writeJSON(w, map[string]any{"ok": false, "error": "item is already at max rarity (Divine)"})
		return
	}

	g.Rarity = g.Rarity + 1
	g.Stats = g.Stats.Scaled(1.3) // +30% stats
	g.GearLevel++

	dataBytes, _ := json.Marshal(g)

	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if !deductTokens(w, tx, uid, 10) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Upgraded weapon to %s tier (+30%% stats)!", g.Rarity.String()), "tokens": tokens - 10})
}

// handleAbyssTransmute converts a weapon into a class-suitable random weapon.
func (s *WebServer) handleAbyssTransmute(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		InvID int64 `json:"inv_id"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	if gold < 100 {
		writeJSON(w, map[string]any{"ok": false, "error": "Not enough gold (requires 100 gold)"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	if err := tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gearID, &itemData); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}

	g, ok := s.bot.makeGear(gearID, itemData)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown item"})
		return
	}

	isWeapon := g.Slot == content.SlotMainHand || g.Slot == content.SlotOffHand || g.Slot == content.SlotRanged
	if !isWeapon {
		writeJSON(w, map[string]any{"ok": false, "error": "transmutation requires a weapon"})
		return
	}

	// Transmutation rebuilds the weapon from a fresh catalog base, which cannot carry
	// over per-item customization. Refuse when the source carries any such state rather
	// than silently destroying gemstones, runes, insurance, affixes or its identity.
	if len(g.Gemstones) > 0 || g.Rune != "" || g.Insured || g.Cursed || g.Eldritch || g.Unidentified {
		writeJSON(w, map[string]any{"ok": false, "error": "cannot transmute customized gear (gems, runes, insurance, or affixes)"})
		return
	}

	// Load user stats to select suitable class weapon
	userStats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())
	
	// Determine weapon pool based on highest stat
	var selected content.Gear
	weaponPool := []string{}

	if userStats.INT > userStats.STR && userStats.INT > userStats.SPD {
		// Mage offhands/weapons
		weaponPool = []string{"ABYSS_TIDAL_SCEPTER", "ABYSS_LIFEBLOOM_STAFF", "ABYSS_MANA_BATTERY"}
	} else if userStats.SPD > userStats.STR {
		// Rogue/ranger weapons
		weaponPool = []string{"ABYSS_ZEPHYR_DAGGER", "ABYSS_NECROTIC_DAGGER", "ABYSS_RANGED", "ABYSS_CRYSTALLINE_DAGGER"}
	} else {
		// Warrior weapons
		weaponPool = []string{"ABYSS_FIREBRAND_SWORD", "ABYSS_EARTHSHAKER_HAMMER", "ABYSS_RUNE_CLAYMORE", "ABYSS_WYRM_TOOTH"}
	}

	selectedID := weaponPool[rand.IntN(len(weaponPool))]
	selected, ok = content.GetGearByID(selectedID)
	if !ok {
		selected = g
	}

	selected.Rarity = g.Rarity
	selected.GearLevel = g.GearLevel
	selected.Sockets = g.Sockets

	dataBytes, _ := json.Marshal(selected)
	if _, err := tx.Exec("UPDATE user_inventory SET gear_id=$1, item_data=$2 WHERE id=$3 AND client_uid=$4", selected.ID, string(dataBytes), req.InvID, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if !deductGold(w, tx, uid, 100) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Transmuted weapon into a suitable %s!", selected.Name), "gold": gold - 100})
}

// handleAbyssConvertMana converts excess mana (INT stat / converts 2:1 to HP).
func (s *WebServer) handleAbyssConvertMana(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Amount int `json:"amount"` // Mana to convert
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	if req.Amount <= 0 || req.Amount%2 != 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "amount must be positive and divisible by 2"})
		return
	}

	// Convert 2 Mana to 1 Max HP
	hpGain := req.Amount / 2

	var upgradesJSON sql.NullString
	_ = s.bot.DB.QueryRow("SELECT abyss_upgrades FROM users WHERE client_uid=$1", uid).Scan(&upgradesJSON)

	upgrades := make(map[string]int)
	if upgradesJSON.Valid && upgradesJSON.String != "" {
		_ = json.Unmarshal([]byte(upgradesJSON.String), &upgrades)
	}

	// Check if user has enough converted mana stats
	userStats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())
	if userStats.MNA < req.Amount {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough mana stats to convert"})
		return
	}

	upgrades["converted_hp"] += hpGain
	upgrades["converted_mana_reduction"] += req.Amount

	upgradesBytes, _ := json.Marshal(upgrades)
	_, err := s.bot.DB.Exec("UPDATE users SET abyss_upgrades=$1 WHERE client_uid=$2", string(upgradesBytes), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db update"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Converted %d mana into +%d Max HP!", req.Amount, hpGain)})
}

// handleAbyssResetTalents resets all Deep-Delver upgrades and refunds their tokens.
func (s *WebServer) handleAbyssResetTalents(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var upVigor, upGreed, upFortune, upWard, upInterest, upTribute, upInsight int
	var tokens int64
	err = tx.QueryRow(`SELECT abyss_up_vigor, abyss_up_greed, abyss_up_fortune, abyss_up_ward,
	                          abyss_up_interest, abyss_up_tribute, abyss_up_insight, abyss_tokens 
	                     FROM users WHERE client_uid=$1`, uid).Scan(
		&upVigor, &upGreed, &upFortune, &upWard, &upInterest, &upTribute, &upInsight, &tokens,
	)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "user not found"})
		return
	}

	// Spend sum calculation
	calcRefund := func(level int) int64 {
		sum := int64(0)
		for l := 1; l <= level; l++ {
			sum += int64(l) * 10
		}
		return sum
	}

	refund := calcRefund(upVigor) + calcRefund(upGreed) + calcRefund(upFortune) + calcRefund(upWard) +
		calcRefund(upInterest) + calcRefund(upTribute) + calcRefund(upInsight)

	if refund <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "no talent points to reset"})
		return
	}

	// Mana-conversion state (converted_hp / converted_mana_reduction) lives in
	// abyss_upgrades but is not a Deep-Delver talent, so a talent reset must not wipe
	// it. Rebuild the JSON from just those keys instead of blanking the whole column.
	var upgradesJSON sql.NullString
	_ = tx.QueryRow("SELECT abyss_upgrades FROM users WHERE client_uid=$1", uid).Scan(&upgradesJSON)
	upgrades := map[string]int{}
	if upgradesJSON.Valid && upgradesJSON.String != "" {
		_ = json.Unmarshal([]byte(upgradesJSON.String), &upgrades)
	}
	preserved := map[string]int{}
	for _, k := range []string{"converted_hp", "converted_mana_reduction"} {
		if v, ok := upgrades[k]; ok {
			preserved[k] = v
		}
	}
	preservedBytes, _ := json.Marshal(preserved)

	// Reset columns
	_, err = tx.Exec(`UPDATE users
	                     SET abyss_up_vigor=0, abyss_up_greed=0, abyss_up_fortune=0, abyss_up_ward=0,
	                         abyss_up_interest=0, abyss_up_tribute=0, abyss_up_insight=0,
	                         abyss_tokens = abyss_tokens + $1, abyss_upgrades = $3::jsonb
	                   WHERE client_uid=$2`, refund, uid, string(preservedBytes))
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db update"})
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Talents reset successfully! Refunded %d tokens.", refund), "tokens": tokens + refund})
}

// handleAbyssInsureItem spends 200 gold to permanently mark a gear piece as insured.
func (s *WebServer) handleAbyssInsureItem(w http.ResponseWriter, r *http.Request, uid string) {
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

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	if gold < 200 {
		writeJSON(w, map[string]any{"ok": false, "error": "Not enough gold (requires 200 gold)"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	var queryErr error

	if req.InvID > 0 {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2", req.Slot, uid).Scan(&gearID, &itemData)
	} else {
		writeJSON(w, map[string]any{"ok": false, "error": "missing item specifier"})
		return
	}

	if queryErr != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}

	g, ok := s.bot.makeGear(gearID, itemData)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown item"})
		return
	}

	if g.Unidentified {
		writeJSON(w, map[string]any{"ok": false, "error": "identify the item first"})
		return
	}

	if g.Insured {
		writeJSON(w, map[string]any{"ok": false, "error": "item is already insured"})
		return
	}

	g.Insured = true
	dataBytes, _ := json.Marshal(g)

	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if !deductGold(w, tx, uid, 200) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "msg": "Item marked as Insured! It will no longer take durability loss.", "gold": gold - 200})
}
