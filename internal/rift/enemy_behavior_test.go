package rift

import (
	"math"
	"testing"
)

func findEvent(events []Event, kind string) *Event {
	for i := range events {
		if events[i].Kind == kind {
			return &events[i]
		}
	}
	return nil
}

func TestRangedEnemySeparationPreventsStacking(t *testing.T) {
	r := testRun()
	r.Player.X = 100
	r.Player.Y = 410
	// Two archers placed at the exact same location
	r.Enemies = []Actor{
		{ID: "archer-1", Kind: "archer", X: 500, Y: 410, HP: 100, MaxHP: 100, Speed: 80},
		{ID: "archer-2", Kind: "archer", X: 501, Y: 410, HP: 100, MaxHP: 100, Speed: 80},
	}

	initialDist := math.Hypot(r.Enemies[0].X-r.Enemies[1].X, r.Enemies[0].Y-r.Enemies[1].Y)
	for step := 0; step < 10; step++ {
		r.enemyTick(0, 0.05)
		r.enemyTick(1, 0.05)
	}

	finalDist := math.Hypot(r.Enemies[0].X-r.Enemies[1].X, r.Enemies[0].Y-r.Enemies[1].Y)
	if finalDist <= initialDist {
		t.Fatalf("archers did not separate: initial=%f, final=%f", initialDist, finalDist)
	}
}

func TestShieldedEnemiesRearAttackBypassesBlock(t *testing.T) {
	r := testRun()
	r.Player.X = 100
	r.Player.Facing = 1 // Player facing right
	// Knight at 150, facing right (away from player) -> Player strikes the knight's back!
	r.Enemies = []Actor{
		{ID: "knight", Kind: "knight", X: 150, Y: 410, HP: 100, MaxHP: 100, Armor: 0.30, Shield: true, Facing: 1},
	}

	r.hurtEnemyPiercing(0, 40, "hit", 0)

	if r.Stats.RearStrikes != 1 {
		t.Fatalf("expected 1 rear strike, got %d", r.Stats.RearStrikes)
	}
	ev := findEvent(r.Events, "backstab")
	if ev == nil {
		t.Fatal("expected backstab event for rear strike on shielded enemy")
	}

	// Now knight is facing left (toward player) -> frontal strike is blocked by shield
	r.Enemies[0].Facing = -1
	r.Events = nil
	r.hurtEnemyPiercing(0, 40, "hit", 0)
	if r.Stats.RearStrikes != 1 {
		t.Fatalf("frontal strike should not count as rear strike, got %d", r.Stats.RearStrikes)
	}
	if findEvent(r.Events, "backstab") != nil {
		t.Fatal("frontal strike should not emit backstab")
	}
}

func TestFragileCasterRepositioningInterval(t *testing.T) {
	r := testRun()
	r.Player.X = 100
	r.Player.Y = 410
	r.Enemies = []Actor{
		{ID: "archer", Kind: "archer", X: 350, Y: 410, HP: 100, MaxHP: 100, Windup: 0.01, Speed: 80},
	}

	// Trigger projectile release
	r.enemyTick(0, 0.02)

	if len(r.Projectiles) != 1 {
		t.Fatalf("expected 1 projectile, got %d", len(r.Projectiles))
	}
	if r.Enemies[0].RepositionTimer <= 0 {
		t.Fatalf("expected archer to enter repositioning interval, got %f", r.Enemies[0].RepositionTimer)
	}

	// Progress past attack pose duration (0.40s) into repositioning run
	for i := 0; i < 9; i++ {
		r.enemyTick(0, 0.05)
	}
	if r.Enemies[0].Pose != "run" && r.Enemies[0].RepositionTimer > 0 {
		t.Fatalf("archer in repositioning interval should be in run pose, got %s", r.Enemies[0].Pose)
	}
}

