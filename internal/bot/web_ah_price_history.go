package bot

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
)

const ahPriceHistoryLimit = 12

type ahPriceHistoryView struct {
	Points    string
	Count     int
	Min       int64
	Max       int64
	Latest    int64
	Direction string
}

func buildAHPriceHistory(prices []int64) ahPriceHistoryView {
	if len(prices) == 0 {
		return ahPriceHistoryView{}
	}
	view := ahPriceHistoryView{Count: len(prices), Min: prices[0], Max: prices[0], Latest: prices[len(prices)-1]}
	for _, price := range prices[1:] {
		view.Min = min(view.Min, price)
		view.Max = max(view.Max, price)
	}
	var points strings.Builder
	for index, price := range prices {
		x := 50
		if len(prices) > 1 {
			x = index * 100 / (len(prices) - 1)
		}
		y := 15
		if view.Max > view.Min {
			ratio := float64(price-view.Min) / float64(view.Max-view.Min)
			y = 27 - int(ratio*24)
		}
		if index > 0 {
			points.WriteByte(' ')
		}
		_, _ = fmt.Fprintf(&points, "%d,%d", x, y)
	}
	view.Points = points.String()
	if prices[len(prices)-1] > prices[0] {
		view.Direction = "up"
	} else if prices[len(prices)-1] < prices[0] {
		view.Direction = "down"
	} else {
		view.Direction = "flat"
	}
	return view
}

func (b *Bot) ahPriceHistories(listings []ahListingView) map[string]ahPriceHistoryView {
	ids := make([]string, 0, len(listings))
	seen := make(map[string]bool, len(listings))
	for _, listing := range listings {
		if listing.ItemID != "" && !seen[listing.ItemID] {
			seen[listing.ItemID] = true
			ids = append(ids, listing.ItemID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	rows, err := b.DB.Query(`SELECT item_id,price FROM (
		SELECT item_id,price,sold_at,
			ROW_NUMBER() OVER (PARTITION BY item_id ORDER BY sold_at DESC) AS sale_rank
		FROM auction_house WHERE sold_at IS NOT NULL AND item_id=ANY($1)
		  AND (item_type <> 'gear' OR LOWER(COALESCE(item_data->>'unidentified','false')) <> 'true')
	) sales WHERE sale_rank <= $2 ORDER BY item_id,sold_at ASC`, pq.Array(ids), ahPriceHistoryLimit)
	if err != nil {
		return nil
	}
	defer func() { _ = rows.Close() }()
	prices := make(map[string][]int64, len(ids))
	for rows.Next() {
		var itemID string
		var price int64
		if err := rows.Scan(&itemID, &price); err == nil {
			prices[itemID] = append(prices[itemID], price)
		}
	}
	if rows.Err() != nil {
		return nil
	}
	views := make(map[string]ahPriceHistoryView, len(prices))
	for itemID, series := range prices {
		views[itemID] = buildAHPriceHistory(series)
	}
	return views
}
