package bot

import (
	"fmt"
	"math"

	"ts3news/internal/leveling"
)

type shopExchangeBalance struct {
	Gold  int64 `json:"gold"`
	XP    int   `json:"xp"`
	Level int   `json:"level"`
}

type shopExchangeQuote struct {
	Direction string              `json:"direction"`
	Amount    int64               `json:"amount"`
	Rate      int64               `json:"rate"`
	NextRate  int64               `json:"next_rate"`
	Spend     int64               `json:"spend"`
	Gain      int64               `json:"gain"`
	Unspent   int64               `json:"unspent"`
	Before    shopExchangeBalance `json:"before"`
	After     shopExchangeBalance `json:"after"`
	LevelLoss bool                `json:"level_loss"`
}

// Both preview and purchase use this calculation; only the purchase persists it.
func makeShopExchangeQuote(direction string, amount, gold int64, xp int, rate int64) (shopExchangeQuote, error) {
	q := shopExchangeQuote{Direction: direction, Amount: amount, Rate: rate,
		Before: shopExchangeBalance{Gold: gold, XP: xp, Level: leveling.LevelForXP(xp)}}
	q.After = q.Before
	if amount <= 0 {
		return q, fmt.Errorf("enter a positive whole amount")
	}
	switch direction {
	case "gold_to_xp":
		remaining := int64(leveling.XPForLevel(PrestigeThreshold)) - int64(xp)
		if remaining <= 0 {
			return q, fmt.Errorf("prestige before buying more XP")
		}
		if rate <= 0 {
			return q, fmt.Errorf("exchange pricing is unavailable; refresh the preview")
		}
		q.Gain = min(amount/rate, remaining)
		q.Spend = q.Gain * rate
		if q.Spend <= 0 {
			return q, fmt.Errorf("enter at least %d gold", rate)
		}
		if gold < q.Spend {
			return q, fmt.Errorf("not enough gold: this exchange needs %d gold; you have %d", q.Spend, gold)
		}
		q.After.Gold -= q.Spend
		q.After.XP += int(q.Gain)
		q.NextRate = rate
		if rate <= math.MaxInt64-goldPerXP/10 {
			q.NextRate += goldPerXP / 10
		}
	case "xp_to_gold":
		q.Rate = xpPerGold
		q.Spend = amount - amount%xpPerGold
		q.Gain = q.Spend / xpPerGold
		if q.Spend <= 0 || int64(xp) < q.Spend {
			return q, fmt.Errorf("not enough XP: enter between %d and %d XP", xpPerGold, xp)
		}
		if q.Gain > math.MaxInt64-gold {
			return q, fmt.Errorf("your gold balance cannot hold this exchange; spend some gold first")
		}
		q.After.Gold += q.Gain
		q.After.XP -= int(q.Spend)
	default:
		return q, fmt.Errorf("choose Gold to XP or XP to gold")
	}
	q.Unspent = amount - q.Spend
	q.After.Level = leveling.LevelForXP(q.After.XP)
	q.LevelLoss = q.After.Level < q.Before.Level
	return q, nil
}
