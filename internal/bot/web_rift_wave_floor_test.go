package bot

import (
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftWaveFloorSnapshotValidation(t *testing.T) {
	for _, mode := range []string{"valid", "legacy", "narrow_bypass", "overlap", "missing", "changed", "timer", "collapsed_timer", "rest_gap", "complete_gap", "wrong_kind"} {
		t.Run(mode, func(t *testing.T) {
			r := rift.NewRunAtLevel("panels", rift.Build{HP: 100}, time.Now(), nil, 1)
			box := rift.Obstacle{X: 500, Y: 380, W: 100, H: 60}
			r.Level.Rooms[0].Objective = "survive_waves"
			r.Level.Rooms[0].FragileFloor = []rift.Obstacle{box}
			r.RoomObjective = &rift.RoomObjective{Kind: "survive_waves", Target: 3, Wave: 1, FloorSegments: []rift.WaveFloorSegment{{Obstacle: box, CollapseIn: 3}}}
			f := &r.RoomObjective.FloorSegments[0]
			switch mode {
			case "legacy":
				r.Level.Rooms[0].FragileFloor = nil
				r.RoomObjective.FloorSegments = nil
			case "narrow_bypass":
				r.Level.Rooms[0].FragileFloor[0].Y = 330
			case "overlap":
				r.Level.Rooms[0].FragileFloor = append(r.Level.Rooms[0].FragileFloor, box)
			case "missing":
				r.RoomObjective.FloorSegments = nil
			case "changed":
				f.X++
			case "timer":
				f.CollapseIn = -1
			case "collapsed_timer":
				f.Collapsed = true
			case "rest_gap":
				f.Collapsed = true
				f.CollapseIn = 0
				r.RoomObjective.NextWaveSeconds = 1
			case "complete_gap":
				f.Collapsed = true
				f.CollapseIn = 0
				r.RoomObjective.Complete = true
			case "wrong_kind":
				r.RoomObjective.Kind = "hold_circle"
			}
			raw, err := encodeRift(r)
			if err != nil {
				t.Fatal(err)
			}
			_, err = decodeRift(raw)
			if (err == nil) != (mode == "valid" || mode == "legacy") {
				t.Fatalf("mode=%s decode=%v", mode, err)
			}
		})
	}
}
