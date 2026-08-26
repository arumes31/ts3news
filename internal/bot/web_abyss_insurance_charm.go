package bot

import "database/sql"

const (
	abyssInsuranceCharmID  = "abyss_insurance_charm"
	abyssInsuranceCharmPct = 50
)

func abyssConsumableCountsTowardCarryCap(id string) bool {
	return id != abyssInsuranceCharmID
}

func (b *Bot) abyssInsuranceCharmCount(uid string) int {
	var count int
	if err := b.DB.QueryRow(
		"SELECT remaining_fights FROM user_consumables WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights > 0",
		uid, abyssInsuranceCharmID,
	).Scan(&count); err != nil {
		return 0
	}
	return count
}

func abyssInsuranceCharmEligible(run abyssRun, pacts []string, hardcore, anchorActive bool) bool {
	if !run.Active || run.Escrow <= 0 || run.Insured > 0 || hardcore || anchorActive || abyssGraceProtected(run.Depth, hardcore) {
		return false
	}
	return !abyssHasPact(pacts, "uninsured") && !abyssHasPact(pacts, "abstinence")
}

// consumeAbyssInsuranceCharm spends one passive charm through the caller's
// forfeiture transaction. A later payout, history, or commit failure therefore
// restores the charge instead of losing it without settling the run.
func consumeAbyssInsuranceCharm(tx *sql.Tx, uid string) (bool, error) {
	result, err := tx.Exec(
		`UPDATE user_consumables
		 SET remaining_fights = remaining_fights - 1
		 WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights > 0`,
		uid, abyssInsuranceCharmID,
	)
	if err != nil {
		return false, err
	}
	updated, err := result.RowsAffected()
	if err != nil || updated == 0 {
		return false, err
	}
	if _, err := tx.Exec(
		"DELETE FROM user_consumables WHERE client_uid=$1 AND cons_id=$2 AND remaining_fights <= 0",
		uid, abyssInsuranceCharmID,
	); err != nil {
		return false, err
	}
	return true, nil
}
