package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestDropProvenanceSurvivesSaving(t *testing.T) {
	r := testRun()
	r.Level = &Level{ID: 43}
	r.Room = 2
	r.Enemies = []Actor{{ID: "boss", Kind: "boss", HP: 20, MaxHP: 20}}
	r.hurtEnemy(0, 1000, "hit")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if len(restored.Drops) != 1 || restored.Drops[0].Mission != 43 || restored.Drops[0].Tier != 3 {
		t.Fatalf("drop lost its discovery location: %+v", restored.Drops)
	}
}

func TestSkillUsageCountsOnlySuccessfulOwnedCasts(t *testing.T) {
	r := testRun()
	r.Build.Skills = []Skill{{ID: "one", Name: "One", Kind: "shield", Cost: 10, Cooldown: 2}, {ID: "two", Name: "Two", Kind: "shield", Cost: 12.5, Cooldown: 2}}
	r.Player.Mana = 100
	r.cast("missing")
	r.cast("one")
	r.cast("one")
	r.Player.Cooldown = 0
	r.Player.Mana = 0
	r.cast("two")
	if r.Stats.SkillsCast != 1 || r.Stats.SkillUses["one"] != 1 || len(r.Stats.SkillUses) != 1 {
		t.Fatalf("rejected cast counted: %+v", r.Stats)
	}
	if r.Stats.SkillMana["one"] != 10 || len(r.Stats.SkillMana) != 1 || r.Stats.ManaSpent != 10 {
		t.Fatalf("rejected cast spent mana: %+v", r.Stats)
	}
	r.Player.Mana = 100
	r.cast("two")
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Stats.SkillUses["one"] != 1 || restored.Stats.SkillUses["two"] != 1 || restored.Stats.SkillsCast != 2 {
		t.Fatalf("usage lost: %+v", restored.Stats)
	}
	if restored.Stats.SkillMana["one"] != 10 || restored.Stats.SkillMana["two"] != 12.5 || restored.Stats.ManaSpent != 22.5 {
		t.Fatalf("mana accounting lost: %+v", restored.Stats)
	}
}

func TestCombatStatsCountEffectiveDamageAndKillsOnce(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{ID: "boss", Kind: "boss", HP: 20, MaxHP: 20}}
	r.hurtEnemy(0, 1000, "hit")
	r.hurtEnemy(0, 1000, "hit")
	if r.Stats.DamageDealt != 20 || r.Stats.Kills != 1 || r.Stats.Bosses != 1 || r.Stats.LargestHit != 20 {
		t.Fatalf("overkill or repeated death inflated stats: %+v", r.Stats)
	}
}

func TestSkillHitsCountDamagedTargetsAndSurviveProjectileSave(t *testing.T) {
	r := testRun()
	r.Build.Skills = []Skill{{ID: "sweep", Kind: "slash", Power: 2}}
	r.Enemies = []Actor{
		{ID: "near", HP: 10, MaxHP: 10, X: r.Player.X + 50, Y: r.Player.Y},
		{ID: "near2", HP: 100, MaxHP: 100, X: r.Player.X + 70, Y: r.Player.Y},
		{ID: "dead", HP: 0, MaxHP: 10, X: r.Player.X + 60, Y: r.Player.Y},
		{ID: "far", HP: 100, MaxHP: 100, X: r.Player.X + 500, Y: r.Player.Y},
	}
	r.cast("sweep")
	r.skillHit(0, 100, r.Build.Skills[0], 0, "")
	r.skillHit(1, 0, r.Build.Skills[0], 0, "")
	if r.Stats.SkillHits["sweep"] != 2 {
		t.Fatalf("hits must count damaged targets, excluding dead targets and zero damage: %+v", r.Stats)
	}
	r = testRun()
	r.Enemies = []Actor{{ID: "target", HP: 100, MaxHP: 100, X: r.Player.X + 60, Y: r.Player.Y, Cooldown: 10}}
	r.cast("fire")
	if len(r.Stats.SkillHits) != 0 {
		t.Fatal("projectile launch counted as a hit")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	restored.tick(Input{}, 1.0/30)
	if restored.Stats.SkillHits["fire"] != 1 {
		t.Fatalf("restored projectile lost attribution: %+v", restored.Stats)
	}
}

func TestSkillHealingCountsActualClassAndDirectRecovery(t *testing.T) {
	for _, class := range []string{"bloodblade", "alchemist"} {
		t.Run(class, func(t *testing.T) {
			r := testRun()
			r.Build.Class = class
			r.Resource = 3
			r.Build.Skills = []Skill{{ID: "restore", Kind: "heal", Role: "finisher", Heal: .15}}
			r.Player.HP = r.Player.MaxHP - 40
			r.cast("restore")
			if r.Stats.SkillHealing["restore"] != 40 || r.Stats.Healing != 40 {
				t.Fatalf("class recovery or overheal accounting incorrect: %+v", r.Stats)
			}
			r.Player.Cooldown = 0
			r.cast("restore")
			if r.Stats.SkillHealing["restore"] != 40 {
				t.Fatal("overheal counted")
			}
			r.Player.HP -= 5
			r.healPlayer(5)
			if r.Stats.SkillHealing["restore"] != 40 || r.Stats.Healing != 45 {
				t.Fatal("unattributed healing assigned to previous skill")
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var restored Run
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Fatal(err)
			}
			if restored.Stats.SkillHealing["restore"] != 40 {
				t.Fatal("saved healing lost")
			}
		})
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
