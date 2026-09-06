package bot

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/leveling"
)

// Exchange rates: initially spend 10,000 gold to gain 1 XP; spend 10 XP to gain 5 gold
// (i.e. 2 XP per gold).
const (
	goldPerXP     = 10_000 // base gold spent to gain 1 XP, +10% per weekly purchase
	xpPerGold     = 2      // XP spent to gain 1 gold (10 XP → 5 gold)
	shopStockSize = 48
)

// gearPrice is the reference gear value, scaled by combat power and rarity.
// Vendor/sell value is half this; shop-only rarity premiums live in shopGearPrice.
func gearPrice(g content.Gear) int64 {
	base := g.CombatRating()*12 + float64(g.Stats.Score())*6
	multiplier := max(int64(g.Rarity)+1, 1)
	if base >= float64(math.MaxInt64/multiplier) {
		return math.MaxInt64
	}
	p := int64(base) * multiplier
	if p < 25 {
		p = 25
	}
	return p
}

func itoa(n int) string     { return strconv.Itoa(n) }
func ftoa(f float64) string { return strconv.FormatFloat(f, 'g', -1, 64) }

// Shop stock rotates on windows whose length is a deterministic, pseudo-random
// value between shopMinHours and shopMaxHours. All players share one rotation.
const (
	shopMinHours   = 1
	shopMaxHours   = 6
	shopAnchorUnix = 1735689600 // 2025-01-01T00:00:00Z — fixed rotation origin
)

// shopWindowDuration returns the length (in seconds) of shop window idx, in
// [shopMinHours, shopMaxHours] hours, derived deterministically from idx.
func shopWindowDuration(idx int64) int64 {
	// #nosec G115 -- idx is always non-negative (window index)
	h := uint64(idx)*0x9E3779B97F4A7C15 + 0x123456789
	h ^= h >> 29
	span := uint64(shopMaxHours - shopMinHours + 1)
	hours := int64(h%span) + shopMinHours
	return hours * 3600
}

// shopWindow returns the current rotation window's stock seed and end time by
// walking fixed-origin windows until the one containing now.
func shopWindow(now time.Time) (seed int64, endsAt time.Time) {
	start := int64(shopAnchorUnix)
	nowU := now.Unix()
	if nowU < start {
		return 0, time.Unix(start+shopWindowDuration(0), 0).UTC()
	}
	var idx int64
	for {
		dur := shopWindowDuration(idx)
		if start+dur > nowU {
			return idx, time.Unix(start+dur, 0).UTC()
		}
		start += dur
		idx++
	}
}

type shopItemView struct {
	itemAtlasView
	gear            content.Gear
	RarityBoosted   bool
	EternalBonusPct string

	ID          string
	Name        string
	Slot        string
	Icon        string
	Rarity      string
	RarityColor string
	CR          float64
	Score       int
	Price       int64
	IsUpgrade   bool
	Featured    bool // the Mythic/Divine showcase relic (priced in the millions)
	Effects     []string
	Stats       []statKV
	Specials    []itemSpecialView
	XPBonusPct  int
	Element     string
	InspectJSON string
	Comparison  shopComparisonView
}

// featuredShopView builds the shop card for the seed's showcase relic.
func featuredShopView(seed int64, equippedGear map[string]content.Gear) shopItemView {
	g := content.FeaturedShopItem(seed)
	gearView := toGearView(g.Slot, g)
	effs := make([]string, 0, len(g.BonusEffects)+1)
	if g.Special != content.EffectNone {
		effs = append(effs, string(g.Special))
	}
	for _, e := range g.BonusEffects {
		effs = append(effs, string(e))
	}
	return shopItemView{
		gear:          g,
		itemAtlasView: gearView.itemAtlasView,
		ID:            g.ID,
		Name:          g.Name,
		Slot:          string(g.Slot),
		Icon:          content.SlotIcon(g.Slot),
		Rarity:        g.Rarity.String(),
		RarityColor:   g.Rarity.Color(),
		CR:            g.CombatRating(),
		Score:         g.Stats.Score(),
		Price:         shopGearPrice(g),
		IsUpgrade:     isGearUpgrade(g, equippedGear),
		Featured:      true,
		Effects:       effs,
		Stats:         gearView.Stats,
		Specials:      gearSpecialViews(g),
		XPBonusPct:    gearView.XPBonusPct,
		Element:       gearView.Element,
		InspectJSON:   gearView.InspectJSON,
		Comparison:    shopGearComparison(g, equippedGear),
	}
}

