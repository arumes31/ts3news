package bot

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftHazardPracticeIntensitySelection(t *testing.T) {
	for _, tc := range []struct {
		level            string
		period, duration float64
	}{{"gentle", 5, .35}, {"standard", 3.5, .45}, {"intense", 2.8, .6}} {
		var req riftRequest
		if err := json.Unmarshal([]byte(`{"hazard_intensity":"`+tc.level+`"}`), &req); err != nil {
			t.Fatal(err)
		}
		r, err := newRiftPractice(req, "intensity", rift.Build{HP: 100}, "hazard", time.Unix(100, 0))
		if err != nil {
			t.Fatal(err)
		}
		h := r.Practice.Arena.Hazards[0]
		if h.Period != tc.period || h.Duration != tc.duration {
			t.Fatalf("%s wrong cadence: %+v", tc.level, h)
		}
		if err = resetRiftPractice(r, riftRequest{}, time.Unix(200, 0)); err != nil {
			t.Fatal(err)
		}
		if r.Practice.Arena.Hazards[0].Period != tc.period {
			t.Fatal("reset forgot intensity")
		}
	}
}

func TestRiftHazardPracticeRejectsInvalidSelectionWithoutReset(t *testing.T) {
	r, err := newRiftPractice(riftRequest{}, "intensity", rift.Build{HP: 100}, "hazard", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	r.Player.HP = 25
	before, _ := json.Marshal(r)
	if err = resetRiftPractice(r, riftRequest{HazardIntensity: "unknown"}, time.Unix(200, 0)); err == nil {
		t.Fatal("accepted invalid selection")
	}
	after, _ := json.Marshal(r)
	if string(before) != string(after) {
		t.Fatal("invalid selection reset the drill")
	}
	if _, err = newRiftPractice(riftRequest{HazardIntensity: "gentle"}, "wrong", rift.Build{HP: 100}, "guard", time.Unix(100, 0)); err == nil {
		t.Fatal("accepted hazard intensity in another drill")
	}
}
