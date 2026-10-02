package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRangedPracticeRequiresRealAimedProjectiles(t *testing.T) {
	build := testRun().Build
	build.Skills = []Skill{{ID: "shot", Kind: "fire", Power: 1}}
	r, err := NewPracticeRun("ranged", build, "ranged", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	tick := 0
	shoot := func() {
		r.Player.Cooldown = 0
		r.SkillTimers["shot"] = 0
		for i := 0; i < 60; i++ {
			tick++
			in := Input{}
			if i == 0 {
				in.Skill = "shot"
			}
			r.Step(in, time.UnixMilli(100000+int64(tick)*50))
		}
	}
	r.Player.Facing = -1
	shoot()
	if r.Practice.RangedHits != 0 {
		t.Fatal("rearward shot hit")
	}
	r.Player.Facing = 1
	r.Player.Y -= 80
	shoot()
	if r.Practice.RangedHits != 0 {
		t.Fatal("wrong lane hit")
	}
	r.Player.Y = r.Enemies[0].Y
	r.skillHit(0, 10, Skill{Kind: "slash"}, 0, "")
	r.practiceTick()
	if r.Practice.RangedHits != 0 || r.Practice.Completed {
		t.Fatal("melee completed ranged drill")
	}
	for i := 0; i < 3; i++ {
		shoot()
	}
	if !r.Practice.Completed || r.Practice.RangedHits != 3 || len(r.Drops) != 0 || len(r.History) != 0 || r.Gold != 0 {
		t.Fatalf("invalid completion: %+v", r.Practice)
	}
	raw, _ := json.Marshal(r)
	var saved Run
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Practice.RangedHits != 3 {
		t.Fatal("progress lost")
	}
	if err := saved.ResetPractice(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if saved.Practice.RangedHits != 0 {
		t.Fatal("reset retained hits")
	}
	if _, err := NewPracticeRun("invalid", Build{HP: 100, Skills: []Skill{{Kind: "slash"}}}, "ranged", time.Now()); err == nil {
		t.Fatal("missing projectile accepted")
	}
}
