package rift

import (
	"encoding/json"
	"math"
	"testing"
)

func reactionTestRun() *Run {
	r := terrainTestRun()
	r.SkillTimers = map[string]float64{}
	r.FirstHitGrace = false
	r.Build.Armor = 0
	r.Player.X, r.Player.Y, r.Player.HP = 100, 320, 100
	r.Level.Rooms[0].Cover = []TerrainCover{
		{Obstacle: Obstacle{480, 395, 20, 30}, ID: "a", Material: "wood", HP: 60, MaxHP: 60, Volatile: true},
		{Obstacle: Obstacle{560, 395, 20, 30}, ID: "b", Material: "wood", HP: 60, MaxHP: 60, Volatile: true},
		{Obstacle: Obstacle{640, 395, 20, 30}, ID: "c", Material: "wood", HP: 60, MaxHP: 60, Volatile: true},
	}
	r.Enemies = []Actor{{ID: "target", Kind: "goblin", X: 570, Y: 410, HP: 100, MaxHP: 100, Armor: .2}}
	return r
}

func TestTerrainReactionWarnsWholeClusterAndHitsOnceAcrossSave(t *testing.T) {
	r := reactionTestRun()
	r.damageTerrainCover(0, 60)
	for _, c := range r.Level.Rooms[0].Cover {
		if c.HP != 0 || c.BlastFuse != 1.2 {
			t.Fatalf("cluster did not arm: %+v", c)
		}
	}
	r.terrainReactionTick(.6)
	if r.Enemies[0].HP != 100 {
		t.Fatal("warning damaged enemy")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	saved.terrainReactionTick(.6)
	// The overlapping cluster inflicts one region-scaled dose, with enemy armor.
	if math.Abs(saved.Enemies[0].HP-(100-saved.hazardContactDamage(saved.Level.Region)*.8)) > 1e-8 {
		t.Fatalf("stacked or missing damage: %v", saved.Enemies[0].HP)
	}
	hp := saved.Enemies[0].HP
	saved.terrainReactionTick(2)
	if saved.Enemies[0].HP != hp {
		t.Fatal("spent cluster fired again")
	}
	if saved.Stats.DamageDealt != 0 || saved.AttackChain != 0 {
		t.Fatal("terrain inflated player attacks")
	}
}

func TestTerrainReactionDoesNotCrossStoneOrArmOrdinaryWood(t *testing.T) {
	r := reactionTestRun()
	r.Level.Rooms[0].HighCover = []Obstacle{{530, 315, 10, 175}}
	r.damageTerrainCover(0, 60)
	if r.Level.Rooms[0].Cover[1].HP != 60 {
		t.Fatal("chain crossed wall")
	}
	r.terrainReactionTick(1.2)
	if r.Enemies[0].HP != 100 {
		t.Fatal("blast crossed wall")
	}
	r = reactionTestRun()
	r.Level.Rooms[0].Cover[1].Volatile = false
	r.damageTerrainCover(0, 60)
	if r.Level.Rooms[0].Cover[1].HP != 60 || r.Level.Rooms[0].Cover[2].HP != 60 {
		t.Fatal("ordinary wood joined chain")
	}
}

func TestTerrainReactionBoundedAndInvalidDamageIgnored(t *testing.T) {
	r := reactionTestRun()
	c := r.Level.Rooms[0].Cover[0]
	for i := 0; i < 12; i++ {
		c.ID += "x"
		r.Level.Rooms[0].Cover = append(r.Level.Rooms[0].Cover, c)
	}
	for _, damage := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		r.damageTerrainCover(0, damage)
	}
	if r.Level.Rooms[0].Cover[0].HP != 60 {
		t.Fatal("invalid damage armed cluster")
	}
	r.damageTerrainCover(0, 60)
	armed := 0
	for _, c := range r.Level.Rooms[0].Cover {
		if c.BlastFuse > 0 {
			armed++
		}
	}
	if armed != 8 {
		t.Fatalf("armed %d props; want hard cap 8", armed)
	}
}

