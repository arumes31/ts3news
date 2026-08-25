package bot

import (
	"math"

	"ts3news/internal/content"
)

type abyssHUDPageState struct {
	EscrowPerFloor      int64
	EscrowSoftCap       int64
	EscrowEfficiencyPct int
	FloorsCleared       int
	InterestRatePct     float64
	InterestTotalPct    float64
	Jackpot             int64
	Pacts               []abyssPact
	IntelConcealed      bool
	Contract            *abyssContractPactView
}

func abyssRunFloorsCleared(run abyssRun) int {
	floors := max(0, run.Depth-run.CheckpointStart)
	if run.Active && (run.Downed || run.FloorType != "combat") && floors > 0 {
		floors--
	}
	return floors
}

func (b *Bot) abyssHUDPageState(uid string, run abyssRun, st abyssStats, equipped map[content.GearSlot]content.Gear) abyssHUDPageState {
	floors := abyssRunFloorsCleared(run)
	hasLuckyCoin := false
	if trinket, ok := equipped[content.SlotTrinket1]; ok && trinket.ID == "ABYSS_LUCKY_COIN" {
		hasLuckyCoin = true
	}
	rate := abyssGreedyInterestRate(abyssEffectiveInterest(st.UpInterest, hasLuckyCoin), run.Depth)
	state := abyssHUDPageState{
		FloorsCleared:       floors,
		InterestRatePct:     rate * 100,
		InterestTotalPct:    (math.Pow(1+rate, float64(floors)) - 1) * 100,
		Jackpot:             b.getJackpot("abyss"),
		EscrowSoftCap:       abyssEscrowSoftCap(run.Depth),
		EscrowEfficiencyPct: 100,
	}
	if run.Escrow >= state.EscrowSoftCap {
		state.EscrowEfficiencyPct = 25
	}
	if floors > 0 {
		state.EscrowPerFloor = run.Escrow / int64(floors)
	}
	runPacts := b.abyssRunPacts(uid)
	runFlags := b.loadRunFlags(uid)
	state.Pacts = abyssVisiblePacts(runPacts, runFlags)
	state.IntelConcealed = abyssHasPact(runPacts, "blind")
	state.Contract = abyssContractViewFromFlags(runFlags, run.Depth)
	return state
}

func (b *Bot) abyssEquippedDurability(uid string) map[content.GearSlot]int {
	result := map[content.GearSlot]int{}
	rows, err := b.DB.Query("SELECT slot, durability FROM user_gear WHERE client_uid=$1", uid)
	if err != nil {
		return result
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var slot content.GearSlot
		var durability int
		if rows.Scan(&slot, &durability) == nil {
			result[slot] = durability
		}
	}
	return result
}
