package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func hazardLaneRun() *Run {
	r := laneSlamRun()
	r.Clock = 2
	r.SkillTimers = map[string]float64{}
	r.Level.Rooms[0].Hazards = []Hazard{{Kind: "poison", Obstacle: Obstacle{35, 315, 1530, 175}, Period: 10, Duration: 4}}
	r.enemyTick(0, .02)
	return r
}

func TestLaneReservationSuppressesHazardContact(t *testing.T) {
	for lane := 0; lane < 3; lane++ {
		r := hazardLaneRun()
		r.Player.Y = 315 + (float64(lane)+.5)*bossLaneHeight
		hp := r.Player.HP
		r.hazardTick()
		if lane == 1 {
			if r.Player.HP >= hp {
				t.Fatal("danger lane lost its hazard")
			}
		} else if r.Player.HP != hp || r.Stats.HazardContacts != 0 || r.SkillTimers["slowed"] > 0 || r.SkillTimers["hazard-0"] > 0 {
			t.Fatalf("safe lane %d still applies hazard damage, slow, or contact cooldown", lane)
		}
	}
}

func TestLaneReservationSaveImpactAndExpiry(t *testing.T) {
	r := hazardLaneRun()
	r.Player.Y = 330
	r.enemyTick(0, 1.6)
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	hp := saved.Player.HP
	saved.hazardTick()
	if saved.Player.HP != hp {
		t.Fatal("impact reservation lost across save")
	}
	for step := 0; step < 21; step++ {
		saved.tick(Input{}, .02)
	}
	saved.hazardTick()
	if saved.Player.HP >= hp {
		t.Fatal("reservation did not expire")
	}
}

func TestLaneReservationCancellationAndOverlaps(t *testing.T) {
	for _, mode := range []string{"cancel", "dead", "overlap"} {
		t.Run(mode, func(t *testing.T) {
			r := hazardLaneRun()
			r.Player.Y = 330
			switch mode {
			case "cancel":
				r.Enemies[0].Windup = 0
			case "dead":
				r.Enemies[0].HP = 0
			case "overlap":
				other := r.Enemies[0]
				other.ID, other.SlamLane = "other", 0
				r.Enemies = append(r.Enemies, other)
			}
			hp := r.Player.HP
			r.hazardTick()
			if r.Player.HP >= hp {
				t.Fatal("invalid safe lane suppressed hazards")
			}
		})
	}
	r := hazardLaneRun()
	r.Player.Y = 330
	hp := r.Player.HP
	r.hurtPlayerFromEnemy(40, 500, 330, "other")
	if r.Player.HP >= hp {
		t.Fatal("reservation made player immune to enemies")
	}
}

func TestLaneReservationPauseAndRoomReset(t *testing.T) {
	r := hazardLaneRun()
	r.Player.Y = 330
	r.enemyTick(0, 1.6)
	r.Paused = true
	r.Step(Input{}, time.UnixMilli(r.LastMS+1000))
	if r.SkillTimers[bossLaneReservationKeys[1]] != .4 {
		t.Fatal("pause consumed impact reservation")
	}
	r.spawnRoom()
	for _, key := range bossLaneReservationKeys {
		if r.SkillTimers[key] != 0 {
			t.Fatal("impact reservation leaked into next room")
		}
	}
}

func TestLaneReservationImpactCannotOverrideAnotherWarning(t *testing.T) {
	r := hazardLaneRun()
	r.Player.Y = 470
	r.enemyTick(0, 1.6)
	r.Enemies[0].Windup = 1.6
	r.Enemies[0].AttackName = "Lane Slam"
	r.Enemies[0].SlamLane = 0
	for lane := 0; lane < 3; lane++ {
		want := lane == 2
		if r.reservedBossArea(r.Player.X, 315+(float64(lane)+.5)*bossLaneHeight) != want {
			t.Fatalf("wrong reservation for simultaneous impact and warning in lane %d", lane)
		}
	}
}
