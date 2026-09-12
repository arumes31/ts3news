package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

const shopBuffPriceStep int64 = 1_000_000

// Counts are permanent and deliberately independent of weekly exchange history.
type shopBuffState struct {
	Rarity   int64 `json:"rarity"`
	Quantity int64 `json:"quantity"`
}

func shopBuffPrice(owned int64) int64 {
	// Clamp before adding so even a corrupt maximum counter cannot wrap the price.
	return (min(max(owned, 0), int64(999)) + 1) * shopBuffPriceStep
}

// shopBuffTotalPrice sums the rising prices and the capped tail without looping
// over a user-controlled amount. Reject overflow before charging the wallet.
func shopBuffTotalPrice(owned, amount int64) (int64, error) {
	if owned < 0 || amount < 1 || amount > math.MaxInt64-owned {
		return 0, errors.New("Choose a valid whole number of tokens.")
	}
	rising := min(amount, max(int64(999)-owned, 0))
	var total int64
	if rising > 0 {
		total = rising * (2*owned + rising + 1) / 2 * shopBuffPriceStep
	}
	capped := amount - rising
	const capPrice = 1000 * shopBuffPriceStep
	if capped > (math.MaxInt64-total)/capPrice {
		return 0, errors.New("The total token price is too large.")
	}
	return total + capped*capPrice, nil
}

func parseShopBuffState(raw string) (shopBuffState, error) {
	var state *shopBuffState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return shopBuffState{}, fmt.Errorf("decode permanent shop buffs: %w", err)
	}
	if state == nil || state.Rarity < 0 || state.Quantity < 0 {
		return shopBuffState{}, errors.New("invalid permanent shop buff counters")
	}
	return *state, nil
}

type shopBuffReader interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func shopBuffKey(uid string) string { return "shop_permanent_buffs_" + uid }

func loadShopBuffs(ctx context.Context, reader shopBuffReader, uid string) (shopBuffState, error) {
	var raw string
	err := reader.QueryRowContext(ctx, "SELECT value FROM app_meta WHERE key=$1", shopBuffKey(uid)).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return shopBuffState{}, nil
	}
	if err != nil {
		return shopBuffState{}, fmt.Errorf("load permanent shop buffs: %w", err)
	}
	return parseShopBuffState(raw)
}
