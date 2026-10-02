package rift

import (
	"math"
	"testing"
	"time"
)

func TestDirectionalDodgeAndInvulnerability(t *testing.T) {
	now := time.Unix(100, 0)
	r := testRun()
	r.Status = "fighting"

	// 1. Initial dodge forward
	r.Step(Input{X: 1, Dodge: true}, now.Add(50*time.Millisecond))
	if r.Stats.Dodges != 0 {
		t.Fatalf("expected 0 evasions before any attack arrives, got %d", r.Stats.Dodges)
	}
	if r.SkillTimers["dodge_invulnerability"] <= 0 {
		t.Fatal("expected dodge invulnerability timer")
	}
	if r.SkillTimers["dodge_cooldown"] <= 0 {
		t.Fatal("expected dodge cooldown timer")
	}
	if r.Player.Pose != "dodge" {
		t.Fatalf("expected dodge pose, got %s", r.Player.Pose)
	}

	// 2. Invulnerability protects against hurtPlayer and enemy hits, registering a dodge
	beforeHP := r.Player.HP
	r.hurtPlayer(50, r.Player.X+20, r.Player.Y)
	if r.Player.HP != beforeHP {
		t.Fatal("dodge invulnerability failed to prevent damage")
	}
	if r.Stats.Dodges != 1 {
		t.Fatalf("expected 1 successful evasion recorded, got %d", r.Stats.Dodges)
	}

	// 3. Repeated dodge while on cooldown is rejected
	r.Step(Input{X: 1, Dodge: true}, now.Add(100*time.Millisecond))
	if r.SkillTimers["dodge_cooldown"] < 0.8 {
		t.Fatal("dodge cooldown was reset or bypassed")
	}

	// 4. After cooldown expires, dodge can be used again
	r.SkillTimers["dodge_cooldown"] = 0
	r.Step(Input{X: -1, Dodge: true}, now.Add(1200*time.Millisecond))
	if r.SkillTimers["dodge_invulnerability"] <= 0 {
		t.Fatal("expected new dodge invulnerability timer")
	}
	if r.Player.Facing != -1 {
		t.Fatalf("expected facing left (-1) after dodge, got %f", r.Player.Facing)
	}
}

func TestRoomEntryGrace(t *testing.T) {
	r := testRun()
	r.Status = "cleared"
	r.Room = 0

	// Advance to next room
	if !r.NextRoom() {
		t.Fatal("NextRoom failed")
	}
	if r.SkillTimers["room_entry_grace"] < 1.0 {
		t.Fatalf("expected room entry grace >= 1.0s, got %f", r.SkillTimers["room_entry_grace"])
	}

	// Enemy attack during grace is suppressed
	beforeHP := r.Player.HP
	r.hurtPlayerFromEnemy(30, r.Player.X+20, r.Player.Y, "enemy-1")
	if r.Player.HP != beforeHP || r.Stats.DamageTaken != 0 {
		t.Fatal("room entry grace failed to suppress enemy damage")
	}

	// Hazard damage during grace is suppressed
	r.hurtPlayerFromHazard(20, r.Player.X, r.Player.Y)
	if r.Player.HP != beforeHP || r.Stats.DamageTaken != 0 {
		t.Fatal("room entry grace failed to suppress hazard damage")
	}

	// After grace expires, damage lands
	r.SkillTimers["room_entry_grace"] = 0
	r.hurtPlayerFromEnemy(30, r.Player.X+20, r.Player.Y, "enemy-1")
	if r.Player.HP >= beforeHP || r.Stats.DamageTaken == 0 {
		t.Fatal("damage did not land after room entry grace expired")
	}
}

func TestFirstHitGraceOnReconnect(t *testing.T) {
	now := time.Unix(100, 0)
	r := testRun()
	r.Status = "fighting"
	r.LastMS = now.UnixMilli()

	// Reconnect after 3 seconds of missed updates
	reconnectTime := now.Add(3 * time.Second)
	r.recoverConnection(reconnectTime)

	if !r.FirstHitGrace {
		t.Fatal("expected FirstHitGrace to be true on reconnect")
	}
	if r.SkillTimers["connection_grace"] < 1.0 {
		t.Fatal("expected connection_grace timer")
	}

	// Expire connection grace window so hit can be received
	r.SkillTimers["connection_grace"] = 0

	// First hit receives grace (50% mitigation)
	r.Build.Armor = 0
	hpBefore := r.Player.HP
	r.hurtPlayer(40, 0, 0)
	firstLoss := hpBefore - r.Player.HP
	if firstLoss != 20 {
		t.Fatalf("expected first hit loss of 20 (50%% of 40), got %f", firstLoss)
	}
	if r.FirstHitGrace {
		t.Fatal("expected FirstHitGrace to be consumed after first hit")
	}

	// Second hit takes full damage
	hpAfterFirst := r.Player.HP
	r.hurtPlayer(40, 0, 0)
	secondLoss := hpAfterFirst - r.Player.HP
	if secondLoss != 40 {
		t.Fatalf("expected second hit loss of 40, got %f", secondLoss)
	}
}

