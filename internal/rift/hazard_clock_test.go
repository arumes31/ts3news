package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestHazardClockAndSavedTimestampNeverRewind(t *testing.T) {
	r := circleTestRun()
	r.RoomObjective = nil
	r.Enemies = []Actor{{ID: "far", Kind: "goblin", X: 1500, Y: 400, HP: 100, Knockdown: 100}}
	r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 100, 40}, Kind: "fire", Period: 7, Duration: 1}}}
	r.Clock = 1.19
	start := r.LastMS
	r.Step(Input{}, time.UnixMilli(start+20))
	clock, stamp, phase := r.Clock, r.SavedAtMS, r.Arena().Hazards[0].Phase(r.Clock)
	for _, delta := range []int64{-1000, 0, 10, 20} {
		r.Step(Input{}, time.UnixMilli(start+delta))
		if r.Clock != clock || r.LastMS != stamp || r.SavedAtMS != stamp || r.Arena().Hazards[0].Phase(r.Clock) != phase {
			t.Fatalf("timestamp %d rewound state: clock=%v saved=%d last=%d", delta, r.Clock, r.SavedAtMS, r.LastMS)
		}
	}
	r.Step(Input{}, time.UnixMilli(start+40))
	if math.Abs(r.Clock-clock-.02) > 1e-9 {
		t.Fatal("backward input changed next forward delta")
	}
}

func TestHazardPhasesFreezeAcrossPauseSaveAndResume(t *testing.T) {
	for _, kind := range []string{"fire", "ice", "poison", "thorns", "rune", "radiant", "void"} {
		for _, clock := range []float64{1.19, 1.3, 6.99} {
			t.Run(kind+"/"+time.Duration(clock*float64(time.Second)).String(), func(t *testing.T) {
				r := circleTestRun()
				r.RoomObjective = nil
				r.Build.Armor = 0
				r.Player.X = 520
				r.Player.Y = 370
				r.Level.Rooms[0] = Arena{Hazards: []Hazard{{Obstacle: Obstacle{500, 350, 100, 40}, Kind: kind, Period: 7, Duration: 1}}}
				r.Clock = clock
				r.SkillTimers["hazard-hit"] = .2
				r.SkillTimers["hazard-0"] = .4
				stamp := r.LastMS
				r.SetPaused(true, time.UnixMilli(stamp))
				hp := r.Player.HP
				r.Step(Input{}, time.UnixMilli(stamp+3600000))
				raw, err := json.Marshal(r)
				if err != nil {
					t.Fatal(err)
				}
				var saved Run
				if err = json.Unmarshal(raw, &saved); err != nil {
					t.Fatal(err)
				}
				h := saved.Arena().Hazards[0]
				if saved.Clock != clock || h.Phase(saved.Clock) != h.Phase(clock) || saved.Player.HP != hp || saved.SkillTimers["hazard-hit"] != .2 || saved.SkillTimers["hazard-0"] != .4 {
					t.Fatal("pause/save advanced hazard phase, damage or cooldown")
				}
				saved.SetPaused(false, time.UnixMilli(stamp+3600000))
				saved.Step(Input{}, time.UnixMilli(stamp+3600020))
				if math.Abs(saved.Clock-clock-.02) > 1e-9 || math.Abs(h.Phase(saved.Clock)-h.Phase(clock+.02)) > 1e-9 {
					t.Fatal("resume used paused wall time")
				}
				if math.Abs(saved.SkillTimers["hazard-hit"]-.18) > 1e-9 {
					t.Fatal("resume did not continue hazard cooldown")
				}
			})
		}
	}
}