func TestSmallEnemyFlanksAroundPlayerGuard(t *testing.T) {
	r := testRun()
	r.Player.X = 200
	r.Player.Y = 410
	r.Player.Facing = 1
	r.Player.Guard = true // Player holding guard facing right

	// Goblin is at 280 (in front of the player's shield)
	r.Enemies = []Actor{
		{ID: "goblin", Kind: "goblin", X: 280, Y: 410, HP: 100, MaxHP: 100, Speed: 115},
	}

	r.enemyTick(0, 0.05)

	if r.Stats.FlankAttempts == 0 {
		t.Fatalf("expected goblin to initiate flank attempt against guarding player, got %d", r.Stats.FlankAttempts)
	}
	ev := findEvent(r.Events, "flank_attempt")
	if ev == nil {
		t.Fatal("expected flank_attempt event to be emitted")
	}
}

func TestIdlePatrolSeparatedFromActivePursuitAndAwareness(t *testing.T) {
	r := testRun()
	r.Player.X = 100
	r.Player.Y = 410

	// Sentry on patrol at 500 (distance 400 > 240, out of alert range)
	r.Enemies = []Actor{
		{ID: "sentry", Kind: "goblin", X: 500, Y: 410, HP: 100, MaxHP: 100, Patrol: true, PatrolOriginX: 500, PatrolDir: -1, Speed: 80},
	}

	r.enemyTick(0, 0.05)
	if r.Enemies[0].Alerted {
		t.Fatal("enemy out of range should not alert")
	}
	if r.Enemies[0].Pose != "walk" {
		t.Fatalf("unalerted enemy on patrol should have walk pose, got %s", r.Enemies[0].Pose)
	}

	// Move player within detection radius (220px)
	r.Player.X = 350
	r.enemyTick(0, 0.05)

	if !r.Enemies[0].Alerted {
		t.Fatal("enemy within detection range should become alerted")
	}
	if r.Enemies[0].AwarenessTimer <= 0 {
		t.Fatalf("enemy should enter awareness phase, got %f", r.Enemies[0].AwarenessTimer)
	}
	if r.Enemies[0].Pose != "alert" {
		t.Fatalf("enemy in awareness should have alert pose, got %s", r.Enemies[0].Pose)
	}
	ev := findEvent(r.Events, "enemy_aware")
	if ev == nil {
		t.Fatal("expected enemy_aware event to be emitted")
	}

	// Enemy cannot attack while in awareness phase
	r.Enemies[0].Cooldown = 0
	r.enemyTick(0, 0.05)
	if r.Enemies[0].Windup > 0 {
		t.Fatal("enemy in awareness phase must not begin attack windup")
	}
}

func TestAlertPropagationRadius(t *testing.T) {
	r := testRun()
	r.Player.X = 350
	r.Player.Y = 410

	// Three enemies on patrol:
	// enemy 0 at 400 (distance 50 -> triggers alert)
	// enemy 1 at 550 (distance 150 from enemy 0 <= 220 -> propagates alert)
	// enemy 2 at 850 (distance 450 from enemy 0 > 220 -> remains unalerted on patrol)
	r.Enemies = []Actor{
		{ID: "e0", Kind: "goblin", X: 400, Y: 410, HP: 100, MaxHP: 100, Patrol: true, PatrolOriginX: 400, Speed: 80},
		{ID: "e1", Kind: "goblin", X: 550, Y: 410, HP: 100, MaxHP: 100, Patrol: true, PatrolOriginX: 550, Speed: 80},
		{ID: "e2", Kind: "goblin", X: 850, Y: 410, HP: 100, MaxHP: 100, Patrol: true, PatrolOriginX: 850, Speed: 80},
	}

	r.enemyTick(0, 0.05)

	if !r.Enemies[0].Alerted {
		t.Fatal("e0 should be alerted by close player")
	}
	if !r.Enemies[1].Alerted {
		t.Fatal("e1 should be alerted via propagation from e0")
	}
	if r.Enemies[2].Alerted {
		t.Fatal("e2 is beyond propagation radius and must remain unalerted on patrol")
	}
	ev := findEvent(r.Events, "alert_propagate")
	if ev == nil {
		t.Fatal("expected alert_propagate event to be emitted")
	}
}

