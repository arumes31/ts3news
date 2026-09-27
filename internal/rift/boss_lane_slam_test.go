package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func laneSlamRun() *Run {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	e := AdaptMonster(content.Mob{Name: i18n.T("mob.chronos"), Type: content.MobLegendary})
	e.ID, e.X, e.Y = "chronos", 1000, 410
	r.Enemies = []Actor{e}
	r.Level = &Level{Rooms: []Arena{{}}}
	return r
}

func TestChronosCyclesFullWidthLanesAcrossSave(t *testing.T) {
	r := laneSlamRun()
	for attack, want := range []int{1, 2, 0, 1} {
		e := &r.Enemies[0]
		e.Attacks = attack * 2
		e.Cooldown, e.PoseTime = 0, 0
		e.Pose = "idle"
		plan := r.NextBossAttack(*e)
		if plan.Kind != "lane_slam" || plan.Windup != 1.4 {
			t.Fatalf("missing lane plan: %+v", plan)
		}
		r.enemyTick(0, .02)
		if e.SlamLane != want || e.Windup != 1.4 {
			t.Fatalf("wrong lane/warning: %+v", e)
		}
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		saved.Player.X = 80
		saved.Player.Y = 315 + (float64(want)+.5)*175/3
		before := saved.Player.HP
		saved.enemyTick(0, 1.4)
		if saved.Player.HP >= before || saved.Enemies[0].Cooldown != 2.3 || findEvent(saved.Events, "lane_slam") == nil {
			t.Fatal("saved full-width slam did not resolve")
		}
		r = &saved
	}
}

func TestLaneSlamClearsOtherLanesAndAllowsJump(t *testing.T) {
	for _, mode := range []string{"other-lane", "jump", "dodge", "guard-break"} {
		r := laneSlamRun()
		r.enemyTick(0, .02)
		switch mode {
		case "other-lane":
			r.Player.Y = 330
		case "jump":
			r.Player.Jump = .5
		case "dodge":
			r.SkillTimers["dodge_invulnerability"] = 1
		case "guard-break":
			r.Enemies[0].BossStagger = 80
			r.hurtEnemy(0, 1, "hit")
		}
		before := r.Player.HP
		r.enemyTick(0, 1.4)
		if r.Player.HP != before {
			t.Fatalf("failed %s counterplay", mode)
		}
		if mode == "guard-break" && findEvent(r.Events, "lane_slam") != nil {
			t.Fatal("cancelled lane slam fired")
		}
	}
	r := laneSlamRun()
	r.Enemies[0].Attacks = 1
	if r.NextBossAttack(r.Enemies[0]).Kind != "projectile" {
		t.Fatal("lane attack replaced ordinary volley")
	}
	r = laneSlamRun()
	r.Practice = &PracticeState{SlowTelegraphs: true}
	r.enemyTick(0, .02)
	if r.Enemies[0].Windup != 2.8 {
		t.Fatal("lane slam ignored slow practice")
	}
}

func TestCampaignLaneSlamsAllowWalkingEscapeAtEntry(t *testing.T) {
	checked := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			for lane := 0; lane < 3; lane++ {
				x, y := 160.0, 315+(float64(lane)+.5)*bossLaneHeight
				blocked := false
				for _, wall := range arena.solidObstacles() {
					if contains(wall, x, y, 10) {
						blocked = true
					}
				}
				if blocked {
					continue
				}
				escaped := false
				for _, direction := range []float64{-1, 1} {
					r := laneSlamRun()
					copyLevel := level
					r.Level = &copyLevel
					r.Room = room
					r.Player.X, r.Player.Y = x, y
					r.enemyTick(0, .02)
					if r.Enemies[0].SlamLane != lane || r.Enemies[0].Windup != 1.4 {
						t.Fatal("entry warning not committed")
					}
					hp := r.Player.HP
					for step := 0; step < 15; step++ {
						r.tick(Input{}, .02)
					}
					for step := 0; step < 56; step++ {
						r.tick(Input{Y: direction}, .02)
					}
					if r.Player.HP == hp && r.Enemies[0].Attacks == 1 && bossLane(r.Player.Y) != lane {
						escaped = true
						break
					}
				}
				checked++
				if !escaped {
					t.Fatalf("no walking escape: mission %d room %d lane %d", level.ID, room, lane)
				}
			}
		}
	}
	if checked < 800 {
		t.Fatalf("insufficient entry coverage: %d", checked)
	}
	t.Logf("Checked %d legal campaign lane entries", checked)
}

func TestBossLaneBoundariesMatchDisplayedBands(t *testing.T) {
	for lane := 0; lane < 3; lane++ {
		start := 315 + float64(lane)*bossLaneHeight
		if bossLane(start) != lane {
			t.Fatalf("lane %d start mapped to %d", lane, bossLane(start))
		}
		if lane > 0 && bossLane(start-1e-7) != lane-1 {
			t.Fatal("previous lane ended early")
		}
	}
	if bossLane(490) != 2 {
		t.Fatal("bottom arena edge has no lane")
	}
}
