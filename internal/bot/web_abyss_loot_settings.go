package bot

import (
	"encoding/json"
	"net/http"
	"strings"

	"ts3news/internal/content"
)

type abyssLootSettings struct {
	TargetCategory         string `json:"target_category,omitempty"`
	AutoSalvageMax         int    `json:"auto_salvage_max"`
	DuplicateLegendConvert bool   `json:"duplicate_legend_convert"`
}

func abyssLootSettingsKey(uid string) string { return "abyss_loot_settings_" + uid }
func abyssLootReservedKey(uid string) string { return "abyss_loot_reserved_" + uid }

func normalizeAbyssLootSettings(settings abyssLootSettings) abyssLootSettings {
	switch strings.ToLower(strings.TrimSpace(settings.TargetCategory)) {
	case "weapon", "armor", "jewelry":
		settings.TargetCategory = strings.ToLower(strings.TrimSpace(settings.TargetCategory))
	default:
		settings.TargetCategory = ""
	}
	if settings.AutoSalvageMax < -1 {
		settings.AutoSalvageMax = -1
	}
	// Automatic salvage is intentionally capped below Rare. Valuable drops must
	// always reach the backpack where the owner can explicitly reserve them.
	if settings.AutoSalvageMax > int(content.RarityUncommon) {
		settings.AutoSalvageMax = int(content.RarityUncommon)
	}
	return settings
}

func (b *Bot) loadAbyssLootSettings(uid string) abyssLootSettings {
	var raw string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssLootSettingsKey(uid)).Scan(&raw)
	settings := abyssLootSettings{AutoSalvageMax: -1}
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &settings)
	}
	return normalizeAbyssLootSettings(settings)
}

func (b *Bot) saveAbyssLootSettings(uid string, settings abyssLootSettings) error {
	settings = normalizeAbyssLootSettings(settings)
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssLootSettingsKey(uid), string(data))
	return err
}

func (b *Bot) loadAbyssReservedLoot(uid string) map[int64]bool {
	reserved := map[int64]bool{}
	var raw string
	_ = b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssLootReservedKey(uid)).Scan(&raw)
	var ids []int64
	if raw != "" && json.Unmarshal([]byte(raw), &ids) == nil {
		for _, id := range ids {
			if id > 0 {
				reserved[id] = true
			}
		}
	}
	return reserved
}

func abyssReservedLootIDs(reserved map[int64]bool) []int64 {
	ids := make([]int64, 0, len(reserved))
	for id, keep := range reserved {
		if id > 0 && keep {
			ids = append(ids, id)
		}
	}
	return ids
}

func (b *Bot) saveAbyssReservedLoot(uid string, reserved map[int64]bool) error {
	ids := make([]int64, 0, len(reserved))
	for id, keep := range reserved {
		if id > 0 && keep {
			ids = append(ids, id)
		}
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	_, err = b.DB.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssLootReservedKey(uid), string(data))
	return err
}

func (b *Bot) shouldAutoSalvageAbyssGear(uid string, gear content.Gear) bool {
	settings := b.loadAbyssLootSettings(uid)
	return settings.AutoSalvageMax >= 0 && int(gear.Rarity) <= settings.AutoSalvageMax
}

func (s *WebServer) handleAbyssLootSettings(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method == http.MethodGet {
		writeJSON(w, map[string]any{
			"ok": true, "settings": s.bot.loadAbyssLootSettings(uid),
			"reserved_ids": abyssReservedLootIDs(s.bot.loadAbyssReservedLoot(uid)),
		})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "GET or POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req abyssLootSettings
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	req = normalizeAbyssLootSettings(req)
	if s.bot.saveAbyssLootSettings(uid, req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "settings": req})
}

func (s *WebServer) handleAbyssLootReserve(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		InvID int64 `json:"inv_id"`
	}
	if readJSON(r, &req) != nil || req.InvID <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	var exists bool
	if s.bot.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM user_inventory WHERE id=$1 AND client_uid=$2)", req.InvID, uid).Scan(&exists) != nil || !exists {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found"})
		return
	}
	reserved := s.bot.loadAbyssReservedLoot(uid)
	reserved[req.InvID] = !reserved[req.InvID]
	if !reserved[req.InvID] {
		delete(reserved, req.InvID)
	}
	if s.bot.saveAbyssReservedLoot(uid, reserved) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "inv_id": req.InvID, "reserved": reserved[req.InvID], "count": len(reserved)})
}
