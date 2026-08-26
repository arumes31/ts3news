package bot

import (
	"database/sql"
	"encoding/json"
	"math/rand/v2"
	"net/http"
)

const (
	abyssRunFlagDoubleBonus      = "double_bonus"
	abyssRunFlagDoubleBonusDepth = "double_bonus_depth"
	abyssRunFlagHardcore         = "hardcore"
	abyssPartialBankFeePct       = 10
	abyssTransportBankFeePct     = 15
	abyssRestFloorGap            = 7
	abyssOverkillDamagePerGold   = 10
	abyssOverkillRewardCapPct    = 25
)

func abyssRestFloorDue(lastRestDepth, nextDepth int) bool {
	return nextDepth > 0 && nextDepth-lastRestDepth >= abyssRestFloorGap
}

func abyssComebackEligible(deathsToday int) bool {
	return deathsToday >= 3
}

func abyssInsuranceLoyaltyPct(lifetimeBanked int64) int {
	if lifetimeBanked <= 0 {
		return 0
	}
	return min(int(lifetimeBanked/1_000_000), 15)
}

func abyssHardcoreRun(flags map[string]int64) bool {
	return flags[abyssRunFlagHardcore] == 1
}

func abyssGraceProtected(depth int, hardcore bool) bool {
	return depth >= 1 && depth <= 3 && !hardcore
}

func abyssHardcoreFloorReward(bonus int64, hardcore bool) int64 {
	if hardcore {
		return bonus * 2
	}
	return bonus
}

type abyssForfeitPolicy struct {
	Refund       int64
	CountDeath   bool
	PreserveLoot bool
}

type abyssEscrowGrowth struct {
	Escrow        int64
	Bonus         int64
	SoftCap       int64
	EfficiencyPct int
}

type abyssEscrowRewardInput struct {
	Escrow         int64
	InterestGain   int64
	FloorBonus     int64
	Depth          int
	OverkillDamage int
}

func abyssEscrowSoftCap(depth int) int64 {
	return 50_000 + int64(max(depth, 1))*10_000
}

func diminishAbyssEscrowGain(current, gain, cap int64) int64 {
	if gain <= 0 {
		return 0
	}
	room := max(cap-current, 0)
	full := min(gain, room)
	return full + (gain-full)/4
}

func applyAbyssEscrowSoftCap(escrow, interestGain, bonus int64, depth int) abyssEscrowGrowth {
	cap := abyssEscrowSoftCap(depth)
	adjustedInterest := diminishAbyssEscrowGain(escrow, interestGain, cap)
	afterInterest := escrow + adjustedInterest
	adjustedBonus := diminishAbyssEscrowGain(afterInterest, bonus, cap)
	rawGain := max(interestGain, 0) + max(bonus, 0)
	adjustedGain := adjustedInterest + adjustedBonus
	efficiency := 100
	if rawGain > 0 {
		efficiency = int(adjustedGain * 100 / rawGain)
	}
	return abyssEscrowGrowth{
		Escrow:        afterInterest + adjustedBonus,
		Bonus:         adjustedBonus,
		SoftCap:       cap,
		EfficiencyPct: efficiency,
	}
}

func abyssOverkillGold(overkillDamage int, floorBonus int64) int64 {
	if overkillDamage <= 0 || floorBonus <= 0 {
		return 0
	}
	converted := (int64(overkillDamage)-1)/abyssOverkillDamagePerGold + 1
	cap := max(floorBonus*abyssOverkillRewardCapPct/100, 1)
	return min(converted, cap)
}

func applyAbyssEscrowReward(input abyssEscrowRewardInput) (abyssEscrowGrowth, int64) {
	base := applyAbyssEscrowSoftCap(input.Escrow, input.InterestGain, input.FloorBonus, input.Depth)
	overkillGold := abyssOverkillGold(input.OverkillDamage, input.FloorBonus)
	if overkillGold == 0 {
		return base, 0
	}
	growth := applyAbyssEscrowSoftCap(
		input.Escrow,
		input.InterestGain,
		input.FloorBonus+overkillGold,
		input.Depth,
	)
	return growth, growth.Escrow - base.Escrow
}

func planAbyssForfeit(escrow int64, insured, depth int, hardcore bool) abyssForfeitPolicy {
	if abyssGraceProtected(depth, hardcore) {
		return abyssForfeitPolicy{Refund: escrow, PreserveLoot: true}
	}
	refund := int64(0)
	if !hardcore && insured > 0 {
		refund = escrow * int64(min(insured, 100)) / 100
	}
	return abyssForfeitPolicy{Refund: refund, CountDeath: true}
}

