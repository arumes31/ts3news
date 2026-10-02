package bot

import (
	"testing"
	"ts3news/internal/rift"
)

func TestRiftHazardSwitchValidation(t *testing.T) {
	for _, mode := range []string{"valid", "outside", "overcharge", "used_live", "used_partial", "used_off"} {
		t.Run(mode, func(t *testing.T) {
			r := riftVitalsFixture()
			level := rift.Campaign()[70]
			r.Level = &level
			s := r.Level.Rooms[0].HazardSwitch
			switch mode {
			case "outside":
				s.X = -1
			case "overcharge":
				s.Charge = 1
			case "used_live":
				s.Used = true
				s.Charge = .6
			case "used_partial":
				s.Used = true
				s.Charge = .3
			case "used_off":
				s.Used = true
				s.Charge = .6
				for i := range r.Level.Rooms[0].Hazards {
					r.Level.Rooms[0].Hazards[i].Disabled = true
				}
			}
			raw, err := encodeRift(r)
			if err != nil {
				t.Fatal(err)
			}
			_, err = decodeRift(raw)
			valid := mode == "valid" || mode == "used_off"
			if (err == nil) != valid {
				t.Fatalf("validation: %v", err)
			}
		})
	}
}
