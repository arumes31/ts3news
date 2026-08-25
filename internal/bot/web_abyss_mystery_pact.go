package bot

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"strings"
)

const (
	abyssMysteryPactKey     = "mystery"
	abyssRunFlagMysteryPact = "mystery_pact"
	abyssMysteryPactReward  = 0.40
	abyssMysteryPactLabel   = "Mystery Pact"
)

type abyssMysteryPactReveal struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Desc  string `json:"desc"`
}

func canonicalAbyssPactRequest(requested []string) []string {
	canonical := strings.Fields(abyssValidatePacts(requested))
	for _, key := range requested {
		if key == abyssMysteryPactKey {
			return append(canonical, abyssMysteryPactKey)
		}
	}
	return canonical
}

func splitAbyssPactRequest(requested []string) ([]string, bool) {
	canonical := canonicalAbyssPactRequest(requested)
	if len(canonical) > 0 && canonical[len(canonical)-1] == abyssMysteryPactKey {
		return canonical[:len(canonical)-1], true
	}
	return canonical, false
}

// resolveAbyssMysteryPact uses a cryptographic draw so the hidden choice cannot
// be predicted from a run seed. Explicitly selected pacts are excluded.
func resolveAbyssMysteryPact(source io.Reader, selected []string) (abyssPact, int, error) {
	excluded := make(map[string]bool, len(selected))
	for _, key := range selected {
		excluded[key] = true
	}
	eligible := make([]int, 0, len(abyssPactCatalog))
	for index, pact := range abyssPactCatalog {
		if !excluded[pact.Key] {
			eligible = append(eligible, index)
		}
	}
	if len(eligible) == 0 {
		return abyssPact{}, 0, fmt.Errorf("no mystery pact is available")
	}
	draw, err := rand.Int(source, big.NewInt(int64(len(eligible))))
	if err != nil {
		return abyssPact{}, 0, fmt.Errorf("resolve mystery pact: %w", err)
	}
	index := eligible[draw.Int64()]
	return abyssPactCatalog[index], index + 1, nil
}

func abyssMysteryPactFromFlags(flags map[string]int64) (abyssPact, bool) {
	index := int(flags[abyssRunFlagMysteryPact]) - 1
	if index < 0 || index >= len(abyssPactCatalog) {
		return abyssPact{}, false
	}
	return abyssPactCatalog[index], true
}

func abyssMysteryPactView() abyssPact {
	return abyssPact{
		Key: abyssMysteryPactKey, Label: abyssMysteryPactLabel,
		Desc:   "A server-selected pact is active and will be revealed when the run ends.",
		Reward: abyssMysteryPactReward, Danger: 1,
	}
}

func abyssVisiblePacts(pacts []string, flags map[string]int64) []abyssPact {
	hidden, mystery := abyssMysteryPactFromFlags(flags)
	visible := make([]abyssPact, 0, len(pacts)+1)
	for _, key := range pacts {
		if mystery && key == hidden.Key {
			continue
		}
		if pact, ok := abyssPactByKey(key); ok {
			visible = append(visible, pact)
		}
	}
	if mystery {
		visible = append(visible, abyssMysteryPactView())
	}
	return visible
}

func abyssMysteryRevealFromFlags(flags map[string]int64) *abyssMysteryPactReveal {
	pact, ok := abyssMysteryPactFromFlags(flags)
	if !ok {
		return nil
	}
	return &abyssMysteryPactReveal{Key: pact.Key, Label: pact.Label, Desc: pact.Desc}
}

// abyssPactTokenRiskPct is immutable for the lifetime of a run. The hidden
// pact's ordinary reward is deliberately omitted so token quotes cannot reveal
// its identity; the advertised Mystery Pact premium remains eligible.
func abyssPactTokenRiskPct(pacts []string, flags map[string]int64) float64 {
	hidden, mystery := abyssMysteryPactFromFlags(flags)
	bonus := 0.0
	for _, key := range pacts {
		if mystery && key == hidden.Key {
			continue
		}
		if pact, ok := abyssPactByKey(key); ok {
			bonus += pact.Reward * 100
		}
	}
	if mystery {
		bonus += abyssMysteryPactReward * 100
	}
	return pctNumber1(bonus)
}

func redactAbyssMysteryPactBreakdown(breakdown abyssPactRewardBreakdown, flags map[string]int64) abyssPactRewardBreakdown {
	hidden, ok := abyssMysteryPactFromFlags(flags)
	if !ok {
		return breakdown
	}
	lines := breakdown.Lines[:0]
	for _, line := range breakdown.Lines {
		if line.Key != hidden.Key {
			lines = append(lines, line)
		}
	}
	breakdown.Lines = append(lines, abyssPactRewardLine{
		Key: abyssMysteryPactKey, Label: abyssMysteryPactLabel,
		BaseBonusPct:  abyssMysteryPactReward * 100,
		TotalBonusPct: abyssMysteryPactReward * 100,
	})
	synergies := breakdown.Synergies[:0]
	for _, synergy := range breakdown.Synergies {
		if synergy.PactKey != hidden.Key {
			synergies = append(synergies, synergy)
		}
	}
	breakdown.Synergies = synergies
	return breakdown
}
