package rift

import (
	"encoding/json"
	"testing"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

func channelRun() *Run {
	r := testRun()
	r.Player.X = 500
	r.Player.Y = 410
	e := AdaptMonster(content.Mob{Name: i18n.T("mob.chronos"), Type: content.MobLegendary})
	e.ID = "chronos"
	e.X = 650
	e.Y = 410
	e.Attacks = 3
	r.Enemies = []Actor{e}
	r.Level = &Level{Rooms: []Arena{{}}}
	return r
}
func TestBossChannelSavedReleaseAndInterrupt(t *testing.T) {
	for _, interrupt := range []bool{false, true} {
		r := channelRun()
		r.enemyTick(0, .02)
		if r.Enemies[0].AttackName != "Time Pulse" || r.Enemies[0].Windup != 2 {
			t.Fatal("missing channel")
		}
		r.enemyTick(0, 1.9)
		if r.Player.HP != r.Player.MaxHP {
			t.Fatal("early impact")
		}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		if interrupt {
			saved.hurtEnemy(0, 1, "hit")
		}
		saved.enemyTick(0, .11)
		if interrupt {
			if saved.Player.HP != saved.Player.MaxHP || saved.Enemies[0].Attacks != 4 || findEvent(saved.Events, "boss_channel_interrupt") == nil {
				t.Fatal("hit did not cancel and advance channel")
			}
		} else {
			if saved.Player.HP >= saved.Player.MaxHP || findEvent(saved.Events, "boss_channel_pulse") == nil {
				t.Fatal("saved channel failed to release")
			}
		}
	}
}
func TestBossChannelZeroDamageAndCounterplay(t *testing.T) {
	r := channelRun()
	r.enemyTick(0, .02)
	r.hurtEnemy(0, 0, "hit")
	if r.Enemies[0].Windup != 2 {
		t.Fatal("zero damage cancelled channel")
	}
	for _, mode := range []string{"move", "jump", "dodge"} {
		r := channelRun()
		r.enemyTick(0, .02)
		switch mode {
		case "move":
			r.Player.X += 150
		case "jump":
			r.Player.Jump = .2
		case "dodge":
			r.SkillTimers["dodge_invulnerability"] = 1
		}
		r.enemyTick(0, 2)
		if r.Player.HP != r.Player.MaxHP {
			t.Fatal(mode)
		}
	}
	r = channelRun()
	r.Practice = &PracticeState{SlowTelegraphs: true}
	r.enemyTick(0, .02)
	if r.Enemies[0].Windup != 4 {
		t.Fatal("slow practice")
	}
}

func TestBossChannelControlAndPhaseCancel(t *testing.T) {
	for _, mode := range []string{"knockdown", "stagger", "lethal", "phase"} {
		r := channelRun()
		r.enemyTick(0, .02)
		switch mode {
		case "knockdown":
			r.Enemies[0].Knockdown = 1
		case "stagger":
			r.Enemies[0].Pose = "stagger"
			r.Enemies[0].PoseTime = 1
		case "lethal":
			r.hurtEnemy(0, 100000, "hit")
		case "phase":
			r.hurtEnemy(0, r.Enemies[0].HP*.8, "hit")
		}
		for j := 0; j < 20; j++ {
			r.enemyTick(0, .1)
		}
		if findEvent(r.Events, "boss_channel_pulse") != nil || r.Player.HP != r.Player.MaxHP {
			t.Fatal(mode + " failed to cancel")
		}
	}
}
func TestBossChannelKeepsSequenceAndLockedGeometry(t *testing.T) {
	r := channelRun()
	for i, want := range []string{"lane_slam", "projectile", "lane_slam", "channel", "lane_slam", "projectile", "lane_slam", "channel"} {
		r.Enemies[0].Attacks = i
		if r.NextBossAttack(r.Enemies[0]).Kind != want {
			t.Fatal("sequence", i)
		}
	}
	r = channelRun()
	r.enemyTick(0, .02)
	r.Player.X += 120
	r.Player.Y += 50
	r.enemyTick(0, 2)
	if r.Player.HP != r.Player.MaxHP {
		t.Fatal("rectangular corner outside ellipse still hit")
	}
	e := findEvent(r.Events, "boss_channel_pulse")
	if e == nil || e.X != 500 || e.Y != 410 {
		t.Fatal("channel retargeted")
	}
}
