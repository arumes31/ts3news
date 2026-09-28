package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func hazardRingRun() *Run {
	r := ringRun()
	r.Clock = 2
	r.SkillTimers = map[string]float64{}
	r.Level.Rooms[0].Hazards = []Hazard{{Kind: "poison", Obstacle: Obstacle{35, 315, 1530, 175}, Period: 10, Duration: 4}}
	r.enemyTick(0, .02)
	return r
}

func TestRingReservationOnlyProtectsCenterAndGap(t *testing.T) {
	for _, c := range []struct {
		name string
		x, y float64
		safe bool
	}{
		{"center", 650, 410, true}, {"inner-edge", 750, 410, true},
		{"gap", 650, 340, true}, {"gap-edge", 725, 376.25, true},
		{"pulse", 800, 410, false}, {"outside", 420, 410, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := hazardRingRun()
			r.Player.X, r.Player.Y = c.x, c.y
			hp := r.Player.HP
			r.hazardTick()
			if (r.Player.HP == hp) != c.safe {
				t.Fatal("incorrect hazard contact in ring escape area")
			}
			if c.safe && (r.Stats.HazardContacts != 0 || r.SkillTimers["slowed"] > 0) {
				t.Fatal("safe area applied contact/slow")
			}
		})
	}
}

func TestRingReservationImpactSavePauseExpiry(t *testing.T) {
	r := hazardRingRun()
	r.Player.X, r.Player.Y = 650, 340
	r.enemyTick(0, 1.6)
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	hp := saved.Player.HP
	saved.hazardTick()
	if saved.Player.HP != hp {
		t.Fatal("impact shelter lost across save")
	}
	saved.Paused = true
	saved.Step(Input{}, time.UnixMilli(saved.LastMS+1000))
	saved.hazardTick()
	if saved.Player.HP != hp {
		t.Fatal("pause expired shelter")
	}
	saved.Paused = false
	for step := 0; step < 24; step++ {
		saved.tick(Input{}, .02)
	}
	if saved.Player.HP >= hp {
		t.Fatal("ring shelter did not expire")
	}
}

func TestRingReservationCancelledWarningAndLaneConflict(t *testing.T) {
	for _, mode := range []string{"cancel", "dead", "lane", "ring"} {
		t.Run(mode, func(t *testing.T) {
			r := hazardRingRun()
			r.Player.X, r.Player.Y = 650, 340
			switch mode {
			case "cancel":
				r.Enemies[0].Windup = 0
			case "dead":
				r.Enemies[0].HP = 0
			case "lane":
				e := laneSlamRun().Enemies[0]
				e.Windup = 1
				e.AttackName = "Lane Slam"
				e.SlamLane = 0
				r.Enemies = append(r.Enemies, e)
			case "ring":
				e := r.Enemies[0]
				e.ID = "second-ring"
				e.TargetX = 500
				e.TargetY = 340
				r.Enemies = append(r.Enemies, e)
			}
			hp := r.Player.HP
			r.hazardTick()
			if r.Player.HP >= hp {
				t.Fatal("conflicting/cancelled warning granted shelter")
			}
		})
	}
}

func TestRingReservationImpactSurvivesDefeatAndClearsAtRoomStart(t *testing.T) {
	r := hazardRingRun()
	r.Player.X, r.Player.Y = 650, 340
	r.enemyTick(0, 1.6)
	r.Enemies[0].HP = 0
	hp := r.Player.HP
	r.hazardTick()
	if r.Player.HP != hp {
		t.Fatal("defeat removed an already released impact reservation")
	}
	r.spawnRoom()
	if r.SkillTimers["hazard-ring-void-lord"] != 0 {
		t.Fatal("ring reservation leaked into next room")
	}
}

func TestRingReservationDoesNotShieldEnemyDamage(t *testing.T) {
	r := hazardRingRun()
	r.Player.X, r.Player.Y = 650, 340
	hp := r.Player.HP
	r.hurtPlayerFromEnemy(40, 650, 340, "other")
	if r.Player.HP >= hp {
		t.Fatal("ring shelter blocked an unrelated enemy")
	}
}
