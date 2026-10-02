package bot

import (
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftRoundSnapshotValidation(t *testing.T) {
	for _, mode := range []string{"valid", "legacy", "small", "wide", "off_map", "entry_outside", "exit_outside"} {
		t.Run(mode, func(t *testing.T) {
			run := rift.NewRunAtLevel("round", rift.Build{HP: 100}, time.Now(), nil, 1)
			room := &run.Level.Rooms[0]
			room.Round = &rift.RoundArena{X: 800, Y: 402.5, RadiusX: 740, RadiusY: 87.5}
			room.Entrance = &rift.ArenaEntrance{X: 160, Y: 402.5}
			room.Exit = &rift.ArenaEntrance{X: 1440, Y: 402.5}
			switch mode {
			case "legacy":
				room.Round = nil
			case "small":
				room.Round.RadiusY = 60
			case "wide":
				room.Round.RadiusX = 800
			case "off_map":
				room.Round.Y = 450
			case "entry_outside":
				room.Entrance.Y = 320
			case "exit_outside":
				room.Exit.Y = 490
			}
			raw, err := encodeRift(run)
			if err != nil {
				t.Fatal(err)
			}
			got, err := decodeRift(raw)
			valid := mode == "valid" || mode == "legacy"
			if (err == nil) != valid {
				t.Fatalf("decode error=%v valid=%v", err, valid)
			}
			if mode == "valid" && *got.Level.Rooms[0].Round != *room.Round {
				t.Fatal("round geometry changed on recovery")
			}
		})
	}
}
