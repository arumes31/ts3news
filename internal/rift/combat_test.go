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
