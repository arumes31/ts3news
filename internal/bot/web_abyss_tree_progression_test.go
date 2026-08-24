package bot

import (
	"testing"

	"ts3news/internal/content"
)

func TestBuildAbyssTreeProgressionPointSources(t *testing.T) {
	progression := buildAbyssTreeProgression(content.AbyssTree(), nil, 100, 50, 2, 11, 75)
	if progression.GrossBase != 472 || progression.TotalPoints != 547 {
		t.Fatalf("gross/total points = %d/%d, want 472/547", progression.GrossBase, progression.TotalPoints)
	}
	if len(progression.PointSources) != 5 {
		t.Fatalf("point source count = %d, want 5", len(progression.PointSources))
	}
	floors := progression.PointSources[2]
	if floors.Earned != 2 || floors.Progress != 3 || floors.Target != 4 || floors.NextReward != 1 {
		t.Fatalf("lifetime floor source = %+v", floors)
	}
}

func TestBuildAbyssTreeProgressionCapsBasePoints(t *testing.T) {
	progression := buildAbyssTreeProgression(content.AbyssTree(), nil, 1000, 1000, 10, 1000, 500)
	if progression.GrossBase <= progression.BasePointCap || progression.TotalPoints != 1500 {
		t.Fatalf("capped progression = gross %d, cap %d, total %d", progression.GrossBase, progression.BasePointCap, progression.TotalPoints)
	}
}

func TestAbyssSectorMasteryAndAchievements(t *testing.T) {
	tree := content.AbyssTree()
	allocated := make([]int, 0, 30)
	for _, node := range tree.Nodes {
		if node.Sector == 0 && len(allocated) < 25 {
			allocated = append(allocated, node.ID)
		}
	}
	sectors := calculateAbyssSectorMastery(tree, allocated)
	if sectors[0].Allocated != 25 || sectors[0].Level != 2 || sectors[0].NextMilestone != 50 || sectors[0].Cosmetic != "Discipline sigil" {
		t.Fatalf("war mastery = %+v", sectors[0])
	}
	achievements := calculateAbyssTreeAchievements(tree, allocated, sectors)
	if achievements[0].Progress != 1 || achievements[1].Progress != 25 || achievements[2].Progress != 1 {
		t.Fatalf("achievement progress = %+v", achievements)
	}
}

func TestCalculateAbyssArchetypesDetectsFocusedBuilds(t *testing.T) {
	tree := content.AbyssTree()
	keys := map[string]bool{
		"skill_damage": false, "str_pct": false, "ult_damage": false,
	}
	allocated := []int{}
	for _, node := range tree.Nodes {
		for key := range node.Pct {
			if _, wanted := keys[key]; wanted && !keys[key] {
				keys[key] = true
				allocated = append(allocated, node.ID)
			}
		}
	}
	scores, dominant := calculateAbyssArchetypes(tree, allocated)
	if dominant != "glass_cannon" {
		t.Fatalf("dominant archetype = %q, want glass_cannon; scores=%+v", dominant, scores)
	}
	glassDetected := false
	for _, score := range scores {
		if score.Key == "glass_cannon" {
			glassDetected = score.Detected
		}
	}
	if !glassDetected {
		t.Fatalf("glass-cannon archetype was not detected: %+v", scores)
	}
	if got := sortedAbyssArchetypeKeys(scores); len(got) != 4 || got[0] != "control" || got[3] != "sustain" {
		t.Fatalf("sorted archetype keys = %v", got)
	}
}
