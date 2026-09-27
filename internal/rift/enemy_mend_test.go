package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func enemyMendRun() *Run {
	r := testRun()
	r.Player.X, r.Player.Y = 100, 410
	r.Enemies = []Actor{
		{ID: "healer", Kind: "archer", Healer: true, X: 500, Y: 410, HP: 100, MaxHP: 100, Damage: 20, Speed: 80},
		{ID: "large", Kind: "knight", X: 600, Y: 410, HP: 120, MaxHP: 200},
		{ID: "wounded", Kind: "goblin", X: 450, Y: 410, HP: 20, MaxHP: 100},
	}
	return r
}

func TestEnemyMendPrioritizesWoundedAllyAndPersistsCast(t *testing.T) {
	r := enemyMendRun()
	r.enemyTick(0, .02)
	if r.Enemies[0].AttackName != "Mend ally" || r.Enemies[0].Windup != .8 {
		t.Fatal("missing mend telegraph")
	}
	if r.Enemies[2].HP != 20 {
		t.Fatal("healed before warning completed")
	}
	for n := 0; n < 20; n++ {
		r.enemyTick(0, .02)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 21; n++ {
		saved.enemyTick(0, .02)
	}
	if saved.Enemies[2].HP != 32 || saved.Enemies[1].HP != 120 {
		t.Fatal("mend did not heal the lowest health ratio ally")
	}
	if saved.Stats.Healing != 0 || saved.Stats.Kills != 0 || saved.Gold != 0 {
		t.Fatal("enemy heal changed player healing or rewards")
	}
	for n := 0; n < 100; n++ {
		saved.enemyTick(0, .02)
	}
	if saved.Enemies[2].HP != 32 {
		t.Fatal("healer ignored mend cooldown")
	}
}

func TestEnemyMendCancelsOnHitDeathOrLostTarget(t *testing.T) {
	for _, mode := range []string{"hit", "knockdown", "dead-target", "far-target", "wall"} {
		r := enemyMendRun()
		r.enemyTick(0, .02)
		switch mode {
		case "hit":
			r.hurtEnemy(0, 10, "hit")
		case "knockdown":
			r.Enemies[0].Knockdown = .4
		case "dead-target":
			r.Enemies[2].HP = 0
		case "far-target":
			r.Enemies[2].X = 1100
		case "wall":
			r.Level = &Level{Rooms: []Arena{{HighCover: []Obstacle{{470, 350, 10, 100}}}}}
		}
		before := r.Enemies[2].HP
		for n := 0; n < 70; n++ {
			r.enemyTick(0, .02)
		}
		if r.Enemies[2].HP != before {
			t.Fatalf("invalid mend completed after %s", mode)
		}
	}
}

func TestEnemyMendCapsHealingAndFallsBackToAttack(t *testing.T) {
	r := enemyMendRun()
	r.Enemies = r.Enemies[:2]
	r.Enemies[1].HP = 100
	r.Enemies[1].MaxHP = 1000
	for n := 0; n < 50; n++ {
		r.enemyTick(0, .02)
	}
	if r.Enemies[1].HP != 130 {
		t.Fatal("mend exceeded absolute cap")
	}
	r = enemyMendRun()
	r.Enemies[1].HP = 200
	r.Enemies[2].HP = 100
	r.enemyTick(0, .02)
	if r.Enemies[0].AttackName == "Mend ally" || r.Enemies[0].Windup == 0 {
		t.Fatal("healthy allies prevented normal attack")
	}
	r = enemyMendRun()
	r.Enemies[0].Windup = .01
	r.enemyTick(0, .02)
	if len(r.Projectiles) != 1 {
		t.Fatal("healing priority canceled a committed attack")
	}
	mob := content.Mob{Name: i18n.T("mob.frost_lich"), Type: content.MobElite}
	if !AdaptMonster(mob).Healer {
		t.Fatal("canonical Frost Lich lacks support move")
	}
}

func TestEnemyMendNeverOverhealsOrTargetsSelfPropsAndTreasure(t *testing.T) {
	r := enemyMendRun()
	r.Enemies[0].HP = 1
	r.Enemies[1].HP = 200
	r.Enemies[2].HP = 99
	r.Enemies = append(r.Enemies, Actor{ID: "prop", Kind: "totem", X: 510, Y: 410, HP: 1, MaxHP: 100}, Actor{ID: "treasure", Kind: "treasure", X: 520, Y: 410, HP: 1, MaxHP: 100})
	r.enemyTick(0, .02)
	if r.Enemies[0].HealTarget != "wounded" {
		t.Fatal("selected self, prop or treasure")
	}
	for n := 0; n < 41; n++ {
		r.enemyTick(0, .02)
	}
	if r.Enemies[2].HP != 100 || r.Enemies[0].HP != 1 || r.Enemies[3].HP != 1 || r.Enemies[4].HP != 1 {
		t.Fatal("overheal or invalid target mutation")
	}
}

func TestEnemyMendRespectsAttackerBudgetAndLockedTarget(t *testing.T) {
	r := enemyMendRun()
	r.Level = &Level{Rooms: []Arena{{MaxAttackers: 1}}}
	r.Enemies[1].Windup = .5
	r.enemyTick(0, .02)
	if r.Enemies[0].HealTarget != "" {
		t.Fatal("mend exceeded action budget")
	}
	r = enemyMendRun()
	r.enemyTick(0, .02)
	r.Enemies[1].HP = 1
	for n := 0; n < 41; n++ {
		r.enemyTick(0, .02)
	}
	if r.Enemies[1].HP != 1 || r.Enemies[2].HP != 32 {
		t.Fatal("cast silently retargeted after warning")
	}
}
