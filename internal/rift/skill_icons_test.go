package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
)

func TestCatalogIconsUseCanonicalSkillsForSavedBuilds(t *testing.T) {
	for _, base := range content.AbyssClasses() {
		styles := []string{base.ID}
		for _, sub := range base.Subclasses {
			styles = append(styles, sub.ID)
		}
		for _, style := range styles {
			for _, source := range content.AbyssClassSkills(style) {
				saved := Skill{ID: source.ID, Kind: "obsolete", Name: "Saved name"}
				data, err := json.Marshal(saved)
				if err != nil {
					t.Fatal(err)
				}
				var value struct {
					Icon string `json:"icon"`
					Name string `json:"name"`
				}
				if err = json.Unmarshal(data, &value); err != nil {
					t.Fatal(err)
				}
				if value.Icon != content.SkillIcon(source.Type) || value.Name != "Saved name" {
					t.Fatalf("canonical icon or saved name incorrect: %s", data)
				}
			}
		}
	}
	if (Skill{ID: "unknown"}).CatalogIcon() != "✦" || (Skill{Kind: "ultimate"}).CatalogIcon() != "★" {
		t.Fatal("fallback icons missing")
	}
}
