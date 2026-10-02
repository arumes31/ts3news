package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func surgeRun() *Run {
	r := testRun()
	r.Player.X = 100
	r.Player.Y = 320
	e := AdaptMonster(content.Mob{Name: i18n.T("mob.ancient_dragon"), Type: content.MobBoss})
	e.ID = "dragon"
	e.X = 1400
	e.Y = 480
	e.Attacks = 7
	r.Enemies = []Actor{e}
	r.Level = &Level{Rooms: []Arena{{}}}
	return r
}
func TestBossSurgeSavedWarningAndWholeArenaDamage(t *testing.T) {
	for _, position := range [][2]float64{{35, 315}, {1565, 490}, {800, 400}} {
		r := surgeRun()
		r.Player.X = position[0]
		r.Player.Y = position[1]
		r.enemyTick(0, .02)
		if r.Enemies[0].AttackName != "Ground Surge" || r.Enemies[0].Windup != 2 {
			t.Fatal("no arena-wide telegraph")
		}
		r.enemyTick(0, 1.9)
		if r.Player.HP != r.Player.MaxHP {
			t.Fatal("early hit")
		}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var s Run
		if err = json.Unmarshal(data, &s); err != nil {
			t.Fatal(err)
		}
		s.enemyTick(0, .11)
		if s.Player.HP >= s.Player.MaxHP || s.Enemies[0].Attacks != 8 || findEvent(s.Events, "boss_surge") == nil {
			t.Fatal("saved surge missed floor")
		}
	}
}
func TestBossSurgeJumpDodgeAndGuard(t *testing.T) {
	losses := map[string]float64{}
	for _, mode := range []string{"standing", "guard", "jump", "dodge", "barrier"} {
		r := surgeRun()
		r.enemyTick(0, .02)
		switch mode {
		case "guard":
			r.Player.Guard = true
			r.Player.Facing = 1
		case "jump":
			r.Player.Jump = .3
		case "dodge":
			r.SkillTimers["dodge_invulnerability"] = 1
		case "barrier":
			r.Barrier = 1000
		}
		r.enemyTick(0, 2)
		losses[mode] = r.Player.MaxHP - r.Player.HP
		if r.Stats.Guards != 0 {
			t.Fatal("surge awarded guard")
		}
	}
	if losses["standing"] <= 0 || losses["guard"] != losses["standing"] || losses["jump"] != 0 || losses["dodge"] != 0 || losses["barrier"] != 0 {
		t.Fatal(losses)
	}
}

func TestBossSurgeRealJumpAcrossAllFinalArenas(t *testing.T) {
	for _, level := range Campaign() {
		r := NewRunAtLevel("surge-jump", Build{HP: 10000}, time.Unix(0, 0), content.AbyssMobCatalog(), level.ID)
		r.Room = 2
		r.spawnRoom()
		r.RoomObjective = nil
		boss := surgeRun().Enemies[0]
		boss.X = r.Enemies[0].X
		boss.Y = r.Enemies[0].Y
		r.Enemies = []Actor{boss}
		r.enemyTick(0, .02)
		hp := r.Player.HP
		for j := 0; j < 80; j++ {
			r.tick(Input{}, .02)
		}
		r.tick(Input{Jump: true}, .02)
		for j := 0; j < 20; j++ {
			r.tick(Input{}, .02)
		}
		if r.Player.HP != hp || r.Enemies[0].Attacks != 8 || findEvent(r.Events, "boss_surge") == nil {
			t.Fatalf("mission%d real jump failed", level.ID)
		}
	}
}
func TestBossSurgeSlowPracticeCancellationAndChargePriority(t *testing.T) {
	r := surgeRun()
	r.Practice = &PracticeState{SlowTelegraphs: true}
	r.enemyTick(0, .02)
	if r.Enemies[0].Windup != 4 {
		t.Fatal("slow warning")
	}
	r = surgeRun()
	r.Player.X = 1250
	r.Player.Y = 480
	r.Enemies[0].Attacks = 23
	if r.wantsBossCharge(r.Enemies[0]) || r.NextBossAttack(r.Enemies[0]).Kind != "surge" {
		t.Fatal("charge took surge turn")
	}
	r.enemyTick(0, .02)
	r.Enemies[0].BossStagger = 80
	r.hurtEnemy(0, 1, "hit")
	r.enemyTick(0, 2)
	if findEvent(r.Events, "boss_surge") != nil {
		t.Fatal("guard break did not cancel surge")
	}
}
