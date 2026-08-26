package bot

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"ts3news/internal/content"
)

// Token Shop
// -----------------------------------------------------------------------------
// Abyss Tokens previously only bought Deep-Delver upgrades, which cap at level 5 —
// so veterans had nowhere to spend them. The Token Shop is an open-ended sink: a
// fixed catalog of consumables, gear and relics bought with tokens. Each purchase
// debits tokens with a guarded UPDATE (so a concurrent spend can't overdraw) and
// then grants through the same live granters the loot path uses.

// abyssShopItem is one catalog entry: a token and/or gold cost and the reward
// key the handler switches on to grant it. CostGold is 0 for the (default)
// tokens-only items; the Emergency Revive Potion is the first gold-priced entry.
type abyssShopItem struct {
	Key      string
	Name     string
	Desc     string
	Cost     int64 // Abyss tokens
	CostGold int64 // gold, 0 = tokens-only item
}

var abyssShopCatalog = []abyssShopItem{
	{"great_potions", "Great Health Potions ×3", "Three large in-combat heals.", 6, 0},
	{"repair_kits", "Master Repair Kits ×2", "Fully restore gear durability, twice.", 5, 0},
	{"phoenix", "Phoenix Feather", "Revives you once when you fall in battle.", 9, 0},
	{"elixir_of_life", "Elixir of Life", "Fully restores your health (100%).", 8, 0},
	{"giant_strength", "Giant Strength Elixirs ×2", "Massively boost Strength for 3 fights.", 7, 0},
	{"speed_elixir_pack", "Speed Elixirs ×2", "Boost Speed by +25 for 3 fights.", 5, 0},
	{"lucky_draught_pack", "Lucky Draughts ×2", "Boost Luck by +20 for 3 fights.", 5, 0},
	{"affix_suppressor", "Affix Suppressor", "Ignore the daily affix for one entire run.", 12, 0},
	{"abyss_gear", "Abyss Gear Cache", "A random Abyss-exclusive gear piece.", 15, 0},
	{"epic_gear", "Epic Abyss Cache", "A guaranteed Epic-or-better Abyss piece.", 30, 0},
	{"relic", "Unstable Relic", "A random Unique item.", 40, 0},
	{"emergency_revive", "Emergency Revive Potion", "Single-use: instantly revive to full HP if you fall, beyond your normal one-per-run revival.", 0, 100000},
	{"insanity_void_aura", "Void Aura", "Rotating Insanity cosmetic title with no combat power.", 60, 0},
	{"insanity_glass_crown", "Glass Crown", "Rotating Insanity cosmetic title with no combat power.", 60, 0},
	{"insanity_depth_trail", "Depth Trail", "Rotating Insanity cosmetic title with no combat power.", 60, 0},
}

// abyssShopIndex maps item key → catalog entry, built once from the catalog so
// lookups stay O(1) instead of a linear scan per purchase.
var abyssShopIndex = func() map[string]abyssShopItem {
	m := make(map[string]abyssShopItem, len(abyssShopCatalog))
	for _, it := range abyssShopCatalog {
		m[it.Key] = it
	}
	return m
}()

func abyssShopByKey(key string) (abyssShopItem, bool) {
	it, ok := abyssShopIndex[key]
	return it, ok
}

