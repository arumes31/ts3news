package bot

import (
	"database/sql"
	"log"
	"net/http"
	"time"
)

type abyssBossVendorItem struct {
	Key         string
	Name        string
	Desc        string
	Cost        int64
	Material    string
	MaterialN   int
	Material2   string
	Material2N  int
	AbyssTokens int64
}

var abyssBossVendorCatalog = []abyssBossVendorItem{
	{Key: "core_cache", Name: "Warden's Core Cache", Desc: "Three Umbral Cores for advanced forge work.", Cost: 1, Material: "core", MaterialN: 3},
	{Key: "prism_cache", Name: "Sovereign Prism", Desc: "One Eldritch Prism from a defeated boss's vault.", Cost: 2, Material: "prism", MaterialN: 1},
	{Key: "token_cache", Name: "Delver's Purse", Desc: "Convert one trophy into ten flexible Abyss Tokens.", Cost: 1, AbyssTokens: 10},
	{Key: "grand_cache", Name: "Grand Trophy Cache", Desc: "Five cores, two prisms, and twenty-five Abyss Tokens.", Cost: 5, Material: "core", MaterialN: 5, Material2: "prism", Material2N: 2, AbyssTokens: 25},
}

var abyssBossVendorIndex = func() map[string]abyssBossVendorItem {
	items := make(map[string]abyssBossVendorItem, len(abyssBossVendorCatalog))
	for _, item := range abyssBossVendorCatalog {
		items[item.Key] = item
	}
	return items
}()

func (b *Bot) abyssBossTokens(uid string) int64 {
	var tokens int64
	_ = b.DB.QueryRow("SELECT abyss_boss_tokens FROM users WHERE client_uid=$1", uid).Scan(&tokens)
	return tokens
}

func (b *Bot) recordAbyssBossKillWithToken(uid, bossName string, depth int, killTime time.Duration, tier string) bool {
	tx, err := b.DB.Begin()
	if err != nil {
		log.Printf("boss trophy transaction failed to start for %s: %v", uid, err)
		return false
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(
		"INSERT INTO abyss_boss_kills (client_uid,boss_name,depth,kill_time_ms,tier) VALUES ($1,$2,$3,$4,$5)",
		uid, bossName, depth, max(int64(0), killTime.Milliseconds()), tier,
	); err != nil {
		log.Printf("boss kill record failed for %s: %v", uid, err)
		return false
	}
	if _, err := tx.Exec("UPDATE users SET abyss_boss_tokens=abyss_boss_tokens+1 WHERE client_uid=$1", uid); err != nil {
		log.Printf("boss trophy award failed for %s: %v", uid, err)
		return false
	}
	if err := tx.Commit(); err != nil {
		log.Printf("boss trophy transaction failed to commit for %s: %v", uid, err)
		return false
	}
	return true
}

func grantAbyssBossVendorMaterial(tx *sql.Tx, uid, material string, amount int) error {
	if material == "" || amount <= 0 {
		return nil
	}
	if _, err := tx.Exec(`INSERT INTO user_materials (client_uid,mat_id,count) VALUES ($1,$2,$3)
		ON CONFLICT (client_uid,mat_id) DO UPDATE SET count=user_materials.count+$3`, uid, material, amount); err != nil {
		return err
	}
	_, err := tx.Exec(`INSERT INTO abyss_forge_material_flow (client_uid,mat_id,direction,amount)
		VALUES ($1,$2,'source',$3)`, uid, material, amount)
	return err
}

func (s *WebServer) handleAbyssBossVendorBuy(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}
	var req struct {
		Item string `json:"item"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid request"})
		return
	}
	item, ok := abyssBossVendorIndex[req.Item]
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown trophy reward"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	var balance int64
	err = tx.QueryRow(`UPDATE users SET abyss_boss_tokens=abyss_boss_tokens-$1
		WHERE client_uid=$2 AND abyss_boss_tokens>=$1 RETURNING abyss_boss_tokens`, item.Cost, uid).Scan(&balance)
	if err != nil {
		if err == sql.ErrNoRows {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough Boss Tokens"})
		} else {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
		}
		return
	}
	if item.Material != "" {
		if err := grantAbyssBossVendorMaterial(tx, uid, item.Material, item.MaterialN); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if item.Material2 != "" {
		if err := grantAbyssBossVendorMaterial(tx, uid, item.Material2, item.Material2N); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if item.AbyssTokens > 0 {
		if _, err := tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens+$1 WHERE client_uid=$2", item.AbyssTokens, uid); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("boss vendor commit failed for %s: %v", uid, err)
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if item.Material != "" {
		recordAbyssMaterialFlow(uid, item.Material, "source", item.MaterialN)
	}
	if item.Material2 != "" {
		recordAbyssMaterialFlow(uid, item.Material2, "source", item.Material2N)
	}
	writeJSON(w, map[string]any{
		"ok": true, "msg": item.Name + " claimed.", "boss_tokens": balance,
		"tokens": s.bot.abyssTokens(uid), "materials": s.bot.loadMaterials(uid),
	})
}
