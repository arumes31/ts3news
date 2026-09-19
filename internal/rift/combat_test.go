package rift

import (
	"testing"
	"time"
)

func testRun() *Run {
	return NewRun("test", Build{Name: "Delver", HP: 240, Damage: 24, Armor: 5, Skills: []Skill{{ID: "fire", Name: "Fire bolt", Kind: "fire", Power: 2, Cost: 20, Cooldown: 3}}}, time.Unix(100, 0))
}

func TestThirdStrikeInterruptsSmallEnemyButNotBoss(t *testing.T) {
	for _, kind := range []string{"goblin", "boss"} {
		t.Run(kind, func(t *testing.T) {
			r := testRun()
			r.Combo = 2
			r.Enemies = []Actor{{ID: "target", Kind: kind, X: r.Player.X + 35, Y: r.Player.Y, HP: 1000, MaxHP: 1000, Windup: .4}}
			r.Step(Input{Attack: true}, time.Unix(100, 100_000_000))
			if kind == "goblin" && (r.Enemies[0].Knockdown <= 0 || r.Enemies[0].Windup != 0) {
				t.Fatal("combo did not interrupt and knock down enemy")
			}
			if kind == "boss" && r.Enemies[0].Knockdown != 0 {
				t.Fatal("boss must resist knockdown")
			}
		})
	}
}

func TestMovementBoundedByServerTime(t *testing.T) {
	r := testRun()
	x := r.Player.X
	r.Step(Input{X: 1}, time.Unix(100, 0))
	if r.Player.X != x {
		t.Fatal("no elapsed server time must mean no movement")
	}
	r.Step(Input{X: 1}, time.Unix(200, 0))
	if r.Player.X-x > 55 {
		t.Fatal("disconnect must not fast-forward combat")
	}
}

func TestAttackRequiresFacingAndDepth(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{ID: "enemy", X: r.Player.X + 35, Y: r.Player.Y + 80, HP: 100, MaxHP: 100}}
	r.Step(Input{Attack: true}, time.Unix(100, 200_000_000))
	if r.Enemies[0].HP != 100 {
		t.Fatal("attack crossed depth lanes")
	}
	r.Player.Cooldown = 0
	r.Enemies[0].Y = r.Player.Y
	r.Step(Input{Attack: true}, time.Unix(100, 400_000_000))
	if r.Enemies[0].HP >= 100 {
		t.Fatal("in-range attack did not hit")
	}
}

func TestSkillCostAndCooldown(t *testing.T) {
	r := testRun()
	r.Step(Input{Skill: "fire"}, time.Unix(100, 100_000_000))
	if r.Player.Mana > 81 || r.SkillTimers["fire"] <= 0 {
		t.Fatal("cast must spend mana and start cooldown")
	}
	before := len(r.Projectiles)
	r.Step(Input{Skill: "fire"}, time.Unix(100, 200_000_000))
	if len(r.Projectiles) != before {
		t.Fatal("cooldown was bypassed")
	}
	r.Step(Input{Skill: "forged"}, time.Unix(100, 300_000_000))
	if len(r.Projectiles) != before {
		t.Fatal("unknown skill accepted")
	}
}

func TestDefeatDropsOnlyPendingLoot(t *testing.T) {
	r := testRun()
	r.Gold = 45
	r.BankedGold = 20
	r.Player.HP = 0
	r.Step(Input{}, time.Unix(100, 100_000_000))
	if r.Status != "defeated" || r.Gold != 0 || r.BankedGold != 20 {
		t.Fatal("defeat did not preserve banking boundary")
	}
}

func TestPauseAndRoomGate(t *testing.T) {
	r := testRun()
	if r.NextRoom() {
		t.Fatal("advanced past living enemies")
	}
	r.Paused = true
	x := r.Player.X
	r.Step(Input{X: 1, Attack: true}, time.Unix(100, 200_000_000))
	if x != r.Player.X {
		t.Fatal("paused run advanced")
	}
	r.Status = "cleared"
	if !r.NextRoom() || r.Room != 1 {
		t.Fatal("cleared room did not advance")
	}
}

