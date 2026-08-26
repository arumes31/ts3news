package bot

import (
	"database/sql"
	"strconv"
	"time"
)

const (
	abyssRunFlagPerfect             = "perfect_run"
	abyssRunFlagDeathWish           = "death_wish"
	abyssRunFlagAnchorRune          = "anchor_rune"
	abyssRunFlagColdMuscles         = "cold_muscles"
	abyssRunFlagDefensiveMomentum   = "def_momentum"
	abyssRunFlagCheckpointTokenCost = "checkpoint_token_cost"
	abyssRunFlagSecondLastStandUsed = "second_last_stand_used"
	abyssRunFlagHybrid              = "hybrid"
)

type abyssCoreLoopView struct {
	GreedyGripStacks       int   `json:"greedy_grip_stacks"`
	GreedyInterestPct      int   `json:"greedy_interest_pct"`
	GreedyDEFPct           int   `json:"greedy_def_pct"`
	HeavyPocketsPenaltyPct int   `json:"heavy_pockets_penalty_pct"`
	IdleDangerPct          int   `json:"idle_danger_pct"`
	DeathWish              bool  `json:"death_wish"`
	AnchorRune             bool  `json:"anchor_rune"`
	PerfectRun             bool  `json:"perfect_run"`
	BankConfirm            bool  `json:"bank_confirm"`
	EchoSeed               int64 `json:"echo_seed"`
	FranticFeePct          int   `json:"frantic_fee_pct"`
	LowHP                  bool  `json:"low_hp"`
	InsuranceMin           int   `json:"insurance_min"`
	InsuranceMax           int   `json:"insurance_max"`
	InsuranceStep          int   `json:"insurance_step"`
	Insured                int   `json:"insured"`
	FreeInsuranceReady     bool  `json:"free_insurance_ready"`
	BankStreak             int   `json:"bank_streak"`
	BanksToFreeInsurance   int   `json:"banks_to_free_insurance"`
	OvercapGold            int64 `json:"overcap_gold"`
	OvercapTokens          int   `json:"overcap_tokens"`
	RestShrinkReady        bool  `json:"rest_shrink_ready"`
	RestShrinkCache        int64 `json:"rest_shrink_cache"`
	RestShrinkTokens       int64 `json:"rest_shrink_tokens"`
	RafflePot              int64 `json:"raffle_pot"`
	DownedTimeoutSeconds   int   `json:"downed_timeout_seconds"`
	ReviveStreak           int   `json:"revive_streak"`
	ReviveChancePct        int   `json:"revive_chance_pct"`
	Hybrid                 bool  `json:"hybrid"`
	TokenAnte              int   `json:"token_ante"`
	RiskDialPct            int   `json:"risk_dial_pct"`
	ColdMusclesFloors      int64 `json:"cold_muscles_floors"`
	DefensiveMomentum      int64 `json:"defensive_momentum"`
}

func (b *Bot) abyssCoreLoopStatus(uid string, run abyssRun) abyssCoreLoopView {
	flags := b.loadRunFlags(uid)
	cache, tokens := abyssRestCacheConversion(run.Escrow)
	stats := b.loadAbyssStats(uid)
	freeInsuranceReady := b.abyssFreeInsuranceReady(uid)
	overcap := abyssOvercapBankConversion(run.Escrow, run.Depth)
	maxHP := b.abyssCombatStats(uid).HP
	status := abyssCoreLoopView{
		GreedyGripStacks:       abyssGreedyGripStacks(run.Depth),
		HeavyPocketsPenaltyPct: abyssHeavyPocketsPct(run.Escrow),
		IdleDangerPct:          abyssIdleDangerPct(run.LastActionAt, time.Now()),
		DeathWish:              flags[abyssRunFlagDeathWish] == 1,
		AnchorRune:             flags[abyssRunFlagAnchorRune] == 1,
		PerfectRun:             flags[abyssRunFlagPerfect] == 1,
		BankConfirm:            !b.abyssBankConfirmDisabled(uid),
		EchoSeed:               b.peekAbyssEchoSeed(uid),
		FranticFeePct:          abyssFranticFeePct,
		LowHP:                  run.Active && maxHP > 0 && run.CurHP*100 < maxHP*abyssFranticHPThresholdPct,
		InsuranceMin:           10,
		InsuranceMax:           90,
		InsuranceStep:          5,
		Insured:                run.Insured,
		FreeInsuranceReady:     freeInsuranceReady,
		BankStreak:             stats.Streak,
		BanksToFreeInsurance:   abyssBanksUntilFreeInsurance(stats.Streak, freeInsuranceReady),
		OvercapGold:            overcap.Gold,
		OvercapTokens:          overcap.Tokens,
		RestShrinkCache:        cache,
		RestShrinkTokens:       tokens,
		RafflePot:              b.abyssRafflePot(),
		Hybrid:                 flags[abyssRunFlagHybrid] == 1,
		TokenAnte:              int(flags[abyssRunFlagTokenAnte]),
		RiskDialPct:            int(flags[abyssRunFlagRiskDialPct]),
		ColdMusclesFloors:      flags[abyssRunFlagColdMuscles],
		DefensiveMomentum:      flags[abyssRunFlagDefensiveMomentum],
	}
	status.ReviveStreak = b.abyssReviveStreak(uid)
	status.ReviveChancePct = abyssReviveOfferChancePct(status.ReviveStreak, stats.UpMercy)
	status.GreedyInterestPct = status.GreedyGripStacks * 2
	status.GreedyDEFPct = status.GreedyGripStacks * 2
	status.RestShrinkReady = run.Active && !run.Downed && run.FloorType == "rest" && tokens > 0 && flags["cache_shrink_depth"] != int64(run.Depth)
	if run.Active && run.Downed && !run.LastActionAt.IsZero() {
		status.DownedTimeoutSeconds = max(int(time.Until(run.LastActionAt.Add(abyssDownedTimeout)).Seconds()), 0)
	}
	return status
}

