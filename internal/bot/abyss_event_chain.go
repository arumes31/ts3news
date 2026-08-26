package bot

const (
	abyssEventChainType             = "sigil_chain"
	abyssRunFlagEventChainActive    = "event_chain_active"
	abyssRunFlagEventSigils         = "event_sigils"
	abyssRunFlagEventChainDeadline  = "event_chain_deadline"
	abyssRunFlagEventChainNextDepth = "event_chain_next_depth"
	abyssRunFlagEventChains         = "event_chains"
	abyssEventChainWindow           = 10
	abyssEventChainFirstOffset      = 2
	abyssEventChainInterval         = 3
	abyssEventChainSigilsRequired   = 3
)

type abyssEventChainView struct {
	Active      bool  `json:"active"`
	Sigils      int   `json:"sigils"`
	Required    int   `json:"required"`
	Deadline    int   `json:"deadline,omitempty"`
	FloorsLeft  int   `json:"floors_left"`
	NextDepth   int   `json:"next_depth,omitempty"`
	Chains      int64 `json:"chains"`
	FoundFirst  bool  `json:"-"`
	FoundSecond bool  `json:"-"`
	FoundThird  bool  `json:"-"`

	Collected   bool  `json:"collected,omitempty"`
	Completed   bool  `json:"completed,omitempty"`
	Expired     bool  `json:"expired,omitempty"`
	ChestReward int64 `json:"chest_reward,omitempty"`
}

func abyssEventChainFromFlags(flags map[string]int64, depth int) abyssEventChainView {
	view := abyssEventChainView{
		Required: abyssEventChainSigilsRequired,
		Chains:   max(flags[abyssRunFlagEventChains], 0),
	}
	deadline := int(flags[abyssRunFlagEventChainDeadline])
	if flags[abyssRunFlagEventChainActive] <= 0 || deadline < depth {
		return view
	}
	view.Active = true
	view.Sigils = min(max(int(flags[abyssRunFlagEventSigils]), 0), abyssEventChainSigilsRequired-1)
	view.Deadline = deadline
	view.FloorsLeft = max(deadline-depth, 0)
	view.NextDepth = int(flags[abyssRunFlagEventChainNextDepth])
	view.FoundFirst = view.Sigils >= 1
	view.FoundSecond = view.Sigils >= 2
	view.FoundThird = view.Sigils >= 3
	return view
}

func startAbyssEventChain(flags map[string]int64, depth int) (abyssEventChainView, bool) {
	if current := abyssEventChainFromFlags(flags, depth); current.Active {
		return current, false
	}
	clearAbyssEventChain(flags)
	flags[abyssRunFlagEventChainActive] = 1
	flags[abyssRunFlagEventSigils] = 0
	flags[abyssRunFlagEventChainDeadline] = int64(depth + abyssEventChainWindow)
	flags[abyssRunFlagEventChainNextDepth] = int64(depth + abyssEventChainFirstOffset)
	return abyssEventChainFromFlags(flags, depth), true
}

func advanceAbyssEventChain(flags map[string]int64, depth, level int) abyssEventChainView {
	if flags[abyssRunFlagEventChainActive] <= 0 {
		return abyssEventChainFromFlags(flags, depth)
	}
	deadline := int(flags[abyssRunFlagEventChainDeadline])
	if depth > deadline {
		clearAbyssEventChain(flags)
		view := abyssEventChainFromFlags(flags, depth)
		view.Expired = true
		return view
	}
	view := abyssEventChainFromFlags(flags, depth)
	if !view.Active || depth < view.NextDepth {
		return view
	}

	view.Collected = true
	view.Sigils++
	flags[abyssRunFlagEventSigils] = int64(view.Sigils)
	if view.Sigils < abyssEventChainSigilsRequired {
		nextDepth := max(view.NextDepth+abyssEventChainInterval, depth+1)
		flags[abyssRunFlagEventChainNextDepth] = int64(nextDepth)
		return abyssEventChainFromFlags(flags, depth).withCollected()
	}

	flags[abyssRunFlagEventChains]++
	chains := flags[abyssRunFlagEventChains]
	clearAbyssEventChain(flags)
	return abyssEventChainView{
		Sigils:      abyssEventChainSigilsRequired,
		Required:    abyssEventChainSigilsRequired,
		Chains:      chains,
		Collected:   true,
		Completed:   true,
		ChestReward: abyssEventChainChestReward(depth, level),
		FoundFirst:  true,
		FoundSecond: true,
		FoundThird:  true,
	}
}

func applyAbyssEventChainVictory(
	flags map[string]int64,
	depth int,
	level int,
	escrow int64,
) (int64, abyssEventChainView) {
	view := advanceAbyssEventChain(flags, depth, level)
	return escrow + view.ChestReward, view
}

func (view abyssEventChainView) withCollected() abyssEventChainView {
	view.Collected = true
	return view
}

func (view abyssEventChainView) relevant() bool {
	return view.Active || view.Collected || view.Completed || view.Expired
}

func abyssEventChainChestReward(depth, level int) int64 {
	return max(int64(1500), abyssFloorBonus(max(depth, 1), max(level, 1))*3)
}

func clearAbyssEventChain(flags map[string]int64) {
	delete(flags, abyssRunFlagEventChainActive)
	delete(flags, abyssRunFlagEventSigils)
	delete(flags, abyssRunFlagEventChainDeadline)
	delete(flags, abyssRunFlagEventChainNextDepth)
}
