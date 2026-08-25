package bot

import (
	"database/sql"
	"net/http"
)

type abyssBossCosmetic struct {
	Key     string
	Name    string
	Kind    string
	Icon    string
	MinTier string
	Lore    string
}

var abyssBossCosmeticCatalog = []abyssBossCosmetic{
	{Key: "boss_mount_bone_runner", Name: "Bone Runner", Kind: "mount", Icon: "♞", MinTier: "normal", Lore: "A pale courser assembled from threshold relics."},
	{Key: "boss_banner_cracked_sigil", Name: "Cracked Sigil", Kind: "banner", Icon: "⚑", MinTier: "normal", Lore: "The first mark carried back from below."},
	{Key: "boss_mount_shadow_drake", Name: "Shadow Drake", Kind: "mount", Icon: "◆", MinTier: "nightmare", Lore: "A winged silhouette that refuses torchlight."},
	{Key: "boss_banner_drowned_pennant", Name: "Drowned Pennant", Kind: "banner", Icon: "⚑", MinTier: "nightmare", Lore: "Still dripping with water from a sunless sea."},
	{Key: "boss_mount_cinder_warhorse", Name: "Cinder Warhorse", Kind: "mount", Icon: "♞", MinTier: "hell", Lore: "Each hoofbeat scatters harmless ember-light."},
	{Key: "boss_banner_void_standard", Name: "Void Standard", Kind: "banner", Icon: "⚐", MinTier: "hell", Lore: "Its black field reflects stars that do not exist."},
	{Key: "boss_mount_eyeless_leviathan", Name: "Eyeless Leviathan", Kind: "mount", Icon: "◈", MinTier: "insanity", Lore: "A ceremonial likeness of the thing beneath."},
	{Key: "boss_banner_crownless", Name: "Crownless Banner", Kind: "banner", Icon: "⚚", MinTier: "insanity", Lore: "Proof that even sovereigns can kneel."},
}

var abyssBossCosmeticByKey = func() map[string]abyssBossCosmetic {
	out := make(map[string]abyssBossCosmetic, len(abyssBossCosmeticCatalog))
	for _, cosmetic := range abyssBossCosmeticCatalog {
		out[cosmetic.Key] = cosmetic
	}
	return out
}()

func abyssBossCosmeticTierRank(tier string) int {
	for rank, key := range abyssTierOrder {
		if key == tier {
			return rank
		}
	}
	return 0
}

func abyssBossCosmeticDropChance(tier string) float64 {
	return []float64{0.02, 0.04, 0.07, 0.12}[abyssBossCosmeticTierRank(tier)]
}

func abyssRollBossCosmetic(tier string, dropRoll, choiceRoll float64) (abyssBossCosmetic, bool) {
	if dropRoll < 0 || dropRoll >= abyssBossCosmeticDropChance(tier) {
		return abyssBossCosmetic{}, false
	}
	rank := abyssBossCosmeticTierRank(tier)
	eligible := make([]abyssBossCosmetic, 0, len(abyssBossCosmeticCatalog))
	for _, cosmetic := range abyssBossCosmeticCatalog {
		if abyssBossCosmeticTierRank(cosmetic.MinTier) <= rank {
			eligible = append(eligible, cosmetic)
		}
	}
	if len(eligible) == 0 {
		return abyssBossCosmetic{}, false
	}
	choice := min(max(int(choiceRoll*float64(len(eligible))), 0), len(eligible)-1)
	return eligible[choice], true
}

func grantAbyssBossCosmeticTx(tx *sql.Tx, uid, tier string, dropRoll, choiceRoll float64) (abyssBossCosmetic, bool, error) {
	cosmetic, ok := abyssRollBossCosmetic(tier, dropRoll, choiceRoll)
	if !ok {
		return abyssBossCosmetic{}, false, nil
	}
	res, err := tx.Exec(`INSERT INTO abyss_shop_cosmetics (client_uid,cosmetic_key) VALUES ($1,$2)
		ON CONFLICT (client_uid,cosmetic_key) DO NOTHING`, uid, cosmetic.Key)
	if err != nil {
		return abyssBossCosmetic{}, false, err
	}
	inserted, _ := res.RowsAffected()
	return cosmetic, inserted > 0, nil
}

type abyssBossCosmeticView struct {
	abyssBossCosmetic
	Owned  bool
	Active bool
}

type abyssBossCosmeticCollectionView struct {
	Items        []abyssBossCosmeticView
	Owned        int
	Total        int
	ActiveMount  string
	ActiveBanner string
	Rates        string
}

func (b *Bot) abyssBossCosmeticCollectionWithOwned(uid string, owned map[string]bool) abyssBossCosmeticCollectionView {
	view := abyssBossCosmeticCollectionView{Total: len(abyssBossCosmeticCatalog), Rates: "Normal 2% · Nightmare 4% · Hell 7% · Insanity 12%"}
	_ = b.DB.QueryRow("SELECT mount_key,banner_key FROM abyss_boss_cosmetic_loadouts WHERE client_uid=$1", uid).Scan(&view.ActiveMount, &view.ActiveBanner)
	for _, cosmetic := range abyssBossCosmeticCatalog {
		item := abyssBossCosmeticView{abyssBossCosmetic: cosmetic, Owned: owned[cosmetic.Key]}
		item.Active = item.Owned && (cosmetic.Key == view.ActiveMount || cosmetic.Key == view.ActiveBanner)
		if item.Owned {
			view.Owned++
		}
		view.Items = append(view.Items, item)
	}
	return view
}

func (s *WebServer) handleAbyssBossCosmeticEquip(w http.ResponseWriter, r *http.Request, uid string) {
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
		Key string `json:"key"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "invalid cosmetic"})
		return
	}
	cosmetic, known := abyssBossCosmeticByKey[req.Key]
	if !known {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown cosmetic"})
		return
	}
	column := map[string]string{"mount": "mount_key", "banner": "banner_key"}[cosmetic.Kind]
	query := `INSERT INTO abyss_boss_cosmetic_loadouts (client_uid,` + column + `)
		SELECT $1,$2 WHERE EXISTS (SELECT 1 FROM abyss_shop_cosmetics WHERE client_uid=$1 AND cosmetic_key=$2)
		ON CONFLICT (client_uid) DO UPDATE SET ` + column + `=EXCLUDED.` + column + `,updated_at=NOW()`
	res, err := s.bot.DB.Exec(query, uid, cosmetic.Key)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "cosmetic not owned"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "kind": cosmetic.Kind, "key": cosmetic.Key, "msg": cosmetic.Name + " equipped. Cosmetic only; combat power unchanged."})
}
