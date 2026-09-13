package bot

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"

	"ts3news/internal/db"
)

type arcadeReceipt struct {
	Epoch   string        `json:"epoch"`
	Choice  string        `json:"choice"`
	Outcome arcadeOutcome `json:"outcome"`
}

func arcadeVIP(points int) vipTier {
	result := vipTiers[0]
	for _, tier := range vipTiers {
		if points >= tier.Points {
			result = tier
		}
	}
	return result
}

// The progressive pool is funded from losses, never reseeded on a claim.
// All games can claim it on a profitable base result.
func arcadeJackpotEligible(out arcadeOutcome, draw int) bool {
	return out.Payout > out.Bet && draw == 0
}

func arcadeLossReturns(out arcadeOutcome, vip vipTier) (rebate, contribution int64) {
	if out.Payout == 0 {
		rebate = out.Bet * int64(vip.Rebate) / 100
	}
	if out.Payout < out.Bet {
		contribution = (out.Bet - out.Payout) / 400
	}
	return
}

// settleArcade commits the wager, all rewards, VIP progress, history and receipt
// together. The account lock serializes retries before examining the receipt.
func (b *Bot) settleArcade(ctx context.Context, uid, game, choice string, bet int64, requestID string) (arcadeOutcome, error) {
	if len(requestID) < 16 || len(requestID) > 80 {
		return arcadeOutcome{}, errors.New("missing or invalid round ID; reload the arcade")
	}
	key := fmt.Sprintf("arcade_receipt:%x", sha256.Sum256([]byte(uid+"\x00"+requestID)))
	tx, err := b.DB.BeginTx(ctx, nil)
	if err != nil {
		return arcadeOutcome{}, fmt.Errorf("begin arcade: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var before int64
	var points int
	if err := tx.QueryRowContext(ctx, "SELECT gold, vip_points FROM users WHERE client_uid=$1 FOR UPDATE", uid).Scan(&before, &points); err != nil {
		return arcadeOutcome{}, fmt.Errorf("load arcade account: %w", err)
	}
	var epoch string
	if err := tx.QueryRowContext(ctx, "SELECT COALESCE((SELECT value FROM app_meta WHERE key='gold_economy_version'),'0')").Scan(&epoch); err != nil {
		return arcadeOutcome{}, err
	}
	var saved string
	err = tx.QueryRowContext(ctx, "SELECT value FROM app_meta WHERE key=$1", key).Scan(&saved)
	if err == nil {
		var receipt arcadeReceipt
		if err := json.Unmarshal([]byte(saved), &receipt); err != nil {
			return arcadeOutcome{}, err
		}
		if receipt.Choice != choice || receipt.Outcome.Game != game || receipt.Outcome.Bet != bet {
			return arcadeOutcome{}, errors.New("round ID already used for another wager")
		}
		if receipt.Epoch != epoch {
			return arcadeOutcome{}, errors.New("round belongs to an earlier economy; no new wager was placed")
		}
		receipt.Outcome.Gold = before
		return receipt.Outcome, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return arcadeOutcome{}, err
	}
	if before < bet {
		return arcadeOutcome{}, errors.New("not enough gold")
	}
	if err := db.SetEconomyContext(ctx, tx, "arcade", requestID, "", ""); err != nil {
		return arcadeOutcome{}, err
	}
	// #nosec G404 -- server-side game simulation, not credential generation.
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	out := playArcade(rng, game, bet, choice)
	if !out.OK {
		return out, errors.New(out.Error)
	}
	out.BasePayout = out.Payout
	points = min(5_000_000, points+int(bet/10))
	var contribution int64
	out.Rebate, contribution = arcadeLossReturns(out, arcadeVIP(points))
	if _, err := tx.ExecContext(ctx, "INSERT INTO arcade_jackpots (game_key,amount) VALUES ('global',0) ON CONFLICT DO NOTHING"); err != nil {
		return out, err
	}
	var pool int64
	if err := tx.QueryRowContext(ctx, "SELECT amount FROM arcade_jackpots WHERE game_key='global' FOR UPDATE").Scan(&pool); err != nil {
		return out, err
	}
	if arcadeJackpotEligible(out, rng.IntN(100)) && pool > 0 {
		out.JackpotWin, out.JackpotAmount = true, arcadeJackpotAward(pool, bet)
		out.Detail = "GLOBAL JACKPOT! " + out.Detail
		pool -= out.JackpotAmount
	}
	out.Payout += out.JackpotAmount + out.Rebate
	out.Net = out.Payout - bet
	out.Win = out.Net > 0
	out.NewJackpot = pool + contribution
	if _, err := tx.ExecContext(ctx, "UPDATE arcade_jackpots SET amount=$1, updated_at=NOW() WHERE game_key='global'", out.NewJackpot); err != nil {
		return out, err
	}
	if err := tx.QueryRowContext(ctx, "/* economy:bot.Bot.settleArcade */ UPDATE users SET gold=gold+$1, vip_points=$2 WHERE client_uid=$3 RETURNING gold", out.Net, points, uid).Scan(&out.Gold); err != nil {
		return out, err
	}
	// Paid wagers award only budgeted gold; vendable gear would subsidize small bets.
	if _, err := tx.ExecContext(ctx, "INSERT INTO game_results (client_uid,game,won,net) VALUES ($1,'arcade',$2,$3)", uid, out.Win, out.Net); err != nil {
		return out, err
	}
	data, err := json.Marshal(arcadeReceipt{Epoch: epoch, Choice: choice, Outcome: out})
	if err != nil {
		return out, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO app_meta (key,value) VALUES ($1,$2)", key, string(data)); err != nil {
		return out, err
	}
	if err := tx.Commit(); err != nil {
		return out, err
	}
	return out, nil
}

func arcadeJackpotAward(pool, bet int64) int64 { return max(int64(0), min(pool, bet)) }
