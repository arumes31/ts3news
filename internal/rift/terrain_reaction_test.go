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
