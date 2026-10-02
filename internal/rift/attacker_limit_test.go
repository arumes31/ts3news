package rift

import (
	"encoding/json"
	"testing"
)

func TestRoomAttackerLimitGatesWindups(t *testing.T) {
	r := testRun()
	r.Player.X = 500
	r.Player.Y = 410
	var arena Arena
	if err := json.Unmarshal([]byte(`{"max_attackers":2}`), &arena); err != nil {
		t.Fatal(err)
	}
	r.Level = &Level{Rooms: []Arena{arena}}
	r.Enemies = nil
	for i := 0; i < 5; i++ {
		r.Enemies = append(r.Enemies, Actor{Kind: "goblin", X: 540, Y: 410, HP: 100, MaxHP: 100})
	}
	for i := range r.Enemies {
		r.enemyTick(i, .01)
	}
	active := 0
	for _, e := range r.Enemies {
		if e.Windup > 0 {
			active++
		}
	}
	if active != 2 {
		t.Fatalf("active windups=%d, want 2", active)
	}
	r.Enemies[0].Windup = 0
	r.Enemies[0].Pose = "attack"
	r.Enemies[0].PoseTime = .3
	r.enemyTick(2, .01)
	if r.Enemies[2].Windup > 0 {
		t.Fatal("strike animation freed slot too early")
	}
	r.Enemies[0].PoseTime = 0
	r.Enemies[0].Pose = "idle"
	r.Enemies[0].Cooldown = 1
	r.enemyTick(2, .01)
	if r.Enemies[2].Windup <= 0 {
		t.Fatal("recovery did not free a slot")
	}
}

func TestAttackerLimitDefaultsBoundsAndSave(t *testing.T) {
	for _, tc := range []struct{ configured, want int }{{0, 3}, {1, 1}, {99, 8}} {
		r := testRun()
		r.Player.X = 500
		r.Player.Y = 410
		r.Level = &Level{Rooms: []Arena{{MaxAttackers: tc.configured}}}
		r.Enemies = []Actor{{HP: 0, Windup: 1}}
		for range 9 {
			r.Enemies = append(r.Enemies, Actor{Kind: "goblin", X: 540, Y: 410, HP: 100, MaxHP: 100})
		}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		if saved.Arena().MaxAttackers != tc.configured {
			t.Fatal("save lost authored limit")
		}
		for i := range saved.Enemies {
			saved.enemyTick(i, .01)
		}
		active := 0
		for _, e := range saved.Enemies {
			if e.HP > 0 && e.Windup > 0 {
				active++
			}
		}
		if active != tc.want {
			t.Fatalf("configured %d: active=%d, want %d", tc.configured, active, tc.want)
		}
	}
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			if arena.MaxAttackers != room+2 {
				t.Fatal("campaign tier limit missing")
			}
		}
	}
}
