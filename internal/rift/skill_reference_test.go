package rift

import (
	"encoding/json"
	"testing"
)

func TestSkillReferenceMatchesAreaHitBoundaries(t *testing.T) {
	for _, kind := range []string{"slash", "quake", "ultimate"} {
		r := testRun()
		s := Skill{ID: "area", Kind: kind, Power: 1}
		r.Build.Skills = []Skill{s}
		ref := s.Reference()
		r.Enemies = []Actor{
			{ID: "inside", HP: 100, MaxHP: 100, X: r.Player.X + ref.Horizontal - 1, Y: r.Player.Y + ref.Depth - 1},
			{ID: "depth-edge", HP: 100, MaxHP: 100, X: r.Player.X, Y: r.Player.Y + ref.Depth},
			{ID: "range-edge", HP: 100, MaxHP: 100, X: r.Player.X + ref.Horizontal, Y: r.Player.Y},
		}
		r.cast(s.ID)
		if r.Enemies[0].HP >= 100 || r.Enemies[1].HP != 100 || r.Enemies[2].HP != 100 {
			t.Fatalf("%s reference differs from combat: %+v", kind, r.Enemies)
		}
	}
}

func TestSkillReferenceSerializationAndRecoveryEffects(t *testing.T) {
	for _, s := range []Skill{{ID: "heal", Kind: "heal"}, {ID: "shield", Kind: "shield"}, {ID: "bolt", Kind: "fire", Heal: .2}} {
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			ID        string         `json:"id"`
			Reference SkillReference `json:"reference"`
		}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.ID != s.ID || decoded.Reference != s.Reference() {
			t.Fatal("serialized reference differs")
		}
	}
	if (Skill{Kind: "heal"}).Reference().Healing != .15 || !(Skill{Kind: "shield"}).Reference().Barrier {
		t.Fatal("recovery effects missing")
	}
	if ref := (Skill{Kind: "fire"}).Reference(); ref.Target != "projectile" || ref.Depth != 30 || ref.Horizontal != 35 {
		t.Fatal("projectile collision reference incorrect")
	}
}