func TestTerrainReactionPlayerEscapeAndImmunity(t *testing.T) {
	for _, mode := range []string{"contact", "jump", "dodge", "connection", "outside"} {
		t.Run(mode, func(t *testing.T) {
			r := reactionTestRun()
			r.Player.X, r.Player.Y = 570, 410
			switch mode {
			case "jump":
				r.Player.Jump = .2
			case "dodge":
				r.SkillTimers["dodge_invulnerability"] = 1
			case "connection":
				r.SkillTimers["connection_grace"] = 1
			case "outside":
				r.Player.Y = 320
			}
			r.damageTerrainCover(0, 60)
			r.terrainReactionTick(1.2)
			if (r.Player.HP < 100) != (mode == "contact") {
				t.Fatalf("wrong contact eligibility: %v", r.Player.HP)
			}
		})
	}
}

func TestTerrainReactionSkipsPropsAndBurrowedEnemies(t *testing.T) {
	for _, mode := range []string{"totem", "burrowed", "airborne"} {
		r := reactionTestRun()
		switch mode {
		case "totem":
			r.Enemies[0].Kind = "totem"
		case "burrowed":
			r.Enemies[0].Burrowed = true
		case "airborne":
			r.Enemies[0].Jump = .2
		}
		r.damageTerrainCover(0, 60)
		r.terrainReactionTick(1.2)
		if r.Enemies[0].HP != 100 {
			t.Fatalf("hit protected enemy %s", mode)
		}
	}
}

func TestTerrainReactionFreezeAndLethalAttribution(t *testing.T) {
	r := reactionTestRun()
	r.damageTerrainCover(0, 60)
	for _, mode := range []string{"paused", "cleared", "invalid"} {
		r.Paused, r.Status = mode == "paused", "fighting"
		if mode == "cleared" {
			r.Status = "cleared"
		}
		dt := 2.0
		if mode == "invalid" {
			dt = math.NaN()
		}
		r.terrainReactionTick(dt)
		if r.Level.Rooms[0].Cover[0].BlastFuse != 1.2 {
			t.Fatal("inactive or invalid tick advanced fuse")
		}
	}
	r.Paused, r.Status = false, "fighting"
	r.Player.X, r.Player.Y, r.Player.HP = 570, 410, 1
	r.Enemies[0].HP = 1
	r.terrainReactionTick(1.2)
	r.terrainReactionTick(1.2)
	if r.Player.HP != 0 || r.DefeatedByHazard == nil || r.DefeatedByHazard.Kind != "volatile cover" || !r.DefeatedByHazard.Jumpable {
		t.Fatal("missing lethal hazard attribution")
	}
	if r.Stats.Kills != 1 || len(r.Drops) != 1 {
		t.Fatal("blast kill did not drop exactly once")
	}
	if r.Stats.HazardDamageTaken != 1 || r.Stats.EnemyDamageTaken != 0 {
		t.Fatal("wrong player damage accounting")
	}
}

func TestTerrainReactionDamageHasAbsoluteRegionalCap(t *testing.T) {
	r := reactionTestRun()
	r.Level.Region = 100000
	r.Enemies[0].Armor = 0
	r.damageTerrainCover(0, 60)
	r.terrainReactionTick(1.2)
	if r.Enemies[0].HP != 79 {
		t.Fatal("malformed region exceeded 21 raw damage")
	}
}

func TestTerrainReactionCombatTickPreservesFullWarning(t *testing.T) {
	r := terrainTestRun()
	r.Level.Rooms[0].Cover[0].Volatile = true
	r.Build.Damage = 60
	r.Enemies[0].X, r.Enemies[0].Speed, r.Enemies[0].Cooldown = 1400, 0, 100
	r.tick(Input{Attack: true}, .02)
	if r.Level.Rooms[0].Cover[0].BlastFuse != 1.2 {
		t.Fatal("trigger tick consumed warning time")
	}
	hp := r.Player.HP
	for i := 0; i < 59; i++ {
		r.tick(Input{}, .02)
	}
	if r.Player.HP != hp || r.Level.Rooms[0].Cover[0].BlastFuse <= 0 {
		t.Fatal("blast fired before full warning")
	}
	r.tick(Input{Jump: true}, .02)
	if r.Level.Rooms[0].Cover[0].BlastFuse != 0 {
		t.Fatal("combat tick did not detonate")
	}
	if r.Player.HP != hp {
		t.Fatal("jump on impact tick failed to evade")
	}
}

