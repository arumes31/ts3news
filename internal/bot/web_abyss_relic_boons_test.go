package bot

import "testing"

func TestAdvanceAbyssRunIdentityAwardsUniqueRelicsAndDrafts(t *testing.T) {
	flags := map[string]int64{}
	first := advanceAbyssRunIdentity(flags, 4)
	if first.Relic == nil || first.Relic.Key != "ember_lens" {
		t.Fatalf("floor 4 relic = %#v", first.Relic)
	}
	second := advanceAbyssRunIdentity(flags, 8)
	if second.Relic == nil || second.Relic.Key == first.Relic.Key {
		t.Fatalf("floor 8 relic = %#v after %#v", second.Relic, first.Relic)
	}

	draft := advanceAbyssRunIdentity(flags, 10).Draft
	if !draft.Pending || draft.Depth != 10 || len(draft.Options) != 3 {
		t.Fatalf("floor 10 draft = %#v", draft)
	}
	if !abyssRunChoicePending(flags) {
		t.Fatal("draft did not block descent")
	}
}

func TestApplyAbyssRunIdentityBuildCapsBoonStacks(t *testing.T) {
	flags := map[string]int64{
		abyssRunRelicFlag(1): 1,
		abyssRunBoonFlag(1):  99,
		abyssRunBoonFlag(4):  2,
	}
	stats := abyssRunIdentityStats(flags)
	if stats.STR != 138 {
		t.Fatalf("STR = %d, want 138", stats.STR)
	}
	if stats.HP != 120 {
		t.Fatalf("HP = %d, want 120", stats.HP)
	}
}

func TestAdvanceAbyssStoryMarksFinaleComplete(t *testing.T) {
	flags := map[string]int64{abyssRunFlagStoryCampaign: 1}
	advance := advanceAbyssRunIdentity(flags, 10)
	if !advance.StoryComplete || flags[abyssRunFlagStoryComplete] != 1 {
		t.Fatalf("finale advance = %#v flags = %#v", advance, flags)
	}
}

func TestAbyssBoonDraftSkipsMaxedBoonsAndNeverDeadlocks(t *testing.T) {
	flags := map[string]int64{abyssRunFlagBoonDraftDepth: 15}
	for _, boon := range abyssRunBoons {
		flags[abyssRunBoonFlag(boon.ID)] = 3
	}
	flags[abyssRunBoonFlag(4)] = 2
	draft := abyssBoonDraftFromFlags(flags)
	if !draft.Pending || len(draft.Options) != 1 || draft.Options[0].ID != 4 {
		t.Fatalf("last useful draft = %#v", draft)
	}
	flags[abyssRunBoonFlag(4)] = 3
	advance := advanceAbyssRunIdentity(flags, 20)
	if advance.Draft.Pending || flags[abyssRunFlagBoonDraftDepth] != 0 || abyssRunChoicePending(flags) {
		t.Fatalf("fully capped draft deadlocked: advance=%#v flags=%#v", advance, flags)
	}
}

func TestClearAbyssRunIdentityFlagsPreservesUnrelatedRunState(t *testing.T) {
	flags := map[string]int64{
		abyssRunFlagStoryCampaign: 1,
		abyssRunFlagBiomeChoice:   2,
		abyssRunRelicFlag(1):      1,
		abyssRunBoonFlag(2):       2,
		"unrelated":               7,
	}
	clearAbyssRunIdentityFlags(flags)
	if len(flags) != 1 || flags["unrelated"] != 7 {
		t.Fatalf("cleared flags = %#v", flags)
	}
}
