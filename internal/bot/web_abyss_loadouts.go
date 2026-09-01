package bot

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type abyssEquipmentPreset struct {
	Name  string            `json:"name"`
	Items map[string]string `json:"items"`
}

type abyssGemPreset struct {
	Name string              `json:"name"`
	Gems map[string][]string `json:"gems"`
}

func abyssEquipmentPresetKey(uid string) string { return "abyss_equipment_presets_" + uid }
func abyssGemPresetKey(uid string) string       { return "abyss_gem_presets_" + uid }

func loadAbyssPresetStore[T any](b *Bot, key string) map[string]T {
	out := map[string]T{}
	var raw string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", key).Scan(&raw)
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	return out
}

func saveAbyssPresetStore[T any](b *Bot, key string, store map[string]T) error {
	data, err := json.Marshal(store)
	if err != nil {
		return err
	}
	_, err = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, key, string(data))
	return err
}

func normalizeAbyssPresetSlot(slot int) (string, bool) {
	if slot < 1 || slot > 3 {
		return "", false
	}
	return strconv.Itoa(slot), true
}

func normalizeAbyssPresetName(name string, slot int) string {
	name = strings.TrimSpace(name)
	if len(name) > 32 {
		name = name[:32]
	}
	if name == "" {
		name = fmt.Sprintf("Loadout %d", slot)
	}
	return name
}

func (s *WebServer) handleAbyssEquipmentLoadout(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	if s.bot.loadAbyssRun(uid).Active {
		writeJSON(w, map[string]any{"ok": false, "error": "equipment loadouts can only change between runs"})
		return
	}
	var req struct {
		Action string `json:"action"`
		Slot   int    `json:"slot"`
		Name   string `json:"name"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	slotKey, ok := normalizeAbyssPresetSlot(req.Slot)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "loadout slot must be 1-3"})
		return
	}
	store := loadAbyssPresetStore[abyssEquipmentPreset](s.bot, abyssEquipmentPresetKey(uid))
	if req.Action == "save" {
		preset := abyssEquipmentPreset{Name: normalizeAbyssPresetName(req.Name, req.Slot), Items: map[string]string{}}
		rows, err := s.bot.DB.Query("SELECT slot, gear_id FROM user_gear WHERE client_uid=$1", uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		for rows.Next() {
			var gearSlot, gearID string
			if rows.Scan(&gearSlot, &gearID) == nil {
				preset.Items[gearSlot] = gearID
			}
		}
		_ = rows.Close()
		store[slotKey] = preset
		if saveAbyssPresetStore(s.bot, abyssEquipmentPresetKey(uid), store) != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "msg": preset.Name + " saved.", "count": len(preset.Items)})
		return
	}
	if req.Action != "apply" {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown loadout action"})
		return
	}
	preset, ok := store[slotKey]
	if !ok || len(preset.Items) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "equipment loadout is empty"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	slots := make([]string, 0, len(preset.Items))
	for gearSlot := range preset.Items {
		slots = append(slots, gearSlot)
	}
	sort.Strings(slots)
	swapped := 0
	for _, gearSlot := range slots {
		desiredID := preset.Items[gearSlot]
		var currentID string
		_ = tx.QueryRow("SELECT gear_id FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, gearSlot).Scan(&currentID)
		if currentID == desiredID {
			continue
		}
		var invID int64
		var durability int
		var itemData sql.NullString
		if tx.QueryRow("SELECT id, durability, item_data FROM user_inventory WHERE client_uid=$1 AND gear_id=$2 ORDER BY id LIMIT 1", uid, desiredID).Scan(&invID, &durability, &itemData) != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "missing saved item " + desiredID})
			return
		}
		gear, known := s.bot.makeGear(desiredID, itemData)
		if !known || string(gear.Slot) != gearSlot {
			writeJSON(w, map[string]any{"ok": false, "error": "saved item no longer matches its slot"})
			return
		}
		if _, err := tx.Exec("DELETE FROM user_inventory WHERE id=$1 AND client_uid=$2", invID, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if err := s.bot.equipGear(tx, uid, gear, durability, itemData); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		swapped++
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": preset.Name + " equipped.", "swapped": swapped})
}

func (s *WebServer) handleAbyssGemPreset(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	if s.bot.loadAbyssRun(uid).Active {
		writeJSON(w, map[string]any{"ok": false, "error": "gem presets can only change between runs"})
		return
	}
	var req struct {
		Action string `json:"action"`
		Slot   int    `json:"slot"`
		Name   string `json:"name"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	slotKey, ok := normalizeAbyssPresetSlot(req.Slot)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "preset slot must be 1-3"})
		return
	}
	store := loadAbyssPresetStore[abyssGemPreset](s.bot, abyssGemPresetKey(uid))
	if req.Action == "save" {
		preset := abyssGemPreset{Name: normalizeAbyssPresetName(req.Name, req.Slot), Gems: map[string][]string{}}
		for gearSlot, gear := range s.bot.getEquippedItems(uid) {
			if abyssGearActiveForCombat(gear) && len(gear.Gemstones) > 0 {
				preset.Gems[string(gearSlot)] = append([]string{}, gear.Gemstones...)
			}
		}
		store[slotKey] = preset
		if saveAbyssPresetStore(s.bot, abyssGemPresetKey(uid), store) != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "msg": preset.Name + " gem layout saved.", "slots": len(preset.Gems)})
		return
	}
	if req.Action != "apply" {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown preset action"})
		return
	}
	preset, ok := store[slotKey]
	if !ok || len(preset.Gems) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "gem preset is empty"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	changed := 0
	for gearSlot, desired := range preset.Gems {
		var gearID string
		var itemData sql.NullString
		if tx.QueryRow("SELECT gear_id, item_data FROM user_gear WHERE client_uid=$1 AND slot=$2", uid, gearSlot).Scan(&gearID, &itemData) != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "a preset gear slot is empty"})
			return
		}
		gear, known := s.bot.makeGear(gearID, itemData)
		if !known {
			writeJSON(w, map[string]any{"ok": false, "error": "socket counts changed since this preset was saved"})
			return
		}
		if !abyssGearActiveForCombat(gear) {
			writeJSON(w, map[string]any{"ok": false, "error": "inactive gear cannot use gem presets"})
			return
		}
		if len(gear.Gemstones) != len(desired) {
			writeJSON(w, map[string]any{"ok": false, "error": "socket counts changed since this preset was saved"})
			return
		}
		for i := range desired {
			wantBase, wantTier := parseGem(desired[i])
			_, haveTier := parseGem(gear.Gemstones[i])
			if wantBase == "" || wantTier != haveTier {
				writeJSON(w, map[string]any{"ok": false, "error": "gem tiers must match; presets only rearrange equal-tier gems"})
				return
			}
			if gear.Gemstones[i] != desired[i] {
				changed++
			}
		}
		gear.Gemstones = append([]string{}, desired...)
		data, _ := json.Marshal(gear)
		if _, err := tx.Exec("UPDATE user_gear SET item_data=$1 WHERE client_uid=$2 AND slot=$3", string(data), uid, gearSlot); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	cost := int64(changed * 150)
	if cost > 0 {
		res, err := tx.Exec("UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold >= $1", cost, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "msg": preset.Name + " applied.", "changed": changed, "cost": cost})
}