func TestTerrainReactionLethalTickCannotAttackAfterDeath(t *testing.T) {
	r := reactionTestRun()
	r.Player.X, r.Player.Y, r.Player.HP = 570, 410, 1
	r.Enemies[0].X, r.Enemies[0].Speed, r.Enemies[0].Cooldown = 1400, 0, 100
	r.damageTerrainCover(0, 60)
	for i := range r.Level.Rooms[0].Cover {
		r.Level.Rooms[0].Cover[i].BlastFuse = .02
	}
	r.tick(Input{Attack: true}, .02)
	if r.Player.HP != 0 || r.Status != "defeated" || r.Stats.Attacks != 0 {
		t.Fatal("lethal blast permitted posthumous attack or failed defeat")
	}
}

func TestTerrainReactionProjectileKeepsNewFuseWhole(t *testing.T) {
	r := terrainTestRun()
	r.Level.Rooms[0].Cover[0].Volatile = true
	r.Enemies[0].X, r.Enemies[0].Speed, r.Enemies[0].Cooldown = 1400, 0, 100
	r.Projectiles = []Projectile{{X: 180, Y: 410, VX: 1000, Life: 2, Power: 100}}
	r.tick(Input{}, .02)
	if r.Level.Rooms[0].Cover[0].BlastFuse != 1.2 || len(r.Projectiles) != 0 {
		t.Fatal("projectile did not arm one full warning and stop")
	}
}

func TestCampaignVolatileClustersHaveClearApproachesAndEscapeLanes(t *testing.T) {
	rooms := 0
	for _, level := range Campaign() {
		for roomIndex, arena := range level.Rooms {
			var indices []int
			for i, c := range arena.Cover {
				if c.Volatile {
					indices = append(indices, i)
				}
			}
			if len(indices) == 0 {
				continue
			}
			rooms++
			if len(indices) != 3 || roomIndex != 0 || level.ID == 1 {
				t.Fatalf("unexpected cluster in mission %d", level.ID)
			}
			for _, index := range indices {
				c := arena.Cover[index]
				from := Actor{X: c.X + c.W/2, Y: c.Y + c.H/2}
				for _, entry := range []*ArenaEntrance{arena.Entrance, arena.Exit} {
					if entry == nil || blastContains(&from, &Actor{X: entry.X, Y: entry.Y}) {
						t.Fatalf("mission %d blast covers entry/exit", level.ID)
					}
				}
				r := reactionTestRun()
				copyLevel := cloneCampaignLevel(level)
				r.Level, r.Room = &copyLevel, roomIndex
				r.damageTerrainCover(index, c.HP)
				for _, linked := range indices {
					if r.Level.Rooms[roomIndex].Cover[linked].BlastFuse != 1.2 {
						t.Fatalf("mission %d cluster disconnected", level.ID)
					}
				}
				// After ignition, even the slowed walking speed can leave the central
				// lane within the warning, allowing 300ms to react before moving.
				for _, direction := range []float64{-1, 1} {
					actor := Actor{X: from.X, Y: from.Y, HP: 100, ID: "player"}
					for step := 0; step < 45; step++ {
						r.moveActor(&actor, 0, direction*141*.6*.02, false)
					}
					for _, linked := range indices {
						other := arena.Cover[linked]
						origin := Actor{X: other.X + other.W/2, Y: other.Y + other.H/2}
						if blastContains(&origin, &actor) {
							t.Fatalf("mission %d blocked escape lane", level.ID)
						}
					}
				}
			}
		}
	}
	if rooms != 9 {
		t.Fatalf("got %d volatile rooms; want nine", rooms)
	}
}
