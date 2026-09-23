package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func TestAerialFinishObjective(t *testing.T) {
	for _, mode := range []string{"airborne", "grounded", "nonlethal", "spell"} {
		t.Run(mode, func(t *testing.T) {
			r := NewRunAtLevel("aerial-goal", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), 1)
			r.Stats.AerialFinishes = 3
			r.beginObjectives()
			r.Level.Rooms[0].Obstacles = nil
			r.Level.Rooms[0].Hazards = nil
			r.Enemies = []Actor{{ID: "target", Kind: "goblin", X: r.Player.X + 40, Y: r.Player.Y, HP: 1, MaxHP: 100, Knockdown: 100}}
			if mode == "nonlethal" {
				r.Enemies[0].HP = 1e6
			}
			input := Input{Attack: true, Jump: mode != "grounded"}
			if mode == "spell" {
				input.Attack = false
				r.Player.Jump = .5
				r.hurtEnemy(0, 100, "fire")
			}
			r.tick(input, .02)
			var goal *ObjectiveProgress
			for i := range r.Objectives.Entries {
				if r.Objectives.Entries[i].ID == "aerial_finish" {
					goal = &r.Objectives.Entries[i]
				}
			}
			if goal == nil {
				t.Fatal("aerial objective missing")
			}
			want := 0
			if mode == "airborne" {
				want = 1
			}
			if int(goal.Current) != want || goal.Status != "active" {
				t.Fatalf("wrong progress: %+v", goal)
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var saved Run
			if err = json.Unmarshal(data, &saved); err != nil {
				t.Fatal(err)
			}
			saved.Room = 2
			saved.Status = "cleared"
			saved.FinishCheckpoint("bank", nil)
			if saved.ObjectiveHistory["Wayfarer"]["aerial_finish"] != want {
				t.Fatal("wrong banked aerial result")
			}
		})
	}
}
