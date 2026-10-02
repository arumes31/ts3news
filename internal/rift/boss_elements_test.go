package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
)

func TestBossElementPhasesUseCanonicalMatchupsAndSurviveSave(t *testing.T) {
	for _, base := range []content.Element{content.ElementFire, content.ElementAir, content.ElementEarth, content.ElementWater} {
		boss := AdaptMonster(content.Mob{Name: "Ward test", Type: content.MobBoss, Element: base})
		if len(boss.ElementalPhases) != 3 || boss.ElementalPhases[0].Element != base {
			t.Fatalf("missing phases for %s", base)
		}
		data, err := json.Marshal(boss)
		if err != nil {
			t.Fatal(err)
		}
		var saved Actor
		if err = json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		for phase := 1; phase <= 3; phase++ {
			saved.Phase = phase
			ward := saved.elementalPhase()
			if ward == nil || ward.Weakness != content.ElementWeakness(ward.Element) {
				t.Fatal("hint differs from canonical rule")
			}
			for _, attack := range []content.Element{content.ElementFire, content.ElementAir, content.ElementEarth, content.ElementWater, content.ElementPhysical, ""} {
				r := testRun()
				saved.HP, saved.MaxHP, saved.Armor = 10000, 10000, 0
				r.Enemies = []Actor{saved}
				r.hurtEnemyElement(0, 100, "hit", 0, attack)
				want := 10000 - 100*content.ElementMultiplier(attack, ward.Element)
				if r.Enemies[0].HP != want {
					t.Fatalf("%s phase %d attack %s: got %v want %v", base, phase, attack, r.Enemies[0].HP, want)
				}
			}
		}
	}
}

func TestBossElementPhasesKeepLegacyAndEnvironmentalDamageNeutral(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		boss := AdaptMonster(content.Mob{Name: "Ward test", Type: content.MobBoss, Element: content.ElementFire})
		boss.HP, boss.MaxHP, boss.Armor = 10000, 10000, 0
		if legacy {
			boss.ElementalPhases = [3]BossElementPhase{}
		}
		r := testRun()
		r.Enemies = []Actor{boss}
		if legacy {
			r.hurtEnemyElement(0, 100, "hit", 0, content.ElementWater)
		} else {
			r.applyEnemyDamage(0, 100, "hit", 0, true)
		}
		if r.Enemies[0].HP != 9900 {
			t.Fatal("legacy or environment received elemental multiplier")
		}
	}
	for _, element := range []content.Element{content.ElementPhysical, "", "unknown"} {
		boss := AdaptMonster(content.Mob{Name: "Neutral", Type: content.MobBoss, Element: element})
		if boss.ElementalPhases != [3]BossElementPhase{} {
			t.Fatal("invented elemental weakness")
		}
	}
}

func elementalAttackRun(base content.Element) *Run {
	r := testRun()
	boss := AdaptMonster(content.Mob{Name: "Ward test", Type: content.MobBoss, Element: base})
	boss.ID = "ward-boss"
	boss.X, boss.Y = r.Player.X+45, r.Player.Y
	boss.HP, boss.MaxHP, boss.Armor, boss.Speed, boss.Cooldown = 10000, 10000, 0, 0, 100
	r.Enemies = []Actor{boss}
	r.Player.Cooldown = 0
	return r
}

func TestBossElementPhasesDirectAttackPaths(t *testing.T) {
	for _, path := range []string{"weapon", "skill", "projectile"} {
		damage := func(element content.Element) float64 {
			r := elementalAttackRun(content.ElementAir)
			skill := Skill{ID: "ward-strike", Name: "Ice-looking fire", Kind: "ice", Element: element}
			switch path {
			case "weapon":
				r.Build.WeaponElement = element
				r.tick(Input{Attack: true}, .02)
			case "skill":
				r.skillHit(0, 100, skill, 0, "")
			case "projectile":
				r.Projectiles = []Projectile{{X: r.Enemies[0].X, Y: r.Enemies[0].Y, Power: 100, Life: 1, Kind: "ice", Skill: skill}}
				r.tick(Input{}, .02)
				if len(r.Projectiles) != 0 {
					t.Fatal("projectile missed fixture target")
				}
			}
			return 10000 - r.Enemies[0].HP
		}
		neutral, advantage := damage(""), damage(content.ElementFire)
		if neutral <= 0 || advantage != neutral*2 {
			t.Fatalf("%s: neutral %v advantage %v", path, neutral, advantage)
		}
	}
}

func TestBossElementPhasesUsePreImpactPhaseThenChange(t *testing.T) {
	r := elementalAttackRun(content.ElementFire)
	threshold := 10 * float64(bossPhaseTraining[1].AtHealthPercent)
	r.Enemies[0].HP, r.Enemies[0].MaxHP = threshold+10, 1000
	r.hurtEnemyElement(0, 10, "hit", 0, content.ElementWater)
	if r.Enemies[0].HP != threshold-10 || r.Enemies[0].Phase != 2 {
		t.Fatalf("crossing hit: %+v", r.Enemies[0])
	}
	r.hurtEnemyElement(0, 10, "hit", 0, content.ElementWater)
	if r.Enemies[0].HP != threshold-20 {
		t.Fatal("next hit retained old phase matchup")
	}
}

func TestBossElementPhasesProjectileUsesImpactWard(t *testing.T) {
	r := elementalAttackRun(content.ElementFire)
	r.Projectiles = []Projectile{{X: r.Enemies[0].X, Y: r.Enemies[0].Y, Power: 100, Life: 1, Kind: "ice", Skill: Skill{ID: "flight", Element: content.ElementWater}}}
	// The ward changes while this Water projectile is in flight: Fire is weak
	// to Water, but the second phase's Air ward takes neutral Water damage.
	r.Enemies[0].Phase = 2
	r.tick(Input{}, .02)
	if r.Enemies[0].HP != 9900 || len(r.Projectiles) != 0 {
		t.Fatal("projectile used its launch phase or visual element")
	}
}

func TestBossElementPhasesNeutralJSONOmitsWard(t *testing.T) {
	for _, boss := range []Actor{{Kind: "boss"}, AdaptMonster(content.Mob{Name: "Physical boss", Type: content.MobBoss, Element: content.ElementPhysical})} {
		raw, err := json.Marshal(boss)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		if _, exists := fields["elemental_phases"]; exists {
			t.Fatal("neutral actor emitted empty phase metadata")
		}
	}
}
