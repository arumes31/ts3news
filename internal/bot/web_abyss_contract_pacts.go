package bot

import "strings"

const (
	abyssRunFlagContract       = "contract_pact"
	abyssRunFlagContractFailed = "contract_failed"
	abyssRunFlagContractTarget = "contract_target"
	abyssContractRewardMult    = 1.20
	abyssContractForfeitPct    = 25
)

type abyssContractPact struct {
	Key   string
	Label string
	Desc  string
}

type abyssContractPactView struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Desc        string `json:"desc"`
	Failed      bool   `json:"failed"`
	TargetDepth int    `json:"target_depth,omitempty"`
}

var abyssContractPacts = []abyssContractPact{
	{Key: "flawless", Label: "Flawless Clause", Desc: "+20% floor cache while unbroken; taking damage forfeits 25% at bank."},
	{Key: "checkpoint", Label: "Checkpoint Clause", Desc: "+20% floor cache; banking before the next checkpoint forfeits 25%."},
}

func normalizeAbyssContractPact(key string) (string, int64) {
	key = strings.ToLower(strings.TrimSpace(key))
	for index, contract := range abyssContractPacts {
		if contract.Key == key {
			return key, int64(index + 1)
		}
	}
	return "", 0
}

func seedAbyssContractPact(flags map[string]int64, key string, startDepth int) {
	_, id := normalizeAbyssContractPact(key)
	if id == 0 {
		return
	}
	flags[abyssRunFlagContract] = id
	if id == 2 {
		flags[abyssRunFlagContractTarget] = int64((max(0, startDepth)/10 + 1) * 10)
	}
}

func abyssContractPactFromFlags(flags map[string]int64) (abyssContractPact, bool) {
	index := int(flags[abyssRunFlagContract]) - 1
	if index < 0 || index >= len(abyssContractPacts) {
		return abyssContractPact{}, false
	}
	return abyssContractPacts[index], true
}

func applyAbyssContractFloor(flags map[string]int64, untouched bool) float64 {
	contract, ok := abyssContractPactFromFlags(flags)
	if !ok {
		return 1
	}
	if contract.Key == "flawless" && !untouched {
		flags[abyssRunFlagContractFailed] = 1
	}
	if flags[abyssRunFlagContractFailed] == 1 {
		return 1
	}
	return abyssContractRewardMult
}

func failAbyssContractOnDefeat(flags map[string]int64) {
	if contract, ok := abyssContractPactFromFlags(flags); ok && contract.Key == "flawless" {
		flags[abyssRunFlagContractFailed] = 1
	}
}

func abyssContractNonCombatRewardMult(flags map[string]int64) float64 {
	if _, ok := abyssContractPactFromFlags(flags); !ok || flags[abyssRunFlagContractFailed] == 1 {
		return 1
	}
	return abyssContractRewardMult
}

func abyssContractViewFromFlags(flags map[string]int64, depth int) *abyssContractPactView {
	contract, ok := abyssContractPactFromFlags(flags)
	if !ok {
		return nil
	}
	target := int(flags[abyssRunFlagContractTarget])
	return &abyssContractPactView{
		Key: contract.Key, Label: contract.Label, Desc: contract.Desc,
		Failed: flags[abyssRunFlagContractFailed] == 1, TargetDepth: target,
	}
}

func abyssContractForfeit(payout int64, flags map[string]int64, depth int, partial bool) int64 {
	if partial || payout <= 0 {
		return 0
	}
	view := abyssContractViewFromFlags(flags, depth)
	if view == nil {
		return 0
	}
	failed := view.Failed || view.Key == "checkpoint" && depth < view.TargetDepth
	if !failed {
		return 0
	}
	return abyssGoldPercent(payout, abyssContractForfeitPct)
}
