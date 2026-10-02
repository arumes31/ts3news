package rift

import "errors"

// ValidHazardIntensity accepts the three drill presets and the legacy default.
func ValidHazardIntensity(value string) bool {
	return value == "" || value == "gentle" || value == "standard" || value == "intense"
}

// ConfigureHazardPractice configures a freshly created or reset hazard drill.
func (r *Run) ConfigureHazardPractice(value string) error {
	if r.Practice == nil || r.Practice.Mode != "hazard" || len(r.Practice.Arena.Hazards) != 1 {
		return errors.New("hazard intensity requires hazard practice")
	}
	if !ValidHazardIntensity(value) {
		return errors.New("unknown hazard intensity")
	}
	if value == "" {
		value = "standard"
	}
	period, duration := 3.5, .45
	switch value {
	case "gentle":
		period, duration = 5, .35
	case "intense":
		period, duration = 2.8, .6
	}
	r.Practice.HazardIntensity = value
	r.Practice.Arena.Hazards[0].Period = period
	r.Practice.Arena.Hazards[0].Duration = duration
	return nil
}

func (r *Run) hazardContactDamage(region int) float64 {
	if r.Practice != nil && r.Practice.Mode == "hazard" {
		switch r.Practice.HazardIntensity {
		case "gentle":
			return 6
		case "intense":
			return 18
		default:
			return 12
		}
	}
	return 12 + float64(region)
}
