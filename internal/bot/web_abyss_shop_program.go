package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	abyssShopLoyaltyPurchases = 10
	abyssShopGiftFeeGold      = int64(2_500)
	abyssSeasonExchangeCost   = int64(25)
)

type abyssShopBundle struct {
	Key   string
	Name  string
	Desc  string
	Cost  int64
	Items map[string]int
}

var abyssShopBundles = []abyssShopBundle{
	{Key: "delver_supply", Name: "Delver Supply Crate", Desc: "3 Great Health Potions, 2 Master Repair Kits, and a passive Insurance Charm.", Cost: 18, Items: map[string]int{"great_health_potion": 3, "master_repair_kit": 2, abyssInsuranceCharmID: 1}},
	{Key: "risk_runner", Name: "Risk Runner Pack", Desc: "Elixir of Life plus 2 Giant Strength and 2 Speed Elixirs.", Cost: 18, Items: map[string]int{"elixir_of_life": 1, "giant_strength_elixir": 2, "speed_elixir": 2}},
}

type abyssShopProgramView struct {
	Gold                  int64
	Tokens                int64
	LoyaltyPunches        int
	LoyaltyTarget         int
	NextTokenPurchaseFree bool
	GiftFeeGold           int64
	Bundles               []abyssShopBundle
	Flash                 *abyssShopItemView
	FlashEnds             string
	SeasonExchangeCost    int64
	SeasonExchangeName    string
	SeasonExchangeOwned   bool
	Materials             []abyssShopCurrencyMaterial
}

type abyssShopCurrencyMaterial struct {
	ID    string
	Name  string
	Icon  string
	Count int64
}

func abyssShopLoyaltyKey(uid string) string { return "abyss_shop_loyalty_" + uid }

func abyssShopLoyaltyPunches(db dbOrTx, uid string, lock bool) (int, error) {
	query := "SELECT value FROM app_meta WHERE key=$1"
	if lock {
		if _, err := db.Exec("INSERT INTO app_meta (key,value) VALUES ($1,'0') ON CONFLICT DO NOTHING", abyssShopLoyaltyKey(uid)); err != nil {
			return 0, err
		}
		query += " FOR UPDATE"
	}
	var raw string
	if err := db.QueryRow(query, abyssShopLoyaltyKey(uid)).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	punches, err := strconv.Atoi(raw)
	if err != nil || punches < 0 || punches >= abyssShopLoyaltyPurchases {
		return 0, nil
	}
	return punches, nil
}

