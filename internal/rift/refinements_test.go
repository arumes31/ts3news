package rift

import (
	"testing"
	"time"
)

func TestUnavailableSkillDoesNotSuppressHeldBasicAttack(t *testing.T) {
	r := testRun()
	r.SkillTimers["fire"] = 3
	r.Enemies = []Actor{{ID: "target", HP: 100, MaxHP: 100, X: r.Player.X + 35, Y: r.Player.Y}}
	r.Step(Input{Attack: true, Skill: "fire"}, time.Unix(100, 100_000_000))
	if r.Enemies[0].HP >= 100 || r.Stats.Attacks != 1 || r.Stats.SkillsCast != 0 {
		t.Fatal("unavailable skill suppressed the basic attack")
	}
}

func TestChronomancerCannotShortenHazardHitProtection(t *testing.T) {
	r := testRun()
	r.Build.Class = "chronomancer"
	r.Resource = 3
	r.SkillTimers["fire"] = 3
	r.SkillTimers["hazard-0"] = .8
	r.SkillTimers["slowed"] = .8
	r.classCast(Skill{ID: "rewind", Role: "finisher"})
	if r.SkillTimers["fire"] != 1.5 {
		t.Fatal("ability cooldown was not rewound")
	}
	if r.SkillTimers["hazard-0"] != .8 || r.SkillTimers["slowed"] != .8 {
		t.Fatal("rewind modified environment timers")
	}
}

func TestCheckpointClearsOldHazardTimersAndTargetMarks(t *testing.T) {
	r := testRun()
	r.Status = "cleared"
	r.Marked = "old-enemy"
	r.SkillTimers["hazard-0"] = .8
	r.SkillTimers["slowed"] = 1
	r.SkillTimers["fire"] = 2
	r.NextRoom()
	if r.Marked != "" || r.SkillTimers["hazard-0"] != 0 || r.SkillTimers["slowed"] != 0 {
		t.Fatal("room-local state leaked into next room")
	}
	if r.SkillTimers["fire"] != 2 {
		t.Fatal("checkpoint reset an equipped skill cooldown")
	}
}
