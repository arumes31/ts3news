// Command brawl-campaign-export writes the authored campaign and its summary as JSON.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"ts3news/internal/rift"
)

type campaignExport struct {
	Schema       int            `json:"schema"`
	MissionCount int            `json:"mission_count"`
	RoomCount    int            `json:"room_count"`
	RegionCount  int            `json:"region_count"`
	Objectives   map[string]int `json:"objective_room_counts"`
	Missions     []rift.Level   `json:"missions"`
}

func execute(args []string, out, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("brawl-campaign-export", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(diagnostics, "No positional arguments are supported; JSON is written to stdout.")
		return 2
	}
	report := campaignExport{Schema: 1, Missions: rift.Campaign(), Objectives: map[string]int{}}
	regions := map[int]bool{}
	report.MissionCount = len(report.Missions)
	for _, mission := range report.Missions {
		regions[mission.Region] = true
		report.RoomCount += len(mission.Rooms)
		for _, room := range mission.Rooms {
			objective := room.Objective
			if objective == "" {
				objective = "combat"
			}
			report.Objectives[objective]++
		}
	}
	report.RegionCount = len(regions)
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(diagnostics, "write campaign export: %v\n", err)
		return 1
	}
	return 0
}
func main() { os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr)) }
