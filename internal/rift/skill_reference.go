package rift

import (
	"encoding/json"
	"strings"
	"ts3news/internal/content"
)

// Projectile target extents are deliberately smaller for the player.
const (
	playerProjectileSpeed    = 530.0
	playerProjectileLifetime = 2.5
	playerProjectileOffset   = 35.0
	projectilePlayerRadiusX  = 25.0
	projectilePlayerRadiusY  = 23.0
	projectileEnemyRadiusX   = 35.0
	projectileEnemyRadiusY   = 30.0
)

// SkillReference exposes the targeting rules used by the action simulation.
type SkillReference struct {
	Travel      float64 `json:"travel,omitempty"`
	SpawnOffset float64 `json:"spawn_offset,omitempty"`
	Target      string  `json:"target"`
	Horizontal  float64 `json:"horizontal"`
	Depth       float64 `json:"depth"`
	Healing     float64 `json:"healing"`
	Barrier     bool    `json:"barrier"`
}

func (s Skill) Reference() SkillReference {
	r := SkillReference{Target: "projectile", Horizontal: projectileEnemyRadiusX, Depth: projectileEnemyRadiusY, Healing: s.Heal}
	switch s.Kind {
	case "shield":
		r.Target = "self"
		r.Horizontal = 0
		r.Depth = 0
		r.Barrier = true
	case "heal":
		r.Target = "self"
		r.Horizontal = 0
		r.Depth = 0
		if s.Heal == 0 {
			r.Healing = .15
		}
	case "slash":
		r.Target = "area"
		r.Horizontal = 155
		r.Depth = 60
	case "quake":
		r.Target = "area"
		r.Horizontal = 260
		r.Depth = 120
	case "ultimate":
		r.Target = "area"
		r.Horizontal = 450
		r.Depth = 180
	}
	if r.Target == "projectile" {
		r.Travel = playerProjectileSpeed * playerProjectileLifetime
		r.SpawnOffset = playerProjectileOffset
	}
	return r
}

// Derive reference data when emitting a snapshot so older saves gain current
// descriptions without retaining stale copies of simulation constants.
func (s Skill) MarshalJSON() ([]byte, error) {
	type plain Skill
	return json.Marshal(struct {
		plain
		Icon      string         `json:"icon"`
		Reference SkillReference `json:"reference"`
	}{plain(s), s.CatalogIcon(), s.Reference()})
}

// CatalogIcon resolves presentation from current content, including old saves.
func (s Skill) CatalogIcon() string {
	if strings.HasPrefix(s.ID, "CLASS_") {
		if end := strings.LastIndex(s.ID, "_"); end > 6 {
			for _, skill := range content.AbyssClassSkills(s.ID[6:end]) {
				if skill.ID == s.ID {
					return content.SkillIcon(skill.Type)
				}
			}
		}
	}
	if skill, ok := content.GetSkillByID(s.ID); ok {
		return content.SkillIcon(skill.Type)
	}
	if _, ok := content.GetUltimateSkillByID(s.ID); ok {
		return content.SkillIcon(content.SkillUltimate)
	}
	if s.Kind == "ultimate" {
		return content.SkillIcon(content.SkillUltimate)
	}
	return content.SkillIcon("")
}
