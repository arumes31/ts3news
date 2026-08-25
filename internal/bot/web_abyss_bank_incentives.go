package bot

import (
	"database/sql"
	"strconv"
)

const (
	abyssBankStreakInsuranceEvery = 3
	abyssOvercapConversionPct     = 10
)

type abyssOvercapConversion struct {
	Gold   int64
	Tokens int
}

func abyssFreeInsuranceKey(uid string) string {
	return "abyss_free_insurance_" + uid
}

func abyssBanksUntilFreeInsurance(streak int, ready bool) int {
	if ready {
		return 0
	}
	return abyssBankStreakInsuranceEvery - max(streak, 0)%abyssBankStreakInsuranceEvery
}

func (b *Bot) abyssFreeInsuranceReady(uid string) bool {
	var value string
	if err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssFreeInsuranceKey(uid)).Scan(&value); err != nil {
		return false
	}
	ready, err := strconv.ParseBool(value)
	return err == nil && ready
}

// consumeAbyssFreeInsurance locks and consumes the one-shot voucher in the
// caller's transaction. A later failure rolls the consumption back with the
// premium and coverage changes.
func consumeAbyssFreeInsurance(tx *sql.Tx, uid string) (bool, error) {
	key := abyssFreeInsuranceKey(uid)
	if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, '0')
		ON CONFLICT (key) DO NOTHING`, key); err != nil {
		return false, err
	}
	var value string
	if err := tx.QueryRow("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", key).Scan(&value); err != nil {
		return false, err
	}
	ready, err := strconv.ParseBool(value)
	if err != nil || !ready {
		return false, nil
	}
	_, err = tx.Exec("UPDATE app_meta SET value='0' WHERE key=$1", key)
	return err == nil, err
}

func awardAbyssBankStreakInsurance(tx *sql.Tx, uid string, streak int) (bool, error) {
	if streak <= 0 || streak%abyssBankStreakInsuranceEvery != 0 {
		return false, nil
	}
	_, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, '1')
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssFreeInsuranceKey(uid))
	return err == nil, err
}

// abyssOvercapBankConversion exchanges a rounded 10% slice of cache above the
// current depth's soft cap at the normal gold-to-token rate. The rounding keeps
// sub-token gold in the bank payout instead of silently destroying it.
func abyssOvercapBankConversion(escrow int64, depth int) abyssOvercapConversion {
	overcap := max(escrow-abyssEscrowSoftCap(depth), int64(0))
	convertible := overcap / (100 / abyssOvercapConversionPct)
	tokens := convertible / int64(abyssTokenBuyGold)
	return abyssOvercapConversion{
		Gold:   tokens * int64(abyssTokenBuyGold),
		Tokens: int(tokens),
	}
}
