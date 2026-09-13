package bot

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestArcadeEffectiveGoldReturn(t *testing.T) {
	// Include every gold liability: base payout, loss-back and the funded pool.
	// Item awards are separate stock grants, not counted as gold payout.
	for game, choices := range gameChoices {
		for _, choice := range choices {
			rng := rand.New(rand.NewPCG(947, uint64(len(game)*137+len(choice))))
			const rounds, bet = 2_000_000, 10000
			var base, extra int64
			for i := 0; i < rounds; i++ {
				out := playArcade(rng, game, bet, choice)
				rebate, pool := arcadeLossReturns(out, arcadeVIP(5_000_000))
				base += out.Payout + pool
				extra += rebate
			}
			for _, total := range []int64{base, base + extra} {
				rtp := float64(total) / float64(rounds*bet)
				// 0.4 percentage-point sampling tolerance, primarily for slots.
				if rtp < .946 || rtp > .984 {
					t.Errorf("%s/%s total RTP %.5f outside target", game, choice, rtp)
				}
			}
		}
	}
}

func TestArcadeRetryReturnsReceiptWithoutAnotherCredit(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	want := arcadeOutcome{OK: true, Game: "dice", Bet: 100, Payout: 900, Net: 800, JackpotAmount: 660, Gold: 1800}
	encoded, _ := json.Marshal(arcadeReceipt{Epoch: "2", Outcome: want})
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gold, vip_points").WillReturnRows(sqlmock.NewRows([]string{"gold", "vip_points"}).AddRow(999, 10))
	mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
	mock.ExpectQuery("SELECT value FROM app_meta").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(encoded)))
	mock.ExpectRollback()
	out, err := (&Bot{DB: database}).settleArcade(context.Background(), "player", "dice", "", 100, "same-logical-round")
	if err != nil || out.Gold != 999 || out.Net != want.Net {
		t.Fatalf("retry %+v: %v", out, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestArcadeFailedPoolWriteRollsBackWager(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT gold, vip_points").WillReturnRows(sqlmock.NewRows([]string{"gold", "vip_points"}).AddRow(1000, 2141229522))
	mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
	mock.ExpectQuery("SELECT value FROM app_meta").WillReturnError(sql.ErrNoRows)
	mock.ExpectExec("SELECT set_config").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO arcade_jackpots").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT amount FROM arcade_jackpots").WillReturnRows(sqlmock.NewRows([]string{"amount"}).AddRow(600))
	mock.ExpectExec("UPDATE arcade_jackpots").WillReturnError(errors.New("write interrupted"))
	mock.ExpectRollback()
	_, err = (&Bot{DB: database}).settleArcade(context.Background(), "player", "dice", "", 100, "failed-logical-round")
	if err == nil {
		t.Fatal("failed settlement accepted")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestArcadeJackpotUsesProfitableResultForEveryGame(t *testing.T) {
	for game := range gameChoices {
		if !arcadeJackpotEligible(arcadeOutcome{Game: game, Bet: 100, Payout: 200}, 0) {
			t.Fatal(game)
		}
		if arcadeJackpotEligible(arcadeOutcome{Game: game, Bet: 100, Payout: 100}, 0) {
			t.Fatal("push claimed pool")
		}
	}
	for _, bet := range []int64{1, 10, 99, 100, 100000000} {
		rebate, pool := arcadeLossReturns(arcadeOutcome{Bet: bet}, arcadeVIP(5_000_000))
		if rebate+pool > bet*2/100 {
			t.Fatal("small wagers mint an unfunded pool")
		}
	}
}

func TestArcadePriorEconomyReceiptIsATombstone(t *testing.T) {
	for _, epoch := range []string{"", "1"} {
		t.Run("epoch"+epoch, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			encoded, err := json.Marshal(arcadeReceipt{Epoch: epoch, Outcome: arcadeOutcome{OK: true, Game: "dice", Bet: 100, Payout: 900, Net: 800, Gold: 1800}})
			if err != nil {
				t.Fatal(err)
			}
			mock.ExpectBegin()
			mock.ExpectQuery("SELECT gold, vip_points").WithArgs("player").WillReturnRows(sqlmock.NewRows([]string{"gold", "vip_points"}).AddRow(0, 0))
			mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"epoch"}).AddRow("2"))
			mock.ExpectQuery("SELECT value FROM app_meta").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(string(encoded)))
			mock.ExpectRollback()
			out, err := (&Bot{DB: database}).settleArcade(context.Background(), "player", "dice", "", 100, "previous-logical-round")
			if err == nil || !strings.Contains(err.Error(), "earlier economy") || out.OK || out.Gold != 0 {
				t.Fatalf("old receipt=%+v err=%v", out, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
