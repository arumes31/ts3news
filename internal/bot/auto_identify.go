package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"ts3news/internal/content"
)

// identifyGearCost follows the item's rarity tier, from Common to Eternal.
func identifyGearCost(rarity content.Rarity) int64 {
	costs := [...]int64{1, 5, 10, 20, 35, 50, 65, 80, 100}
	return costs[min(max(int(rarity), 0), len(costs)-1)]
}

func capIdentifyCharge(cost, gold int64) int64 {
	return min(max(cost, 0), max(gold, 0))
}

// chargeAutoIdentification runs after item locks and before item writes. Both
// the debit and daily free claim roll back if storing the identified gear fails.
func chargeAutoIdentification(ctx context.Context, tx *sql.Tx, uid string, total, firstCost int64) (int64, error) {
	var gold int64
	if err := tx.QueryRowContext(ctx, "SELECT gold FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&gold); err != nil {
		return 0, err
	}
	free, err := claimAbyssDailyIdentify(ctx, tx, uid)
	if err != nil {
		return 0, err
	}
	if free {
		total -= firstCost
	}
	cost := capIdentifyCharge(total, gold)
	if cost > 0 {
		if _, err := tx.ExecContext(ctx, "UPDATE users SET gold = gold - $1 WHERE client_uid=$2", cost, uid); err != nil {
			return 0, err
		}
	}
	return cost, nil
}

// storeAutoIdentifiedDrop keeps formerly hidden drops in inventory, with the
// identification fee and the new item committed together.
func (b *Bot) storeAutoIdentifiedDrop(ctx context.Context, uid string, gear content.Gear) (content.Gear, int64, error) {
	tx, err := b.DB.BeginTx(ctx, nil)
	if err != nil {
		return gear, 0, err
	}
	defer tx.Rollback()
	normalCost := identifyGearCost(gear.Rarity)
	cost, err := chargeAutoIdentification(ctx, tx, uid, normalCost, normalCost)
	if err != nil {
		return gear, 0, err
	}
	gear.Unidentified = false
	data, err := json.Marshal(gear)
	if err != nil {
		return gear, 0, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO user_inventory (client_uid, gear_id, durability, item_data) VALUES ($1,$2,$3,$4)",
		uid, gear.ID, gear.MaxDurability, string(data)); err != nil {
		return gear, 0, err
	}
	if err := tx.Commit(); err != nil {
		return gear, 0, err
	}
	return gear, cost, nil
}

// autoIdentifyItems also handles gear acquired before automatic identification
// was enabled. Only still-hidden rows are locked, so repeated calls cannot bill
// the same item twice. JSON updates preserve every other stored item attribute.
func (b *Bot) autoIdentifyItems(ctx context.Context, uid string) (int, int64, error) {
	tx, err := b.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	type hiddenItem struct {
		id   int64
		slot string
		cost int64
	}
	var items []hiddenItem
	queries := []string{
		"SELECT id, gear_id, item_data FROM user_inventory WHERE client_uid=$1 AND item_data->>'unidentified'='true' ORDER BY id FOR UPDATE",
		"SELECT slot, gear_id, item_data FROM user_gear WHERE client_uid=$1 AND item_data->>'unidentified'='true' ORDER BY slot FOR UPDATE",
	}
	for index, query := range queries {
		rows, err := tx.QueryContext(ctx, query, uid)
		if err != nil {
			return 0, 0, err
		}
		for rows.Next() {
			var item hiddenItem
			var gearID string
			var data sql.NullString
			var key any = &item.id
			if index == 1 {
				key = &item.slot
			}
			if err := rows.Scan(key, &gearID, &data); err != nil {
				rows.Close()
				return 0, 0, err
			}
			gear, ok := b.makeGear(gearID, data)
			if !ok {
				rows.Close()
				return 0, 0, fmt.Errorf("cannot identify unknown gear %q", gearID)
			}
			if gear.Unidentified {
				item.cost = identifyGearCost(gear.Rarity)
				items = append(items, item)
			}
		}
		readErr := rows.Err()
		closeErr := rows.Close()
		if readErr != nil {
			return 0, 0, readErr
		}
		if closeErr != nil {
			return 0, 0, closeErr
		}
	}
	if len(items) == 0 {
		return 0, 0, nil
	}
	var total int64
	for _, item := range items {
		total += item.cost
	}
	cost, err := chargeAutoIdentification(ctx, tx, uid, total, items[0].cost)
	if err != nil {
		return 0, 0, err
	}
	for _, item := range items {
		query := "UPDATE user_inventory SET item_data=jsonb_set(item_data, '{unidentified}', 'false') WHERE id=$1 AND client_uid=$2"
		var key any = item.id
		if item.slot != "" {
			query = "UPDATE user_gear SET item_data=jsonb_set(item_data, '{unidentified}', 'false') WHERE slot=$1 AND client_uid=$2"
			key = item.slot
		}
		if _, err := tx.ExecContext(ctx, query, key, uid); err != nil {
			return 0, 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return len(items), cost, nil
}
