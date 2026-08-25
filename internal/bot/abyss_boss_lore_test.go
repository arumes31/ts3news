package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAbyssBossLoreUnlocksFromFirstKillTrophies(t *testing.T) {
	views := abyssBossLoreViews([]abyssTrophyView{{Boss: "Malakor the Voidweaver", Date: "2026-08-25"}})
	if len(views) != 4 {
		t.Fatalf("boss lore entries = %d, want 4", len(views))
	}
	seenText := make(map[string]bool, len(views))
	for _, view := range views {
		if view.Title == "" || view.Text == "" || seenText[view.Text] {
			t.Errorf("boss lore is empty or duplicated: %+v", view)
		}
		seenText[view.Text] = true
		if view.Boss == "Malakor the Voidweaver" {
			if !view.Unlocked || view.FirstSlain != "2026-08-25" {
				t.Errorf("Malakor chronicle = %+v", view)
			}
		} else if view.Unlocked || view.FirstSlain != "" {
			t.Errorf("unbeaten boss chronicle unlocked: %+v", view)
		}
	}
}

func TestAbyssBossLoreCodexContract(t *testing.T) {
	root := abyssAAARepositoryRoot(t)
	checks := map[string][]string{
		filepath.Join(root, "internal", "bot", "webassets", "abyss.html"): {
			"Boss Chronicles", "FIRST-KILL ARCHIVE", ".Social.BossLore", "First slain {{.FirstSlain}}",
		},
		filepath.Join(root, "internal", "bot", "webassets", "abyss_boss_lore.css"): {
			".ab-boss-chronicle-grid", ".ab-boss-chronicle.locked", "@media(max-width:760px)",
		},
	}
	for path, required := range checks {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, token := range required {
			if !strings.Contains(string(raw), token) {
				t.Errorf("%s is missing %q", filepath.Base(path), token)
			}
		}
	}
}
