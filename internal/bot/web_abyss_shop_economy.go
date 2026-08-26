package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"
)

var errAbyssAutoInsuranceFunds = errors.New("not enough gold for automatic insurance")

const (
	abyssPotionSubscriptionDailyGold = int64(25_000)
	abyssRepairSubscriptionDailyGold = int64(10_000)
	abyssRepairSubscriptionDays      = 7
	abyssScratchCost                 = 50
)

var abyssInsanityCosmeticKeys = []string{
	"insanity_void_aura",
	"insanity_glass_crown",
	"insanity_depth_trail",
}

type abyssShopItemView struct {
	abyssShopItem
	EffectiveCost int64
	DiscountPct   int
	DemandPct     int
	DemandSales   int64
	HappyAccident bool
	Insanity      bool
	Owned         bool
	RotationWeek  string
	RotationEnds  string
}

func abyssEconomyDayIndex(now time.Time) int {
	utc := now.UTC()
	return utc.Year()*366 + utc.YearDay()
}

func abyssHappyAccidentIndex(now time.Time, size int) int {
	if size <= 0 {
		return -1
	}
	return abyssEconomyDayIndex(now) % size
}

func abyssTokenBundleRate(purchases int) int64 {
	return 100_000 + int64(max(0, purchases))*20_000
}

func abyssVendorLoyaltyPercent(completedSales int) int {
	if completedSales >= 5 {
		return 2
	}
	return 0
}

func abyssScratchReward(roll float64) int {
	switch {
	case roll < 0.01:
		return 1_000
	case roll < 0.10:
		return 150
	case roll < 0.30:
		return 75
	default:
		return 0
	}
}

func abyssWeeklyInsanityCosmetic(now time.Time) string {
	utc := now.UTC()
	daysSinceMonday := (int(utc.Weekday()) + 6) % 7
	monday := time.Date(utc.Year(), utc.Month(), utc.Day()-daysSinceMonday, 0, 0, 0, 0, time.UTC)
	weekIndex := monday.Unix() / int64((7*24*time.Hour)/time.Second)
	index := int(weekIndex % int64(len(abyssInsanityCosmeticKeys)))
	if index < 0 {
		index += len(abyssInsanityCosmeticKeys)
	}
	return abyssInsanityCosmeticKeys[index]
}

func abyssWeeklyCosmeticReset(now time.Time) time.Time {
	utc := now.UTC()
	daysUntilMonday := (8 - int(utc.Weekday())) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	return time.Date(utc.Year(), utc.Month(), utc.Day()+daysUntilMonday, 0, 0, 0, 0, time.UTC)
}

func abyssShopEffectiveCost(item abyssShopItem, now time.Time) (int64, bool) {
	if item.Cost <= 0 {
		return item.Cost, false
	}
	eligible := make([]abyssShopItem, 0, len(abyssShopCatalog))
	for _, candidate := range abyssShopCatalog {
		if candidate.Cost > 0 && !strings.HasPrefix(candidate.Key, "insanity_") {
			eligible = append(eligible, candidate)
		}
	}
	if len(eligible) == 0 {
		return item.Cost, false
	}
	deal := eligible[abyssHappyAccidentIndex(now, len(eligible))].Key == item.Key
	if !deal {
		return item.Cost, false
	}
	return abyssDiscountedCost(item.Cost), true
}

func (b *Bot) abyssShopViewsWithOwned(uid string, now time.Time, ownedCosmetics map[string]bool) []abyssShopItemView {
	b.maybeDeliverAbyssPotionSubscription(uid, now)
	demand := b.abyssShopDemand(now)
	activeCosmetic := abyssWeeklyInsanityCosmetic(now)
	rotationWeek := abyssEconomyWeek(now)
	rotationEnds := abyssWeeklyCosmeticReset(now).Format("2006-01-02 15:04 UTC")
	out := make([]abyssShopItemView, 0, len(abyssShopCatalog))
	for _, item := range abyssShopCatalog {
		insanity := strings.HasPrefix(item.Key, "insanity_")
		if insanity && item.Key != activeCosmetic {
			continue
		}
		market := demand[item.Key]
		cost, deal := abyssShopPricedCost(item, now, market.Percent)
		out = append(out, abyssShopItemView{
			abyssShopItem: item, EffectiveCost: cost, DiscountPct: map[bool]int{true: 40}[deal],
			DemandPct: market.Percent, DemandSales: market.Purchases,
			HappyAccident: deal, Insanity: insanity, Owned: ownedCosmetics[item.Key],
			RotationWeek: rotationWeek, RotationEnds: rotationEnds,
		})
	}
	return out
}

func (b *Bot) abyssOwnedShopCosmetics(uid string) map[string]bool {
	rows, err := b.DB.Query("SELECT cosmetic_key FROM abyss_shop_cosmetics WHERE client_uid=$1", uid)
	if err != nil {
		return map[string]bool{}
	}
	defer func() { _ = rows.Close() }()
	owned := map[string]bool{}
	for rows.Next() {
		var key string
		if rows.Scan(&key) == nil {
			owned[key] = true
		}
	}
	return owned
}

