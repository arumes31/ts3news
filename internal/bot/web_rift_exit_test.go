package bot

import (
	"testing"
	"ts3news/internal/rift"
)

func TestRiftExitBounds(t *testing.T) {
	for _, place := range []string{"current", "future", "practice"} {
		for _, valid := range []bool{false, true} {
			r := riftVitalsFixture()
			level := rift.Campaign()[0]
			r.Level = &level
			e := &rift.ArenaEntrance{X: 115, Y: 370}
			if !valid {
				e.Y = 600
			}
			switch place {
			case "current":
				r.Level.Rooms[0].Exit = e
			case "future":
				r.Level.Rooms[1].Exit = e
			case "practice":
				r.Practice.Arena.Exit = e
			}
			raw, err := encodeRift(r)
			if err != nil {
				t.Fatal(err)
			}
			_, err = decodeRift(raw)
			if (err == nil) != valid {
				t.Fatalf("%s valid=%v: %v", place, valid, err)
			}
		}
	}
}
