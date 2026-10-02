package rift

import (
	"encoding/json"
	"math"
	"testing"
)

func TestHazardContactPreviewMatchesAppliedDamage(t *testing.T) {
	for _, region := range []int{0, 9} {
		for _, hp := range []float64{1, 3, 8, 30} {
			for _, armor := range []float64{0, 5, 100} {
				for _, guard := range []bool{false, true} {
					for _, barrier := range []float64{0, 2, 50} {
						for _, grace := range []bool{false, true} {
							r := testRun()
							r.Level = &Level{Region: region}
							r.Player.HP = hp
							r.Build.Armor = armor
							r.Player.Guard = guard
							r.Player.GuardStamina = 100
							r.Barrier = barrier
							r.FirstHitGrace = grace
							r.SkillTimers = map[string]float64{}
							before, _ := json.Marshal(r)
							preview := r.HazardContactHealthLoss()
							after, _ := json.Marshal(r)
							if string(before) != string(after) {
								t.Fatal("preview mutated run")
							}
							r.hurtPlayerFromHazard(r.hazardContactDamage(region), r.Player.X, r.Player.Y)
							if math.Abs(preview-(hp-r.Player.HP)) > 1e-9 {
								t.Fatalf("preview %.8f != applied %.8f", preview, hp-r.Player.HP)
							}
						}
					}
				}
			}
		}
	}
}

func TestHazardContactPreviewPracticeAndEvasion(t *testing.T) {
	for _, intensity := range []string{"gentle", "standard", "intense"} {
		r := testRun()
		r.Practice = &PracticeState{Mode: "hazard", HazardIntensity: intensity}
		r.Build.Armor = 0
		r.FirstHitGrace = false
		want := r.hazardContactDamage(0)
		r.SkillTimers = map[string]float64{"dodge_invulnerability": 1, "room_entry_grace": 1}
		r.Player.Jump = 1
		if r.HazardContactHealthLoss() != want {
			t.Fatal("conditional preview confused evasion with damage mitigation")
		}
		r.Player.HP = 0
		if r.HazardContactHealthLoss() != 0 {
			t.Fatal("defeated player has a lethal warning")
		}
	}
}
