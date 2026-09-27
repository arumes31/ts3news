package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func supportRetreatRun() *Run {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	r.Enemies = []Actor{{ID: "support", Kind: "archer", Support: true, X: 600, Y: 410, HP: 100, MaxHP: 100, Speed: 80}, {ID: "defender", Kind: "knight", Shield: true, X: 640, Y: 410, HP: 100, MaxHP: 100}}
	return r
}

func TestSupportRetreatsBehindDefenderThenResumesAttacking(t *testing.T) {
	r := supportRetreatRun()
	r.enemyTick(0, .02)
	if r.Enemies[0].X <= 600 || findEvent(r.Events, "support_retreat") == nil {
		t.Fatal("support did not seek its defender")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Events = nil
	for n := 0; n < 120 && len(saved.Projectiles) == 0; n++ {
		saved.enemyTick(0, .02)
	}
	if saved.Enemies[0].X < 700 || len(saved.Projectiles) == 0 {
		t.Fatal("support failed to get behind defender and resume shooting")
	}
	if findEvent(saved.Events, "support_retreat") != nil {
		t.Fatal("continuous retreat replayed its cue after save")
	}
}

func TestSupportRetreatTracksPlayerSideAndDefenderLane(t *testing.T) {
	r := supportRetreatRun()
	r.Enemies[1].Y = 450
	r.enemyTick(0, .02)
	if r.Enemies[0].Y <= 410 {
		t.Fatal("support ignored defender lane")
	}
	r = supportRetreatRun()
	r.Player.X = 900
	r.Enemies[0].X = 780
	r.Enemies[1].X = 700
	r.enemyTick(0, .02)
	if r.Enemies[0].X >= 780 || r.Enemies[0].Facing != 1 {
		t.Fatal("support did not retreat behind defender after player flanked")
	}
}

func TestSupportRetreatPreservesShotsAndFallsBackWithoutCover(t *testing.T) {
	r := supportRetreatRun()
	r.Enemies[0].Windup = .01
	r.enemyTick(0, .02)
	if len(r.Projectiles) != 1 || r.Enemies[0].X != 600 {
		t.Fatal("retreat interrupted committed shot")
	}
	for _, mode := range []string{"dead", "wall", "not-defender"} {
		r = supportRetreatRun()
		r.Enemies[0].X = 700
		r.Enemies[1].X = 800
		switch mode {
		case "dead":
			r.Enemies[1].HP = 0
		case "wall":
			r.Level = &Level{Rooms: []Arena{{HighCover: []Obstacle{{740, 350, 20, 100}}}}}
		case "not-defender":
			r.Enemies[1].Kind = "archer"
			r.Enemies[1].Shield = false
		}
		r.enemyTick(0, .02)
		if r.Enemies[0].Windup == 0 || findEvent(r.Events, "support_retreat") != nil {
			t.Fatalf("support failed to fall back when %s", mode)
		}
	}
}

func TestFrostLichReceivesSupportRole(t *testing.T) {
	a := AdaptMonster(content.Mob{Name: i18n.T("mob.frost_lich"), Type: content.MobElite})
	if !a.Support || !a.Healer {
		t.Fatal("canonical healer lacks support role")
	}
}

func TestSupportReassessesLostDefenderAndPreservesMend(t *testing.T) {
	r := supportRetreatRun()
	r.enemyTick(0, .02)
	if r.Enemies[0].SupportCoverID != "defender" {
		t.Fatal("no active cover target")
	}
	r.Enemies[1].HP = 0
	r.enemyTick(0, .02)
	if r.Enemies[0].SupportCoverID != "" {
		t.Fatal("support retained dead defender")
	}
	r = supportRetreatRun()
	r.Enemies[0].Healer = true
	r.Enemies[1].HP = 10
	r.enemyTick(0, .02)
	if r.Enemies[0].HealTarget != "defender" || r.Enemies[0].SupportCoverID != "" || r.Enemies[0].X != 600 {
		t.Fatal("retreat overrode healing priority")
	}
	for n := 0; n < 41; n++ {
		r.enemyTick(0, .02)
	}
	if r.Enemies[1].HP <= 10 {
		t.Fatal("support movement interrupted committed mend")
	}
	r = supportRetreatRun()
	r.enemyTick(0, .02)
	r.hurtEnemy(0, 10, "hit")
	if r.Enemies[0].SupportCoverID != "" {
		t.Fatal("hit retained stale retreat cue")
	}
}

func TestSupportFinishesChosenRetreatBeyondInitialPressureRadius(t *testing.T) {
	r := supportRetreatRun()
	r.Enemies[0].X = 700
	r.Enemies[1].X = 800
	for n := 0; n < 180 && len(r.Projectiles) == 0; n++ {
		r.enemyTick(0, .02)
	}
	if r.Enemies[0].X < 860 || len(r.Projectiles) == 0 {
		t.Fatalf("retreat stopped before cover at %.1f", r.Enemies[0].X)
	}
}
