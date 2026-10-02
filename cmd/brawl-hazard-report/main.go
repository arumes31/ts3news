// Command brawl-hazard-report finds hazard phase intervals without a safe route.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"ts3news/internal/rift"
)

type hazardRoomReport struct {
	Mission   int                         `json:"mission"`
	Tier      int                         `json:"tier"`
	Name      string                      `json:"name"`
	Horizon   float64                     `json:"horizon_seconds"`
	Intervals []rift.HazardUnsafeInterval `json:"unsafe_intervals"`
	Error     string                      `json:"error,omitempty"`
}

func execute(args []string, out, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("brawl-hazard-report", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	horizon := flags.Float64("seconds", 30, "analysis horizon in seconds, greater than zero and at most 3600")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if _, err := rift.AuditHazardSafety(rift.Arena{}, *horizon); err != nil || flags.NArg() != 0 {
		_, _ = fmt.Fprintln(diagnostics, "Use -seconds in (0, 3600] and no positional arguments.")
		return 2
	}
	rows := []hazardRoomReport{}
	failed := 0
	for _, level := range rift.Campaign() {
		for room, arena := range level.Rooms {
			intervals, err := rift.AuditHazardSafety(arena, *horizon)
			row := hazardRoomReport{Mission: level.ID, Tier: room + 1, Name: level.Name, Horizon: *horizon, Intervals: intervals}
			if err != nil {
				row.Error = err.Error()
			}
			if err != nil || len(intervals) > 0 {
				failed++
			}
			rows = append(rows, row)
		}
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rows); err != nil {
		_, _ = fmt.Fprintln(diagnostics, err)
		return 2
	}
	_, _ = fmt.Fprintf(diagnostics, "%d rooms; %d with unsafe intervals or invalid hazards in %.2f seconds.\n", len(rows), failed, *horizon)
	if failed > 0 {
		return 1
	}
	return 0
}
func main() { os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr)) }
