package bot

import "database/sql"

type abyssEntryRoute struct {
	Depth           int
	CheckpointStart int
	ExpressUntil    int
	TokenCost       int64
	GoldCost        int64
}

func planAbyssEntryRoute(start string, checkpoint, bestDepth int) (abyssEntryRoute, string) {
	var route abyssEntryRoute
	switch start {
	case "checkpoint":
		if checkpoint <= 0 || checkpoint%10 != 0 || checkpoint > bestDepth {
			return route, "invalid checkpoint — pick a multiple of 10 you have already reached"
		}
		route.Depth = checkpoint
		route.CheckpointStart = checkpoint
		route.TokenCost = int64(checkpoint / 2)
	case "express":
		if bestDepth < 8 {
			return route, "the express elevator unlocks at best depth 8"
		}
		route.Depth = bestDepth - 5
		route.ExpressUntil = bestDepth
		route.GoldCost = int64(route.Depth) * abyssExpressGoldPerDepth
	}
	return route, ""
}

func claimAbyssDailyFreeEntry(tx *sql.Tx, uid string, paidEntry bool) (bool, error) {
	if !paidEntry {
		return false, nil
	}
	result, err := tx.Exec(`UPDATE users SET abyss_free_entry_date = CURRENT_DATE
		WHERE client_uid=$1 AND (abyss_free_entry_date IS NULL OR abyss_free_entry_date < CURRENT_DATE)`, uid)
	if err != nil {
		return false, err
	}
	claimed, err := result.RowsAffected()
	return claimed > 0, err
}

func applyAbyssRouteReward(bonus int64, depth int, run abyssRun) (int64, bool) {
	if run.CheckpointStart > 0 {
		bonus = bonus * 3 / 4
	}
	if run.ExpressUntil > 0 && depth <= run.ExpressUntil {
		return 0, true
	}
	return bonus, false
}

func abyssMomentumStrength(momentum int) int {
	if momentum < 0 {
		return 0
	}
	if momentum > 10 {
		momentum = 10
	}
	return momentum * 2
}