// handleAbyssShopBuy spends tokens on a catalog item. The token debit is a guarded
// UPDATE so it can't overdraw under a concurrent purchase; the reward is granted
// only if the debit actually consumed tokens.
func (s *WebServer) handleAbyssShopBuy(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()

	var req struct {
		Item       string `json:"item"`
		QuotedCost *int64 `json:"quoted_cost"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	item, ok := abyssShopByKey(req.Item)
	if !ok {
		writeJSON(w, map[string]any{"ok": false, "error": "unknown item"})
		return
	}
	now := time.Now()
	if strings.HasPrefix(item.Key, "insanity_") && item.Key != abyssWeeklyInsanityCosmetic(now) {
		writeJSON(w, map[string]any{"ok": false, "error": "that Insanity cosmetic is not in this week's rotation"})
		return
	}
	market := s.bot.abyssShopDemand(now)[item.Key]
	tokenCost, _ := abyssShopPricedCost(item, now, market.Percent)
	if item.Cost > 0 && req.QuotedCost != nil && *req.QuotedCost != tokenCost {
		writeJSON(w, map[string]any{"ok": false, "error": "shop price changed; refresh and review the new total", "current_cost": tokenCost})
		return
	}
	if strings.HasPrefix(item.Key, "insanity_") {
		s.buyAbyssShopCosmetic(w, uid, item, tokenCost)
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	chargedTokens, punches, loyaltyFree := tokenCost, 0, false
	if item.Cost > 0 {
		chargedTokens, punches, loyaltyFree, err = applyAbyssShopLoyalty(tx, uid, tokenCost)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	res, err := tx.Exec(`UPDATE users SET gold=gold-$1,abyss_tokens=abyss_tokens-$2
		WHERE client_uid=$3 AND gold >= $1 AND abyss_tokens >= $2`, item.CostGold, chargedTokens, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if count, _ := res.RowsAffected(); count == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold or tokens"})
		return
	}

	// Debit, grant, loyalty, and demand accounting share this transaction. A
	// failed reward can therefore never consume currency or a loyalty punch.
	msg := "Purchased " + item.Name + "!"
	switch item.Key {
	case "great_potions":
		err = grantAbyssShopConsumable(tx, uid, "great_health_potion", 3)
	case "repair_kits":
		err = grantAbyssShopConsumable(tx, uid, "master_repair_kit", 2)
	case "phoenix":
		err = grantAbyssShopConsumable(tx, uid, "phoenix_feather", 1)
	case "elixir_of_life":
		err = grantAbyssShopConsumable(tx, uid, "elixir_of_life", 1)
	case "giant_strength":
		err = grantAbyssShopConsumable(tx, uid, "giant_strength_elixir", 2)
	case "speed_elixir_pack":
		err = grantAbyssShopConsumable(tx, uid, "speed_elixir", 2)
	case "lucky_draught_pack":
		err = grantAbyssShopConsumable(tx, uid, "lucky_draught", 2)
	case "affix_suppressor":
		err = grantAbyssShopConsumable(tx, uid, "abyss_affix_suppressor", 1)
	case "abyss_gear":
		g := content.RandomAbyssGearDrop()
		err = grantAbyssShopGear(tx, uid, g)
		msg = "Cache opened: " + g.Rarity.String() + " " + g.Name + "!"
	case "epic_gear":
		g := rollAbyssBonusGear(50)
		err = grantAbyssShopGear(tx, uid, g)
		msg = "Epic cache opened: " + g.Rarity.String() + " " + g.Name + "!"
	case "relic":
		ui := content.RandomUniqueItem()
		result, insertErr := tx.Exec(`INSERT INTO user_unique_items (client_uid,item_name,rarity,power)
			VALUES ($1,$2,$3,$4) ON CONFLICT (client_uid,item_name) DO NOTHING`, uid, ui.Name, ui.Rarity, ui.Power)
		err = insertErr
		if insertErr == nil {
			if inserted, _ := result.RowsAffected(); inserted == 0 {
				writeJSON(w, map[string]any{"ok": false, "error": ui.Name + " already owned — nothing charged"})
				return
			}
		}
		msg = "Relic acquired: " + ui.Name + " [" + ui.Rarity.String() + "]!"
	case "emergency_revive":
		err = grantAbyssShopConsumable(tx, uid, "abyss_emergency_revive", 1)
	}
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "reward could not be delivered"})
		return
	}
	if abyssShopDemandEligible(item) {
		if err := recordAbyssShopDemandWith(tx, item.Key, now); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
	}
	var gold, tokens int64
	if err := tx.QueryRow("SELECT gold,abyss_tokens FROM users WHERE client_uid=$1", uid).Scan(&gold, &tokens); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if loyaltyFree {
		msg += " Loyalty reward: no tokens charged."
	}
	writeJSON(w, map[string]any{
		"ok": true, "msg": msg, "tokens": tokens, "gold": gold,
		"loyalty_punches": punches, "loyalty_free": loyaltyFree,
		"consumables": s.bot.getConsumables(uid),
	})
}

func grantAbyssShopGear(tx dbOrTx, uid string, gear content.Gear) error {
	data, err := json.Marshal(gear)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO user_inventory (client_uid,gear_id,durability,item_data) VALUES ($1,$2,$3,$4)", uid, gear.ID, gear.MaxDurability, string(data))
	return err
}
