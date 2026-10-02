package bot

import (
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftBridgeSnapshotValidation(t *testing.T) {
	for _, mode := range []string{"valid", "legacy", "narrow", "off_map", "duplicate", "overlap", "nearby", "empty_id", "long_id"} {
		t.Run(mode, func(t *testing.T) {
			run := rift.NewRunAtLevel("bridge", rift.Build{HP: 100}, time.Now(), nil, 1)
			bridges := []rift.NarrowBridge{{ID: "deck", Obstacle: rift.Obstacle{X: 400, Y: 370, W: 200, H: 90}}}
			switch mode {
			case "legacy":
				bridges = nil
			case "narrow":
				bridges[0].H = 79
			case "off_map":
				bridges[0].Y = 450
			case "duplicate":
				bridges = append(bridges, bridges[0])
				bridges[1].X = 900
			case "overlap":
				bridges = append(bridges, bridges[0])
				bridges[1].ID = "second"
			case "nearby":
				bridges = append(bridges, bridges[0])
				bridges[1].ID = "second"
				bridges[1].X = 650
			case "empty_id":
				bridges[0].ID = ""
			case "long_id":
				bridges[0].ID = string(make([]byte, 81))
			}
			run.Level.Rooms[0].Bridges = bridges
			raw, err := encodeRift(run)
			if err != nil {
				t.Fatal(err)
			}
			got, err := decodeRift(raw)
			valid := mode == "valid" || mode == "legacy"
			if (err == nil) != valid {
				t.Fatalf("decode error=%v, valid=%v", err, valid)
			}
			if valid && len(got.Level.Rooms[0].Bridges) != len(bridges) {
				t.Fatal("saved bridge geometry changed")
			}
		})
	}
}