func (s *WebServer) buyAbyssShopCosmetic(w http.ResponseWriter, uid string, item abyssShopItem, tokenCost int64) {
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec(`INSERT INTO abyss_shop_cosmetics (client_uid,cosmetic_key) VALUES ($1,$2)
		ON CONFLICT (client_uid,cosmetic_key) DO NOTHING`, uid, item.Key)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	newlyOwned, _ := res.RowsAffected()
	punches, loyaltyFree := 0, false
	if newlyOwned > 0 {
		var charged int64
		charged, punches, loyaltyFree, err = applyAbyssShopLoyalty(tx, uid, tokenCost)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		res, err = tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens-$1 WHERE client_uid=$2 AND abyss_tokens >= $1", charged, uid)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": "db"})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			writeJSON(w, map[string]any{"ok": false, "error": "not enough tokens"})
			return
		}
	}
	title := map[string]string{
		"insanity_void_aura": "Void-Touched", "insanity_glass_crown": "Glass Sovereign", "insanity_depth_trail": "Depthwalker",
	}[item.Key]
	if _, err := tx.Exec(`UPDATE users SET title=$2,title_mult=1,title_expires=NOW()+INTERVAL '30 days',title_source='abyss_shop'
		WHERE client_uid=$1`, uid, title); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	message := "Cosmetic equipped: " + title + "."
	if newlyOwned > 0 {
		message = "Permanent cosmetic unlocked and equipped: " + title + "."
	}
	if loyaltyFree {
		message += " Loyalty reward: no tokens charged."
	}
	writeJSON(w, map[string]any{"ok": true, "owned": true, "newly_owned": newlyOwned > 0, "tokens": s.bot.abyssTokens(uid),
		"loyalty_punches": punches, "loyalty_free": loyaltyFree, "msg": message})
}

func (b *Bot) maybeDeliverAbyssPotionSubscription(uid string, now time.Time) {
	tx, err := b.DB.Begin()
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback() }()
	var active bool
	var delivered sql.NullTime
	if err := tx.QueryRow(`SELECT potion_subscription, potion_delivery_date
		FROM abyss_economy_profiles WHERE client_uid=$1 FOR UPDATE`, uid).Scan(&active, &delivered); err != nil || !active {
		return
	}
	today := now.UTC().Truncate(24 * time.Hour)
	if delivered.Valid && !delivered.Time.Before(today) {
		return
	}
	res, err := tx.Exec("UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold >= $1", abyssPotionSubscriptionDailyGold, uid)
	if err != nil {
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return
	}
	if _, err := tx.Exec(`INSERT INTO user_consumables (client_uid, cons_id, remaining_fights)
		VALUES ($1,'great_health_potion',2)
		ON CONFLICT (client_uid,cons_id) DO UPDATE SET remaining_fights=user_consumables.remaining_fights+2`, uid); err != nil {
		return
	}
	if _, err := tx.Exec("UPDATE abyss_economy_profiles SET potion_delivery_date=$2::date WHERE client_uid=$1", uid, today); err != nil {
		return
	}
	if _, err := tx.Exec(`INSERT INTO abyss_economy_events (client_uid,kind,message,amount)
		VALUES ($1,'subscription','Potion subscription delivered two Great Health Potions.',$2)`, uid, -abyssPotionSubscriptionDailyGold); err != nil {
		return
	}
	_ = tx.Commit()
}

