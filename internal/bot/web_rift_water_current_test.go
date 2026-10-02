package bot

import (
	"testing"
	"ts3news/internal/rift"
)

func TestRiftWaterCurrentValidation(t *testing.T) {
	for _, place := range []string{"current", "future", "practice"} {
		for name, change := range map[string]func(*rift.WaterCurrent){
			"valid": func(c *rift.WaterCurrent) {}, "fast": func(c *rift.WaterCurrent) { c.VX = 41 }, "diagonal fast": func(c *rift.WaterCurrent) { c.VY = 35 }, "still": func(c *rift.WaterCurrent) { c.VX = 0 }, "outside": func(c *rift.WaterCurrent) { c.X = 1500 }, "negative size": func(c *rift.WaterCurrent) { c.W = -1 },
		} {
			t.Run(place+"/"+name, func(t *testing.T) {
				r := riftVitalsFixture()
				level := rift.Campaign()[0]
				r.Level = &level
				c := rift.WaterCurrent{Obstacle: rift.Obstacle{X: 460, Y: 400, W: 500, H: 65}, VX: 35}
				change(&c)
				switch place {
				case "current":
					r.Level.Rooms[0].WaterCurrents = []rift.WaterCurrent{c}
				case "future":
					r.Level.Rooms[1].WaterCurrents = []rift.WaterCurrent{c}
				case "practice":
					r.Practice.Arena.WaterCurrents = []rift.WaterCurrent{c}
				}
				saved, err := encodeRift(r)
				if err != nil {
					t.Fatal(err)
				}
				_, err = decodeRift(saved)
				if (err == nil) != (name == "valid") {
					t.Fatalf("validation: %v", err)
				}
			})
		}
	}
}
