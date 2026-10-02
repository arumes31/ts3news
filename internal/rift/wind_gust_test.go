package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func windTestRun(t *testing.T) *Run {
	t.Helper()
	r := circleTestRun()
	r.RoomObjective = nil
	if err := json.Unmarshal([]byte(`{"name":"Gale corridor","wind_gusts":[{"x":400,"y":330,"w":600,"h":150,"period":8,"offset":0,"duration":3,"vx":60}]}`), &r.Level.Rooms[0]); err != nil {
		t.Fatal(err)
	}
	r.Clock = 1.3
	r.Projectiles = []Projectile{{ID: 1, X: 500, Y: 380, VX: 100, Life: 4, Power: 10}, {ID: 2, X: 800, Y: 420, VX: -100, Life: 4, Power: 10, Enemy: true}}
	return r
}

func TestWindAffectsBothTeamsWithoutChangingBaseVelocity(t *testing.T) {
	r := windTestRun(t)
	r.Step(Input{}, time.UnixMilli(r.LastMS+20))
	if len(r.Projectiles) != 2 {
		t.Fatal("shots disappeared")
	}
	a, b := r.Projectiles[0], r.Projectiles[1]
	if math.Abs(a.X-503.2) > 1e-8 || math.Abs(b.X-799.2) > 1e-8 || a.VX != 100 || b.VX != -100 {
		t.Fatalf("unexpected drift: %+v %+v", a, b)
	}
}

func TestWindWarningRestOutsideAndPause(t *testing.T) {
	for _, mode := range []string{"warning", "rest", "outside", "paused", "cleared"} {
		t.Run(mode, func(t *testing.T) {
			r := windTestRun(t)
			switch mode {
			case "warning":
				r.Clock = .3
			case "rest":
				r.Clock = 5
			case "outside":
				r.Projectiles[0].Y = 320
			case "paused":
				r.Paused = true
			case "cleared":
				r.Status = "cleared"
			}
			r.Step(Input{}, time.UnixMilli(r.LastMS+20))
			want := 502.0
			if mode == "paused" {
				want = 500
			}
			if len(r.Projectiles) == 0 || r.Projectiles[0].X != want {
				t.Fatalf("%s moved unexpectedly: %+v", mode, r.Projectiles)
			}
		})
	}
}

func TestWindResumeMatchesContinuousTrajectory(t *testing.T) {
	r := windTestRun(t)
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		now := time.UnixMilli(r.LastMS + 20)
		r.Step(Input{}, now)
		saved.Step(Input{}, now)
		if len(r.Projectiles) != len(saved.Projectiles) {
			t.Fatal("saved shot count diverged")
		}
		for j, p := range r.Projectiles {
			if p.X != saved.Projectiles[j].X || p.Y != saved.Projectiles[j].Y {
				t.Fatal("saved trajectory diverged")
			}
		}
	}
}

func TestWindDriftStillCollidesWithCover(t *testing.T) {
	r := windTestRun(t)
	r.Level.Rooms[0].HighCover = []Obstacle{{502.5, 350, 10, 100}}
	r.Step(Input{}, time.UnixMilli(r.LastMS+20))
	if len(r.Projectiles) != 1 || r.Projectiles[0].ID != 2 {
		t.Fatal("wind-blown shot bypassed cover")
	}
}

func TestWindCuesOncePerCycleAndSaved(t *testing.T) {
	r := windTestRun(t)
	r.Clock = 0
	r.Projectiles = nil
	counts := map[string]int{}
	seen := map[int]bool{}
	for i := 0; i < 500; i++ {
		r.Step(Input{}, time.UnixMilli(r.LastMS+20))
		for _, e := range r.Events {
			if !seen[e.ID] {
				counts[e.Kind]++
				seen[e.ID] = true
			}
		}
	}
	if counts["wind_warning"] != 2 || counts["wind_gust"] != 2 {
		t.Fatalf("cue counts: %v", counts)
	}
	raw, _ := json.Marshal(r)
	var saved Run
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Step(Input{}, time.UnixMilli(saved.LastMS+20))
	for _, e := range saved.Events {
		if e.Kind == "wind_gust" && !seen[e.ID] {
			t.Fatal("resume replayed gust cue")
		}
	}
}

func TestWindCampaignCopiesAreDetached(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for _, arena := range level.Rooms {
			for _, w := range arena.WindGusts {
				count++
				if level.Region != 3 || math.Abs(w.VX) != 60 {
					t.Fatal("unexpected wind authoring")
				}
			}
		}
	}
	if count != 10 {
		t.Fatalf("wind corridors=%d", count)
	}
	levels := Campaign()
	levels[30].Rooms[1].WindGusts[0].VX = 1
	if Campaign()[30].Rooms[1].WindGusts[0].VX == 1 {
		t.Fatal("campaign wind alias")
	}
	r := windTestRun(t)
	r.Level.Rooms[0].WindGusts = append(r.Level.Rooms[0].WindGusts, r.Level.Rooms[0].WindGusts[0])
	if r.projectileWind(500, 380) != 80 {
		t.Fatal("overlapping gust exceeded cap")
	}
}
