package bot

import "fmt"

const (
	abyssAutoStopHP        = "hp"
	abyssAutoStopDepth     = "depth"
	abyssAutoStopLegendary = "legendary"
)

// abyssAutoDescendRules are optional safeguards for a planned multi-floor
// descent. Zero values disable the corresponding rule.
type abyssAutoDescendRules struct {
	HPBelowPct      int  `json:"hp_below_pct"`
	TargetDepth     int  `json:"target_depth"`
	StopOnLegendary bool `json:"stop_on_legendary"`
}

func (r abyssAutoDescendRules) validate(currentDepth, plannedFloors int) error {
	if r.HPBelowPct < 0 || r.HPBelowPct > 99 {
		return fmt.Errorf("HP stop must be between 1 and 99 percent")
	}
	if r.TargetDepth < 0 {
		return fmt.Errorf("target depth cannot be negative")
	}
	if r.TargetDepth > 0 && (r.TargetDepth <= currentDepth || r.TargetDepth > currentDepth+plannedFloors) {
		return fmt.Errorf("target depth must be within the planned queue")
	}
	return nil
}

// stopReason evaluates authoritative state. Safety has priority when several
// rules trigger on the same settled floor.
func (r abyssAutoDescendRules) stopReason(depth, hp, maxHP int, legendaryDrop bool) string {
	if r.HPBelowPct > 0 && maxHP > 0 && int64(hp)*100 < int64(maxHP)*int64(r.HPBelowPct) {
		return abyssAutoStopHP
	}
	if r.StopOnLegendary && legendaryDrop {
		return abyssAutoStopLegendary
	}
	if r.TargetDepth > 0 && depth >= r.TargetDepth {
		return abyssAutoStopDepth
	}
	return ""
}

func addAbyssAutoStopResponse(out map[string]any, reason string) {
	if reason == "" {
		return
	}
	out["auto_stopped"] = true
	out["stop_reason"] = reason
}
