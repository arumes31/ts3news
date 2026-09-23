package rift

import "encoding/json"

// Projectile target extents are deliberately smaller for the player.
const (
	projectilePlayerRadiusX = 25.0
	projectilePlayerRadiusY = 23.0
	projectileEnemyRadiusX  = 35.0
	projectileEnemyRadiusY  = 30.0
)

// SkillReference exposes the targeting rules used by the action simulation.
type SkillReference struct {
	Target     string  `json:"target"`
	Horizontal float64 `json:"horizontal"`
	Depth      float64 `json:"depth"`
	Healing    float64 `json:"healing"`
	Barrier    bool    `json:"barrier"`
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
	return r
}

// Derive reference data when emitting a snapshot so older saves gain current
// descriptions without retaining stale copies of simulation constants.
func (s Skill) MarshalJSON() ([]byte, error) {
	type plain Skill
	return json.Marshal(struct {
		plain
		Reference SkillReference `json:"reference"`
	}{plain(s), s.Reference()})
}