// applyAbyssShopLoyalty locks and advances the bounded punch card. The tenth
// token-priced purchase is free; gold-only services never consume a punch.
func applyAbyssShopLoyalty(tx *sql.Tx, uid string, quotedCost int64) (charged int64, punches int, free bool, err error) {
	punches, err = abyssShopLoyaltyPunches(tx, uid, true)
	if err != nil {
		return 0, 0, false, err
	}
	free = punches == abyssShopLoyaltyPurchases-1
	if free {
		charged = 0
		punches = 0
	} else {
		charged = quotedCost
		punches++
	}
	_, err = tx.Exec(`INSERT INTO app_meta (key,value) VALUES ($1,$2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssShopLoyaltyKey(uid), strconv.Itoa(punches))
	return charged, punches, free, err
}

func abyssShopBundleByKey(key string) (abyssShopBundle, bool) {
	for _, bundle := range abyssShopBundles {
		if bundle.Key == key {
			return bundle, true
		}
	}
	return abyssShopBundle{}, false
}

func grantAbyssShopConsumable(db dbOrTx, uid, consID string, quantity int) error {
	if quantity <= 0 {
		return nil
	}
	_, err := db.Exec(`INSERT INTO user_consumables (client_uid,cons_id,remaining_fights) VALUES ($1,$2,$3)
		ON CONFLICT (client_uid,cons_id) DO UPDATE SET remaining_fights=user_consumables.remaining_fights+EXCLUDED.remaining_fights`, uid, consID, quantity)
	return err
}

func (b *Bot) abyssShopProgram(uid string, now time.Time, shop []abyssShopItemView, materialCounts map[string]int64, ownedCosmetics map[string]bool) abyssShopProgramView {
	punches, _ := abyssShopLoyaltyPunches(b.DB, uid, false)
	view := abyssShopProgramView{
		Gold: b.abyssGold(uid), Tokens: b.abyssTokens(uid), LoyaltyPunches: punches,
		LoyaltyTarget: abyssShopLoyaltyPurchases, NextTokenPurchaseFree: punches == abyssShopLoyaltyPurchases-1,
		GiftFeeGold: abyssShopGiftFeeGold, Bundles: abyssShopBundles,
		FlashEnds:          now.UTC().Truncate(24 * time.Hour).Add(24 * time.Hour).Format("02 Jan 15:04 UTC"),
		SeasonExchangeCost: abyssSeasonExchangeCost,
	}
	view.Materials = make([]abyssShopCurrencyMaterial, 0, len(abyssMaterials))
	for _, material := range abyssMaterials {
		view.Materials = append(view.Materials, abyssShopCurrencyMaterial{
			ID: material.ID, Name: material.Name, Icon: material.Icon, Count: materialCounts[material.ID],
		})
	}
	for index := range shop {
		if shop[index].HappyAccident {
			flash := shop[index]
			view.Flash = &flash
			break
		}
	}
	previous := abyssSeasonCampaignAt(now.UTC().Add(-time.Duration(abyssSeasonWeeks) * abyssSeasonWeek))
	view.SeasonExchangeName = previous.RewardWord + " Legacy Cache"
	key := abyssSeasonLegacyCosmeticKey(previous)
	view.SeasonExchangeOwned = ownedCosmetics[key]
	return view
}

func (s *WebServer) handleAbyssShopBundleBuy(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		Bundle     string `json:"bundle"`
		QuotedCost int64  `json:"quoted_cost"`
	}
	if readJSON(r, &req) != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	bundle, ok := abyssShopBundleByKey(req.Bundle)
	if !ok || req.QuotedCost != bundle.Cost {
		writeJSON(w, map[string]any{"ok": false, "error": "bundle changed; refresh and review the total"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	charged, punches, free, err := applyAbyssShopLoyalty(tx, uid, bundle.Cost)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	res, err := tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens-$1 WHERE client_uid=$2 AND abyss_tokens >= $1", charged, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if count, _ := res.RowsAffected(); count == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens"})
		return
	}
	for consID, quantity := range bundle.Items {
		if err := grantAbyssShopConsumable(tx, uid, consID, quantity); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "tokens": s.bot.abyssTokens(uid), "consumables": s.bot.getConsumables(uid),
		"loyalty_punches": punches, "loyalty_free": free, "msg": fmt.Sprintf("%s opened for %d tokens.", bundle.Name, charged)})
}

func abyssSeasonLegacyCosmeticKey(campaign abyssSeasonCampaign) string {
	return "season_" + campaign.ID + "_legacy_cache"
}

func (s *WebServer) handleAbyssSeasonExchange(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	previous := abyssSeasonCampaignAt(time.Now().UTC().Add(-time.Duration(abyssSeasonWeeks) * abyssSeasonWeek))
	key := abyssSeasonLegacyCosmeticKey(previous)
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.Exec(`INSERT INTO abyss_shop_cosmetics (client_uid,cosmetic_key) VALUES ($1,$2)
		ON CONFLICT (client_uid,cosmetic_key) DO NOTHING`, uid, key)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	inserted, _ := result.RowsAffected()
	if inserted == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "previous-season legacy cosmetic already owned"})
		return
	}
	result, err = tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens-$1 WHERE client_uid=$2 AND abyss_tokens >= $1", abyssSeasonExchangeCost, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if count, _ := result.RowsAffected(); count == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "tokens": s.bot.abyssTokens(uid), "owned": true,
		"msg": previous.RewardWord + " Legacy Cache converted into a permanent cosmetic. Tokens never hard reset."})
}
