package bot

import (
	"testing"
	"ts3news/internal/content"
)

func TestRiftTerminologyUsesAbyssDefinitions(t *testing.T) {
	names := riftClassNames()
	for _, class := range content.AbyssClasses() {
		if names[class.ID] != class.Name {
			t.Fatal("base class label drift")
		}
		for _, sub := range class.Subclasses {
			build := riftBuildFromUser(UserInCombat{AbyssClass: class.ID, AbyssSubclass: sub.ID, Skills: content.AbyssClassSkills(sub.ID)}, "Test", 20)
			if names[sub.ID] != sub.Name || build.ClassName != sub.Name || build.Resource != sub.Resource || build.Sequence != sub.Sequence {
				t.Fatalf("class terminology drift: %s", sub.ID)
			}
			if len(build.Signatures) != 2 || build.Signatures[0].Name != sub.Builder || build.Signatures[1].Name != sub.Finisher {
				t.Fatalf("ability terminology drift: %s", sub.ID)
			}
		}
	}
	names["vanguard"] = "Changed"
	if riftClassNames()["vanguard"] == "Changed" {
		t.Fatal("catalog aliases caller state")
	}
}
