package bot

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

// Standard healing consumables share the Abyss inventory and definitions.
// Corrupted potions require their backlash mechanic and are not offered here.
func riftPotionAmount(id string, maxHP float64) (float64, error) {
	potion, ok := content.GetConsumableByID(id)
	if !ok || potion.Type != content.ConsumableHealing || content.IsCorruptedConsumable(id) {
		return 0, errors.New("this item is not a supported healing potion")
	}
	amount := potion.EffectValue
	if amount <= 1 {
		amount *= maxHP
	}
	return math.Max(1, math.Floor(amount)), nil
}

func useRiftPotion(ctx context.Context, tx *sql.Tx, uid string, run *rift.Run, id string) error {
	amount, err := riftPotionAmount(id, run.Player.MaxHP)
	if err != nil {
		return err
	}
	if err = run.UseHealingPotion(amount); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "UPDATE user_consumables SET remaining_fights=remaining_fights-1 WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights>0", uid, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return errors.New("healing potion is no longer available")
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM user_consumables WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights<=0", uid, id)
	return err
}

// riftPotionOption describes owned, supported base-potency healing items only.
type riftPotionOption struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Count        int     `json:"count"`
	HealHP       float64 `json:"heal_hp,omitempty"`
	HealFraction float64 `json:"heal_fraction,omitempty"`
}

func (b *Bot) riftPotions(ctx context.Context, uid string) ([]riftPotionOption, error) {
	rows, err := b.DB.QueryContext(ctx, "SELECT cons_id, remaining_fights FROM user_consumables WHERE client_uid=$1 AND remaining_fights>0 ORDER BY cons_id", uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []riftPotionOption{}
	for rows.Next() {
		var id string
		var count int
		if err := rows.Scan(&id, &count); err != nil {
			return nil, err
		}
		if count <= 0 {
			continue
		}
		if _, err := riftPotionAmount(id, 100); err != nil {
			continue
		}
		potion, _ := content.GetConsumableByID(id)
		item := riftPotionOption{ID: id, Name: potion.Name, Count: count}
		if potion.EffectValue <= 1 {
			item.HealFraction = potion.EffectValue
		} else {
			item.HealHP = potion.EffectValue
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
