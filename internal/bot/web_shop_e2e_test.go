//go:build e2e

package bot

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	"ts3news/internal/content"
	"ts3news/internal/leveling"
)

// Isolated in-memory wallet: browser tests never modify a real player account.
func registerShopReviewE2EFixture(mux *http.ServeMux, server *WebServer) {
	var lock sync.Mutex
	type fixtureWallet struct {
		gold  int64
		xp    int
		count int64
		buffs shopBuffState
	}
	wallets := map[string]*fixtureWallet{}
	getWallet := func(w http.ResponseWriter, r *http.Request) *fixtureWallet {
		if cookie, err := r.Cookie("shop_fixture"); err == nil {
			if wallet := wallets[cookie.Value]; wallet != nil {
				return wallet
			}
		}
		id := strconv.Itoa(len(wallets) + 1)
		wallet := &fixtureWallet{gold: 25_000_000, xp: 10_000}
		if r.URL.Query().Get("buff_fixture") == "boosted" {
			wallet.buffs = shopBuffState{Rarity: 1000, Quantity: 1000}
		}
		if r.URL.Query().Get("buff_fixture") == "endless" {
			wallet.buffs = shopBuffState{Rarity: 10000, Quantity: 10000}
			wallet.gold = 3_000_000_000
		}
		wallets[id] = wallet
		http.SetCookie(w, &http.Cookie{Name: "shop_fixture", Value: id, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
		return wallet
	}
	equipped := map[string]content.Gear{}
	for _, item := range content.ShopStock(42, shopStockSize) {
		if _, exists := equipped[string(item.Slot)]; exists {
			continue
		}
		item.Name = "Equipped " + item.Name
		item.Stats.STR += 90
		item.Stats.DEF += 10
		equipped[string(item.Slot)] = item
	}
	mux.HandleFunc("/shop", func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		defer lock.Unlock()
		wallet := getWallet(w, r)
		gold, xp := wallet.gold, wallet.xp
		page, _ := strconv.ParseInt(r.URL.Query().Get("stock_page"), 10, 64)
		pagination := shopStockPagination(42, "shop-e2e", wallet.buffs, page)
		level := leveling.LevelForXP(xp)
		if err := server.tmpl.ExecuteTemplate(w, "shop", map[string]any{
			"Title": "Shop", "Nav": "shop", "EnableAbyss": true,
			"U":     &webUser{UID: "shop-e2e", Nickname: "Shop Tester", Gold: gold, XP: xp, Level: level, LevelName: leveling.LevelName(level)},
			"Stock": personalizedShopStockPage(42, "shop-e2e", wallet.buffs, equipped, pagination.Page), "GoldPerXP": shopXPExchangeRate(wallet.count), "XPPerGold": xpPerGold,
			"StockPagination": pagination,
			"Buffs":           shopBuffViews(wallet.buffs), "StockRevision": shopStockRevision(42, "shop-e2e", wallet.buffs),
			"PrestigeXP": leveling.XPForLevel(PrestigeThreshold), "RefreshIn": int64(3600),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("/api/shop/exchange", func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		defer lock.Unlock()
		wallet := getWallet(w, r)
		gold, xp, count := wallet.gold, wallet.xp, wallet.count
		var req struct {
			Direction        string               `json:"direction"`
			Amount           int64                `json:"amount"`
			Preview          bool                 `json:"preview"`
			Rate             int64                `json:"rate"`
			Expected         *shopExchangeBalance `json:"expected"`
			ConfirmLevelLoss bool                 `json:"confirm_level_loss"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Invalid fixture request."})
			return
		}
		rate := shopXPExchangeRate(count)
		q, err := makeShopExchangeQuote(req.Direction, req.Amount, gold, xp, rate)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if req.Preview {
			writeJSON(w, map[string]any{"ok": true, "quote": q})
			return
		}
		if req.Expected == nil || req.Expected.Gold != gold || req.Expected.XP != xp || (req.Direction == "gold_to_xp" && req.Rate != rate) || (q.LevelLoss && !req.ConfirmLevelLoss) {
			writeJSON(w, map[string]any{"ok": false, "review_required": true, "error": "Your balance or rate changed. Review a fresh preview before converting."})
			return
		}
		gold, xp = q.After.Gold, q.After.XP
		if req.Direction == "gold_to_xp" {
			count++
		}
		wallet.gold, wallet.xp, wallet.count = gold, xp, count
		writeJSON(w, map[string]any{"ok": true, "gold": gold, "xp": xp, "level": q.After.Level, "level_name": leveling.LevelName(q.After.Level), "gold_per_xp": q.NextRate})
	})
	mux.HandleFunc("/api/shop/buffs", func(w http.ResponseWriter, r *http.Request) {
		lock.Lock()
		defer lock.Unlock()
		wallet := getWallet(w, r)
		var req shopBuffPurchaseRequest
		if json.NewDecoder(r.Body).Decode(&req) != nil || (req.Kind != "rarity" && req.Kind != "quantity") || req.ExpectedOwned == nil || req.ExpectedGold == nil {
			writeJSON(w, map[string]any{"ok": false, "error": "Invalid fixture request."})
			return
		}
		owned := &wallet.buffs.Rarity
		if req.Kind == "quantity" {
			owned = &wallet.buffs.Quantity
		}
		if *req.ExpectedGold != wallet.gold || *req.ExpectedOwned != *owned {
			writeJSON(w, map[string]any{"ok": false, "review_required": true, "error": "Your balance or tokens changed. Refresh to review."})
			return
		}
		price := shopBuffPrice(*owned)
		if price > wallet.gold {
			writeJSON(w, map[string]any{"ok": false, "error": "Not enough gold."})
			return
		}
		wallet.gold -= price
		*owned++
		writeJSON(w, map[string]any{"ok": true, "gold": wallet.gold, "buffs": wallet.buffs, "price": price, "next_price": shopBuffPrice(*owned)})
	})
}