func abyssReviveOfferChancePct(streak, mercy int) int {
	return min(100, 45+8*max(mercy, 0)+5*min(max(streak, 0), 5))
}

func abyssColdMusclesOnEntry(startDepth int) int64 {
	if startDepth > 0 {
		return 2
	}
	return 0
}

func abyssRecordPushReward(reward int64, depth, bestDepth int) int64 {
	if depth > bestDepth {
		return reward * 103 / 100
	}
	return reward
}

func abyssNextDefensiveMomentum(stacks int64, untouched bool) int64 {
	if !untouched {
		return 0
	}
	return min(max(stacks, 0)+1, int64(10))
}

func abyssHybridDangerMultiplier(tier abyssTier) float64 {
	next, ok := abyssNextTier(tier.Key)
	if !ok || tier.DiffMult <= 0 {
		return 1
	}
	return next.DiffMult / tier.DiffMult
}

func abyssHybridRewardMultiplier(tier abyssTier) float64 {
	next, ok := abyssNextTier(tier.Key)
	if !ok || tier.RewardMult <= 0 {
		return 1
	}
	return (tier.RewardMult + next.RewardMult) / 2 / tier.RewardMult
}

func abyssHybridSurge(active bool, depth int) bool {
	return active && depth > 0 && depth%5 == 0
}

func abyssFatigueDamage(maxHP, round int) int {
	if maxHP <= 0 || round <= 30 {
		return 0
	}
	return max(1, maxHP*(round-30)/100)
}

func clearAbyssPerfectRunInTx(tx *sql.Tx, uid string) error {
	flags, err := loadAbyssRunFlagsInTx(tx, uid)
	if err != nil {
		return err
	}
	flags[abyssRunFlagPerfect] = 0
	return saveAbyssRunFlagsInTx(tx, uid, flags)
}

func abyssFranticBankFee(cache int64, currentHP, maxHP int) int64 {
	if cache <= 0 || maxHP <= 0 || currentHP*100 >= maxHP*abyssFranticHPThresholdPct {
		return 0
	}
	return cache * abyssFranticFeePct / 100
}

func abyssInsurancePercentValid(percent int) bool {
	return percent >= 10 && percent <= 90 && percent%5 == 0
}

func abyssCheapskateEligible(cost, escrow int64) bool {
	return cost >= 0 && escrow > 0 && cost <= (escrow-1)/20
}

func abyssRestCacheConversion(escrow int64) (removed int64, tokens int64) {
	if escrow <= 0 {
		return 0, 0
	}
	removed = escrow / 2
	// The token-shop baseline is 100k gold per token; conversion pays 70% of
	// that value, making the sanctuary route deliberately less efficient.
	tokens = removed * 70 / 100 / abyssTokenBuyGold
	return removed, tokens
}

func abyssDeathWishFloorReward(reward int64, active bool) int64 {
	if active {
		return reward * 2
	}
	return reward
}

func abyssAnchorRefund(refund, escrow int64, active bool) int64 {
	if active && refund < escrow/2 {
		return escrow / 2
	}
	return refund
}

func abyssEchoBankSeed(payout int64, doubled bool) int64 {
	seed := max(payout, int64(0)) * 5 / 100
	if doubled {
		seed *= 2
	}
	return seed
}

func abyssLastStandOffer(run abyssRun, flags map[string]int64) (cost int64, available bool) {
	base := abyssLastStandCost(run.Depth)
	if !run.LastStandUsed {
		return base, true
	}
	if flags[abyssRunFlagSecondLastStandUsed] == 0 {
		return base * 3, true
	}
	return base * 3, false
}

func saveAbyssEchoSeed(tx *sql.Tx, uid string, amount int64) error {
	_, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssEchoSeedKey(uid), itoa64(amount))
	return err
}

func recordAbyssRaffleEntry(tx *sql.Tx, uid string, fee int64, now time.Time) error {
	day := abyssRaffleDay(now)
	if fee > 0 {
		if _, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
			ON CONFLICT (key) DO UPDATE SET value=(COALESCE(NULLIF(app_meta.value, '')::bigint, 0) + $3)::text`,
			"abyss_raffle_pot_"+day, itoa64(fee), fee); err != nil {
			return err
		}
	}
	_, err := tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, '1') ON CONFLICT (key) DO NOTHING`,
		"abyss_raffle_entry_"+day+"_"+uid)
	return err
}

func itoa64(value int64) string {
	return strconv.FormatInt(value, 10)
}