func (b *Bot) abyssRepairSubscriptionActive(uid string, now time.Time) bool {
	var active bool
	_ = b.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM abyss_economy_profiles
		WHERE client_uid=$1 AND repair_until > $2)`, uid, now).Scan(&active)
	return active
}

func (b *Bot) abyssAutoInsureEnabled(uid string) bool {
	var enabled bool
	_ = b.DB.QueryRow(`SELECT EXISTS(SELECT 1 FROM abyss_economy_profiles
		WHERE client_uid=$1 AND auto_insure=TRUE)`, uid).Scan(&enabled)
	return enabled
}

func applyAbyssAutoInsurance(tx *sql.Tx, uid string, plan abyssAutoInsurancePlan) (cost int64, free bool, err error) {
	if !plan.Applied {
		return 0, false, nil
	}
	var availableGold int64
	if err := tx.QueryRow("SELECT gold FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&availableGold); err != nil {
		return 0, false, err
	}
	free, err = consumeAbyssFreeInsurance(tx, uid)
	if err != nil {
		return 0, false, err
	}
	if free {
		return 0, true, nil
	}
	if availableGold < plan.Cost {
		return 0, false, errAbyssAutoInsuranceFunds
	}
	if _, err := tx.Exec("UPDATE users SET gold=gold-$1 WHERE client_uid=$2", plan.Cost, uid); err != nil {
		return 0, false, err
	}
	return plan.Cost, false, nil
}

func abyssRepairSubscriptionCharge(cost int64, covered bool) int64 {
	if covered {
		return 0
	}
	return max(int64(0), cost)
}

func (s *WebServer) handleAbyssPotionSubscription(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	_, err := s.bot.DB.Exec(`INSERT INTO abyss_economy_profiles (client_uid,potion_subscription)
		VALUES ($1,$2) ON CONFLICT (client_uid) DO UPDATE SET potion_subscription=$2`, uid, req.Enabled)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if req.Enabled {
		s.bot.maybeDeliverAbyssPotionSubscription(uid, time.Now())
	}
	writeJSON(w, map[string]any{"ok": true, "enabled": req.Enabled,
		"gold": s.bot.abyssGold(uid), "tokens": s.bot.abyssTokens(uid), "consumables": s.bot.getConsumables(uid),
		"msg": fmt.Sprintf("Potion subscription %s · %dg per delivered day.", map[bool]string{true: "enabled", false: "paused"}[req.Enabled], abyssPotionSubscriptionDailyGold)})
}

func (s *WebServer) handleAbyssAutoInsure(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "bad request"})
		return
	}
	_, err := s.bot.DB.Exec(`INSERT INTO abyss_economy_profiles (client_uid,auto_insure)
		VALUES ($1,$2) ON CONFLICT (client_uid) DO UPDATE SET auto_insure=$2`, uid, req.Enabled)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	state := map[bool]string{true: "enabled", false: "paused"}[req.Enabled]
	writeJSON(w, map[string]any{"ok": true, "enabled": req.Enabled,
		"msg": "Auto-insure " + state + " · compatible runs start with 25% cache cover."})
}

func (s *WebServer) handleAbyssRepairSubscription(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	cost := abyssRepairSubscriptionDailyGold * abyssRepairSubscriptionDays
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec("UPDATE users SET gold=gold-$1 WHERE client_uid=$2 AND gold >= $1", cost, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	if _, err := tx.Exec(`INSERT INTO abyss_economy_profiles (client_uid,repair_until)
		VALUES ($1,NOW()+INTERVAL '7 days') ON CONFLICT (client_uid) DO UPDATE SET
		repair_until=GREATEST(COALESCE(abyss_economy_profiles.repair_until,NOW()),NOW())+INTERVAL '7 days'`, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "gold": s.bot.abyssGold(uid),
		"msg": fmt.Sprintf("Repair plan active for %d days · %dg/day, all durability loss covered.", abyssRepairSubscriptionDays, abyssRepairSubscriptionDailyGold)})
}

func (s *WebServer) handleAbyssTokenBundle(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	var req struct {
		Count int `json:"count"`
	}
	if err := readJSON(r, &req); err != nil || req.Count < 1 || req.Count > 25 {
		writeJSON(w, map[string]any{"ok": false, "error": "count must be 1–25"})
		return
	}
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`INSERT INTO abyss_economy_profiles (client_uid) VALUES ($1) ON CONFLICT DO NOTHING`, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	var day sql.NullTime
	var bought int
	if err := tx.QueryRow(`SELECT token_bundle_date,token_bundle_count FROM abyss_economy_profiles
		WHERE client_uid=$1 FOR UPDATE`, uid).Scan(&day, &bought); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if !day.Valid || day.Time.UTC().Format("2006-01-02") != time.Now().UTC().Format("2006-01-02") {
		bought = 0
	}
	var cost int64
	for i := 0; i < req.Count; i++ {
		cost += abyssTokenBundleRate(bought + i)
	}
	res, err := tx.Exec("UPDATE users SET gold=gold-$1,abyss_tokens=abyss_tokens+$2 WHERE client_uid=$3 AND gold >= $1", cost, req.Count, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "not enough gold"})
		return
	}
	if _, err := tx.Exec(`UPDATE abyss_economy_profiles SET token_bundle_date=CURRENT_DATE,token_bundle_count=$2 WHERE client_uid=$1`, uid, bought+req.Count); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "cost": cost, "tokens": s.bot.abyssTokens(uid), "gold": s.bot.abyssGold(uid),
		"next_rate": abyssTokenBundleRate(bought + req.Count), "msg": fmt.Sprintf("Bought %d tokens for %dg; next token costs %dg today.", req.Count, cost, abyssTokenBundleRate(bought+req.Count))})
}

func (s *WebServer) handleAbyssScratch(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`INSERT INTO abyss_economy_profiles (client_uid) VALUES ($1) ON CONFLICT DO NOTHING`, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	res, err := tx.Exec(`UPDATE abyss_economy_profiles SET scratch_date=CURRENT_DATE
		WHERE client_uid=$1 AND (scratch_date IS NULL OR scratch_date<CURRENT_DATE)`, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "scratch card already played today"})
		return
	}
	reward := abyssScratchReward(rand.Float64()) // #nosec G404 -- posted game odds, not a security token
	res, err = tx.Exec("UPDATE users SET abyss_tokens=abyss_tokens-$1+$2 WHERE client_uid=$3 AND abyss_tokens >= $1", abyssScratchCost, reward, uid)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "need 50 tokens"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "reward": reward, "tokens": s.bot.abyssTokens(uid),
		"msg": fmt.Sprintf("Scratch card settled: %d tokens returned. Odds: 70%%/0 · 20%%/75 · 9%%/150 · 1%%/1000.", reward)})
}

func abyssDiscountedCost(cost int64) int64 {
	return max(1, cost*60/100)
}
