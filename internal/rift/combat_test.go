package rift

import (
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
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

func TestPerfectGuardEventWithinWindowAndNormalGuardAfter(t *testing.T) {
	r := testRun()
	now := time.Unix(100, 0)
	r.LastMS = now.UnixMilli()

	// Initial step raising guard
	r.Step(Input{Guard: true}, now.Add(50*time.Millisecond))
	if !r.Player.Guard {
		t.Fatal("player should be guarding")
	}
	if r.SkillTimers["perfect_guard"] <= 0 {
		t.Fatal("perfect_guard timer should be active immediately after raising guard")
	}

	// Incoming attack within window emits "perfect_guard"
	r.hurtPlayer(30, r.Player.X+20, r.Player.Y)
	if len(r.Events) == 0 || r.Events[len(r.Events)-1].Kind != "perfect_guard" {
		t.Fatalf("expected last event to be perfect_guard, got %+v", r.Events)
	}

	// Step forward beyond the 0.22s window while holding guard (two steps since dt is capped at 0.2s)
	r.Step(Input{Guard: true}, now.Add(200*time.Millisecond))
	r.Step(Input{Guard: true}, now.Add(350*time.Millisecond))
	if r.SkillTimers["perfect_guard"] > 0 {
		t.Fatal("perfect_guard timer should have expired after 300ms")
	}

	// Incoming attack after window emits standard "block"
	r.hurtPlayer(30, r.Player.X+20, r.Player.Y)
	if len(r.Events) == 0 || r.Events[len(r.Events)-1].Kind != "block" {
		t.Fatalf("expected last event to be block, got %+v", r.Events)
	}

	// Release guard
	r.Step(Input{Guard: false}, now.Add(400*time.Millisecond))
	if r.Player.Guard {
		t.Fatal("player should no longer be guarding")
	}
	r.hurtPlayer(30, r.Player.X+20, r.Player.Y)
	if len(r.Events) == 0 || r.Events[len(r.Events)-1].Kind != "hurt" {
		t.Fatalf("expected last event to be hurt, got %+v", r.Events)
	}
}

func TestTreasureGoblinEscapeEventAndNoKillCredit(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{
		{ID: "goblin-1", Kind: "treasure", X: 150, Y: 300, HP: 50, MaxHP: 50},
		{ID: "goblin-2", Kind: "treasure", X: 350, Y: 300, HP: 50, MaxHP: 50},
	}

	// Goblin 0 escapes
	r.EscapeEnemy(0)
	if r.Enemies[0].HP != 0 || r.Enemies[0].Pose != "escape" {
		t.Fatalf("expected goblin 0 to have HP=0 and Pose=escape, got HP=%v Pose=%s", r.Enemies[0].HP, r.Enemies[0].Pose)
	}
	if len(r.Events) == 0 || r.Events[len(r.Events)-1].Kind != "treasure_escape" {
		t.Fatalf("expected treasure_escape event, got %+v", r.Events)
	}
	lastEvent := r.Events[len(r.Events)-1]
	if lastEvent.X != 150 || lastEvent.Y != 300 {
		t.Fatalf("expected event at (150, 300), got (%v, %v)", lastEvent.X, lastEvent.Y)
	}
	if r.Stats.TreasureGoblins != 0 || r.Stats.Kills != 0 || len(r.Drops) != 0 {
		t.Fatalf("escaped goblin should not give stats or drops: %+v drops=%d", r.Stats, len(r.Drops))
	}

	// Goblin 1 is defeated by player
	r.hurtEnemy(1, 100, "hit")
	if r.Enemies[1].HP != 0 {
		t.Fatalf("expected goblin 1 to be defeated, got HP=%v", r.Enemies[1].HP)
	}
	if len(r.Events) == 0 || r.Events[len(r.Events)-1].Kind != "treasure_death" {
		t.Fatalf("expected treasure_death event, got %+v", r.Events)
	}
	if r.Stats.TreasureGoblins != 1 || r.Stats.Kills != 1 || len(r.Drops) != 1 {
		t.Fatalf("defeated goblin should grant 1 treasure goblin, 1 kill, 1 drop: %+v drops=%d", r.Stats, len(r.Drops))
	}
}

func TestRareItemDiscoveryEventEmittedOnCollection(t *testing.T) {
	r := testRun()
	now := time.Unix(100, 0)
	r.LastMS = now.UnixMilli()

	// 1. Drop with only gold emits "pickup"
	r.Player.X = 160
	r.Player.Y = 410
	r.Drops = []Drop{
		{ID: "drop-gold", X: 160, Y: 410, Gold: 50},
	}
	r.Step(Input{}, now.Add(50*time.Millisecond))
	if !r.Drops[0].Collected || r.Gold != 50 {
		t.Fatalf("expected drop-gold collected with gold=50, got collected=%v gold=%v", r.Drops[0].Collected, r.Gold)
	}
	if len(r.Events) == 0 || r.Events[len(r.Events)-1].Kind != "pickup" || r.Events[len(r.Events)-1].Value != 50 {
		t.Fatalf("expected pickup event with value 50, got %+v", r.Events)
	}

	// 2. Drop with common gear (Rarity < RarityRare) emits "pickup"
	r.Drops = append(r.Drops, Drop{
		ID:   "drop-common",
		X:    160,
		Y:    410,
		Gold: 20,
		Gear: &content.Gear{Name: "Iron Dagger", Rarity: content.RarityCommon},
	})
	r.Step(Input{}, now.Add(100*time.Millisecond))
	if !r.Drops[1].Collected || r.Gold != 70 {
		t.Fatalf("expected drop-common collected with gold=70, got collected=%v gold=%v", r.Drops[1].Collected, r.Gold)
	}
	if len(r.Events) == 0 || r.Events[len(r.Events)-1].Kind != "pickup" {
		t.Fatalf("expected pickup event for common gear, got %+v", r.Events)
	}

	// 3. Drop with Rare gear emits "rare_item" with value = Rarity
	r.Drops = append(r.Drops, Drop{
		ID:   "drop-rare",
		X:    160,
		Y:    410,
		Gold: 100,
		Gear: &content.Gear{Name: "Sapphire Wand", Rarity: content.RarityRare},
	})
	r.Step(Input{}, now.Add(150*time.Millisecond))
	if !r.Drops[2].Collected || r.Gold != 170 {
		t.Fatalf("expected drop-rare collected with gold=170, got collected=%v gold=%v", r.Drops[2].Collected, r.Gold)
	}
	if len(r.Events) == 0 || r.Events[len(r.Events)-1].Kind != "rare_item" || r.Events[len(r.Events)-1].Value != float64(content.RarityRare) {
		t.Fatalf("expected rare_item event with value %v, got %+v", float64(content.RarityRare), r.Events)
	}

	// 4. Drop with Legendary gear emits "rare_item" with value = Rarity
	r.Drops = append(r.Drops, Drop{
		ID:   "drop-legendary",
		X:    160,
		Y:    410,
		Gold: 500,
		Gear: &content.Gear{Name: "Sunforged Claymore", Rarity: content.RarityLegendary},
	})
	r.Step(Input{}, now.Add(200*time.Millisecond))
	if !r.Drops[3].Collected || r.Gold != 670 {
		t.Fatalf("expected drop-legendary collected with gold=670, got collected=%v gold=%v", r.Drops[3].Collected, r.Gold)
	}
	if len(r.Events) == 0 || r.Events[len(r.Events)-1].Kind != "rare_item" || r.Events[len(r.Events)-1].Value != float64(content.RarityLegendary) {
		t.Fatalf("expected rare_item event with value %v, got %+v", float64(content.RarityLegendary), r.Events)
	}
}

func TestLandingEventVariesByJumpIntensity(t *testing.T) {
	now := time.Unix(100, 0)
	r := NewRun("test-jump-landing", testRun().Build, now)
	r.Status = "fighting"

	// 1. Stationary hop (no directional movement): emits light landing (intensity <= 0.4)
	r.Step(Input{Jump: true}, now.Add(50*time.Millisecond))
	if r.Player.Jump == 0 {
		t.Fatal("expected player to be jumping")
	}
	currentTime := now.Add(50 * time.Millisecond)
	for i := 0; i < 20 && r.Player.Jump > 0; i++ {
		currentTime = currentTime.Add(50 * time.Millisecond)
		r.Step(Input{}, currentTime)
	}
	if r.Player.Jump != 0 {
		t.Fatalf("expected player to land, got Jump=%v", r.Player.Jump)
	}

	var stationaryLand *Event
	for i := len(r.Events) - 1; i >= 0; i-- {
		if r.Events[i].Kind == "land" {
			stationaryLand = &r.Events[i]
			break
		}
	}
	if stationaryLand == nil {
		t.Fatalf("expected 'land' event for stationary hop, events=%+v", r.Events)
	}
	if stationaryLand.Value > 0.4 {
		t.Fatalf("expected stationary hop landing intensity <= 0.4, got %v", stationaryLand.Value)
	}

	// Wait for jump cooldown
	currentTime = currentTime.Add(time.Second)
	r.SkillTimers["jump"] = 0

	// 2. Moving running leap (held directional movement): emits higher intensity (>= 0.6)
	currentTime = currentTime.Add(50 * time.Millisecond)
	r.Step(Input{Jump: true, X: 1}, currentTime)
	for i := 0; i < 20 && r.Player.Jump > 0; i++ {
		currentTime = currentTime.Add(50 * time.Millisecond)
		r.Step(Input{X: 1}, currentTime)
	}
	if r.Player.Jump != 0 {
		t.Fatalf("expected moving leap player to land, got Jump=%v", r.Player.Jump)
	}

	var movingLand *Event
	for i := len(r.Events) - 1; i >= 0; i-- {
		if r.Events[i].Kind == "land" {
			movingLand = &r.Events[i]
			break
		}
	}
	if movingLand == nil {
		t.Fatalf("expected 'land' event for moving leap, events=%+v", r.Events)
	}
	if movingLand.Value < 0.6 {
		t.Fatalf("expected moving leap landing intensity >= 0.6, got %v", movingLand.Value)
	}

	// 3. Knockdown landing (swatted out of air or slammed down): emits heavy landing (intensity = 1.0)
	currentTime = currentTime.Add(time.Second)
	r.SkillTimers["jump"] = 0
	currentTime = currentTime.Add(50 * time.Millisecond)
	r.Step(Input{Jump: true}, currentTime)
	r.Player.Knockdown = 0.5
	for i := 0; i < 20 && r.Player.Jump > 0; i++ {
		currentTime = currentTime.Add(50 * time.Millisecond)
		r.Step(Input{}, currentTime)
	}
	var knockdownLand *Event
	for i := len(r.Events) - 1; i >= 0; i-- {
		if r.Events[i].Kind == "land" {
			knockdownLand = &r.Events[i]
			break
		}
	}
	if knockdownLand == nil || knockdownLand.Value < 0.85 {
		t.Fatalf("expected heavy landing intensity >= 0.85 on knockdown, got %+v", knockdownLand)
	}

	// 4. Direct Land helper
	r.Land(0.92)
	last := r.Events[len(r.Events)-1]
	if last.Kind != "land" || last.Value != 0.92 {
		t.Fatalf("expected direct Land event with value 0.92, got %+v", last)
	}
}

func TestWeaponFamilyClassification(t *testing.T) {
	tests := []struct {
		weapon string
		class  string
		want   string
	}{
		{"Trusty Longsword", "vanguard", "blade"},
		{"Sunforged Claymore", "berserker", "blade"},
		{"Firebrand Greatsword", "bloodblade", "blade"},
		{"Earthshaker Warhammer", "geomancer", "blunt"},
		{"Spiked Mace", "warrior", "blunt"},
		{"Necrotic Dagger", "assassin", "pierce"},
		{"Soul Reaper Scythe", "reaver", "pierce"},
		{"Harvester's Gulthook", "adventurer", "pierce"},
		{"Lifebloom Staff", "elementalist", "arcane"},
		{"Tidal Wave Scepter", "oracle", "blunt"},
		{"Pocket Bow", "marksman", "ranged"},
		{"Unarmed", "", "fist"},
		{"Iron Knuckles", "brawler", "fist"},
		{"", "runesmith", "blunt"},
		{"", "chronomancer", "arcane"},
	}

	for _, tt := range tests {
		got := WeaponFamily(tt.weapon, tt.class)
		if got != tt.want {
			t.Errorf("WeaponFamily(%q, %q) = %q, want %q", tt.weapon, tt.class, got, tt.want)
		}
	}
}

func TestWeaponImpactEventVariesByWeaponFamily(t *testing.T) {
	families := []struct {
		weapon string
		class  string
		want   string
	}{
		{"Claymore", "vanguard", "hit_blade"},
		{"Warhammer", "geomancer", "hit_blunt"},
		{"Dagger", "marksman", "hit_pierce"},
		{"Staff", "elementalist", "hit_arcane"},
		{"Unarmed", "", "hit_fist"},
	}

	for _, tc := range families {
		now := time.Unix(100, 0)
		r, err := NewPracticeRun("test-"+tc.want, Build{
			Name:   "Tester",
			Class:  tc.class,
			Weapon: tc.weapon,
			Damage: 10,
			HP:     100,
		}, "combo", now)
		if err != nil {
			t.Fatalf("failed to create practice run: %v", err)
		}

		r.Player.X = 220
		r.Player.Y = 400
		r.Enemies[0].X = 260
		r.Enemies[0].Y = 400
		r.Player.Facing = 1

		r.Step(Input{Attack: true}, now.Add(50*time.Millisecond))

		var hitEvent *Event
		for i := len(r.Events) - 1; i >= 0; i-- {
			if strings.HasPrefix(r.Events[i].Kind, "hit") {
				hitEvent = &r.Events[i]
				break
			}
		}

		if hitEvent == nil {
			t.Fatalf("expected hit event for weapon %q, got events: %+v", tc.weapon, r.Events)
		}
		if hitEvent.Kind != tc.want {
			t.Fatalf("for weapon %q expected event kind %q, got %q", tc.weapon, tc.want, hitEvent.Kind)
		}
		if r.Practice.Hits != 1 {
			t.Fatalf("expected Practice.Hits to increment for weapon %q, got %d", tc.weapon, r.Practice.Hits)
		}
	}
}

func TestEnemyHurtEventsVaryByCreatureFamily(t *testing.T) {
	creatures := []struct {
		kind string
		want string
	}{
		{"goblin", "goblin_hurt"},
		{"knight", "knight_hurt"},
		{"archer", "archer_hurt"},
		{"treasure", "treasure_hurt"},
		{"boss", "boss_hurt"},
	}

	for _, tc := range creatures {
		now := time.Unix(100, 0)
		r := NewRun("test-hurt-"+tc.kind, testRun().Build, now)
		r.Enemies = []Actor{
			{ID: "enemy-1", Kind: tc.kind, X: 300, Y: 350, HP: 500, MaxHP: 500},
		}

		r.hurtEnemy(0, 25, "hit")

		var hurtEvent *Event
		for i := len(r.Events) - 1; i >= 0; i-- {
			if strings.HasSuffix(r.Events[i].Kind, "_hurt") {
				hurtEvent = &r.Events[i]
				break
			}
		}

		if hurtEvent == nil {
			t.Fatalf("expected creature hurt event for %q, got events: %+v", tc.kind, r.Events)
		}
		if hurtEvent.Kind != tc.want {
			t.Fatalf("for %q expected event %q, got %q", tc.kind, tc.want, hurtEvent.Kind)
		}
		if hurtEvent.Value <= 0 || hurtEvent.Value > 25 {
			t.Fatalf("expected hurt event value between 0 and 25, got %v", hurtEvent.Value)
		}
	}
}

func TestEnemyDeathEventsVaryByCreatureFamily(t *testing.T) {
	creatures := []struct {
		kind string
		want string
	}{
		{"goblin", "goblin_death"},
		{"knight", "knight_death"},
		{"archer", "archer_death"},
		{"treasure", "treasure_death"},
		{"boss", "boss_death"},
		{"wolf", "wolf_death"},
		{"spore", "spore_death"},
		{"unknown_beast", "goblin_death"}, // default fallback
	}

	for _, tc := range creatures {
		now := time.Unix(100, 0)
		r := NewRun("test-death-"+tc.kind, testRun().Build, now)
		r.Enemies = []Actor{
			{ID: "enemy-1", Kind: tc.kind, X: 300, Y: 350, HP: 10, MaxHP: 10},
		}

		// Deal lethal damage
		r.hurtEnemy(0, 50, "hit")

		var deathEvent *Event
		for i := len(r.Events) - 1; i >= 0; i-- {
			if strings.HasSuffix(r.Events[i].Kind, "_death") {
				deathEvent = &r.Events[i]
				break
			}
		}

		if deathEvent == nil {
			t.Fatalf("expected creature death event for %q, got events: %+v", tc.kind, r.Events)
		}
		if deathEvent.Kind != tc.want {
			t.Fatalf("for %q expected event %q, got %q", tc.kind, tc.want, deathEvent.Kind)
		}
	}
}

func TestPlayerLandingPose(t *testing.T) {
	now := time.Unix(100, 0)
	r := NewRun("test-player-landing-pose", testRun().Build, now)
	r.Status = "fighting"
	r.Enemies = nil

	// 1. Initial takeoff into jump
	r.Step(Input{Jump: true}, now.Add(50*time.Millisecond))
	if r.Player.Jump == 0 {
		t.Fatal("expected player to be jumping")
	}
	if r.Player.Pose != "jump" {
		t.Fatalf("expected player pose 'jump', got %q", r.Player.Pose)
	}

	// 2. Advance through jump until touchdown
	currentTime := now.Add(50 * time.Millisecond)
	for i := 0; i < 20 && r.Player.Jump > 0; i++ {
		currentTime = currentTime.Add(50 * time.Millisecond)
		r.Step(Input{}, currentTime)
	}
	if r.Player.Jump != 0 {
		t.Fatalf("expected player to land, got Jump=%v", r.Player.Jump)
	}

	// Immediately upon landing, player should be in "land" pose with active PoseTime
	if r.Player.Pose != "land" {
		t.Fatalf("expected player pose 'land' immediately upon landing, got %q", r.Player.Pose)
	}
	if r.Player.PoseTime <= 0 {
		t.Fatalf("expected positive PoseTime on landing, got %v", r.Player.PoseTime)
	}

	// 3. Advancing past PoseTime (0.14s) without directional input returns player to "idle"
	currentTime = currentTime.Add(150 * time.Millisecond)
	r.Step(Input{}, currentTime)
	if r.Player.Pose != "idle" {
		t.Fatalf("expected player pose 'idle' after landing recovery, got %q", r.Player.Pose)
	}

	// 4. Leap with directional movement held recovers into "run"
	r.SkillTimers["jump"] = 0
	currentTime = currentTime.Add(50 * time.Millisecond)
	r.Step(Input{Jump: true, X: 1}, currentTime)
	for i := 0; i < 20 && r.Player.Jump > 0; i++ {
		currentTime = currentTime.Add(50 * time.Millisecond)
		r.Step(Input{X: 1}, currentTime)
	}
	if r.Player.Pose != "land" {
		t.Fatalf("expected player pose 'land' upon moving leap touchdown, got %q", r.Player.Pose)
	}
	// Once recovery expires with X held, transitions to "run"
	currentTime = currentTime.Add(150 * time.Millisecond)
	r.Step(Input{X: 1}, currentTime)
	if r.Player.Pose != "run" {
		t.Fatalf("expected player pose 'run' when moving after landing recovery, got %q", r.Player.Pose)
	}

	// 5. Attacking immediately cancels landing recovery
	r.SkillTimers["jump"] = 0
	currentTime = currentTime.Add(50 * time.Millisecond)
	r.Step(Input{Jump: true}, currentTime)
	for i := 0; i < 20 && r.Player.Jump > 0; i++ {
		currentTime = currentTime.Add(50 * time.Millisecond)
		r.Step(Input{}, currentTime)
	}
	if r.Player.Pose != "land" {
		t.Fatalf("expected pose 'land', got %q", r.Player.Pose)
	}
	// Attack on next frame cancels landing
	currentTime = currentTime.Add(30 * time.Millisecond)
	r.Step(Input{Attack: true}, currentTime)
	if r.Player.Pose != "attack" {
		t.Fatalf("expected attack to cancel landing recovery, got %q", r.Player.Pose)
	}
}

func TestPlayerGuardedWalkingPose(t *testing.T) {
	now := time.Unix(100, 0)
	r := NewRun("test-guarded-walking-pose", testRun().Build, now)
	r.Status = "fighting"
	r.Enemies = nil

	// 1. Stationary guard
	r.Step(Input{Guard: true}, now.Add(50*time.Millisecond))
	if !r.Player.Guard {
		t.Fatal("expected player to be guarding")
	}
	if r.Player.Pose != "guard" {
		t.Fatalf("expected stationary guarded player to have pose 'guard', got %q", r.Player.Pose)
	}

	// 2. Moving while guarding: transitions to guard_walk
	r.Step(Input{Guard: true, X: 1}, now.Add(100*time.Millisecond))
	if !r.Player.Guard {
		t.Fatal("expected player to remain guarding")
	}
	if r.Player.Pose != "guard_walk" {
		t.Fatalf("expected moving guarded player to have pose 'guard_walk', got %q", r.Player.Pose)
	}

	// 3. Releasing movement while continuing guard returns to guard
	r.Step(Input{Guard: true}, now.Add(150*time.Millisecond))
	if r.Player.Pose != "guard" {
		t.Fatalf("expected stationary guarded player to return to pose 'guard', got %q", r.Player.Pose)
	}

	// 4. Releasing guard while moving transitions to run
	r.Step(Input{X: 1}, now.Add(200*time.Millisecond))
	if r.Player.Guard {
		t.Fatal("expected player to stop guarding")
	}
	if r.Player.Pose != "run" {
		t.Fatalf("expected moving player to transition to pose 'run', got %q", r.Player.Pose)
	}
}

func TestDirectionalHitRecoilOffsets(t *testing.T) {
	now := time.Unix(100, 0)

	// 1. Enemy directional hit recoil from player
	t.Run("enemy_hit_recoil_directions_and_resistance", func(t *testing.T) {
		r := NewRun("test-enemy-recoil", testRun().Build, now)
		r.Status = "fighting"
		r.Player.X = 500
		r.Player.Y = 300
		r.Player.Facing = 1

		// Normal goblin to right: +10 recoil
		r.Enemies = []Actor{{ID: "goblin-right", Kind: "goblin", X: 540, Y: 300, HP: 100, MaxHP: 100}}
		r.hurtEnemy(0, 10, "slash")
		if r.Enemies[0].RecoilX != 10.0 {
			t.Fatalf("expected goblin on right to have +10.0 recoil_x, got %f", r.Enemies[0].RecoilX)
		}

		// Boss to right: +4.0 recoil (heavy resistance)
		r.Enemies = []Actor{{ID: "boss-right", Kind: "boss", X: 540, Y: 300, HP: 500, MaxHP: 500}}
		r.hurtEnemy(0, 10, "slash")
		if r.Enemies[0].RecoilX != 4.0 {
			t.Fatalf("expected boss to have +4.0 recoil_x, got %f", r.Enemies[0].RecoilX)
		}

		// Knight to right: +6.5 recoil
		r.Enemies = []Actor{{ID: "knight-right", Kind: "knight", X: 540, Y: 300, HP: 200, MaxHP: 200}}
		r.hurtEnemy(0, 10, "slash")
		if r.Enemies[0].RecoilX != 6.5 {
			t.Fatalf("expected knight to have +6.5 recoil_x, got %f", r.Enemies[0].RecoilX)
		}

		// Enemy to left: hit while player faces left
		r.Player.Facing = -1
		r.Enemies = []Actor{{ID: "goblin-left", Kind: "goblin", X: 460, Y: 300, HP: 100, MaxHP: 100}}
		r.hurtEnemy(0, 10, "slash")
		if r.Enemies[0].RecoilX != -10.0 {
			t.Fatalf("expected goblin on left to have -10.0 recoil_x, got %f", r.Enemies[0].RecoilX)
		}

		// Enemy at same X: follows player facing
		r.Player.Facing = 1
		r.Enemies = []Actor{{ID: "goblin-center", Kind: "goblin", X: 500, Y: 300, HP: 100, MaxHP: 100}}
		r.hurtEnemy(0, 10, "slash")
		if r.Enemies[0].RecoilX != 10.0 {
			t.Fatalf("expected center goblin with facing 1 to have +10.0 recoil_x, got %f", r.Enemies[0].RecoilX)
		}
	})

	// 2. Player hit recoil and guard bracing
	t.Run("player_hit_recoil_and_guard_bracing", func(t *testing.T) {
		r := NewRun("test-player-recoil", testRun().Build, now)
		r.Status = "fighting"
		r.Player.X = 500
		r.Player.Y = 300
		r.Player.Facing = 1

		// Damage from right (x = 550): player pushed left (-9.0)
		r.hurtPlayer(20, 550, 300)
		if r.Player.RecoilX != -9.0 {
			t.Fatalf("expected unguard player hit from right to have -9.0 recoil_x, got %f", r.Player.RecoilX)
		}

		// Damage from left (x = 450): player pushed right (+9.0)
		r.hurtPlayer(20, 450, 300)
		if r.Player.RecoilX != 9.0 {
			t.Fatalf("expected unguard player hit from left to have +9.0 recoil_x, got %f", r.Player.RecoilX)
		}

		// Guarding player hit: braced (+3.0 or -3.0)
		r.Player.Guard = true
		r.hurtPlayer(20, 550, 300)
		if r.Player.RecoilX != -3.0 {
			t.Fatalf("expected guarded player hit from right to have braced -3.0 recoil_x, got %f", r.Player.RecoilX)
		}
	})

	// 3. Recoil decay over time and reset at pose_time == 0
	t.Run("recoil_decay_and_reset", func(t *testing.T) {
		r := NewRun("test-decay-recoil", testRun().Build, now)
		r.Status = "fighting"
		r.Player.X = 500
		r.Player.Y = 300
		r.Enemies = []Actor{{ID: "target", Kind: "goblin", X: 540, Y: 300, HP: 100, MaxHP: 100}}

		// Hit enemy
		r.Step(Input{Attack: true}, now.Add(50*time.Millisecond))
		if r.Enemies[0].RecoilX <= 0 {
			t.Fatalf("expected positive recoil, got %f", r.Enemies[0].RecoilX)
		}
		initialRecoil := r.Enemies[0].RecoilX

		// Advance 50ms: recoil should decay
		r.Step(Input{}, now.Add(100*time.Millisecond))
		if r.Enemies[0].RecoilX >= initialRecoil || r.Enemies[0].RecoilX <= 0 {
			t.Fatalf("expected decayed recoil between 0 and %f, got %f", initialRecoil, r.Enemies[0].RecoilX)
		}

		// Advance past pose_time (.2s total): recoil must reset to 0
		r.Step(Input{}, now.Add(350*time.Millisecond))
		if r.Enemies[0].RecoilX != 0 {
			t.Fatalf("expected recoil to reset to 0 after pose_time expires, got %f", r.Enemies[0].RecoilX)
		}

		// Defeating enemy resets recoil to 0
		r.Enemies[0].HP = 1
		r.Enemies[0].RecoilX = 10.0
		r.hurtEnemy(0, 50, "slash")
		if r.Enemies[0].RecoilX != 0 {
			t.Fatalf("expected defeated enemy recoil to reset to 0, got %f", r.Enemies[0].RecoilX)
		}
	})
}

func TestThirdStrikeImpactAccent(t *testing.T) {
	now := time.Unix(100, 0)

	hasEvent := func(events []Event, kind string) bool {
		for _, e := range events {
			if e.Kind == kind {
				return true
			}
		}
		return false
	}

	// 1. Strikes 1 and 2 do not emit third_strike event
	t.Run("strikes_1_and_2_no_third_strike_event", func(t *testing.T) {
		r := NewRun("test-combo-1-2", testRun().Build, now)
		r.Status = "fighting"
		r.Player.X = 500
		r.Player.Y = 300
		r.Player.Facing = 1
		r.Enemies = []Actor{{ID: "goblin", Kind: "goblin", X: 540, Y: 300, HP: 500, MaxHP: 500}}

		// Strike 1
		r.Step(Input{Attack: true}, now.Add(50*time.Millisecond))
		if r.Combo != 1 {
			t.Fatalf("expected combo 1, got %d", r.Combo)
		}
		if hasEvent(r.Events, "third_strike") {
			t.Fatal("strike 1 must not emit third_strike event")
		}

		// Strike 2
		r.Player.Cooldown = 0
		r.Events = nil
		r.Step(Input{Attack: true}, now.Add(450*time.Millisecond))
		if r.Combo != 2 {
			t.Fatalf("expected combo 2, got %d", r.Combo)
		}
		if hasEvent(r.Events, "third_strike") {
			t.Fatal("strike 2 must not emit third_strike event")
		}
	})

	// 2. Strike 3 hitting enemy emits third_strike impact event and knocks down enemy
	t.Run("strike_3_emits_third_strike_on_hit", func(t *testing.T) {
		r := NewRun("test-combo-3", testRun().Build, now)
		r.Status = "fighting"
		r.Combo = 2
		r.Player.X = 500
		r.Player.Y = 300
		r.Player.Facing = 1
		r.Enemies = []Actor{{ID: "goblin", Kind: "goblin", X: 540, Y: 300, HP: 500, MaxHP: 500}}

		r.Step(Input{Attack: true}, now.Add(50*time.Millisecond))
		if r.Combo != 3 {
			t.Fatalf("expected combo 3, got %d", r.Combo)
		}
		if !hasEvent(r.Events, "third_strike") {
			t.Fatal("strike 3 hitting enemy must emit third_strike event")
		}
		if r.Enemies[0].Knockdown <= 0 {
			t.Fatal("normal goblin must receive knockdown on third strike")
		}
	})

	// 3. Strike 3 on boss emits third_strike impact even though boss resists knockdown
	t.Run("strike_3_boss_emits_third_strike_resists_knockdown", func(t *testing.T) {
		r := NewRun("test-combo-3-boss", testRun().Build, now)
		r.Status = "fighting"
		r.Combo = 2
		r.Player.X = 500
		r.Player.Y = 300
		r.Player.Facing = 1
		r.Enemies = []Actor{{ID: "boss", Kind: "boss", X: 540, Y: 300, HP: 1000, MaxHP: 1000}}

		r.Step(Input{Attack: true}, now.Add(50*time.Millisecond))
		if !hasEvent(r.Events, "third_strike") {
			t.Fatal("strike 3 hitting boss must emit third_strike event")
		}
		if r.Enemies[0].Knockdown != 0 {
			t.Fatal("boss must resist knockdown on third strike")
		}
	})

	// 4. Strike 3 swing that misses all enemies does not emit third_strike impact
	t.Run("strike_3_whiff_no_third_strike_impact", func(t *testing.T) {
		r := NewRun("test-combo-3-whiff", testRun().Build, now)
		r.Status = "fighting"
		r.Combo = 2
		r.Player.X = 500
		r.Player.Y = 300
		r.Player.Facing = 1
		r.Enemies = []Actor{{ID: "goblin-far", Kind: "goblin", X: 700, Y: 300, HP: 500, MaxHP: 500}}

		r.Step(Input{Attack: true}, now.Add(50*time.Millisecond))
		if r.Combo != 3 {
			t.Fatalf("expected combo 3, got %d", r.Combo)
		}
		if hasEvent(r.Events, "third_strike") {
			t.Fatal("whiffed strike 3 must not emit third_strike impact event")
		}
	})
}

func TestUltimateAnticipationPose(t *testing.T) {
	now := time.Unix(100, 0)

	findEvent := func(events []Event, kind string) *Event {
		for i := range events {
			if events[i].Kind == kind {
				return &events[i]
			}
		}
		return nil
	}

	build := testRun().Build
	build.Ultimate = &Skill{ID: "cataclysm", Name: "Cataclysm", Kind: "ultimate", Power: 4, Cost: 0, Cooldown: 12}

	// 1. Regular skill cast sets pose "cast" with .4s, not ultimate_anticipation
	t.Run("regular_skill_uses_cast_pose", func(t *testing.T) {
		r := NewRun("test-regular-cast", build, now)
		r.cast("fire")
		if r.Player.Pose != "cast" {
			t.Fatalf("expected regular skill to set pose 'cast', got %q", r.Player.Pose)
		}
		if r.Player.PoseTime != 0.4 {
			t.Fatalf("expected pose_time 0.4, got %f", r.Player.PoseTime)
		}
		if findEvent(r.Events, "ultimate_anticipation") != nil {
			t.Fatal("regular skill must not emit ultimate_anticipation event")
		}
	})

	// 2. Ultimate skill cast sets pose "ultimate_anticipation" with 0.55s and emits event
	t.Run("ultimate_sets_anticipation_pose_and_event", func(t *testing.T) {
		r := NewRun("test-ultimate-cast", build, now)
		r.Status = "fighting"
		r.Player.X = 450
		r.Player.Y = 320
		r.cast("cataclysm")
		if r.Player.Pose != "ultimate_anticipation" {
			t.Fatalf("expected ultimate to set pose 'ultimate_anticipation', got %q", r.Player.Pose)
		}
		if r.Player.PoseTime != 0.55 {
			t.Fatalf("expected pose_time 0.55 for ultimate anticipation, got %f", r.Player.PoseTime)
		}
		ev := findEvent(r.Events, "ultimate_anticipation")
		if ev == nil {
			t.Fatal("ultimate cast must emit ultimate_anticipation event")
		}
		if ev.X != r.Player.X || ev.Y != r.Player.Y-35 {
			t.Fatalf("expected event coordinates (%f, %f), got (%f, %f)", r.Player.X, r.Player.Y-35, ev.X, ev.Y)
		}

		// After 0.55s, transitions to recovery pose
		cur := now
		for i := 0; i < 6; i++ {
			cur = cur.Add(100 * time.Millisecond)
			r.Step(Input{}, cur)
		}
		if r.Player.Pose != "recovery" {
			t.Fatalf("expected pose to transition to 'recovery' after ultimate anticipation expires, got %q", r.Player.Pose)
		}

		// Smooth recovery to idle after recovery duration (0.22s) expires
		for i := 0; i < 3; i++ {
			cur = cur.Add(100 * time.Millisecond)
			r.Step(Input{}, cur)
		}
		if r.Player.Pose != "idle" {
			t.Fatalf("expected pose to recover to 'idle' after recovery expires, got %q", r.Player.Pose)
		}
	})
}

func TestHeavyAbilityRecoveryPose(t *testing.T) {
	now := time.Unix(100, 0)

	findEvent := func(events []Event, kind string) *Event {
		for i := range events {
			if events[i].Kind == kind {
				return &events[i]
			}
		}
		return nil
	}

	build := testRun().Build
	build.Signatures = []Skill{
		{ID: "build", Role: "builder", Kind: "shield", Cost: 0, Cooldown: 0},
		{ID: "finish", Role: "finisher", Kind: "slash", Cost: 0, Cooldown: 0, Power: 2},
	}
	build.Skills = []Skill{
		{ID: "fire", Name: "Fireball", Kind: "fire", Power: 1.5, Cost: 0, Cooldown: 2},
		{ID: "tremor", Name: "Earth Tremor", Kind: "quake", Power: 2.5, Cost: 0, Cooldown: 4},
	}
	build.Ultimate = &Skill{ID: "cataclysm", Name: "Cataclysm", Kind: "ultimate", Power: 4, Cost: 0, Cooldown: 12}

	// 1. Regular skill cast (fireball) skips recovery pose
	t.Run("regular_skill_skips_recovery", func(t *testing.T) {
		r := NewRun("test-reg-skill", build, now)
		r.Status = "fighting"
		r.cast("fire")
		if r.Player.Pose != "cast" {
			t.Fatalf("expected initial pose 'cast', got %q", r.Player.Pose)
		}
		cur := now
		for i := 0; i < 5; i++ {
			cur = cur.Add(100 * time.Millisecond)
			r.Step(Input{}, cur)
		}
		if r.Player.Pose != "idle" {
			t.Fatalf("expected regular skill to recover directly to 'idle', got %q", r.Player.Pose)
		}
		if findEvent(r.Events, "heavy_recovery") != nil {
			t.Fatal("regular skill must not emit heavy_recovery event")
		}
	})

	// 2. Uncharged finisher (0 charges) skips recovery pose
	t.Run("uncharged_finisher_skips_recovery", func(t *testing.T) {
		r := NewRun("test-empty-finish", build, now)
		r.Status = "fighting"
		r.Resource = 0
		r.cast("finish")
		cur := now
		for i := 0; i < 5; i++ {
			cur = cur.Add(100 * time.Millisecond)
			r.Step(Input{}, cur)
		}
		if r.Player.Pose != "idle" {
			t.Fatalf("expected uncharged finisher to recover directly to 'idle', got %q", r.Player.Pose)
		}
		if findEvent(r.Events, "heavy_recovery") != nil {
			t.Fatal("uncharged finisher must not emit heavy_recovery event")
		}
	})

	// 3. Charged finisher (2 charges) enters recovery pose and emits heavy_recovery event
	t.Run("charged_finisher_enters_recovery", func(t *testing.T) {
		r := NewRun("test-charged-finish", build, now)
		r.Status = "fighting"
		r.Resource = 2
		r.cast("finish")
		if r.Player.Pose != "cast" {
			t.Fatalf("expected initial pose 'cast', got %q", r.Player.Pose)
		}
		cur := now
		// Step 4 times (400ms = cast duration)
		for i := 0; i < 4; i++ {
			cur = cur.Add(100 * time.Millisecond)
			r.Step(Input{}, cur)
		}
		if r.Player.Pose != "recovery" {
			t.Fatalf("expected pose to transition to 'recovery' after charged finisher cast, got %q", r.Player.Pose)
		}
		if r.Player.PoseTime <= 0 || r.Player.PoseTime > 0.25 {
			t.Fatalf("expected pose_time ~0.22, got %f", r.Player.PoseTime)
		}
		ev := findEvent(r.Events, "heavy_recovery")
		if ev == nil {
			t.Fatal("charged finisher must emit heavy_recovery event")
		}

		// While in recovery, cannot attack or jump
		r.Step(Input{Attack: true}, cur.Add(20*time.Millisecond))
		if r.Player.Pose != "recovery" {
			t.Fatalf("attack must not override active recovery pose, got %q", r.Player.Pose)
		}

		// Step until recovery duration expires (300ms more)
		for i := 0; i < 3; i++ {
			cur = cur.Add(100 * time.Millisecond)
			r.Step(Input{}, cur)
		}
		if r.Player.Pose != "idle" {
			t.Fatalf("expected pose to recover to 'idle' after recovery expires, got %q", r.Player.Pose)
		}
	})

	// 4. Heavy quake ability enters recovery pose
	t.Run("quake_enters_recovery", func(t *testing.T) {
		r := NewRun("test-quake", build, now)
		r.Status = "fighting"
		r.cast("tremor")
		cur := now
		for i := 0; i < 4; i++ {
			cur = cur.Add(100 * time.Millisecond)
			r.Step(Input{}, cur)
		}
		if r.Player.Pose != "recovery" {
			t.Fatalf("expected quake to transition to 'recovery', got %q", r.Player.Pose)
		}
		for i := 0; i < 3; i++ {
			cur = cur.Add(100 * time.Millisecond)
			r.Step(Input{}, cur)
		}
		if r.Player.Pose != "idle" {
			t.Fatalf("expected quake recovery to end in 'idle', got %q", r.Player.Pose)
		}
	})

	// 5. Movement during recovery stays in recovery until pose_time expires, then transitions to run
	t.Run("moving_during_recovery_transitions_to_run", func(t *testing.T) {
		r := NewRun("test-recovery-move", build, now)
		r.Status = "fighting"
		r.Resource = 3
		r.cast("finish")
		cur := now
		// Reach recovery
		for i := 0; i < 4; i++ {
			cur = cur.Add(100 * time.Millisecond)
			r.Step(Input{}, cur)
		}
		if r.Player.Pose != "recovery" {
			t.Fatalf("expected recovery pose, got %q", r.Player.Pose)
		}
		// Hold movement key during recovery
		cur = cur.Add(50 * time.Millisecond)
		r.Step(Input{X: 1}, cur)
		if r.Player.Pose != "recovery" {
			t.Fatalf("holding movement should not immediately interrupt recovery pose, got %q", r.Player.Pose)
		}
		// Step past recovery duration with movement held
		for i := 0; i < 3; i++ {
			cur = cur.Add(100 * time.Millisecond)
			r.Step(Input{X: 1}, cur)
		}
		if r.Player.Pose != "run" {
			t.Fatalf("expected transition to 'run' when moving after recovery expires, got %q", r.Player.Pose)
		}
	})

	// 6. Taking damage interrupts recovery
	t.Run("taking_damage_interrupts_recovery", func(t *testing.T) {
		r := NewRun("test-recovery-damage", build, now)
		r.Status = "fighting"
		r.Resource = 2
		r.cast("finish")
		cur := now
		for i := 0; i < 4; i++ {
			cur = cur.Add(100 * time.Millisecond)
			r.Step(Input{}, cur)
		}
		if r.Player.Pose != "recovery" {
			t.Fatalf("expected recovery pose, got %q", r.Player.Pose)
		}
		// Enemy hits player
		r.hurtPlayer(15, r.Player.X+50, r.Player.Y)
		if r.Player.Pose != "hit" {
			t.Fatalf("taking damage must interrupt recovery with 'hit' pose, got %q", r.Player.Pose)
		}
	})
}
