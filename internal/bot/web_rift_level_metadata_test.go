package bot

import (
	"reflect"
	"testing"
	"time"

	"ts3news/internal/rift"
)

func TestRiftDecodeRejectsInvalidLevelMetadata(t *testing.T) {
	changes := map[string]func(*rift.Level){
		"zero id":           func(l *rift.Level) { l.ID = 0 },
		"large id":          func(l *rift.Level) { l.ID = rift.LevelCount + 1 },
		"negative region":   func(l *rift.Level) { l.Region = -1 },
		"large region":      func(l *rift.Level) { l.Region = 10 },
		"wrong region":      func(l *rift.Level) { l.Region = 1 },
		"missing tier":      func(l *rift.Level) { l.Rooms = l.Rooms[:2] },
		"extra tier":        func(l *rift.Level) { l.Rooms = append(l.Rooms, l.Rooms[0]) },
		"empty name":        func(l *rift.Level) { l.Name = " " },
		"empty region name": func(l *rift.Level) { l.RegionName = "" },
		"empty difficulty":  func(l *rift.Level) { l.Difficulty = " " },
		"empty tier name":   func(l *rift.Level) { l.Rooms[2].Name = " " },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			r := rift.NewRunAtLevel("metadata", rift.Build{HP: 100}, time.Now(), nil, 1)
			change(r.Level)
			saved, err := encodeRift(r)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = decodeRift(saved); err == nil {
				t.Fatal("invalid level metadata accepted")
			}
		})
	}
}

func TestRiftDecodePreservesHistoricalLevelMetadata(t *testing.T) {
	r := rift.NewRunAtLevel("historic", rift.Build{HP: 100}, time.Now(), nil, 27)
	r.Level.Name = "Earlier mission title"
	r.Level.RegionName = "Earlier region"
	r.Level.Difficulty = "Retired challenge"
	r.Level.Rooms[0].Name = "Earlier tier"
	r.Level.Rooms[0].Obstacles = []rift.Obstacle{{X: 750, Y: 400, W: 20, H: 20}}
	before := *r.Level
	definition := r.MissionDefinition
	saved, err := encodeRift(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decodeRift(saved)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, *got.Level) || got.MissionDefinition != definition {
		t.Fatal("historical content replaced")
	}
	r.Level = nil
	saved, err = encodeRift(r)
	if err != nil {
		t.Fatal(err)
	}
	got, err = decodeRift(saved)
	if err != nil || got.Level != nil {
		t.Fatalf("legacy level-less save rejected or rewritten: %v", err)
	}
}
