package bot

import (
	"log"
	"time"
)

const (
	abyssShopDemandDays       = 7
	abyssShopDemandRetention  = 8
	abyssShopDemandAdjustment = 10
)

type abyssShopDemand struct {
	Purchases int64
	Percent   int
}

func abyssShopDemandEligible(item abyssShopItem) bool {
	return item.Cost > 0 && len(item.Key) > 0 && !isAbyssInsanityShopItem(item)
}

func isAbyssInsanityShopItem(item abyssShopItem) bool {
	return len(item.Key) >= len("insanity_") && item.Key[:len("insanity_")] == "insanity_"
}

func abyssShopDemandAdjustments(counts map[string]int64) map[string]abyssShopDemand {
	eligible := 0
	var total int64
	for _, item := range abyssShopCatalog {
		if !abyssShopDemandEligible(item) {
			continue
		}
		eligible++
		total += max(int64(0), counts[item.Key])
	}
	out := make(map[string]abyssShopDemand, eligible)
	marketReady := eligible > 0 && total >= int64(eligible*2)
	for _, item := range abyssShopCatalog {
		if !abyssShopDemandEligible(item) {
			continue
		}
		purchases := max(int64(0), counts[item.Key])
		percent := 0
		if marketReady {
			scaled := purchases * int64(eligible)
			switch {
			case scaled*2 >= total*3:
				percent = abyssShopDemandAdjustment
			case scaled*2 <= total:
				percent = -abyssShopDemandAdjustment
			}
		}
		out[item.Key] = abyssShopDemand{Purchases: purchases, Percent: percent}
	}
	return out
}

func abyssShopDemandCost(cost int64, percent int) int64 {
	if cost <= 0 || percent == 0 {
		return cost
	}
	return max(int64(1), (cost*int64(100+percent)+50)/100)
}

func abyssShopPricedCost(item abyssShopItem, now time.Time, demandPercent int) (int64, bool) {
	cost := abyssShopDemandCost(item.Cost, demandPercent)
	_, deal := abyssShopEffectiveCost(item, now)
	if deal {
		cost = abyssDiscountedCost(cost)
	}
	return cost, deal
}

func (b *Bot) abyssShopDemand(now time.Time) map[string]abyssShopDemand {
	end := now.UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -abyssShopDemandDays)
	rows, err := b.DB.Query(`SELECT item_key,COALESCE(SUM(purchases),0) FROM abyss_shop_demand
		WHERE demand_day >= $1::date AND demand_day < $2::date GROUP BY item_key`, start, end)
	if err != nil {
		return abyssShopDemandAdjustments(nil)
	}
	defer func() { _ = rows.Close() }()
	counts := make(map[string]int64)
	for rows.Next() {
		var key string
		var purchases int64
		if rows.Scan(&key, &purchases) == nil {
			counts[key] = purchases
		}
	}
	if rows.Err() != nil {
		return abyssShopDemandAdjustments(nil)
	}
	return abyssShopDemandAdjustments(counts)
}

func (b *Bot) recordAbyssShopDemand(itemKey string, now time.Time) {
	_, err := b.DB.Exec(`WITH recorded AS (
		INSERT INTO abyss_shop_demand (demand_day,item_key,purchases) VALUES ($1::date,$2,1)
		ON CONFLICT (demand_day,item_key) DO UPDATE SET purchases=abyss_shop_demand.purchases+1 RETURNING 1
	) DELETE FROM abyss_shop_demand WHERE demand_day < $1::date - $3::int`,
		now.UTC().Truncate(24*time.Hour), itemKey, abyssShopDemandRetention)
	if err != nil {
		log.Printf("recording Abyss shop demand for %s: %v", itemKey, err)
	}
}
