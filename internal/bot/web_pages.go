package bot

import (
	"encoding/json"
	"log"
	"net/http"

	"ts3news/internal/content"
)

// statKV is a single non-zero gear stat for display.
type statKV struct {
	Label string
	Value int
}

// gearView is a template-friendly view of a gear piece.
type gearView struct {
	InvID       int64
	Slot        string
	Icon        string
	ID          string
	Name        string
	Rarity      string
	RarityColor string
	CR          float64
	Score       int
	Durability  int
	Empty       bool

	// Detail surfaced in the armoury/inventory.
	Element    string
	Effect     string
	EffectDesc string
	XPBonusPct int
	Stats      []statKV
}

// gearEffectDescriptions maps each special effect to a short player-facing blurb.
var gearEffectDescriptions = map[content.ItemEffect]string{
	content.EffectThorns:         "Reflects 10% of damage taken",
	content.EffectVampiric:       "Heals for 5% of damage dealt",
	content.EffectBerserk:        "+20% STR while below 50% HP",
	content.EffectLucky:          "+10% Luck",
	content.EffectTreasureHunter: "+5% item find chance",
	content.EffectQuick:          "+10% Speed",
	content.EffectBulwark:        "+10% Defense",
	content.EffectRadiant:        "+10% XP gained",
	content.EffectFragile:        "+30% STR but double durability loss",
	content.EffectSteady:         "-50% stun chance",
	content.EffectMindControl:    "Chance to capture low-health mobs",
	content.EffectRegenStack:     "Permanent regen stack on victory",
	content.EffectPhoenix:        "Revive once per fight at 50% HP",
	content.EffectStealth:        "Skip first-round mob damage",
	content.EffectParry:          "10% chance to negate a hit and counter",
	content.EffectCleanse:        "Removes a negative effect each turn",
}

// gearStatList returns the gear's non-zero combat stats, largest first.
func gearStatList(s content.Stats) []statKV {
	pairs := []statKV{
		{"HP", s.HP}, {"STR", s.STR}, {"DEF", s.DEF}, {"SPD", s.SPD},
		{"CRT%", s.CRT}, {"DGE%", s.DGE}, {"LCK", s.LCK}, {"INT", s.INT}, {"STA", s.STA},
	}
	var out []statKV
	for _, p := range pairs {
		if p.Value != 0 {
			out = append(out, p)
		}
	}
	return out
}

func toGearView(slot content.GearSlot, g content.Gear) gearView {
	v := gearView{
		Slot:        string(slot),
		Icon:        content.SlotIcon(slot),
		ID:          g.ID,
		Name:        g.Name,
		Rarity:      g.Rarity.String(),
		RarityColor: g.Rarity.Color(),
		CR:          g.CombatRating(),
		Score:       g.Stats.Score(),
		Stats:       gearStatList(g.Stats),
		XPBonusPct:  int((g.XPMultiplier - 1.0) * 100),
	}
	if g.Element != "" && g.Element != content.ElementPhysical {
		v.Element = string(g.Element)
	}
	if g.Special != content.EffectNone {
		v.Effect = string(g.Special)
		v.EffectDesc = gearEffectDescriptions[g.Special]
	}
	return v
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("web: json encode failed: %v", err)
	}
}

func (s *WebServer) handleArmory(w http.ResponseWriter, r *http.Request, uid string) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}

	equipped := s.bot.getEquippedItems(uid)
	var slots []gearView
	for _, slot := range content.AllSlots {
		if g, ok := equipped[slot]; ok {
			slots = append(slots, toGearView(slot, g))
		} else {
			slots = append(slots, gearView{Slot: string(slot), Icon: content.SlotIcon(slot), Empty: true})
		}
	}

	skills := s.bot.getSkills(uid)

	s.render(w, "armory", map[string]any{
		"Title": "Armoury",
		"Nav":   "armory",
		"U":     u,
		"Slots": slots,
		"Skills": skills,
	})
}

func (s *WebServer) handleInventory(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}

	items := s.bot.inventoryItems(uid)
	cons := s.bot.consumableCounts(uid)

	s.render(w, "inventory", map[string]any{
		"Title":      "Inventory",
		"Nav":        "inventory",
		"U":          u,
		"Items":      items,
		"Consumables": cons,
	})
}

// inventoryItems returns the user's owned, unequipped gear.
func (b *Bot) inventoryItems(uid string) []gearView {
	rows, err := b.DB.Query("SELECT id, gear_id, durability FROM user_inventory WHERE client_uid=$1 ORDER BY id DESC", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []gearView
	for rows.Next() {
		var id int64
		var gid string
		var dur int
		if err := rows.Scan(&id, &gid, &dur); err != nil {
			continue
		}
		g, ok := content.GetGearByID(gid)
		if !ok {
			continue
		}
		v := toGearView(g.Slot, g)
		v.InvID = id
		v.Durability = dur
		out = append(out, v)
	}
	return out
}

type consumableView struct {
	Name  string
	Count int
}

func (b *Bot) consumableCounts(uid string) []consumableView {
	rows, err := b.DB.Query("SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid=$1", uid)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	var out []consumableView
	for rows.Next() {
		var cid string
		var n int
		if err := rows.Scan(&cid, &n); err != nil {
			continue
		}
		name := cid
		if c, ok := content.GetConsumableByID(cid); ok {
			name = c.Name
		}
		out = append(out, consumableView{Name: name, Count: n})
	}
	return out
}

// handleEquipAPI equips an inventory item, swapping any currently-equipped piece
// in that slot back into the inventory.
func (s *WebServer) handleEquipAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InvID int64 `json:"inv_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	// Look up the inventory item.
	var gid string
	var dur int
	if err := s.bot.DB.QueryRow("SELECT gear_id, durability FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gid, &dur); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	g, ok := content.GetGearByID(gid)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown gear"})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "tx"})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Remove from inventory.
	if _, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "remove"})
		return
	}

	// Displace whatever is in the slot back into the inventory.
	var oldGID string
	var oldDur int
	switch err := tx.QueryRow("SELECT gear_id, durability FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, string(g.Slot)).Scan(&oldGID, &oldDur); err {
	case nil:
		if _, err := tx.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability) VALUES ($1, $2, $3)", uid, oldGID, oldDur); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "displace"})
			return
		}
	default:
		// empty slot, nothing to displace
	}

	// Equip the new piece.
	if _, err := tx.Exec(
		`INSERT INTO user_gear (client_uid, slot, gear_id, durability) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (client_uid, slot) DO UPDATE SET gear_id=$3, durability=$4`,
		uid, string(g.Slot), g.ID, dur,
	); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "equip"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "commit"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "equipped": g.Name, "slot": string(g.Slot)})
}

// handleSellAPI vendors an inventory item for half its fair price.
func (s *WebServer) handleSellAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InvID int64 `json:"inv_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}

	var gid string
	if err := s.bot.DB.QueryRow("SELECT gear_id FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid).Scan(&gid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	g, ok := content.GetGearByID(gid)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown gear"})
		return
	}
	value := gearPrice(g) / 2
	if value < 1 {
		value = 1
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "tx"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2", req.InvID, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "remove"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "already sold"})
		return
	}
	var gold int64
	if err := tx.QueryRow("UPDATE users SET gold = gold + $1 WHERE client_uid=$2 RETURNING gold", value, uid).Scan(&gold); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "gold"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "commit"})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "value": value, "gold": gold})
}
