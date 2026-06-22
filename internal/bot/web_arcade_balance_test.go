package bot

import (
	"math/rand/v2"
	"testing"
)

// gameChoices enumerates every arcade game and the choices a player can make
// ("" = the game takes no choice). The balance test verifies each game — and
// each individual choice — returns to the player less than it takes in, i.e. the
// house always keeps an edge and no option is exploitably player-favoured.
var gameChoices = map[string][]string{
	// Originals.
	"slots":    {""},
	"dice":     {""},
	"coinflip": {"heads", "tails"},
	"wheel":    {""},
	"highlow":  {"high", "low"},
}

func rtp(t *testing.T, game, choice string, rounds int, bet int64) float64 {
	t.Helper()
	// Fixed seed → deterministic, non-flaky RTP measurement.
	rng := rand.New(rand.NewPCG(0xA5CADE, uint64(len(game)*131+len(choice))))
	var paid int64
	for i := 0; i < rounds; i++ {
		out := playArcade(rng, game, bet, choice)
		if !out.OK {
			t.Fatalf("game %q choice %q returned not-OK: %s", game, choice, out.Error)
		}
		paid += out.Payout
	}
	return float64(paid) / float64(int64(rounds)*bet)
}

func TestArcadeBalance(t *testing.T) {
	const rounds = 400000
	const bet = 100
	// House edge band: every game/choice must return between 85% and 99.5% of
	// stakes over the long run. Above 100% would mean the player profits.
	const minRTP, maxRTP = 0.85, 0.995

	for game, choices := range gameChoices {
		for _, ch := range choices {
			got := rtp(t, game, ch, rounds, bet)
			label := game
			if ch != "" {
				label = game + "/" + ch
			}
			if got < minRTP || got > maxRTP {
				t.Errorf("%-18s RTP %.4f out of band [%.2f, %.3f]", label, got, minRTP, maxRTP)
			} else {
				t.Logf("%-18s RTP %.4f", label, got)
			}
		}
	}
}

// TestEveryGameIsDispatchable ensures no game in the balance table is silently
// unknown (which would surface as a refund-only "unknown game" at runtime).
func TestEveryGameIsDispatchable(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for game, choices := range gameChoices {
		out := playArcade(rng, game, 100, choices[0])
		if !out.OK {
			t.Errorf("game %q not dispatchable: %s", game, out.Error)
		}
	}
}
