package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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

// handleAbyssIdentify identifies an inventory or equipped item. The first
// successful identification each UTC day is free.
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

	tx, err := s.beginForgeRequestTx(w)
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
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2 FOR UPDATE", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2 FOR UPDATE", req.Slot, uid).Scan(&gearID, &itemData)
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
	dataBytes, err := json.Marshal(g)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	normalCost := s.bot.forgeGoldCost(uid, 100, g.Rarity)
	cost, dailyFree, chargeOK := s.dailyIdentifyCharge(w, r, tx, uid, normalCost, normalCost)
	if !chargeOK {
		return
	}
	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if cost > 0 && !deductGold(w, tx, uid, cost) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	msg := "Item successfully identified!"
	if dailyFree {
		msg = "Daily free identification used — item successfully identified!"
	}
	writeJSON(w, map[string]any{"ok": true, "msg": msg, "gold": gold, "cost": cost, "daily_free": dailyFree})
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

	tx, err := s.beginForgeRequestTx(w)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	var queryErr error

	if req.InvID > 0 {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2 FOR UPDATE", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2 FOR UPDATE", req.Slot, uid).Scan(&gearID, &itemData)
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

	dataBytes, err := json.Marshal(g)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if !deductGold(w, tx, uid, s.bot.forgeGoldCost(uid, 50, g.Rarity)) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Successfully socketed %s into your gear!", gem), "gold": gold})
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
		InvID  int64  `json:"inv_id"`
		Slot   string `json:"slot"`
		Rune   string `json:"rune"` // Fire, Water, Earth, Air, Physical
		Family string `json:"rune_family"`
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
	defensive := strings.EqualFold(strings.TrimSpace(req.Family), "defensive")
	storedRune := runeType
	if defensive {
		storedRune = content.DefensiveRuneName(content.Element(runeType))
	}

	// Rune library (#118): once a rune type has been etched, re-etching it
	// anywhere costs a third of the price. A failed lookup must abort rather than
	// default to first-time pricing and overcharge an already-known rune.
	var known bool
	if err := s.bot.DB.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM user_runes WHERE client_uid=$1 AND rune=$2)", uid, storedRune).Scan(&known); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	costBase := int64(150)
	if known {
		costBase = 50
	}

	tx, err := s.beginForgeRequestTx(w)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	var queryErr error

	if req.InvID > 0 {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2 FOR UPDATE", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2 FOR UPDATE", req.Slot, uid).Scan(&gearID, &itemData)
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
	if !defensive && !isWeapon {
		writeJSON(w, map[string]any{"ok": false, "error": "runes can only be etched on weapons or shields"})
		return
	}

	g.Rune = storedRune
	g.Element = content.Element(runeType)

	dataBytes, err := json.Marshal(g)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	cost := s.bot.forgeGoldCost(uid, costBase, g.Rarity)
	if !deductGold(w, tx, uid, cost) {
		return
	}
	if _, err := tx.Exec("INSERT INTO user_runes (client_uid, rune) VALUES ($1,$2) ON CONFLICT DO NOTHING", uid, storedRune); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	s.bot.recordForge(uid, "etch", storedRune+" rune", fmt.Sprintf("%dg", cost))
	msg := fmt.Sprintf("Etched %s Rune into your gear!", storedRune)
	if !known {
		msg += " 📖 Rune learned — re-etching it now costs only 50g."
	}
	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{"ok": true, "msg": msg, "gold": gold})
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
		InvID     int64  `json:"inv_id"`
		Slot      string `json:"slot"`
		Stat      string `json:"stat"` // HP, MNA, STR, DEF, SPD, LCK, INT, STA, CRT, DGE
		MaxTokens int64 `json:"max_tokens"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if req.MaxTokens > 0 && req.MaxTokens < 5 {
		writeJSON(w, map[string]any{"ok": false, "error": "recalibration exceeds the requested token cap"})
		return
	}

	stat := req.Stat
	if stat != "HP" && stat != "MNA" && stat != "STR" && stat != "DEF" && stat != "SPD" && stat != "LCK" && stat != "INT" && stat != "STA" && stat != "CRT" && stat != "DGE" {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid stat"})
		return
	}

	tx, err := s.beginForgeRequestTx(w)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	var queryErr error

	if req.InvID > 0 {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2 FOR UPDATE", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2 FOR UPDATE", req.Slot, uid).Scan(&gearID, &itemData)
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
		g.Stats.HP = 200 + rand.IntN(400) // #nosec G404 -- non-cryptographic stat reroll
	case "MNA":
		g.Stats.MNA = 40 + rand.IntN(110) // #nosec G404 -- non-cryptographic stat reroll
	case "STR":
		g.Stats.STR = 40 + rand.IntN(80) // #nosec G404 -- non-cryptographic stat reroll
	case "DEF":
		g.Stats.DEF = 20 + rand.IntN(60) // #nosec G404 -- non-cryptographic stat reroll
	case "SPD":
		g.Stats.SPD = 20 + rand.IntN(60) // #nosec G404 -- non-cryptographic stat reroll
	case "LCK":
		g.Stats.LCK = 15 + rand.IntN(40) // #nosec G404 -- non-cryptographic stat reroll
	case "INT":
		g.Stats.INT = 15 + rand.IntN(45) // #nosec G404 -- non-cryptographic stat reroll
	case "STA":
		g.Stats.STA = 10 + rand.IntN(30) // #nosec G404 -- non-cryptographic stat reroll
	case "CRT":
		g.Stats.CRT = 5 + rand.IntN(15) // #nosec G404 -- non-cryptographic stat reroll
	case "DGE":
		g.Stats.DGE = 5 + rand.IntN(15) // #nosec G404 -- non-cryptographic stat reroll
	}

	dataBytes, err := json.Marshal(g)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

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

	s.bot.recordForge(uid, "recalibrate", stat+" stat", "5 tokens")
	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Successfully recalibrated %s stat!", stat), "tokens": s.bot.abyssTokens(uid)})
}

// abyssUpgradeGearCost is the Abyss-token cost to ascend a piece of gear to the
// given target rarity. Ascending into the top tiers costs dramatically more, so a
// Mythic/Divine piece is a real long-term goal rather than a cheap token dump.
func abyssUpgradeGearCost(target content.Rarity) int64 {
	switch target {
	case content.RarityEternal:
		return 800
	case content.RarityCelestial:
		return 400
	case content.RarityDivine:
		return 200
	case content.RarityMythic:
		return 80
	case content.RarityLegendary:
		return 30
	case content.RarityEpic:
		return 15
	default:
		return 10
	}
}

// handleAbyssUpgradeGear spends Abyss tokens to ascend a piece of gear one rarity
// tier. Cost scales with the target tier, and Mythic/Divine ascensions also roll
// extra bonus combat affixes onto the item.
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

	tx, err := s.beginForgeRequestTx(w)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	var queryErr error

	if req.InvID > 0 {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2 FOR UPDATE", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2 FOR UPDATE", req.Slot, uid).Scan(&gearID, &itemData)
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

	if g.Rarity >= content.RarityEternal {
		writeJSON(w, map[string]any{"ok": false, "error": "item is already at max rarity (Eternal)"})
		return
	}

	target := g.Rarity + 1
	cost := abyssUpgradeGearCost(target)

	g.Rarity = target
	g.Stats = g.Stats.Scaled(1.3) // +30% stats
	g.GearLevel++

	// Ascending into the top tiers imbues extra bonus combat affixes: Mythic gains
	// one, Divine two, Celestial two, Eternal three, so the very best gear is
	// meaningfully more powerful.
	added := 0
	switch target {
	case content.RarityMythic:
		added = 1
	case content.RarityDivine:
		added = 2
	case content.RarityCelestial:
		added = 2
	case content.RarityEternal:
		added = 3
	}
	if added > 0 {
		before := len(g.BonusEffects)
		content.AddBonusEffects(&g, added, rand.IntN)
		added = len(g.BonusEffects) - before
	}

	dataBytes, err := json.Marshal(g)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if !deductTokens(w, tx, uid, cost) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	msg := fmt.Sprintf("Upgraded gear to %s tier (+30%% stats)!", g.Rarity.String())
	if added > 0 {
		effNames := make([]string, 0, added)
		for _, e := range g.BonusEffects[len(g.BonusEffects)-added:] {
			effNames = append(effNames, string(e))
		}
		msg += " New bonus effect(s): " + strings.Join(effNames, ", ")
	}
	writeJSON(w, map[string]any{"ok": true, "msg": msg, "tokens": s.bot.abyssTokens(uid)})
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

	tx, err := s.beginForgeRequestTx(w)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	if err := tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2 FOR UPDATE", req.InvID, uid).Scan(&gearID, &itemData); err != nil {
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
	var weaponPool []string

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

	// #nosec G404 -- non-cryptographic weapon pool pick
	selectedID := weaponPool[rand.IntN(len(weaponPool))]
	selected, ok = content.GetGearByID(selectedID)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "transmute failed"})
		return
	}

	// A fresh catalog base carries no per-level scaling or ascension multiplier, so
	// transmuting an ascended piece would silently strip that power. Refuse instead.
	if selected.GearLevel != g.GearLevel {
		writeJSON(w, map[string]any{"ok": false, "error": "cannot transmute ascended gear (gear level cannot be preserved)"})
		return
	}
	// Attunement is a binding contract: transmuting into a fresh catalog base
	// would silently drop the flag and unbind the item. Refuse instead.
	if g.Attuned {
		writeJSON(w, map[string]any{"ok": false, "error": g.Name + " is attuned to you and cannot be transmuted"})
		return
	}

	selected.Rarity = g.Rarity
	selected.GearLevel = g.GearLevel
	selected.Sockets = g.Sockets

	dataBytes, err := json.Marshal(selected)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if _, err := tx.Exec("UPDATE user_inventory SET gear_id=$1, item_data=$2 WHERE id=$3 AND client_uid=$4", selected.ID, string(dataBytes), req.InvID, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if !deductGold(w, tx, uid, s.bot.forgeGoldCost(uid, 100, g.Rarity)) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Transmuted weapon into a suitable %s!", selected.Name), "gold": gold})
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
		// Fail closed: a corrupt upgrades blob must not be overwritten with an
		// empty map (that would wipe the player's converted HP/mana state).
		if err := json.Unmarshal([]byte(upgradesJSON.String), &upgrades); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}

	// Check if user has enough converted mana stats
	userStats, _, _, _ := s.bot.calculateTotalStats(uid, time.Now())
	if userStats.MNA < req.Amount {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough mana stats to convert"})
		return
	}

	upgrades["converted_hp"] += hpGain
	upgrades["converted_mana_reduction"] += req.Amount

	upgradesBytes, err := json.Marshal(upgrades)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	_, err = s.bot.DB.Exec("UPDATE users SET abyss_upgrades=$1 WHERE client_uid=$2", string(upgradesBytes), uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db update"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "msg": fmt.Sprintf("Converted %d mana into +%d Max HP!", req.Amount, hpGain)})
}

const abyssTalentRespecFee int64 = 10

// handleAbyssResetTalents refunds one progression scope. The legacy request
// without a body keeps its old "all" behavior; the tree UI sends an explicit
// scope so Deep Delver and specialization allocations can be retuned without
// erasing each other. The small fee is deducted from the refund, so a player is
// never trapped because every token is currently allocated.
func (s *WebServer) handleAbyssResetTalents(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Scope string `json:"scope"`
	}
	if err := readJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	if req.Scope == "" {
		req.Scope = "all"
	}
	if req.Scope != "all" && req.Scope != "deep_delver" && req.Scope != "specializations" {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown respec scope"})
		return
	}

	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var upVigor, upGreed, upFortune, upWard, upInterest, upTribute, upInsight int
	var upSwift, upScav, upMercy, upCarto, upQuarter int
	err = tx.QueryRow(`SELECT abyss_up_vigor, abyss_up_greed, abyss_up_fortune, abyss_up_ward,
	                          abyss_up_interest, abyss_up_tribute, abyss_up_insight,
	                          abyss_up_swiftness, abyss_up_scavenger, abyss_up_mercy,
	                          abyss_up_cartographer, abyss_up_quartermaster
	                     FROM users WHERE client_uid=$1`, uid).Scan(
		&upVigor, &upGreed, &upFortune, &upWard, &upInterest, &upTribute, &upInsight,
		&upSwift, &upScav, &upMercy, &upCarto, &upQuarter,
	)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "user not found"})
		return
	}

	// Spend sum calculation
	calcRefund := func(level int) int64 {
		level = min(max(level, 0), content.TalentMaxLevel)
		sum := int64(0)
		for l := 1; l <= level; l++ {
			sum += talentTokenCost(l - 1)
		}
		return sum
	}

	resetDeepDelver := req.Scope == "all" || req.Scope == "deep_delver"
	legacyRefund := int64(0)
	if resetDeepDelver {
		legacyRefund = calcRefund(upVigor) + calcRefund(upGreed) + calcRefund(upFortune) + calcRefund(upWard) +
			calcRefund(upInterest) + calcRefund(upTribute) + calcRefund(upInsight) +
			calcRefund(upSwift) + calcRefund(upScav) + calcRefund(upMercy) +
			calcRefund(upCarto) + calcRefund(upQuarter)
	}

	var levelsJSON string
	_ = tx.QueryRow("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", abyssTalentKey(uid)).Scan(&levelsJSON)
	allLevels := map[string]int{}
	if levelsJSON != "" {
		if err := json.Unmarshal([]byte(levelsJSON), &allLevels); err != nil {
			log.Printf("abyss talent levels corrupt for %s during scoped reset: %v", uid, err)
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	resetLevels, remainingLevels := partitionAbyssTalentLevels(allLevels, req.Scope)
	grossRefund := legacyRefund + abyssTalentRefund(resetLevels)
	if grossRefund <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "no talent points to reset"})
		return
	}
	netRefund := max(int64(0), grossRefund-abyssTalentRespecFee)

	// Mana-conversion state (converted_hp / converted_mana_reduction) lives in
	// abyss_upgrades but is not a Deep-Delver talent, so a talent reset must not wipe
	// it. Rebuild the JSON from just those keys instead of blanking the whole column.
	if resetDeepDelver {
		var upgradesJSON sql.NullString
		_ = tx.QueryRow("SELECT abyss_upgrades FROM users WHERE client_uid=$1", uid).Scan(&upgradesJSON)
		upgrades := map[string]int{}
		if upgradesJSON.Valid && upgradesJSON.String != "" {
			if err := json.Unmarshal([]byte(upgradesJSON.String), &upgrades); err != nil {
				log.Printf("abyss upgrades blob corrupt for %s during talent reset: %v", uid, err)
				writeJSON(w, map[string]any{"ok": false, "error": "db"})
				return
			}
		}
		preserved := map[string]int{}
		for _, key := range []string{"converted_hp", "converted_mana_reduction"} {
			if value, ok := upgrades[key]; ok {
				preserved[key] = value
			}
		}
		preservedBytes, marshalErr := json.Marshal(preserved)
		if marshalErr != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		_, err = tx.Exec(`UPDATE users
		                     SET abyss_up_vigor=0, abyss_up_greed=0, abyss_up_fortune=0, abyss_up_ward=0,
		                         abyss_up_interest=0, abyss_up_tribute=0, abyss_up_insight=0,
		                         abyss_up_swiftness=0, abyss_up_scavenger=0, abyss_up_mercy=0,
		                         abyss_up_cartographer=0, abyss_up_quartermaster=0,
		                         abyss_tokens = abyss_tokens + $1, abyss_upgrades = $3::jsonb
		                   WHERE client_uid=$2`, netRefund, uid, string(preservedBytes))
	} else {
		_, err = tx.Exec("UPDATE users SET abyss_tokens = abyss_tokens + $1 WHERE client_uid=$2", netRefund, uid)
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db update"})
		return
	}

	if len(remainingLevels) == 0 {
		_, err = tx.Exec("DELETE FROM app_meta WHERE key=$1", abyssTalentKey(uid))
	} else {
		remainingBytes, marshalErr := json.Marshal(remainingLevels)
		if marshalErr != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		_, err = tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssTalentKey(uid), string(remainingBytes))
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db update"})
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	writeJSON(w, map[string]any{
		"ok": true, "scope": req.Scope, "gross_refund": grossRefund,
		"fee": abyssTalentRespecFee, "refund": netRefund, "tokens": s.bot.abyssTokens(uid),
		"msg": fmt.Sprintf("Respec complete — %d tokens refunded after the %d-token fee.", netRefund, abyssTalentRespecFee),
	})
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

	tx, err := s.beginForgeRequestTx(w)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gearID string
	var itemData sql.NullString
	var queryErr error

	if req.InvID > 0 {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_inventory WHERE id=$1 AND client_uid=$2 FOR UPDATE", req.InvID, uid).Scan(&gearID, &itemData)
	} else if req.Slot != "" {
		queryErr = tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE slot=$1 AND client_uid=$2 FOR UPDATE", req.Slot, uid).Scan(&gearID, &itemData)
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
	dataBytes, err := json.Marshal(g)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}

	if !writeGearItemData(w, tx, uid, req.InvID, req.Slot, string(dataBytes)) {
		return
	}
	if !deductGold(w, tx, uid, s.bot.forgeGoldCost(uid, 200, g.Rarity)) {
		return
	}

	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{"ok": true, "msg": "Item marked as Insured! It will no longer take durability loss.", "gold": gold})
}
