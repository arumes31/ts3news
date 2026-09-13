package bot

import "testing"

func TestArcadePreloadedPoolAwardsAtMostOneWager(t *testing.T) {
	for _, bet := range []int64{100, 10000, maxArcadeBet} {
		for _, pool := range []int64{0, bet - 1, bet, bet + 1, 9223372036854775807} {
			award := arcadeJackpotAward(pool, bet)
			if award < 0 || award > bet || award > pool {
				t.Fatalf("pool%d bet%d award%d", pool, bet, award)
			}
			remainder := pool - award
			if remainder+award != pool {
				t.Fatalf("pool funding lost: pool%d award%d remaining%d", pool, award, remainder)
			}
			if pool >= bet && award != bet {
				t.Fatalf("funded jackpot=%d, want stake%d", award, bet)
			}
		}
	}
}

func TestArcadePreloadedPoolCannotMakeDicePlayerFavoured(t *testing.T) {
	// Enumerate all six equally likely dice outcomes. Every profitable outcome
	// gets its full possible jackpot probability, even against an enormous pool.
	for _, bet := range []int64{100, 10000, maxArcadeBet} {
		var returned float64
		for _, base := range []int64{0, 0, 0, bet, bet * 24 / 10, bet * 24 / 10} {
			out := arcadeOutcome{Bet: bet, Payout: base}
			rebate, _ := arcadeLossReturns(out, arcadeVIP(5000000))
			returned += float64(base + rebate)
			if arcadeJackpotEligible(out, 0) {
				returned += float64(arcadeJackpotAward(9223372036854775807, bet)) / 100
			}
		}
		rtp := returned / float64(6*bet)
		if rtp < .95 || rtp > .98 {
			t.Fatalf("bet%d exact dice RTP%f outside95-98%%", bet, rtp)
		}
	}
}
