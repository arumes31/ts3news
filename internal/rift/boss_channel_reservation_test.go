package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func hazardChannelRun() *Run {
	r := channelRun()
	r.Clock = 2
	r.SkillTimers = map[string]float64{}
	r.Level.Rooms[0].Hazards = []Hazard{{Kind: "poison", Obstacle: Obstacle{35, 315, 1530, 175}, Period: 10, Duration: 4}}
	r.enemyTick(0, .02)
	return r
}
func TestChannelReservationProtectsOnlyOuterBand(t *testing.T) {
	for _, c := range []struct {
		x, y float64
		safe bool
	}{{500, 410, false}, {625, 410, false}, {625.01, 410, true}, {700, 410, true}, {700.01, 410, false}, {500, 472, false}, {500, 480, true}} {
		r := hazardChannelRun()
		r.Player.X, r.Player.Y = c.x, c.y
		hp := r.Player.HP
		r.hazardTick()
		if (r.Player.HP == hp) != c.safe {
			t.Errorf("wrong reservation at %.2f,%.2f", c.x, c.y)
		}
		if c.safe && (r.Stats.HazardContacts != 0 || r.SkillTimers["slowed"] > 0 || r.SkillTimers["hazard-0"] > 0) {
			t.Fatal("shelter applied contact effects")
		}
	}
}
func TestChannelReservationSavePauseExpiryAndRoomReset(t *testing.T) {
	r := hazardChannelRun()
	r.Player.X = 650
	r.enemyTick(0, 2)
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Enemies[0].HP = 0
	hp := saved.Player.HP
	saved.hazardTick()
	if saved.Player.HP != hp {
		t.Fatal("saved released shelter lost on boss defeat")
	}
	saved.Paused = true
	saved.Step(Input{}, time.UnixMilli(saved.LastMS+1000))
	if saved.SkillTimers["hazard-channel-chronos"] != .45 {
		t.Fatal("pause consumed shelter")
	}
	// Keep combat active for the expiry probe; a cleared room stops hazards.
	saved.Enemies[0].HP = 100
	saved.Paused = false
	for i := 0; i < 24; i++ {
		saved.tick(Input{}, .02)
	}
	if saved.Player.HP >= hp {
		t.Fatal("shelter never expired")
	}
	r.spawnRoom()
	if r.SkillTimers["hazard-channel-chronos"] != 0 {
		t.Fatal("room retained shelter")
	}
}
func TestChannelReservationCancelsAndRespectsOtherDanger(t *testing.T) {
	for _, mode := range []string{"interrupt", "dead", "lane", "ring", "pulse"} {
		r := hazardChannelRun()
		r.Player.X = 650
		switch mode {
		case "interrupt":
			r.hurtEnemy(0, 1, "hit")
		case "dead":
			r.Enemies[0].HP = 0
		case "lane":
			e := r.Enemies[0]
			e.ID = "lane"
			e.AttackName = "Lane Slam"
			e.SlamLane = 1
			r.Enemies = append(r.Enemies, e)
		case "ring":
			e := r.Enemies[0]
			e.ID = "ring"
			e.LaneSlams = false
			e.RingAttack = true
			e.AttackName = "Void Ring"
			e.TargetX = 500
			r.Enemies = append(r.Enemies, e)
		case "pulse":
			e := r.Enemies[0]
			e.ID = "pulse"
			e.TargetX = 650
			r.Enemies = append(r.Enemies, e)
		}
		hp := r.Player.HP
		r.hazardTick()
		if r.Player.HP >= hp {
			t.Fatalf("%s incorrectly sheltered", mode)
		}
	}
	r := hazardChannelRun()
	r.Player.X = 650
	hp := r.Player.HP
	r.hurtPlayerFromEnemy(40, 600, 410, "other")
	if r.Player.HP >= hp {
		t.Fatal("shelter blocked enemy damage")
	}
}
func TestChannelDangerOverridesExistingLaneReservation(t *testing.T) {
	r := hazardChannelRun()
	r.SkillTimers[bossLaneReservationKeys[0]] = 1
	if r.reservedBossArea(500, 410) {
		t.Fatal("pulse danger inherited lane shelter")
	}
}
