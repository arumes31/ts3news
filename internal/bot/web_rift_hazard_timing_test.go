package bot

import (
	"testing"
	"time"

	"ts3news/internal/rift"
)

func TestRiftDecodeRejectsInvalidHazardTiming(t *testing.T) {
	for _, place := range []string{"current", "future", "practice"} {
		for name, change := range map[string]func(*rift.Hazard){
			"zero period":       func(h *rift.Hazard) { h.Period = 0 },
			"negative period":   func(h *rift.Hazard) { h.Period = -1 },
			"huge period":       func(h *rift.Hazard) { h.Period = 1e100 },
			"zero duration":     func(h *rift.Hazard) { h.Duration = 0 },
			"negative duration": func(h *rift.Hazard) { h.Duration = -1 },
			"missing recovery":  func(h *rift.Hazard) { h.Duration = h.Period - 1.2 },
			"negative offset":   func(h *rift.Hazard) { h.Offset = -1 },
			"huge offset":       func(h *rift.Hazard) { h.Offset = 1e100 },
		} {
			t.Run(place+"/"+name, func(t *testing.T) {
				r := riftVitalsFixture()
				h := rift.Hazard{Period: 5, Duration: 1, Offset: .5}
				change(&h)
				switch place {
				case "current":
					level := rift.Campaign()[0]
					r.Level = &level
					r.Level.Rooms[0].Hazards = []rift.Hazard{h}
				case "future":
					level := rift.Campaign()[0]
					r.Level = &level
					r.Level.Rooms[1].Hazards = []rift.Hazard{h}
				case "practice":
					r.Practice.Arena.Hazards = []rift.Hazard{h}
				}
				saved, err := encodeRift(r)
				if err != nil {
					t.Fatal(err)
				}
				if got, err := decodeRift(saved); err == nil || got != nil {
					t.Fatal("invalid hazard accepted")
				}
			})
		}
	}
	for _, clock := range []float64{-1, 1e100} {
		r := riftVitalsFixture()
		r.Clock = clock
		saved, err := encodeRift(r)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := decodeRift(saved); err == nil {
			t.Fatal("invalid simulation clock accepted")
		}
	}
}

func TestRiftDecodePreservesHazardPracticeTiming(t *testing.T) {
	for _, intensity := range []string{"gentle", "standard", "intense"} {
		r, err := rift.NewPracticeRun("hazard-save", rift.Build{HP: 100}, "hazard", time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if err = r.ConfigureHazardPractice(intensity); err != nil {
			t.Fatal(err)
		}
		r.Clock = 12345.6789
		r.Practice.Arena.Hazards[0].Offset = 2*r.Practice.Arena.Hazards[0].Period + .3
		original := r.Practice.Arena.Hazards[0]
		saved, err := encodeRift(r)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decodeRift(saved)
		if err != nil {
			t.Fatal(err)
		}
		h := got.Practice.Arena.Hazards[0]
		if h != original || got.Clock != r.Clock || h.Phase(got.Clock) != original.Phase(r.Clock) {
			t.Fatal("hazard phase changed on reload")
		}
	}
}
