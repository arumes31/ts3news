package content

import (
	"strings"
	"testing"
)

func TestSkillCatalogMetadata(t *testing.T) {
	t.Parallel()

	if err := ValidateSkillCatalog(); err != nil {
		t.Fatalf("validate production skill catalog: %v", err)
	}
	details := SkillDetailsByID([]string{"S_EQ", "S0_1", "S_EQ", "missing"})
	if len(details) != 2 || details[0].ID >= details[1].ID {
		t.Fatalf("skill details are not unique and deterministic: %+v", details)
	}
	for _, detail := range details {
		if detail.TargetMode == "" || detail.ManaCost < 0 || detail.Element == "" || len(detail.Tags) == 0 {
			t.Fatalf("skill detail is incomplete: %+v", detail)
		}
		if detail.PreviewMax < detail.PreviewMin || detail.Archetype == "" || detail.Role == "" || detail.Source == "" {
			t.Fatalf("skill preview/classification is incomplete: %+v", detail)
		}
	}
}

func TestValidateSkillCatalogRejectsInvalidMetadata(t *testing.T) {
	t.Parallel()

	base := Skill{ID: "test", Name: "Test", Type: SkillPhysical, Rarity: RarityCommon, Power: 1}
	enrichSkillMetadata(&base)
	tests := []struct {
		name   string
		mutate func(*Skill)
		want   string
	}{
		{name: "duplicate id", mutate: func(skill *Skill) {}, want: "duplicate"},
		{name: "probability", mutate: func(skill *Skill) { skill.StunChance = 1.1 }, want: "probability"},
		{name: "negative cost", mutate: func(skill *Skill) { skill.ManaCost = -1 }, want: "cost"},
		{name: "target mismatch", mutate: func(skill *Skill) {
			skill.Power = 0
			skill.HealPercent = .2
			skill.TargetMode = SkillTargetEnemy
		}, want: "incompatible target"},
		{name: "description mismatch", mutate: func(skill *Skill) { skill.Description = "flavor only" }, want: "description"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skill := base
			tt.mutate(&skill)
			catalog := []Skill{skill}
			if tt.name == "duplicate id" {
				catalog = append(catalog, skill)
			}
			err := validateSkillCatalog(catalog)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}
