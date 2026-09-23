package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHazardPracticeIntensityDamageAndPersistence(t *testing.T) {
	for _, tc := range []struct {
		intensity string
		damage    float64
	}{{"gentle", 6}, {"standard", 12}, {"intense", 18}} {
		r, err := NewPracticeRun("intensity", Build{HP: 100}, "hazard", time.Unix(100, 0))
		if err != nil {
			t.Fatal(err)
		}
		if err = r.ConfigureHazardPractice(tc.intensity); err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		saved.Clock = 1.3
		before := saved.Player.HP
		saved.hazardTick()
		if before-saved.Player.HP != tc.damage || saved.Practice.HazardIntensity != tc.intensity {
			t.Fatalf("%s damage or persistence mismatch", tc.intensity)
		}
		if saved.Gold != 0 || len(saved.Drops) != 0 {
			t.Fatal("practice granted loot")
		}
	}
	r := circleTestRun()
	r.Level.Region = 9
	if r.hazardContactDamage(9) != 21 {
		t.Fatal("campaign damage changed")
	}
}
