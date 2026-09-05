package bot

import (
	"database/sql"
	"encoding/json"

	"ts3news/internal/leveling"
)

// Focus grants share the floor-state transaction, so a failed completion cannot
// award XP or sealed loot again on retry.
type abyssNonCombatFocusGrant struct {
	XP    int
	Loot  *abyssLootGrant
	Label string
}

func (grant abyssNonCombatFocusGrant) save(tx *sql.Tx, uid string, depth int) (int, error) {
	newLevel := 0
	if grant.XP > 0 {
		var currentXP int
		if err := tx.QueryRow("SELECT xp FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&currentXP); err != nil {
			return 0, err
		}
		totalXP := currentXP + grant.XP
		newLevel = leveling.LevelForXP(totalXP)
		if _, err := tx.Exec("UPDATE users SET xp=$2, level=$3, last_seen=NOW() WHERE client_uid=$1", uid, totalXP, newLevel); err != nil {
			return 0, err
		}
	}
	if grant.Loot != nil {
		data, err := json.Marshal(grant.Loot)
		if err != nil {
			return 0, err
		}
		if _, err := tx.Exec(
			"INSERT INTO abyss_escrow_loot (client_uid, item_type, label, item_data, depth) VALUES ($1,$2,$3,$4,$5)",
			uid, grant.Loot.Type, grant.Label, data, max(depth, 0),
		); err != nil {
			return 0, err
		}
	}
	return newLevel, nil
}
