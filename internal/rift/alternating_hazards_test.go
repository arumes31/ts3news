package rift

import (
	"encoding/json"
	"math"
	"sort"
	"testing"
)

func TestCrossroadsHazardsAlternateAcrossEveryRegionAndSave(t *testing.T) {
	groups := 0
	for _, level := range Campaign() {
		if level.ID%10 != 4 {
			continue
		}
		groups++
		hazards := append([]Hazard(nil), level.Rooms[2].Hazards...)
		sort.Slice(hazards, func(i, j int) bool { return hazards[i].X < hazards[j].X })
		if len(hazards) != 3 {
			t.Fatal("linked group must have three hazards")
		}
		period := hazards[0].Period
		slot := period / 3
		for i, h := range hazards {
			if h.Period != period || h.Duration >= slot {
				t.Fatal("linked group has overlapping active windows")
			}
			onset := math.Mod(1.2-h.Offset+period, period)
			want := math.Mod(1.2+float64(i)*slot, period)
			if math.Abs(onset-want) > 1e-8 {
				t.Fatalf("mission %d hazard %d onset=%v want=%v", level.ID, i, onset, want)
			}
		}
		raw, err := json.Marshal(hazards)
		if err != nil {
			t.Fatal(err)
		}
		var saved []Hazard
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		// Examine every transition on both sides, plus each active-window midpoint.
		for _, h := range saved {
			for _, phase := range []float64{1.2 - 1e-7, 1.2 + 1e-7, 1.2 + h.Duration/2, 1.2 + h.Duration - 1e-7, 1.2 + h.Duration + 1e-7} {
				clock := 2*period - h.Offset + phase
				active := 0
				for _, other := range saved {
					p := other.Phase(clock)
					if p >= 1.2 && p < 1.2+other.Duration {
						active++
					}
				}
				if active > 1 {
					t.Fatalf("mission %d has %d simultaneous linked hazards", level.ID, active)
				}
			}
		}
	}
	if groups != 10 {
		t.Fatalf("got %d alternating rooms", groups)
	}
}