func TestLongRangeCasterPredictsPlayerMovement(t *testing.T) {
	r := testRun()
	r.Player.X = 200
	r.Player.Y = 410
	r.Player.Vx = 200 // Player running right
	r.Player.Vy = 150 // Player running down

	r.Enemies = []Actor{
		{ID: "archer", Kind: "archer", X: 500, Y: 410, HP: 100, MaxHP: 100, Windup: 0.01, Speed: 80},
	}

	r.enemyTick(0, 0.02)

	if len(r.Projectiles) != 1 {
		t.Fatalf("expected 1 projectile, got %d", len(r.Projectiles))
	}
	proj := r.Projectiles[0]
	// Without vertical prediction (direct shot along Y=410), proj.VY would be exactly 0.
	// With downward prediction, proj.VY must lead downwards (proj.VY > 0)!
	if proj.VY <= 0 {
		t.Fatalf("expected projectile to lead downward runner (VY > 0), got proj.VY=%f", proj.VY)
	}
}

func TestPackEnemiesCoordinateAlternatingAttacks(t *testing.T) {
	r := testRun()
	r.Player.X = 460
	r.Player.Y = 410

	// Two pack wolves
	r.Enemies = []Actor{
		{ID: "pack-1", Kind: "goblin", Pack: true, X: 500, Y: 410, HP: 100, MaxHP: 100, Cooldown: 0},
		{ID: "pack-2", Kind: "goblin", Pack: true, X: 505, Y: 410, HP: 100, MaxHP: 100, Cooldown: 0},
	}

	// Pack 1 ticks and starts attack
	r.enemyTick(0, 0.05)
	if r.Enemies[0].Windup <= 0 {
		t.Fatal("first pack enemy should start attack windup")
	}
	if r.PackAttackLockout <= 0 {
		t.Fatalf("expected PackAttackLockout to be active, got %f", r.PackAttackLockout)
	}

	// Pack 2 ticks on same frame -> locked out by pack coordination
	r.enemyTick(1, 0.05)
	if r.Enemies[1].Windup > 0 {
		t.Fatal("second pack enemy should be locked out from attacking simultaneously")
	}
	if r.Stats.PackAttacks != 1 {
		t.Fatalf("expected 1 coordinated pack attack tracked, got %d", r.Stats.PackAttacks)
	}
}

func TestSummonedEnemyArrivalVulnerability(t *testing.T) {
	r := testRun()
	r.Enemies = nil // Clear default room enemies so summon is at index 0
	summon := r.SummonEnemy("goblin", 300, 410)

	if !summon.Summoned {
		t.Fatal("expected Summoned to be true")
	}
	if summon.ArrivalVulnerability <= 0 {
		t.Fatalf("expected ArrivalVulnerability > 0, got %f", summon.ArrivalVulnerability)
	}
	if summon.Pose != "spawn" {
		t.Fatalf("expected spawn pose, got %s", summon.Pose)
	}

	// Cannot attack while materializing
	r.enemyTick(0, 0.05)
	if r.Enemies[0].Windup > 0 {
		t.Fatal("summon in arrival phase must not attack")
	}

	// Punish summon during arrival window -> bonus +30% damage
	r.Player.X = 260
	r.hurtEnemyPiercing(0, 20, "hit", 0)

	if r.Stats.SummonPunishes != 1 {
		t.Fatalf("expected 1 summon punish, got %d", r.Stats.SummonPunishes)
	}
	ev := findEvent(r.Events, "summon_punish")
	if ev == nil {
		t.Fatal("expected summon_punish event")
	}
	// Damage should be 20 * 1.30 * (1 - armor) = 26
	if summon.HP > 60-25 {
		t.Fatalf("summon did not take vulnerability damage: HP=%f", summon.HP)
	}
}

func TestEncounterSummaryCapturesEnemyBehaviorMetrics(t *testing.T) {
	r := testRun()
	r.Stats.PackAttacks = 3
	r.Stats.SummonPunishes = 2
	r.Stats.FlankAttempts = 4
	r.Stats.RearStrikes = 5

	r.RecordEncounterSummary("cleared")

	if r.LastEncounter == nil {
		t.Fatal("expected LastEncounter to be populated")
	}
	enc := r.LastEncounter
	if enc.PackAttacks != 3 || enc.SummonPunishes != 2 || enc.FlankAttempts != 4 || enc.RearStrikes != 5 {
		t.Fatalf("EncounterSummary did not preserve behavior metrics: %+v", enc)
	}
}
