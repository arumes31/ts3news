package bot

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestAbyssEscrowGrowthCannotOverflow(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                    string
		escrow, interest, bonus int64
	}{
		{"screenshot cache", 9_123_939_416_000_000_000, 8_000_000_000_000_000_000, 10_000},
		{"full cache", math.MaxInt64, 100, 100},
		{"large rewards", 0, math.MaxInt64, math.MaxInt64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := applyAbyssEscrowSoftCap(tc.escrow, tc.interest, tc.bonus, 439)
			if got.Escrow < tc.escrow || got.Bonus < 0 || got.Bonus > got.Escrow-tc.escrow {
				t.Fatalf("cache or reward wrapped: %#v", got)
			}
			if got.EfficiencyPct < 0 || got.EfficiencyPct > 100 {
				t.Fatalf("invalid efficiency: %#v", got)
			}
		})
	}
}

func TestAbyssNonCombatRewardUsesCombatCapAndNeverRepaysDeferredFloor(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name          string
		escrow, bonus int64
		rate          float64
		deferred      bool
		want          int64
	}{
		{"ordinary floor", 10_000, 2_000, 0.05, false, 12_500},
		{"over soft cap", 200_000, 8_000, 0.02, false, 203_000},
		{"deferred return", 200_000, 8_000, 0.02, true, 200_000},
		{"screenshot cache", 9_123_939_416_000_000_000, 10_000, 0.87, false, math.MaxInt64},
		{"full deferred cache", math.MaxInt64, 10_000, 0.87, true, math.MaxInt64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := abyssNonCombatReward(tc.escrow, tc.bonus, tc.rate, 10, tc.deferred)
			if got.Escrow != tc.want || (tc.deferred && got.Bonus != 0) {
				t.Fatalf("reward = %#v, want escrow %d", got, tc.want)
			}
		})
	}
}

func TestAbyssHighCacheBankSharesAndInsuranceStayValid(t *testing.T) {
	t.Parallel()
	const escrow int64 = 9_123_939_416_000_000_000
	for _, percent := range []int{25, 50} {
		t.Run(fmt.Sprint(percent), func(t *testing.T) {
			quote, ok := quoteAbyssPartialBank(escrow, 2, percent)
			if !ok || quote.Escrow != abyssGoldPercent(escrow, percent) ||
				quote.Remaining+quote.Escrow != escrow || quote.Gross <= 0 ||
				quote.Payout+quote.Fee != quote.Gross {
				t.Fatalf("partial quote = %#v, accepted = %v", quote, ok)
			}
		})
	}
	quote, ok := quoteAbyssTransportBank(escrow, 2)
	if !ok || quote.Gross != math.MaxInt64 || quote.Fee != abyssGoldPercent(math.MaxInt64, 15) ||
		quote.Payout+quote.Fee != math.MaxInt64 {
		t.Fatalf("transport quote = %#v, accepted = %v", quote, ok)
	}
	if refund := planAbyssForfeit(escrow, 25, 439, false).Refund; refund != abyssGoldPercent(escrow, 25) {
		t.Fatalf("insurance refund = %d", refund)
	}
}

func TestAbyssFocusRewardSharesFloorTransaction(t *testing.T) {
	t.Parallel()
	for _, grant := range []abyssNonCombatFocusGrant{
		{XP: 10, Label: "xp"},
		{Loot: &abyssLootGrant{Type: "mat", MatID: "shard", MatN: 2}, Label: "materials"},
		{Loot: &abyssLootGrant{Type: "tokens", Tokens: 2}, Label: "tokens"},
	} {
		for _, failureAt := range []string{"none", "flags", "reward", "commit"} {
			t.Run(grant.Label+"/"+failureAt, func(t *testing.T) {
				database, mock, err := sqlmock.New()
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = database.Close() }()
				failure := errors.New("save failed")
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE abyss_active SET escrow").WithArgs(int64(100), "delver").WillReturnResult(sqlmock.NewResult(0, 1))
				flags := mock.ExpectExec("INSERT INTO app_meta").WithArgs(abyssRunFlagsKey("delver"), "{}")
				if failureAt == "flags" {
					flags.WillReturnError(failure)
					mock.ExpectRollback()
				} else {
					flags.WillReturnResult(sqlmock.NewResult(0, 1))
					var reward *sqlmock.ExpectedExec
					if grant.XP > 0 {
						mock.ExpectQuery("SELECT xp FROM users").WithArgs("delver").WillReturnRows(sqlmock.NewRows([]string{"xp"}).AddRow(100))
						reward = mock.ExpectExec("UPDATE users SET xp").WithArgs("delver", 110, sqlmock.AnyArg())
					} else {
						reward = mock.ExpectExec("INSERT INTO abyss_escrow_loot").WithArgs("delver", grant.Loot.Type, grant.Label, sqlmock.AnyArg(), 12)
					}
					if failureAt == "reward" {
						reward.WillReturnError(failure)
						mock.ExpectRollback()
					} else {
						reward.WillReturnResult(sqlmock.NewResult(0, 1))
						if failureAt == "commit" {
							mock.ExpectCommit().WillReturnError(failure)
						} else {
							mock.ExpectCommit()
						}
					}
				}
				err = commitAbyssVictoryRunState(database, "delver", 100, map[string]int64{}, func(tx *sql.Tx) error {
					_, err := grant.save(tx, "delver", 12)
					return err
				})
				if failureAt == "none" && err != nil {
					t.Fatal(err)
				}
				if failureAt != "none" && !errors.Is(err, failure) {
					t.Fatalf("error = %v, want %v", err, failure)
				}
				if err := mock.ExpectationsWereMet(); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}
