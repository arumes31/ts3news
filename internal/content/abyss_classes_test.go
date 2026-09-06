package content

import "testing"

func TestAbyssClassesHaveDistinctCompleteBuilds(t *testing.T) {
	classes := AbyssClasses()
	if len(classes) != 6 {
		t.Fatalf("classes = %d", len(classes))
	}
	seen := map[string]bool{}
	for _, class := range classes {
		t.Run(class.ID, func(t *testing.T) {
			if len(class.Subclasses) != 2 || class.ArtKey != "class:"+class.ID {
				t.Fatal("incomplete class", class)
			}
			for _, sub := range class.Subclasses {
				if seen[sub.ID] {
					t.Fatal("duplicate subclass", sub.ID)
				}
				seen[sub.ID] = true
				skills := AbyssClassSkills(sub.ID)
				if len(skills) != 2 || skills[0].ID == skills[1].ID {
					t.Fatal("missing sequence", sub.ID)
				}
				for _, skill := range skills {
					if skill.ManaCost < 5 || skill.ScalingStat == "" || skill.Mechanics == "" {
						t.Fatal("missing skill contract", skill)
					}
				}
			}
		})
	}
	if _, ok := AbyssSubclassByID("unknown"); ok {
		t.Fatal("invalid subclass accepted")
	}
}
