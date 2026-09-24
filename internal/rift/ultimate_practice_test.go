package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUltimatePracticeRequiresEquippedUltimateInTimingWindow(t *testing.T) {
	for _, kind := range []string{"ultimate", "heal", "shield"} {
		t.Run(kind, func(t *testing.T) {
			build := testRun().Build
			build.Ultimate = &Skill{ID: "equipped-ult", Kind: kind, Power: 1, Cost: 10, Cooldown: 20}
			r, err := NewPracticeRun("timing", build, "ultimate", time.Unix(100, 0))
			if err != nil {
				t.Fatal(err)
			}
			r.Clock = 1
			r.cast("equipped-ult")
			r.practiceTick()
			if r.Practice.Completed {
				t.Fatal("early cast completed drill")
			}
			r.Clock = 3
			r.Player.Cooldown = 0
			r.cast("equipped-ult")
			r.practiceTick()
			if r.Practice.Completed {
				t.Fatal("blocked cooldown completed drill")
			}
			r.skillHit(0, 10, Skill{ID: "ordinary", Kind: "ultimate"}, 0, "")
			r.practiceTick()
			if r.Practice.Completed {
				t.Fatal("unequipped ability completed drill")
			}
			r.SkillTimers["equipped-ult"] = 0
			r.Player.Cooldown = 0
			r.cast("equipped-ult")
			r.practiceTick()
			if !r.Practice.Completed || r.Gold != 0 || len(r.Drops) != 0 || len(r.History) != 0 {
				t.Fatal("timed ultimate failed or granted campaign rewards")
			}
			raw, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err = json.Unmarshal(raw, &saved); err != nil {
				t.Fatal(err)
			}
			if !saved.Practice.UltimateTimed {
				t.Fatal("timing progress lost")
			}
			if err = saved.ResetPractice(time.Unix(200, 0)); err != nil {
				t.Fatal(err)
			}
			if saved.Practice.UltimateTimed || saved.Practice.Completed || saved.Build.Ultimate.ID != "equipped-ult" {
				t.Fatal("reset lost build or retained completion")
			}
		})
	}
	if _, err := NewPracticeRun("missing", testRun().Build, "ultimate", time.Now()); err == nil {
		t.Fatal("missing ultimate accepted")
	}
}

func TestUltimatePracticeTimingWindowBoundaries(t *testing.T) {
	r := testRun()
	r.Practice = &PracticeState{Mode: "ultimate"}
	for _, tc := range []struct {
		clock float64
		open  bool
	}{{0, false}, {2.999, false}, {3, true}, {4.999, true}, {5, false}, {8.999, false}, {9, true}} {
		r.Clock = tc.clock
		if r.ultimatePracticeWindow() != tc.open {
			t.Fatalf("incorrect window at %v", tc.clock)
		}
	}
}

func TestUltimatePracticeProjectileUsesImpactTimeAndAim(t *testing.T) {
	for _, tc := range []struct {
		name     string
		clock, y float64
		want     bool
	}{{"open", 3, 410, true}, {"late", 4.99, 410, false}, {"wrong lane", 3, 300, false}} {
		t.Run(tc.name, func(t *testing.T) {
			build := testRun().Build
			build.Ultimate = &Skill{ID: "shot-ult", Kind: "fire", Power: 1}
			r, err := NewPracticeRun("impact", build, "ultimate", time.Unix(100, 0))
			if err != nil {
				t.Fatal(err)
			}
			r.Clock = tc.clock
			r.Projectiles = []Projectile{{X: 220, Y: tc.y, Life: 1, Power: 10, Skill: *build.Ultimate, Kind: "fire"}}
			r.tick(Input{}, .02)
			if r.Practice.Completed != tc.want {
				t.Fatal("completion ignored impact timing or aim")
			}
		})
	}
}
