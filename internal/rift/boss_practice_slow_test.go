package rift

import (
	"testing"
	"time"
)

func TestBossPracticeLongWarningsPreserveAttackRecovery(t *testing.T) {
	for phase := 1; phase <= 3; phase++ {
		r, err := NewPracticeRun("slow", testRun().Build, "boss", time.Unix(100, 0))
		if err != nil {
			t.Fatal(err)
		}
		r.Enemies[0].Phase = phase
		ordinary := r.NextBossAttack(r.Enemies[0])
		r.Practice.SlowTelegraphs = true
		slow := r.NextBossAttack(r.Enemies[0])
		if slow.Windup != ordinary.Windup*2 || slow.Recovery != ordinary.Recovery {
			t.Fatal("slow practice changed wrong timing")
		}
		r.Player.X = 500
		r.enemyTick(0, .01)
		if r.Enemies[0].Windup != slow.Windup {
			t.Fatal("extended warning not used in combat")
		}
		r.enemyTick(0, ordinary.Windup)
		if r.Enemies[0].Attacks != 0 {
			t.Fatal("attack released at normal timing")
		}
		r.enemyTick(0, ordinary.Windup+.01)
		if r.Enemies[0].Attacks != 1 || r.Enemies[0].Cooldown != ordinary.Recovery {
			t.Fatal("extended warning did not release normally")
		}
		if err := r.ResetPractice(time.Unix(200, 0)); err != nil {
			t.Fatal(err)
		}
		if !r.Practice.SlowTelegraphs {
			t.Fatal("reset forgot slow warnings")
		}
	}
}
