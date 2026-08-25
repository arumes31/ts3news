package bot

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPlanAbyssEntryRouteEnforcesCheckpointAndExpressRules(t *testing.T) {
	t.Parallel()

	checkpoint, message := planAbyssEntryRoute("checkpoint", 20, 37)
	if message != "" || checkpoint.Depth != 20 || checkpoint.CheckpointStart != 20 || checkpoint.TokenCost != 10 {
		t.Fatalf("checkpoint route = (%#v, %q), want depth 20 and 10-token cost", checkpoint, message)
	}
	if _, message := planAbyssEntryRoute("checkpoint", 15, 37); message == "" {
		t.Fatal("non-multiple-of-ten checkpoint was accepted")
	}
	if _, message := planAbyssEntryRoute("checkpoint", 40, 37); message == "" {
		t.Fatal("unreached checkpoint was accepted")
	}

	express, message := planAbyssEntryRoute("express", 0, 37)
	if message != "" || express.Depth != 32 || express.ExpressUntil != 37 || express.GoldCost != 32_000 {
		t.Fatalf("express route = (%#v, %q), want best-5 with per-depth cost", express, message)
	}
	if _, message := planAbyssEntryRoute("express", 0, 7); message == "" {
		t.Fatal("locked express route was accepted")
	}
}

func TestClaimAbyssDailyFreeEntryIsTransactionalAndOncePerDay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		paid      bool
		rows      int64
		updateErr error
		wantClaim bool
		wantErr   bool
	}{
		{name: "first paid entry", paid: true, rows: 1, wantClaim: true},
		{name: "already claimed today", paid: true, rows: 0},
		{name: "free tier does not consume claim", paid: false},
		{name: "database failure aborts claim", paid: true, updateErr: errors.New("write failed"), wantErr: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("sqlmock.New: %v", err)
			}
			defer func() { _ = database.Close() }()
			mock.ExpectBegin()
			tx, err := database.Begin()
			if err != nil {
				t.Fatalf("Begin: %v", err)
			}
			if test.paid {
				expectation := mock.ExpectExec("UPDATE users SET abyss_free_entry_date").WithArgs("player")
				if test.updateErr != nil {
					expectation.WillReturnError(test.updateErr)
				} else {
					expectation.WillReturnResult(sqlmock.NewResult(0, test.rows))
				}
			}
			mock.ExpectRollback()
			claimed, err := claimAbyssDailyFreeEntry(tx, "player", test.paid)
			if claimed != test.wantClaim || (err != nil) != test.wantErr {
				t.Fatalf("claim = (%v, %v), want claimed=%v error=%v", claimed, err, test.wantClaim, test.wantErr)
			}
			if err := tx.Rollback(); err != nil {
				t.Fatalf("Rollback: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestApplyAbyssRouteRewardPreservesRouteTradeoffs(t *testing.T) {
	t.Parallel()

	if bonus, skipped := applyAbyssRouteReward(100, 21, abyssRun{CheckpointStart: 20}); bonus != 75 || skipped {
		t.Fatalf("checkpoint reward = (%d, %v), want (75, false)", bonus, skipped)
	}
	if bonus, skipped := applyAbyssRouteReward(100, 37, abyssRun{ExpressUntil: 37}); bonus != 0 || !skipped {
		t.Fatalf("express catch-up reward = (%d, %v), want (0, true)", bonus, skipped)
	}
	if bonus, skipped := applyAbyssRouteReward(100, 38, abyssRun{ExpressUntil: 37}); bonus != 100 || skipped {
		t.Fatalf("express record reward = (%d, %v), want (100, false)", bonus, skipped)
	}
}

func TestAbyssMomentumStrengthCapsAtTwentyPercent(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		momentum int
		want     int
	}{{-1, 0}, {0, 0}, {1, 2}, {7, 14}, {10, 20}, {50, 20}} {
		if got := abyssMomentumStrength(test.momentum); got != test.want {
			t.Errorf("momentum %d strength = %d%%, want %d%%", test.momentum, got, test.want)
		}
	}
}
