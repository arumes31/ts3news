package rift

import (
	"encoding/json"
	"math"
	"testing"
)

func TestMonsterDamageRecordsActualLossByOwner(t *testing.T) {
	for _, mode := range []string{"normal", "armor", "guard", "barrier", "lethal", "grace"} {
		t.Run(mode, func(t *testing.T) {
			r := circleTestRun()
			r.MonsterRecords = nil
			r.Build.Armor = 0
			r.Enemies = []Actor{{ID: "a", Name: "Rat", ArtKey: "monster:Rat"}, {ID: "b", Name: "King", ArtKey: "monster:King", Kind: "boss"}}
			switch mode {
			case "armor":
				r.Build.Armor = 20
			case "guard":
				r.Player.Guard = true
			case "barrier":
				r.Barrier = 100
			case "lethal":
				r.Player.HP = 3
			case "grace":
				r.SkillTimers["connection_grace"] = 1
			}
			before := r.Player.HP
			r.hurtPlayerFromEnemy(30, r.Player.X, r.Player.Y, "a")
			loss := before - r.Player.HP
			if math.Abs(r.MonsterRecords["monster:Rat"].DamageTaken-loss) > 1e-8 {
				t.Fatal("record includes blocked or excess damage")
			}
			if r.MonsterRecords["monster:King"].DamageTaken != 0 {
				t.Fatal("wrong monster credited")
			}
			r.hurtPlayerFromHazard(10, r.Player.X, r.Player.Y)
			r.hurtPlayerFromEnemy(10, r.Player.X, r.Player.Y, "missing")
			if math.Abs(r.MonsterRecords["monster:Rat"].DamageTaken-loss) > 1e-8 {
				t.Fatal("unknown or hazard damage attributed to monster")
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err = json.Unmarshal(data, &saved); err != nil {
				t.Fatal(err)
			}
			next := circleTestRun()
			next.inheritMonsterRecords(&saved)
			if math.Abs(next.MonsterRecords["monster:Rat"].DamageTaken-loss) > 1e-8 {
				t.Fatal("damage lost after save and inheritance")
			}
			next.Enemies = r.Enemies
			next.Build.Armor = 0
			next.hurtPlayerFromEnemy(5, next.Player.X, next.Player.Y, "a")
			if math.Abs(next.MonsterRecords["monster:Rat"].DamageTaken-loss-5) > 1e-8 || math.Abs(saved.MonsterRecords["monster:Rat"].DamageTaken-loss) > 1e-8 {
				t.Fatal("damage inheritance aliases records or fails to accumulate")
			}
		})
	}
}

func TestMonsterDamageExcludesPracticeAndInvalidIdentities(t *testing.T) {
	for _, mode := range []string{"practice", "invalid", "legacy"} {
		r := circleTestRun()
		r.MonsterRecords = nil
		r.Build.Armor = 0
		r.Enemies = []Actor{{ID: "a", Name: "Rat", ArtKey: "monster:Rat"}}
		if mode == "practice" {
			r.Practice = &PracticeState{Mode: "boss"}
		}
		if mode == "invalid" {
			r.Enemies[0].ArtKey = "prop:Rat"
		}
		if mode == "legacy" {
			r.Level = nil
		}
		r.hurtPlayerFromEnemy(10, r.Player.X, r.Player.Y, "a")
		if len(r.MonsterRecords) != 0 {
			t.Fatalf("%s created a damage record", mode)
		}
	}
}

func TestMonsterDamageTracksProjectileAfterOwnerDeath(t *testing.T) {
	r := testRun()
	r.Level = &Level{ID: 1}
	r.Build.Armor = 0
	r.Player.X, r.Player.Y = 500, 410
	r.Enemies = []Actor{{ID: "archer", Name: "Archer", ArtKey: "monster:Archer", Kind: "archer", HP: 0}, {ID: "survivor", HP: 100, X: 900, Y: 410, Cooldown: 5}}
	r.Projectiles = []Projectile{{OwnerID: "archer", Enemy: true, X: 500, Y: 410, Life: 1, Power: 10}}
	r.tick(Input{}, .02)
	if r.MonsterRecords["monster:Archer"].DamageTaken != 10 {
		t.Fatal("projectile lost dead owner's canonical identity")
	}
}