func TestClockNeverReplaysPreviouslySimulatedTime(t *testing.T) {
	r := testRun()
	r.Step(Input{X: 1}, time.Unix(100, 200_000_000))
	x := r.Player.X
	r.Step(Input{X: 1}, time.Unix(100, 100_000_000))
	r.Step(Input{X: 1}, time.Unix(100, 200_000_000))
	if r.Player.X != x {
		t.Fatal("out-of-order server time advanced simulation twice")
	}
}

func TestClearCollectsFloorDropsBeforeCheckpoint(t *testing.T) {
	r := testRun()
	r.Enemies = nil
	r.Drops = []Drop{{ID: "far", X: 1500, Y: 350, Gold: 30}}
	r.Step(Input{}, time.Unix(100, 100_000_000))
	if r.Status != "cleared" || !r.Drops[0].Collected || r.Gold != 30 {
		t.Fatal("checkpoint receipt differs from expedition bag")
	}
}

func TestBossAttackNameDuringWindup(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{
		ID:       "boss1",
		Kind:     "boss",
		X:        r.Player.X + 80,
		Y:        r.Player.Y,
		HP:       1000,
		MaxHP:    1000,
		Cooldown: 0,
	}}
	// Trigger windup
	r.Step(Input{}, time.Unix(100, 100_000_000))
	if r.Enemies[0].Windup <= 0 {
		t.Fatalf("expected boss to be winding up, got %v", r.Enemies[0].Windup)
	}
	if r.Enemies[0].AttackName != "Mossbound Slam" {
		t.Fatalf("expected Mossbound Slam, got %q", r.Enemies[0].AttackName)
	}

	// Test alternating ranged attack
	r.Enemies[0].Attacks = 1
	r.Enemies[0].ArtKey = "boss_dragon"
	r.Enemies[0].Shot = "fire"
	r.Enemies[0].Windup = 0
	r.Enemies[0].Cooldown = 0
	r.Step(Input{}, time.Unix(100, 200_000_000))
	if r.Enemies[0].AttackName != "Cinder Volley" {
		t.Fatalf("expected Cinder Volley, got %q", r.Enemies[0].AttackName)
	}
}

func TestRunCatchupAfterStall(t *testing.T) {
	r := testRun()
	now := time.Now()
	r.LastMS = now.UnixMilli()

	// Normal step within 30ms: not catching up
	r.Step(Input{}, now.Add(30*time.Millisecond))
	if r.Catchup {
		t.Fatalf("expected Catchup to be false for 30ms step, got true")
	}

	// Step after 150ms stall: catching up
	r.Step(Input{}, now.Add(180*time.Millisecond))
	if !r.Catchup {
		t.Fatalf("expected Catchup to be true after 150ms stall, got false")
	}

	// Normal step again: catchup cleared
	r.Step(Input{}, now.Add(210*time.Millisecond))
	if r.Catchup {
		t.Fatalf("expected Catchup to be false on subsequent normal step, got true")
	}
}

func TestRunSavedAtMS(t *testing.T) {
	r := testRun()
	now := time.Now()
	r.LastMS = now.UnixMilli()

	if r.SavedAtMS == 0 {
		t.Fatalf("expected SavedAtMS to be set at creation, got 0")
	}

	// Step updates SavedAtMS
	step := now.Add(100 * time.Millisecond)
	r.Step(Input{}, step)
	if r.SavedAtMS != step.UnixMilli() {
		t.Fatalf("expected SavedAtMS=%d after Step, got %d", step.UnixMilli(), r.SavedAtMS)
	}

	// Pause updates SavedAtMS
	pauseTime := now.Add(200 * time.Millisecond)
	r.SetPaused(true, pauseTime)
	expected := max(r.LastMS, pauseTime.UnixMilli())
	if r.SavedAtMS != expected {
		t.Fatalf("expected SavedAtMS=%d after SetPaused, got %d", expected, r.SavedAtMS)
	}
}
