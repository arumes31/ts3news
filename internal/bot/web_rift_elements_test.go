package bot

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

func TestRiftBuildPreservesCanonicalElementsInsteadOfArtNames(t *testing.T) {
	u := UserInCombat{Equipped: map[content.GearSlot]content.Gear{content.SlotMainHand: {Slot: content.SlotMainHand, Name: "Water blade", Element: content.ElementWater}}, Skills: []content.Skill{{ID: "misleading-art", Name: "Ice flare", Type: content.SkillMagic, Element: content.ElementFire, Power: 1}}}
	build := riftBuildFromUser(u, "Test", 1)
	if build.WeaponElement != content.ElementWater || len(build.Skills) != 1 || build.Skills[0].Element != content.ElementFire || build.Skills[0].Kind != "ice" {
		t.Fatal("canonical element was lost or inferred from presentation")
	}
	raw, err := json.Marshal(build)
	if err != nil {
		t.Fatal(err)
	}
	var saved rift.Build
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.WeaponElement != build.WeaponElement || saved.Skills[0].Element != build.Skills[0].Element {
		t.Fatal("save lost elemental metadata")
	}
	empty := riftBuildFromUser(UserInCombat{}, "Test", 1)
	if empty.WeaponElement != content.ElementPhysical {
		t.Fatal("unarmed build is not neutral")
	}
}

func TestRiftClassSignatureElementsMatchCanonicalSkills(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			skills := content.AbyssClassSkills(sub.ID)
			build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Skills: skills}, "Test", 1)
			for _, converted := range append(build.Signatures, build.Skills...) {
				found := false
				for _, source := range skills {
					if converted.ID == source.ID {
						found = true
						if converted.Element != source.Element {
							t.Fatalf("%s lost element", converted.ID)
						}
					}
				}
				if !found {
					t.Fatal("unknown skill")
				}
			}
		}
	}
}
