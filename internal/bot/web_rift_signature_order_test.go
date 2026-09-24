package bot

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

func TestRiftSignatureOrderIgnoresLearnedSkillOrder(t *testing.T) {
	for _, class := range content.AbyssClasses() {
		for _, sub := range class.Subclasses {
			t.Run(sub.ID, func(t *testing.T) {
				skills := content.AbyssClassSkills(sub.ID)
				if len(skills) != 2 {
					t.Fatalf("update ordering regression for %d canonical signatures", len(skills))
				}
				for _, reverse := range []bool{false, true} {
					ordered := append([]content.Skill{}, skills...)
					if reverse {
						ordered[0], ordered[1] = ordered[1], ordered[0]
					}
					before, _ := json.Marshal(ordered)
					build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Skills: ordered}, "Test", 20)
					data, err := json.Marshal(build)
					if err != nil {
						t.Fatal(err)
					}
					var saved rift.Build
					if err = json.Unmarshal(data, &saved); err != nil {
						t.Fatal(err)
					}
					if len(saved.Signatures) != 2 || saved.Signatures[0].Role != "builder" || saved.Signatures[0].Name != sub.Builder || saved.Signatures[1].Role != "finisher" || saved.Signatures[1].Name != sub.Finisher {
						t.Fatalf("signature controls reordered: %+v", saved.Signatures)
					}
					after, _ := json.Marshal(ordered)
					if string(before) != string(after) {
						t.Fatal("adapter mutated learned skill order")
					}
				}
			})
		}
	}
}