func stockForSeed(seed int64, equippedGear map[string]content.Gear) []shopItemView {
	return personalizedShopStock(seed, "", shopBuffState{}, equippedGear)
}

func regularShopView(g content.Gear, equippedGear map[string]content.Gear) shopItemView {
	gearView := toGearView(g.Slot, g)

	return shopItemView{
		gear:          g,
		itemAtlasView: gearView.itemAtlasView,
		ID:            g.ID,
		Name:          g.Name,
		Slot:          string(g.Slot),
		Icon:          content.SlotIcon(g.Slot),
		Rarity:        g.Rarity.String(),
		RarityColor:   g.Rarity.Color(),
		CR:            g.CombatRating(),
		Score:         g.Stats.Score(),
		Price:         shopGearPrice(g),
		IsUpgrade:     isGearUpgrade(g, equippedGear),
		Effects:       append([]string(nil), gearViewEffectNames(g)...),
		Stats:         gearView.Stats,
		Specials:      gearSpecialViews(g),
		XPBonusPct:    gearView.XPBonusPct,
		Element:       gearView.Element,
		InspectJSON:   gearView.InspectJSON,
		Comparison:    shopGearComparison(g, equippedGear),
	}
}

func gearViewEffectNames(g content.Gear) []string {
	specials := gearSpecialViews(g)
	out := make([]string, 0, len(specials))
	for _, special := range specials {
		out = append(out, special.Name)
	}
	return out
}

func (s *WebServer) handleShopPage(w http.ResponseWriter, r *http.Request, uid string) {
	u, err := s.loadWebUser(uid)
	if err != nil {
		http.Redirect(w, r, "/denied", http.StatusSeeOther)
		return
	}

	// Upgrade badges use the exact equipped instances, including forge/custom data.
	equippedGear := s.bot.equippedGearUpgradeIndex(uid)

	seed, endsAt := shopWindow(time.Now())
	refreshIn := int(time.Until(endsAt).Seconds())
	if refreshIn < 0 {
		refreshIn = 0
	}
	count, err := loadShopXPExchangeCount(s.bot.DB, uid, shopXPExchangeWeek(time.Now()))
	if err != nil {
		http.Error(w, "exchange pricing unavailable", http.StatusServiceUnavailable)
		return
	}
	buffs, err := loadShopBuffs(r.Context(), s.bot.DB, uid)
	if err != nil {
		http.Error(w, "shop bonuses unavailable", http.StatusServiceUnavailable)
		return
	}
	page, _ := strconv.ParseInt(r.URL.Query().Get("stock_page"), 10, 64)
	pagination := shopStockPagination(seed, uid, buffs, page)
	s.render(w, "shop", map[string]any{
		"Title":           "Shop",
		"Nav":             "shop",
		"U":               u,
		"Stock":           personalizedShopStockPage(seed, uid, buffs, equippedGear, pagination.Page),
		"StockPagination": pagination,
		"Buffs":           shopBuffViews(buffs),
		"StockRevision":   shopStockRevision(seed, uid, buffs),
		"RefreshIn":       refreshIn,
		"GoldPerXP":       shopXPExchangeRate(count),
		"PrestigeXP":      leveling.XPForLevel(PrestigeThreshold),
		"XPPerGold":       xpPerGold,
	})
}