func TestDeterministicCombatReplaySeed(t *testing.T) {
	now := time.Unix(100, 0)
	r1 := NewRun("test-seed-xyz", Build{HP: 200}, now)
	r2 := NewRun("test-seed-xyz", Build{HP: 200}, now)
	r3 := NewRun("other-seed-abc", Build{HP: 200}, now)

	if r1.ReplaySeed == 0 {
		t.Fatal("expected non-zero ReplaySeed")
	}
	if r1.ReplaySeed != r2.ReplaySeed {
		t.Fatalf("deterministic seeds diverged: %d != %d", r1.ReplaySeed, r2.ReplaySeed)
	}
	if r1.ReplaySeed == r3.ReplaySeed {
		t.Fatalf("expected different seeds for different IDs: %d == %d", r1.ReplaySeed, r3.ReplaySeed)
	}
	// Verify within JS safe integer range (53 bits)
	if r1.ReplaySeed > (1<<53 - 1) {
		t.Fatalf("seed exceeds JS safe integer limit: %d", r1.ReplaySeed)
	}
}

func TestArmorPiercingDamageBreakdown(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{ID: "knight-target", Kind: "knight", Armor: 0.4, HP: 500, MaxHP: 500}}

	// Deal 100 damage with 50% armor pierce
	r.hurtEnemyPiercing(0, 100, "hit", 0.5)

	if r.Stats.ArmorPiercingDamage <= 0 {
		t.Fatalf("expected ArmorPiercingDamage > 0, got %f", r.Stats.ArmorPiercingDamage)
	}

	// Verify EncounterSummary captures armor piercing damage
	r.RecordEncounterSummary("cleared")
	if r.LastEncounter.ArmorPiercingDamage != r.Stats.ArmorPiercingDamage {
		t.Fatalf("LastEncounter.ArmorPiercingDamage %f != Stats.ArmorPiercingDamage %f",
			r.LastEncounter.ArmorPiercingDamage, r.Stats.ArmorPiercingDamage)
	}
}

func TestUninterruptedAttackChains(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{ID: "target-1", HP: 500, MaxHP: 500}}

	// Connect two hits
	r.hurtEnemy(0, 20, "hit")
	r.hurtEnemy(0, 20, "hit")

	if r.AttackChain != 2 {
		t.Fatalf("expected AttackChain 2, got %d", r.AttackChain)
	}
	if r.Stats.HighestAttackChain != 2 {
		t.Fatalf("expected HighestAttackChain 2, got %d", r.Stats.HighestAttackChain)
	}

	// Taking damage breaks the attack chain
	r.hurtPlayer(15, 0, 0)
	if r.AttackChain != 0 {
		t.Fatalf("expected AttackChain reset to 0 after taking damage, got %d", r.AttackChain)
	}
	// But HighestAttackChain is preserved
	if r.Stats.HighestAttackChain != 2 {
		t.Fatalf("expected HighestAttackChain preserved at 2, got %d", r.Stats.HighestAttackChain)
	}
}

func TestBoundedComboScoring(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{ID: "target-1", HP: 500, MaxHP: 500}}

	r.Combo = 1
	r.hurtEnemy(0, 10, "hit")
	firstScore := r.Stats.ComboScore
	if firstScore <= 0 {
		t.Fatal("expected positive combo score")
	}

	r.Combo = 3
	r.hurtEnemy(0, 10, "hit")
	if r.Stats.ComboScore <= firstScore {
		t.Fatal("expected higher combo score for combo 3")
	}

	// Bounded by 50,000 maximum
	r.awardComboScore(100000)
	if r.Stats.ComboScore != 50000 {
		t.Fatalf("expected combo score capped at 50000, got %d", r.Stats.ComboScore)
	}
}

func TestNormalizedDiagonalKnockback(t *testing.T) {
	r := testRun()
	target := Actor{X: 500, Y: 350}

	// Diagonal knockback with dx=30, dy=30
	// Pre-normalization hypot was 42.43
	// Normalized, total displacement should equal targetDist (30)
	r.knockbackActor(&target, 30, 30)

	dx := target.X - 500
	dy := target.Y - 350
	dist := math.Hypot(dx, dy)
	if math.Abs(dist-30) > 1.0 {
		t.Fatalf("expected normalized diagonal knockback distance ~30, got %f", dist)
	}
}
