package bot

import "testing"

func TestAbyssStoryCampaignHasTenAuthoredFloors(t *testing.T) {
	if len(abyssStoryCampaign) != 10 {
		t.Fatalf("story floors = %d, want 10", len(abyssStoryCampaign))
	}
	for depth, floor := range abyssStoryCampaign {
		if floor.Depth != depth+1 || floor.Title == "" || floor.Subtitle == "" || floor.Affinity == "" || floor.Modifier == "" {
			t.Fatalf("incomplete story floor %d: %#v", depth+1, floor)
		}
	}
	if abyssStoryCampaign[4].Title != "The First Warden" || abyssStoryCampaign[9].Title != "Heart of the Descent" {
		t.Fatalf("boss chapters = %q, %q", abyssStoryCampaign[4].Title, abyssStoryCampaign[9].Title)
	}
}

func TestAbyssBiomeForRunHonorsStoryThenContract(t *testing.T) {
	storyFlags := map[string]int64{abyssRunFlagStoryCampaign: 1}
	biome, label := abyssBiomeForRun(1, "void", 0, storyFlags)
	if biome.Affinity != "fire" || label != "The Sealed Gate" {
		t.Fatalf("story biome = %q %q", biome.Affinity, label)
	}

	contractFlags := map[string]int64{abyssRunFlagBiomeChoice: 3, abyssRunFlagBiomeUntil: 8}
	biome, label = abyssBiomeForRun(7, "void", 0, contractFlags)
	if biome.Affinity != "storm" || label != "Tempest Crown" {
		t.Fatalf("contract biome = %q %q", biome.Affinity, label)
	}
	if _, ok := abyssSelectedBiomeContract(contractFlags, 9); ok {
		t.Fatal("expired contract remained active")
	}
}
