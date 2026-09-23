package rift

import "testing"

func TestSkillHitCannotModifyDefeatedTarget(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{ID: "dead", Kind: "goblin", HP: 0, MaxHP: 100, X: 300, Y: 400, Windup: .4}}
	r.Marked = "living"
	before := r.Enemies[0]
	events := len(r.Events)
	drops := len(r.Drops)
	r.skillHit(0, 100, Skill{ID: "ice-builder", Kind: "ice", Role: "builder"}, 0, "")
	if r.Enemies[0] != before || r.Marked != "living" || len(r.Events) != events {
		t.Fatal("skill modified defeated target or its mark")
	}
	if r.Stats.DamageDealt != 0 || r.Stats.Kills != 0 || len(r.Drops) != drops {
		t.Fatal("dead target awarded combat credit")
	}
}

func TestAttacksSkipDefeatedTargets(t *testing.T) {
	for _, kind := range []string{"melee", "area", "projectile"} {
		t.Run(kind, func(t *testing.T) {
			r := testRun()
			r.Status = "cleared"
			r.Enemies = []Actor{{ID: "dead", HP: 0, MaxHP: 100, X: r.Player.X + 50, Y: r.Player.Y}, {ID: "alive", HP: 1000, MaxHP: 1000, X: r.Player.X + 50, Y: r.Player.Y}}
			before := r.Enemies[0]
			switch kind {
			case "melee":
				r.tick(Input{Attack: true}, 0)
			case "area":
				r.Build.Skills = []Skill{{ID: "sweep", Kind: "slash", Power: 1}}
				r.cast("sweep")
			case "projectile":
				r.Projectiles = []Projectile{{X: r.Player.X + 50, Y: r.Player.Y, Power: 10, Life: 2, Skill: r.Build.Skills[0]}}
				r.tick(Input{}, 0)
			}
			if r.Enemies[0] != before {
				t.Fatal("attack changed defeated target")
			}
			if r.Enemies[1].HP >= 1000 {
				t.Fatal("dead target blocked attack against living target")
			}
			if r.Stats.Kills != 0 || len(r.Drops) != 0 {
				t.Fatal("attack granted duplicate death rewards")
			}
		})
	}
}
