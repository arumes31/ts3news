package rift

import (
	"encoding/json"
	"testing"
)

func TestMissedBossSlamOpensTemporarySavedWeakPoint(t *testing.T) {
	for _, avoid := range []string{"jump", "lane", "hit"} {
		t.Run(avoid, func(t *testing.T) {
			r := testRun()
			r.Player.X, r.Player.Y = 500, 410
			r.Enemies = []Actor{{ID: "boss", Kind: "boss", ArtKey: "monster:test", X: 550, Y: 410, HP: 1000, MaxHP: 1000, Windup: .01, TargetX: 500, TargetY: 410}}
			if avoid == "jump" {
				r.Player.Jump = .5
			}
			if avoid == "lane" {
				r.Player.Y = 480
			}
			r.enemyTick(0, .02)
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err := json.Unmarshal(data, &saved); err != nil {
				t.Fatal(err)
			}
			saved.hurtEnemy(0, 100, "hit")
			want := 875.0
			if avoid == "hit" {
				want = 900
			}
			if saved.Enemies[0].HP != want {
				t.Fatalf("counter damage: hp=%v want=%v", saved.Enemies[0].HP, want)
			}
			saved.enemyTick(0, .81)
			saved.hurtEnemy(0, 100, "hit")
			if saved.Enemies[0].HP != want-100 {
				t.Fatal("weak-point bonus persisted beyond its window")
			}
		})
	}
}
