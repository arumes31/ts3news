package bot

import "testing"

func TestAbyssRunIdentityViewRequiresBiomeAtRest(t *testing.T) {
	run := abyssRun{Active: true, Depth: 6, FloorType: "rest"}
	flags := map[string]int64{
		abyssRunFlagStoryCampaign: 1,
		abyssRunRelicFlag(1):      1,
		abyssRunBoonFlag(2):       2,
	}
	view := abyssRunIdentityViewFrom(run, flags)
	if !view.Active || !view.Story || view.StoryProgress != 6 || len(view.StoryBeats) != 10 {
		t.Fatalf("story view = %#v", view)
	}
	if !view.BiomeChoiceRequired || len(view.BiomeChoices) != 4 {
		t.Fatalf("biome choice view = %#v", view)
	}
	if len(view.Relics) != 1 || len(view.Boons) != 1 || view.Boons[0].Stacks != 2 {
		t.Fatalf("run powers = relics %#v boons %#v", view.Relics, view.Boons)
	}

	flags[abyssRunFlagBiomeChoice] = 2
	flags[abyssRunFlagBiomeUntil] = 11
	flags[abyssRunFlagBiomeSelectedAt] = 6
	view = abyssRunIdentityViewFrom(run, flags)
	if view.BiomeChoiceRequired || view.Biome == nil || view.Biome.Key != "verdant" || view.BiomeUntil != 11 {
		t.Fatalf("bound biome view = %#v", view)
	}
}
