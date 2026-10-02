package rift

import (
	"math"
	"testing"
	"time"
)

func TestGuardStaminaAndBreak(t *testing.T) {
	now := time.Unix(100, 0)
	r := newRunState("test-guard-stamina", testRun().Build, now)
	r.Status = "fighting"

	if r.Player.GuardStamina != 100 {
		t.Fatalf("expected initial guard stamina 100, got %v", r.Player.GuardStamina)
	}

	// Guarding drains stamina over time
	r.tick(Input{Guard: true}, 0.5)
	if r.Player.GuardStamina >= 100 || !r.Player.Guard {
		t.Fatalf("guarding did not drain stamina or enable guard: stamina=%v, guard=%v", r.Player.GuardStamina, r.Player.Guard)
	}

	// Releasing guard regenerates stamina
	prevStamina := r.Player.GuardStamina
	r.tick(Input{Guard: false}, 0.2)
	if r.Player.GuardStamina <= prevStamina {
		t.Fatalf("releasing guard did not regenerate stamina: was %v, now %v", prevStamina, r.Player.GuardStamina)
	}

	// Absorbing heavy hits depletes stamina and triggers guard break
	r.Player.GuardStamina = 15.0
	r.Player.Guard = true
	r.hurtPlayer(60.0, r.Player.X+50, r.Player.Y)

	if r.Player.GuardStamina != 0 {
		t.Fatalf("expected stamina 0 after heavy blow, got %v", r.Player.GuardStamina)
	}
	if r.Player.Guard {
		t.Fatal("expected guard to break, but player is still guarding")
	}
	if r.Player.Pose != "guard_break" {
		t.Fatalf("expected pose 'guard_break', got %q", r.Player.Pose)
	}
	if r.SkillTimers["guard_break_recovery"] <= 0 {
		t.Fatal("expected guard_break_recovery timer to be active")
	}
	if r.Stats.GuardBreaks != 1 {
		t.Fatalf("expected 1 guard break recorded, got %d", r.Stats.GuardBreaks)
	}

	// Cannot re-guard during guard break recovery
	r.tick(Input{Guard: true}, 0.1)
	if r.Player.Guard {
		t.Fatal("player was able to guard while guard-broken")
	}
}

func TestHeavyBasicAttack(t *testing.T) {
	now := time.Unix(100, 0)
	r := newRunState("test-heavy-attack", testRun().Build, now)
	r.Status = "fighting"
	r.Enemies = []Actor{{ID: "goblin-1", Kind: "goblin", X: r.Player.X + 40, Y: r.Player.Y, HP: 500, MaxHP: 500}}

	// Strike 1
	r.tick(Input{Attack: true}, 0)
	if r.Combo != 1 || r.Player.Pose != "attack" {
		t.Fatalf("strike 1 failed: combo=%d, pose=%q", r.Combo, r.Player.Pose)
	}

	// Strike 2
	r.Player.Cooldown = 0
	r.tick(Input{Attack: true}, 0)
	if r.Combo != 2 || r.Player.Pose != "attack" {
		t.Fatalf("strike 2 failed: combo=%d, pose=%q", r.Combo, r.Player.Pose)
	}

	// Strike 3: Heavy Attack
	r.Player.Cooldown = 0
	r.tick(Input{Attack: true}, 0)
	if r.Combo != 3 || r.Player.Pose != "heavy_attack" {
		t.Fatalf("strike 3 failed: combo=%d, pose=%q", r.Combo, r.Player.Pose)
	}
	if r.Stats.HeavyAttacks != 1 {
		t.Fatalf("expected 1 heavy attack tracked, got %d", r.Stats.HeavyAttacks)
	}

	// Check heavy slash event
	found := false
	for _, ev := range r.Events {
		if ev.Kind == "heavy_slash" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("missing heavy_slash event for strike 3")
	}
}

func TestAerialAndGroundedAttacks(t *testing.T) {
	now := time.Unix(100, 0)
	r := newRunState("test-aerial-sweep-launcher", testRun().Build, now)
	r.Status = "fighting"
	r.Enemies = []Actor{{ID: "goblin-1", Kind: "goblin", X: r.Player.X + 40, Y: r.Player.Y, HP: 500, MaxHP: 500}}

	// 1. Aerial basic attack
	r.Player.Jump = 0.5
	r.tick(Input{Attack: true}, 0)
	if r.Player.Pose != "aerial_attack" || r.Stats.AerialAttacks != 1 {
		t.Fatalf("aerial attack failed: pose=%q, aerial_attacks=%d", r.Player.Pose, r.Stats.AerialAttacks)
	}

	// 2. Grounded sweep attack (Down + Attack)
	r.Player.Jump = 0
	r.Player.Cooldown = 0
	r.tick(Input{Attack: true, Y: 0.8}, 0)
	if r.Player.Pose != "sweep_attack" || r.Stats.SweepAttacks != 1 {
		t.Fatalf("sweep attack failed: pose=%q, sweep_attacks=%d", r.Player.Pose, r.Stats.SweepAttacks)
	}
	if r.Enemies[0].Cooldown <= 0 {
		t.Fatal("sweep attack did not stumble enemy with cooldown")
	}

	// 3. Launcher attack against small enemies (Up + Attack)
	r.Player.Cooldown = 0
	r.tick(Input{Attack: true, Y: -0.8}, 0)
	if r.Player.Pose != "launcher_attack" || r.Stats.Launchers != 1 {
		t.Fatalf("launcher attack failed: pose=%q, launchers=%d", r.Player.Pose, r.Stats.Launchers)
	}
	if r.Enemies[0].Jump <= 0 || r.Enemies[0].Knockdown <= 0 {
		t.Fatalf("launcher did not pop enemy into air: jump=%v, knockdown=%v", r.Enemies[0].Jump, r.Enemies[0].Knockdown)
	}
}