// handleExchangeAPI previews or commits exchanges using the same server quote.
// Reviewed clients submit the quoted wallet and rate; legacy callers remain valid.
func (s *WebServer) handleExchangeAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Direction        string               `json:"direction"` // "gold_to_xp" | "xp_to_gold"
		Amount           int64                `json:"amount"`    // amount of the input resource to spend
		Rate             int64                `json:"rate"`      // optional quote; stale prices require a new preview
		Preview          bool                 `json:"preview"`
		Expected         *shopExchangeBalance `json:"expected"`
		ConfirmLevelLoss bool                 `json:"confirm_level_loss"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Amount <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "Enter a positive whole amount and try again."})
		return
	}

	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "The exchange could not start. Your balance is unchanged; try again."})
		return
	}
	defer func() { _ = tx.Rollback() }()

	var gold int64
	var xp int
	// Lock the wallet row before calculating either side of the exchange. Other
	// purchases and rewards update this same row, so the lock prevents this
	// transaction's absolute gold/XP write from erasing a concurrent change.
	if err := tx.QueryRow("SELECT gold, xp FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&gold, &xp); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "Your character balance could not be loaded. Refresh the shop and try again."})
		return
	}

	if !req.Preview && req.Expected != nil && (req.Expected.Gold != gold || req.Expected.XP != xp) {
		writeJSON(w, map[string]any{"ok": false, "review_required": true, "error": "Your balance changed. Review a fresh preview before converting."})
		return
	}
	rate := int64(goldPerXP)
	var count int64
	week := shopXPExchangeWeek(time.Now())
	if req.Direction == "gold_to_xp" {
		remainingXP := int64(leveling.XPForLevel(PrestigeThreshold)) - int64(xp)
		if remainingXP <= 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "prestige before buying more XP"})
			return
		}
		count, err = loadShopXPExchangeCount(tx, uid, week)
		if err != nil || count == math.MaxInt64 {
			writeJSON(w, map[string]any{"ok": false, "error": "Exchange pricing is unavailable. Your balance is unchanged; try again shortly."})
			return
		}
		rate = shopXPExchangeRate(count)
		if !req.Preview && req.Rate != 0 && req.Rate != rate {
			writeJSON(w, map[string]any{"ok": false, "error": "exchange price changed; review the new rate and try again", "gold_per_xp": rate})
			return
		}
	}
	quote, err := makeShopExchangeQuote(req.Direction, req.Amount, gold, xp, rate)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if req.Preview {
		writeJSON(w, map[string]any{"ok": true, "quote": quote})
		return
	}
	if req.Expected != nil && quote.LevelLoss && !req.ConfirmLevelLoss {
		writeJSON(w, map[string]any{"ok": false, "review_required": true, "error": "This exchange lowers your level. Review and confirm the level loss first."})
		return
	}
	if req.Direction == "gold_to_xp" {
		history, _ := json.Marshal(shopXPExchangeState{Week: week, Count: count + 1})
		if _, err := tx.Exec("INSERT INTO app_meta (key,value) VALUES ($1,$2) ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value", shopXPExchangeKey(uid), string(history)); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "The exchange could not be saved. Your balance is unchanged; try again."})
			return
		}
	}
	gold, xp, newLevel := quote.After.Gold, quote.After.XP, quote.After.Level
	if _, err := tx.Exec("UPDATE users SET gold=$1, xp=$2, level=$3 WHERE client_uid=$4", gold, xp, newLevel, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "The exchange could not be saved. Your balance is unchanged; try again."})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "unconfirmed": true, "error": "The exchange result could not be confirmed. Refresh to verify before trying again."})
		return
	}
	var unspentGold int64
	if req.Direction == "gold_to_xp" {
		unspentGold = quote.Unspent
	}
	writeJSON(w, map[string]any{
		"ok": true, "detail": "Exchange complete. Your balance and level are updated below.", "gold": gold, "xp": xp, "level": newLevel,
		"level_name":  leveling.LevelName(newLevel),
		"gold_per_xp": quote.NextRate, "unspent_gold": unspentGold,
	})
}

// handleBuyAPI buys a gear item from today's rotating stock for its fair price
// and places it in the inventory.
func (s *WebServer) handleBuyAPI(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID            string `json:"id"`
		ExpectedGold  *int64 `json:"expected_gold"`
		StockRevision string `json:"stock_revision"`
		StockPage     int64  `json:"stock_page"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "The purchase request could not be read. Refresh the shop and try again."})
		return
	}

	tx, err := s.bot.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "The purchase could not start. Your gold is unchanged; try again."})
		return
	}
	defer func() { _ = tx.Rollback() }()

	// Token purchases lock this same wallet. Resolve stock after taking the lock
	// so a concurrent token purchase cannot change the item being delivered.
	var gold int64
	if err := tx.QueryRowContext(r.Context(), "SELECT gold FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&gold); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "Your balance could not be loaded. Refresh the shop and try again."})
		return
	}
	if req.ExpectedGold != nil && *req.ExpectedGold != gold {
		writeJSON(w, map[string]any{"ok": false, "review_required": true, "error": "Your balance changed. Refresh the shop to review your current balance."})
		return
	}
	buffs, err := loadShopBuffs(r.Context(), tx, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "Your shop bonuses could not be loaded. No gold was spent."})
		return
	}
	seed, _ := shopWindow(time.Now())
	if req.StockPage != shopStockPagination(seed, uid, buffs, req.StockPage).Page {
		writeJSON(w, map[string]any{"ok": false, "review_required": true, "error": "This stock page changed. Refresh the shop."})
		return
	}
	if (req.StockRevision != "" || buffs != (shopBuffState{})) && req.StockRevision != shopStockRevision(seed, uid, buffs) {
		writeJSON(w, map[string]any{"ok": false, "review_required": true, "error": "Your shop bonuses or stock changed. Refresh to review the current items and prices."})
		return
	}
	var chosen *shopItemView
	for _, item := range personalizedShopStockPage(seed, uid, buffs, nil, req.StockPage) {
		if item.ID == req.ID {
			chosen = &item
			break
		}
	}
	if chosen == nil {
		writeJSON(w, map[string]any{"ok": false, "review_required": true, "error": "This item is no longer in stock. Refresh the shop to see the new rotation."})
		return
	}
	g := chosen.gear

	query := "UPDATE users SET gold = gold - $1 WHERE client_uid=$2 AND gold >= $1"
	args := []any{chosen.Price, uid}
	if req.ExpectedGold != nil {
		query += " AND gold = $3"
		args = append(args, *req.ExpectedGold)
	}
	res, err := tx.Exec(query, args...)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "The purchase could not be saved. Your gold is unchanged; try again."})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "review_required": true, "error": "Your balance changed or you need more gold. Refresh the shop to review your current balance."})
		return
	}
	var equippedMsg = ""
	equipped := false
	itemDataBytes, _ := json.Marshal(g)

	if s.bot.shouldEquip(uid, g) {
		// Displace and equip
		if err := s.bot.equipGear(tx, uid, g, g.MaxDurability, string(itemDataBytes)); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "This item could not be equipped. The purchase was cancelled; try again."})
			return
		}
		equippedMsg = " and equipped!"
		equipped = true
	} else {
		// Deliver to inventory
		if _, err := tx.Exec("INSERT INTO user_inventory (client_uid, gear_id, durability, item_data) VALUES ($1, $2, $3, $4)", uid, g.ID, g.MaxDurability, string(itemDataBytes)); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "This item could not be added to inventory. The purchase was cancelled; try again."})
			return
		}
	}

	// Read the post-purchase balance inside the transaction to avoid a race with
	// other concurrent operations between commit and a separate query.
	if err := tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1", uid).Scan(&gold); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "Your remaining balance could not be checked. The purchase was cancelled; try again."})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "unconfirmed": true, "error": "The purchase result could not be confirmed. Refresh to verify before trying again."})
		return
	}

	writeJSON(w, map[string]any{"ok": true, "bought": g.Name + equippedMsg, "gold": gold, "equipped": equipped})
}
