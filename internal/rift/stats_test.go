package rift

import (
	"math"
	"testing"
	"time"
)

func TestCombatStatsCountEffectiveDamageAndKillsOnce(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{ID: "boss", Kind: "boss", HP: 20, MaxHP: 20}}
	r.hurtEnemy(0, 1000, "hit")
	r.hurtEnemy(0, 1000, "hit")
	if r.Stats.DamageDealt != 20 || r.Stats.Kills != 1 || r.Stats.Bosses != 1 || r.Stats.LargestHit != 20 {
		t.Fatalf("overkill or repeated death inflated stats: %+v", r.Stats)
	}
}

func TestCombatStatsSeparateGuardBarrierAndHealthDamage(t *testing.T) {
	r := testRun()
	r.Build.Armor = 0
	r.Player.Guard = true
	r.Barrier = 3
	r.hurtPlayer(100, r.Player.X+20, r.Player.Y)
	if math.Abs(r.Stats.GuardBlocked-82) > .001 || r.Stats.BarrierBlocked != 3 || math.Abs(r.Stats.DamageTaken-15) > .001 || r.Stats.Guards != 1 {
		t.Fatalf("incorrect defense accounting: %+v", r.Stats)
	}
	r.Player.Guard = false
	r.Player.HP = 2
	r.hurtPlayer(100, r.Player.X, r.Player.Y)
	if math.Abs(r.Stats.DamageTaken-17) > .001 {
		t.Fatal("counted damage after health reached zero")
	}
}

func TestCombatStatsExcludeOverhealAndPausedTime(t *testing.T) {
	r := testRun()
	r.Player.HP = r.Player.MaxHP - 5
	r.healPlayer(30)
	r.healPlayer(30)
	if r.Stats.Healing != 5 {
		t.Fatal("counted overhealing")
	}
	r.Step(Input{}, time.Unix(100, 100_000_000))
	elapsed := r.Stats.Seconds
	if elapsed <= 0 {
		t.Fatal("fighting time missing")
	}
	r.Paused = true
	r.Step(Input{}, time.Unix(100, 200_000_000))
	r.Paused = false
	r.Status = "cleared"
	r.Step(Input{}, time.Unix(100, 300_000_000))
	if r.Stats.Seconds != elapsed {
		t.Fatal("pause or checkpoint time inflated combat duration")
	}
}
