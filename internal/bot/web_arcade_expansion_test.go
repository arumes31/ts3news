package bot

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestArcadeRejectsInvalidWagersBeforeAccessingAccount(t *testing.T) {
	for _, body := range []string{
		`{"game":"vault","bet":100,"choice":"4"}`,
		`{"game":"expedition","bet":100,"choice":"cheat"}`,
		`{"game":"coinflip","bet":100,"choice":"edge"}`,
		`{"game":"highlow","bet":100,"choice":"seven"}`,
		`{"game":"memory","bet":100}`,
		`{"game":"dice","bet":0}`,
		`{"game":"dice","bet":100000001}`,
	} {
		t.Run(body, func(t *testing.T) {
			// A nil bot deliberately panics if validation touches the economy.
			server := &WebServer{}
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/arcade/play", strings.NewReader(body))
			server.handleArcadeAPI(response, request, "invalid-arcade-choice")
			var outcome arcadeOutcome
			if err := json.Unmarshal(response.Body.Bytes(), &outcome); err != nil {
				t.Fatal(err)
			}
			if outcome.OK || outcome.Error == "" {
				t.Fatalf("invalid wager was accepted: %s", response.Body.String())
			}
		})
	}
}

func TestArcadeHundredMillionWagerLimit(t *testing.T) {
	for _, game := range []struct{ name, choice string }{
		{"slots", ""}, {"dice", ""}, {"coinflip", "heads"}, {"wheel", ""},
		{"highlow", "high"}, {"vault", "1"}, {"expedition", "abyss"},
	} {
		t.Run(game.name, func(t *testing.T) {
			database, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			server := &WebServer{bot: &Bot{DB: database}}
			for _, wager := range []int64{100001, 100000000} {
				// Accepted wagers still need an affordable balance; stop before RNG and rewards.
				mock.ExpectExec("UPDATE users SET gold = gold -").WithArgs(wager, "delver").WillReturnResult(sqlmock.NewResult(0, 0))
				body := fmt.Sprintf(`{"game":%q,"choice":%q,"bet":%d}`, game.name, game.choice, wager)
				response := httptest.NewRecorder()
				server.handleArcadeAPI(response, httptest.NewRequest(http.MethodPost, "/api/arcade/play", strings.NewReader(body)), "delver")
				if !strings.Contains(response.Body.String(), "not enough gold") {
					t.Fatalf("wager %d rejected before balance check: %s", wager, response.Body.String())
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
			// Identical random draws must scale payouts exactly at the new limit.
			small, large := rand.New(rand.NewPCG(42, 42)), rand.New(rand.NewPCG(42, 42))
			for i := 0; i < 2000; i++ {
				base := playArcade(small, game.name, 100, game.choice)
				out := playArcade(large, game.name, 100000000, game.choice)
				if !out.OK || out.Payout != base.Payout*1000000 {
					t.Fatalf("large wager payout: %+v; base: %+v", out, base)
				}
			}
		})
	}
}

func TestArcadeVaultOutcomes(t *testing.T) {
	rng := rand.New(rand.NewPCG(21, 39))
	for _, choice := range []string{"1", "2", "3"} {
		wins, losses := 0, 0
		for i := 0; i < 100; i++ {
			out := playArcade(rng, "vault", 100, choice)
			if !out.OK || out.Chest < 1 || out.Chest > 3 {
				t.Fatalf("invalid vault outcome: %+v", out)
			}
			if itoa(out.Chest) == choice {
				wins++
				if out.Payout != 285 {
					t.Fatalf("vault win pays %d, want 285", out.Payout)
				}
			} else {
				losses++
				if out.Payout != 0 {
					t.Fatalf("empty chest pays %d", out.Payout)
				}
			}
		}
		if wins == 0 || losses == 0 {
			t.Fatal("both vault outcomes must be exercised")
		}
	}
}

func TestArcadeExpeditionOutcomes(t *testing.T) {
	for _, tc := range []struct {
		choice string
		chance int
		payout int64
	}{
		{"scout", 80, 120}, {"delve", 48, 200}, {"abyss", 24, 400},
	} {
		t.Run(tc.choice, func(t *testing.T) {
			rng := rand.New(rand.NewPCG(52, 11))
			for i := 0; i < 300; i++ {
				out := playArcade(rng, "expedition", 100, tc.choice)
				if !out.OK || out.Roll < 1 || out.Roll > 100 || out.Chance != tc.chance {
					t.Fatalf("invalid expedition outcome: %+v", out)
				}
				want := int64(0)
				if out.Roll <= tc.chance {
					want = tc.payout
				}
				if out.Payout != want {
					t.Fatalf("roll %d pays %d, want %d", out.Roll, out.Payout, want)
				}
			}
		})
	}
}

func TestArcadeExpansionRejectsInvalidChoices(t *testing.T) {
	for _, game := range []string{"vault", "expedition"} {
		for _, choice := range []string{"", "-1", "4", "cheat"} {
			out := playArcade(rand.New(rand.NewPCG(1, 2)), game, 100, choice)
			if out.OK || out.Payout != 0 {
				t.Fatalf("%s accepted invalid choice %q", game, choice)
			}
		}
	}
}
