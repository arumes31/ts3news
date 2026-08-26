package bot

import "testing"

func TestStartAbyssEventChainSchedulesBoundedHunt(t *testing.T) {
	t.Parallel()

	flags := map[string]int64{}
	view, started := startAbyssEventChain(flags, 20)
	if !started || !view.Active {
		t.Fatalf("start = %+v, %v", view, started)
	}
	if view.Sigils != 0 || view.Deadline != 30 || view.FloorsLeft != 10 || view.NextDepth != 22 {
		t.Fatalf("scheduled hunt = %+v", view)
	}

	again, started := startAbyssEventChain(flags, 21)
	if started || again.Deadline != 30 || again.NextDepth != 22 {
		t.Fatalf("active hunt was replaced: %+v, %v", again, started)
	}
}

func TestAdvanceAbyssEventChainCollectsThreeScheduledSigils(t *testing.T) {
	t.Parallel()

	flags := map[string]int64{}
	startAbyssEventChain(flags, 20)
	tests := []struct {
		name      string
		depth     int
		sigils    int
		collected bool
		completed bool
	}{
		{name: "before first mark", depth: 21, sigils: 0},
		{name: "first mark", depth: 22, sigils: 1, collected: true},
		{name: "between marks", depth: 24, sigils: 1},
		{name: "second mark", depth: 25, sigils: 2, collected: true},
		{name: "third mark", depth: 28, sigils: 3, collected: true, completed: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			view := advanceAbyssEventChain(flags, test.depth, 100)
			if view.Sigils != test.sigils || view.Collected != test.collected || view.Completed != test.completed {
				t.Fatalf("advance at depth %d = %+v", test.depth, view)
			}
			if test.completed && view.ChestReward <= 0 {
				t.Fatal("completed chain did not award its chest")
			}
		})
	}
	if flags[abyssRunFlagEventChainActive] != 0 || flags[abyssRunFlagEventChains] != 1 {
		t.Fatalf("completed flags = %#v", flags)
	}
}

func TestAdvanceAbyssEventChainExpiresAfterTenFloors(t *testing.T) {
	t.Parallel()

	flags := map[string]int64{}
	startAbyssEventChain(flags, 7)
	atDeadline := advanceAbyssEventChain(flags, 17, 100)
	if !atDeadline.Active || atDeadline.Expired {
		t.Fatalf("deadline floor should remain eligible: %+v", atDeadline)
	}
	afterDeadline := advanceAbyssEventChain(flags, 18, 100)
	if afterDeadline.Active || !afterDeadline.Expired {
		t.Fatalf("overdue hunt did not expire: %+v", afterDeadline)
	}
	if flags[abyssRunFlagEventChainDeadline] != 0 || flags[abyssRunFlagEventSigils] != 0 {
		t.Fatalf("expired flags = %#v", flags)
	}
}

func TestApplyAbyssEventChainVictoryAddsOnlyCompletedChest(t *testing.T) {
	t.Parallel()

	flags := map[string]int64{}
	startAbyssEventChain(flags, 10)
	escrow, first := applyAbyssEventChainVictory(flags, 12, 100, 5_000)
	if escrow != 5_000 || !first.Collected || first.Completed {
		t.Fatalf("first sigil = escrow %d, view %+v", escrow, first)
	}
	escrow, _ = applyAbyssEventChainVictory(flags, 15, 100, escrow)
	escrow, completed := applyAbyssEventChainVictory(flags, 18, 100, escrow)
	if !completed.Completed || completed.ChestReward <= 0 || escrow != 5_000+completed.ChestReward {
		t.Fatalf("completion = escrow %d, view %+v", escrow, completed)
	}
}

func TestAbyssEventChainChestRewardIsDepthScaledAndBounded(t *testing.T) {
	t.Parallel()

	if got := abyssEventChainChestReward(1, 1); got != 1500 {
		t.Fatalf("minimum chest = %d, want 1500", got)
	}
	shallow := abyssEventChainChestReward(20, 100)
	deep := abyssEventChainChestReward(80, 100)
	if deep <= shallow {
		t.Fatalf("depth scaling = shallow %d, deep %d", shallow, deep)
	}
}
