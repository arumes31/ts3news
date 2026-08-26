package bot

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"ts3news/internal/content"
)

const abyssTransmogKeyPrefix = "gear_appearance:"

type abyssTransmogAppearance struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slot   string `json:"slot"`
	Rarity string `json:"rarity"`
	Rank   int    `json:"rank"`
	Cost   int64  `json:"cost"`
}

type abyssTransmogOwnedGear struct {
	gearID       string
	itemData     sql.NullString
	unidentified bool
}

func abyssTransmogKey(gearID string) string {
	return abyssTransmogKeyPrefix + gearID
}

func abyssTransmogCost(rarity content.Rarity) int64 {
	rank := min(max(int(rarity), 0), int(content.RarityEternal))
	return int64(10_000) << rank
}

func abyssTransmogCatalog() map[string]content.Gear {
	out := make(map[string]content.Gear)
	for _, gear := range content.GearAppearanceCatalog() {
		out[gear.ID] = gear
	}
	return out
}

func (b *Bot) collectAbyssTransmogGear(tx *sql.Tx, uid string) ([]abyssTransmogOwnedGear, error) {
	rows, err := tx.Query(`SELECT gear_id,item_data FROM user_gear WHERE client_uid=$1
		UNION ALL SELECT gear_id,item_data FROM user_inventory WHERE client_uid=$1`, uid)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	owned := make([]abyssTransmogOwnedGear, 0)
	for rows.Next() {
		var row abyssTransmogOwnedGear
		if err := rows.Scan(&row.gearID, &row.itemData); err != nil {
			return nil, err
		}
		gear, ok := b.makeGear(row.gearID, row.itemData)
		row.unidentified = !ok || gear.ID != row.gearID || gear.Unidentified
		owned = append(owned, row)
	}
	return owned, rows.Err()
}

func (b *Bot) syncAbyssTransmogUnlocks(tx *sql.Tx, uid string) (int, error) {
	owned, err := b.collectAbyssTransmogGear(tx, uid)
	if err != nil {
		return 0, err
	}
	catalog := abyssTransmogCatalog()
	unlocked := 0
	for _, item := range owned {
		if item.unidentified {
			continue
		}
		if _, known := catalog[item.gearID]; !known {
			continue
		}
		result, err := tx.Exec(`INSERT INTO abyss_shop_cosmetics (client_uid,cosmetic_key) VALUES ($1,$2)
			ON CONFLICT (client_uid,cosmetic_key) DO NOTHING`, uid, abyssTransmogKey(item.gearID))
		if err != nil {
			return 0, err
		}
		if inserted, _ := result.RowsAffected(); inserted > 0 {
			unlocked++
		}
	}
	return unlocked, nil
}

func abyssTransmogViews(catalog map[string]content.Gear, owned map[string]bool) []abyssTransmogAppearance {
	views := make([]abyssTransmogAppearance, 0, len(owned))
	for key := range owned {
		if !strings.HasPrefix(key, abyssTransmogKeyPrefix) {
			continue
		}
		gear, known := catalog[strings.TrimPrefix(key, abyssTransmogKeyPrefix)]
		if !known {
			continue
		}
		views = append(views, abyssTransmogAppearance{
			ID: gear.ID, Name: gear.Name, Slot: string(gear.Slot), Rarity: gear.Rarity.String(),
			Rank: int(gear.Rarity), Cost: abyssTransmogCost(gear.Rarity),
		})
	}
	sort.Slice(views, func(i, j int) bool {
		if views[i].Slot != views[j].Slot {
			return views[i].Slot < views[j].Slot
		}
		if views[i].Rank != views[j].Rank {
			return views[i].Rank > views[j].Rank
		}
		return views[i].Name < views[j].Name
	})
	return views
}

func (s *WebServer) handleAbyssTransmogState(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
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
	newUnlocks, err := s.bot.syncAbyssTransmogUnlocks(tx, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	rows, err := tx.Query(`SELECT cosmetic_key FROM abyss_shop_cosmetics
		WHERE client_uid=$1 AND cosmetic_key LIKE $2`, uid, abyssTransmogKeyPrefix+"%")
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	owned := make(map[string]bool)
	for rows.Next() {
		var key string
		if scanErr := rows.Scan(&key); scanErr != nil {
			_ = rows.Close()
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		owned[key] = true
	}
	rowsErr := rows.Err()
	_ = rows.Close()
	if rowsErr != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var gold int64
	if err := tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}
	catalog := abyssTransmogCatalog()
	appearances := abyssTransmogViews(catalog, owned)
	writeJSON(w, map[string]any{
		"ok": true, "appearances": appearances, "owned": len(appearances),
		"total": len(catalog), "new_unlocks": newUnlocks, "gold": gold,
	})
}

func (s *WebServer) handleAbyssTransmogApply(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	var request struct {
		InvID        int64  `json:"inv_id"`
		Slot         string `json:"slot"`
		AppearanceID string `json:"appearance_id"`
	}
	if readJSON(r, &request) != nil || (request.InvID > 0) == (request.Slot != "") {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid item"})
		return
	}
	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	target, _, ok := loadForgeItem(tx, s.bot, uid, request.InvID, request.Slot)
	if !ok || target.Unidentified {
		writeJSON(w, map[string]any{"ok": false, "error": "item not found or unidentified"})
		return
	}
	cost := int64(0)
	appearanceName := "Original appearance"
	if request.AppearanceID != "" {
		appearance, known := content.GetGearByID(request.AppearanceID)
		if !known || appearance.Slot != target.Slot {
			writeJSON(w, map[string]any{"ok": false, "error": "incompatible appearance"})
			return
		}
		var owned bool
		if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM abyss_shop_cosmetics
			WHERE client_uid=$1 AND cosmetic_key=$2)`, uid, abyssTransmogKey(appearance.ID)).Scan(&owned); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if !owned {
			writeJSON(w, map[string]any{"ok": false, "error": "appearance not unlocked"})
			return
		}
		cost = abyssTransmogCost(appearance.Rarity)
		appearanceName = appearance.Name
	}
	if target.AppearanceID == request.AppearanceID {
		var gold int64
		if err := tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if err := tx.Commit(); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
			return
		}
		writeJSON(w, map[string]any{
			"ok": true, "appearance_id": request.AppearanceID, "cost": int64(0),
			"gold": gold, "msg": "Appearance already applied.",
		})
		return
	}
	target.AppearanceID = request.AppearanceID
	encoded, err := json.Marshal(target)
	if err != nil || !writeGearItemData(w, tx, uid, request.InvID, request.Slot, string(encoded)) {
		return
	}
	if cost > 0 && !deductGold(w, tx, uid, cost) {
		return
	}
	var gold int64
	if err := tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db commit"})
		return
	}
	message := "Original appearance restored."
	if request.AppearanceID != "" {
		message = appearanceName + " applied. Combat power is unchanged."
	}
	writeJSON(w, map[string]any{
		"ok": true, "appearance_id": request.AppearanceID, "cost": cost,
		"gold": gold, "msg": message,
	})
}