func TestDownedFollowupAndJuggleCeiling(t *testing.T) {
	now := time.Unix(100, 0)
	r := newRunState("test-downed-and-juggle", testRun().Build, now)
	r.Status = "fighting"
	r.Enemies = []Actor{{ID: "target-1", Kind: "goblin", X: r.Player.X + 40, Y: r.Player.Y, HP: 1000, MaxHP: 1000, Knockdown: 0.8}}

	// Hit knocked-down enemy
	hpBefore := r.Enemies[0].HP
	r.tick(Input{Attack: true}, 0)
	if r.Stats.DownedFollowups != 1 {
		t.Fatalf("expected 1 downed followup, got %d", r.Stats.DownedFollowups)
	}
	damageDealt := hpBefore - r.Enemies[0].HP
	if damageDealt <= r.Build.Damage {
		t.Fatalf("expected downed bonus damage, dealt %v <= base %v", damageDealt, r.Build.Damage)
	}

	// Test airborne crowd juggle damage ceiling
	r.Enemies[0].Knockdown = 0
	r.Enemies[0].Jump = 0.5
	r.Player.Cooldown = 0
	r.Combo = 0

	// First juggle hit
	hpJuggle1 := r.Enemies[0].HP
	r.tick(Input{Attack: true}, 0)
	dmg1 := hpJuggle1 - r.Enemies[0].HP
	if r.Enemies[0].JuggleCount != 1 {
		t.Fatalf("expected juggle count 1, got %d", r.Enemies[0].JuggleCount)
	}

	// Second juggle hit
	r.Player.Cooldown = 0
	hpJuggle2 := r.Enemies[0].HP
	r.tick(Input{Attack: true}, 0)
	dmg2 := hpJuggle2 - r.Enemies[0].HP
	if r.Enemies[0].JuggleCount != 2 {
		t.Fatalf("expected juggle count 2, got %d", r.Enemies[0].JuggleCount)
	}
	if dmg2 >= dmg1 {
		t.Fatalf("juggle damage ceiling did not scale damage down: dmg2=%v >= dmg1=%v", dmg2, dmg1)
	}
}

func TestEnemyHitStunResistance(t *testing.T) {
	now := time.Unix(100, 0)
	r := newRunState("test-hit-stun-resist", testRun().Build, now)
	r.Status = "fighting"
	r.Enemies = []Actor{{ID: "target-1", Kind: "knight", X: r.Player.X + 40, Y: r.Player.Y, HP: 1000, MaxHP: 1000, Windup: 0.5}}

	// Interrupt first windup
	r.tick(Input{Attack: true}, 0)
	if r.Enemies[0].InterruptCount != 1 || r.Enemies[0].StunResist <= 0 {
		t.Fatalf("expected interrupt count 1 and stun resist > 0, got count=%d, resist=%v", r.Enemies[0].InterruptCount, r.Enemies[0].StunResist)
	}

	// Interrupt second windup
	r.Player.Cooldown = 0
	r.Enemies[0].Windup = 0.5
	firstRecoil := math.Abs(r.Enemies[0].RecoilX)
	r.tick(Input{Attack: true}, 0)
	secondRecoil := math.Abs(r.Enemies[0].RecoilX)

	if r.Enemies[0].InterruptCount != 2 {
		t.Fatalf("expected interrupt count 2, got %d", r.Enemies[0].InterruptCount)
	}
	if secondRecoil >= firstRecoil {
		t.Fatalf("stun resistance did not reduce recoil distance: second=%v >= first=%v", secondRecoil, firstRecoil)
	}
}

func TestMeleeEnemiesSpreadAcrossApproachLanes(t *testing.T) {
	now := time.Unix(100, 0)
	r := newRunState("test-lane-spread", testRun().Build, now)
	r.Status = "fighting"
	r.Player.X, r.Player.Y = 200, 400
	r.Enemies = []Actor{
		{ID: "goblin-0", Kind: "goblin", X: 500, Y: 400, HP: 100, MaxHP: 100, Speed: 100},
		{ID: "goblin-1", Kind: "goblin", X: 500, Y: 400, HP: 100, MaxHP: 100, Speed: 100},
	}

	// Advance enemy pursuit
	r.enemyTick(0, 0.1)
	r.enemyTick(1, 0.1)

	// Enemy 0 remains on player's lane (400)
	if r.Enemies[0].Y != 400 {
		t.Fatalf("expected enemy 0 on lane 400, got %v", r.Enemies[0].Y)
	}

	// Enemy 1 spreads away from lane 400 (fanning out)
	if r.Enemies[1].Y == 400 {
		t.Fatal("expected enemy 1 to spread across an alternate approach lane, but stayed at 400")
	}
}
