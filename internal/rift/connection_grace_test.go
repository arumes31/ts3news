package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestConnectionGraceRecoveryAndExpiry(t *testing.T) {
	now := time.Unix(100, 0)
	for _, resume := range []bool{false, true} {
		r, err := NewPracticeRun("reconnect", Build{HP: 100}, "hazard", now)
		if err != nil {
			t.Fatal(err)
		}
		r.Paused = false
		if resume {
			r.SetPaused(false, now.Add(3*time.Second))
		} else {
			r.Step(Input{}, now.Add(3*time.Second))
		}
		if r.SkillTimers["connection_grace"] < .99 {
			t.Fatal("missing recovery interval")
		}
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		saved.Clock = 1.3
		before := saved.Player
		saved.hazardTick()
		saved.hurtPlayer(30, before.X, before.Y)
		if saved.Player.HP != before.HP || saved.Stats.HazardContacts != 0 || saved.SkillTimers["slowed"] > 0 {
			t.Fatal("recovery did not suppress contact")
		}
		for i := 1; i <= 14; i++ {
			saved.Step(Input{}, now.Add(3*time.Second+time.Duration(i)*100*time.Millisecond))
		}
		if saved.SkillTimers["connection_grace"] > 0 {
			t.Fatal("grace failed to expire")
		}
		hp := saved.Player.HP
		saved.hurtPlayer(10, saved.Player.X, saved.Player.Y)
		if saved.Player.HP >= hp {
			t.Fatal("damage remained disabled")
		}
	}
}

func TestConnectionGraceIgnoresPauseAndNormalUpdates(t *testing.T) {
	now := time.Unix(100, 0)
	for _, tc := range []struct {
		name   string
		paused bool
		gap    time.Duration
	}{{"normal", false, time.Second}, {"backwards", false, -time.Second}, {"pause", true, 3 * time.Second}} {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := NewPracticeRun("reconnect", Build{HP: 100}, "hazard", now)
			r.Paused = tc.paused
			r.SetPaused(false, now.Add(tc.gap))
			if r.SkillTimers["connection_grace"] > 0 {
				t.Fatal("unexpected grace")
			}
		})
	}
}