type abyssPartialBankQuote struct {
	Escrow    int64
	Gross     int64
	Fee       int64
	Payout    int64
	Remaining int64
}

type abyssBankModeQuote struct {
	Escrow       int64
	Remaining    int64
	PartialFee   int64
	TransportFee int64
}

func quoteAbyssPartialBank(escrow int64, multiplier float64, percent int) (abyssPartialBankQuote, bool) {
	if escrow <= 0 || (percent != 25 && percent != 50) {
		return abyssPartialBankQuote{}, false
	}
	share := escrow * int64(percent) / 100
	if share <= 0 {
		return abyssPartialBankQuote{}, false
	}
	gross := int64(float64(share) * multiplier)
	fee := gross * abyssPartialBankFeePct / 100
	return abyssPartialBankQuote{
		Escrow:    share,
		Gross:     gross,
		Fee:       fee,
		Payout:    gross - fee,
		Remaining: escrow - share,
	}, true
}

func quoteAbyssTransportBank(escrow int64, multiplier float64) (abyssPartialBankQuote, bool) {
	if escrow <= 0 {
		return abyssPartialBankQuote{}, false
	}
	gross := int64(float64(escrow) * multiplier)
	fee := gross * abyssTransportBankFeePct / 100
	return abyssPartialBankQuote{Escrow: escrow, Gross: gross, Fee: fee, Payout: gross - fee}, true
}

func quoteAbyssBankMode(escrow int64, multiplier float64, percent int, transport bool) (abyssBankModeQuote, bool) {
	if percent != 0 {
		if transport {
			return abyssBankModeQuote{}, false
		}
		quote, ok := quoteAbyssPartialBank(escrow, multiplier, percent)
		return abyssBankModeQuote{Escrow: quote.Escrow, Remaining: quote.Remaining, PartialFee: quote.Fee}, ok
	}
	if transport {
		quote, ok := quoteAbyssTransportBank(escrow, multiplier)
		return abyssBankModeQuote{Escrow: quote.Escrow, TransportFee: quote.Fee}, ok
	}
	return abyssBankModeQuote{Escrow: escrow}, true
}

func resolveAbyssDoubleBonus(escrow, bonus int64, won bool) int64 {
	if bonus <= 0 {
		return escrow
	}
	if won {
		return escrow + bonus
	}
	return max(escrow-bonus, 0)
}

func (b *Bot) setPendingAbyssDoubleBonus(uid string, depth int, bonus int64) error {
	flags := b.loadRunFlags(uid)
	flags[abyssRunFlagDoubleBonus] = max(bonus, 0)
	flags[abyssRunFlagDoubleBonusDepth] = int64(depth)
	return b.saveRunFlags(uid, flags)
}

func pendingAbyssDoubleBonus(flags map[string]int64, depth int) int64 {
	if flags[abyssRunFlagDoubleBonusDepth] != int64(depth) {
		return 0
	}
	return max(flags[abyssRunFlagDoubleBonus], 0)
}

func consumePendingAbyssDoubleBonus(tx *sql.Tx, uid string, flags map[string]int64) error {
	flags[abyssRunFlagDoubleBonus] = 0
	flags[abyssRunFlagDoubleBonusDepth] = 0
	encoded, err := json.Marshal(flags)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, abyssRunFlagsKey(uid), string(encoded))
	return err
}

// handleAbyssDoubleBonus resolves the optional one-shot coin flip attached to
// the most recently cleared floor. The escrow mutation and consumed offer are
// committed together, preventing refreshes or repeated requests from rerolling.
func (s *WebServer) handleAbyssDoubleBonus(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	unlock := s.lockAbyss(uid)
	defer unlock()
	if s.rejectDuringLiveCombat(w, uid) {
		return
	}

	run := s.bot.loadAbyssRun(uid)
	if !run.Active || run.Downed {
		writeJSON(w, map[string]any{"ok": false, "error": "no live run"})
		return
	}
	flags := s.bot.loadRunFlags(uid)
	bonus := pendingAbyssDoubleBonus(flags, run.Depth)
	if bonus <= 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "no floor bonus gamble is available"})
		return
	}

	won := rand.IntN(2) == 0 // #nosec G404 -- intentional gameplay coin flip
	newEscrow := resolveAbyssDoubleBonus(run.Escrow, bonus, won)
	tx, err := s.bot.DB.Begin()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec("UPDATE abyss_active SET escrow=$1, last_action_at=NOW() WHERE client_uid=$2", newEscrow, uid); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := consumePendingAbyssDoubleBonus(tx, uid, flags); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "db"})
		return
	}
	writeJSON(w, map[string]any{
		"ok": true, "won": won, "bonus": bonus, "escrow": newEscrow, "depth": run.Depth,
	})
}
