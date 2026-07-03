package bot

import (
	"net/http"

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
	{"abyss_gear", "Abyss Gear Cache", "A random Abyss-exclusive gear piece.", 15, 0},
	{"epic_gear", "Epic Abyss Cache", "A guaranteed Epic-or-better Abyss piece.", 30, 0},
	{"relic", "Unstable Relic", "A random Unique item.", 40, 0},
	{"emergency_revive", "Emergency Revive Potion", "Single-use: instantly revive to full HP if you fall, beyond your normal one-per-run revival.", 0, 100000},
}

func abyssShopByKey(key string) (abyssShopItem, bool) {
	for _, it := range abyssShopCatalog {
		if it.Key == key {
			return it, true
		}
	}
	return abyssShopItem{}, false
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
		Item string `json:"item"`
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

	if item.CostGold > 0 {
		res, err := s.bot.DB.Exec(
			"UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1",
			item.CostGold, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
			return
		}
	} else {
		res, err := s.bot.DB.Exec(
			"UPDATE users SET abyss_tokens = abyss_tokens - $1 WHERE client_uid=$2 AND abyss_tokens >= $1",
			item.Cost, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens"})
			return
		}
	}

	// Grant the reward. Tokens are already debited; the grant helpers mirror the
	// loot path (auto-equip, stacking, dedupe) so shop items behave like drops.
	msg := "Purchased " + item.Name + "!"
	switch item.Key {
	case "great_potions":
		s.bot.grantConsumable(uid, "great_health_potion", 3)
	case "repair_kits":
		s.bot.grantConsumable(uid, "master_repair_kit", 2)
	case "phoenix":
		s.bot.grantConsumable(uid, "phoenix_feather", 1)
	case "elixir_of_life":
		s.bot.grantConsumable(uid, "elixir_of_life", 1)
	case "giant_strength":
		s.bot.grantConsumable(uid, "giant_strength_elixir", 2)
	case "speed_elixir_pack":
		s.bot.grantConsumable(uid, "speed_elixir", 2)
	case "lucky_draught_pack":
		s.bot.grantConsumable(uid, "lucky_draught", 2)
	case "abyss_gear":
		g := content.RandomAbyssGearDrop()
		got := s.bot.awardGearDrop(uid, g)
		msg = "Cache opened: " + got.Prefix + got.ItemName + "!"
	case "epic_gear":
		// Reuse the deep-bank reward roller at a depth that guarantees an Epic floor.
		name := s.bot.awardAbyssBonusGear(uid, 50)
		msg = "Epic cache opened: " + name + "!"
	case "relic":
		ui := content.RandomUniqueItem()
		s.bot.grantAbyssUnique(uid, ui.Name, ui.Rarity, ui.Power)
		msg = "Relic acquired: " + ui.Name + " [" + ui.Rarity.String() + "]!"
	case "emergency_revive":
		s.bot.grantConsumable(uid, "abyss_emergency_revive", 1)
	}

	var gold int64
	_ = s.bot.DB.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold)
	writeJSON(w, map[string]any{
		"ok": true, "msg": msg, "tokens": s.bot.abyssTokens(uid),
		"gold": gold, "consumables": s.bot.getConsumables(uid),
	})
}
