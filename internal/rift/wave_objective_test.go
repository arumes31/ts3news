package rift

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/content"
)

func waveTestRun() *Run {
	r := circleTestRun()
	r.Level.Rooms[0].Objective = "survive_waves"
	r.Enemies = nil
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		r.Enemies = append(r.Enemies, Actor{ID: id, Kind: "goblin", X: 1400, Y: 490, HP: 100, MaxHP: 100, Knockdown: 100})
	}
	r.beginRoomObjective()
	return r
}

func TestWaveRosterPersistsAndDefeatedActorsRemain(t *testing.T) {
	r := waveTestRun()
	if len(r.Enemies) != 2 || len(r.RoomObjective.Waves) != 3 {
		t.Fatal("roster not split")
	}
	r.Enemies[0].HP = 0
	if r.RoomObjective.Waves[0][0].HP != 100 {
		t.Fatal("wave snapshot aliases live actors")
	}
	r.Enemies[1].HP = 0
	r.tick(Input{}, .02)
	if r.Status != "fighting" || r.RoomObjective.NextWaveSeconds != 2.5 {
		t.Fatal("room cleared before reinforcements")
	}
	r.tickWaveObjective(1)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.RoomObjective.NextWaveSeconds != 1.5 {
		t.Fatal("countdown lost")
	}
	saved.tickWaveObjective(1.49)
	if len(saved.Enemies) != 2 {
		t.Fatal("reinforcements arrived early")
	}
	saved.tickWaveObjective(.01)
	if len(saved.Enemies) != 4 || saved.RoomObjective.Wave != 2 || saved.Enemies[0].HP != 0 || saved.Enemies[2].Cooldown < 1.2 {
		t.Fatal("wrong wave transition")
	}
	for i := range saved.Enemies {
		saved.Enemies[i].HP = 0
	}
	saved.tickWaveObjective(.02)
	saved.tickWaveObjective(2.5)
	if len(saved.Enemies) != 7 || saved.RoomObjective.Wave != 3 {
		t.Fatal("third wave missing")
	}
	ids := map[string]bool{}
	for _, enemy := range saved.Enemies {
		if ids[enemy.ID] {
			t.Fatal("duplicate enemy ID")
		}
		ids[enemy.ID] = true
	}
	saved.tick(Input{}, .02)
	if saved.RoomObjective.Complete || saved.Status != "fighting" {
		t.Fatal("living final wave bypassed")
	}
	for i := range saved.Enemies {
		saved.Enemies[i].HP = 0
	}
	saved.tick(Input{}, .02)
	if !saved.RoomObjective.Complete || saved.Status != "cleared" || saved.Stats.RoomsCleared != 1 {
		t.Fatal("final wave did not clear exactly one room")
	}
}

func TestWaveCountdownStopsWhenInactive(t *testing.T) {
	for _, mode := range []string{"paused", "dead", "ended", "alive"} {
		t.Run(mode, func(t *testing.T) {
			r := waveTestRun()
			for i := range r.Enemies {
				r.Enemies[i].HP = 0
			}
			r.tickWaveObjective(.02)
			switch mode {
			case "paused":
				r.Paused = true
			case "dead":
				r.Player.HP = 0
			case "ended":
				r.Status = "defeated"
			case "alive":
				r.Enemies[0].HP = 1
			}
			r.tickWaveObjective(2.5)
			if r.RoomObjective.Wave != 1 || r.RoomObjective.NextWaveSeconds != 2.5 {
				t.Fatal("inactive countdown advanced")
			}
		})
	}
}

func TestWaveCampaignRosterAndPreview(t *testing.T) {
	count := 0
	for _, level := range Campaign() {
		for room, arena := range level.Rooms {
			expected := level.ID%10 == 5 && room == 1
			if (arena.Objective == "survive_waves") != expected {
				t.Fatal("wrong wave placement")
			}
			if !expected {
				continue
			}
			count++
			r := NewRunAtLevel("wave-campaign", testRun().Build, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
			r.Room = room
			r.spawnRoom()
			total := 0
			for _, group := range r.RoomObjective.Waves {
				total += len(group)
			}
			if total != arena.Encounter.Enemies || total != len(r.EncounterPlan[room]) {
				t.Fatal("preview differs from frozen roster")
			}
			if total < 6 {
				t.Fatal("not enough defenders for three waves")
			}
		}
	}
	if count != 10 {
		t.Fatal("missing regional wave rooms")
	}
}
