// Command brawl-reachability-report identifies unreachable campaign target coordinates.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

type roomReport struct {
	Mission  int                        `json:"mission"`
	Tier     int                        `json:"tier"`
	Name     string                     `json:"name"`
	Targets  int                        `json:"targets"`
	Failures []rift.ReachabilityFailure `json:"failures"`
}

func report(seed string) []roomReport {
	rows := []roomReport{}
	for _, level := range rift.Campaign() {
		run := rift.NewRunAtLevel(seed, rift.Build{HP: 300, Damage: 20}, time.Unix(100, 0), content.AbyssMobCatalog(), level.ID)
		for room := range level.Rooms {
			targets := append([]rift.Actor(nil), run.EncounterPlan[room]...)
			ids := map[string]bool{}
			for _, actor := range targets {
				ids[actor.ID] = true
			}
			for _, actor := range run.Enemies {
				if !ids[actor.ID] {
					targets = append(targets, actor)
					ids[actor.ID] = true
				}
			}
			failures := rift.AuditArenaReachability(run.Arena(), run.Player, targets)
			rows = append(rows, roomReport{level.ID, room + 1, level.Name, len(targets), failures})
			run.Status = "cleared"
			run.NextRoom()
		}
	}
	return rows
}

func execute(args []string, out, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("brawl-reachability-report", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	seed := flags.String("seed", "author-reachability", "deterministic encounter seed")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(diagnostics, "Positional arguments are unsupported.")
		return 2
	}
	rows := report(*seed)
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rows); err != nil {
		fmt.Fprintln(diagnostics, err)
		return 2
	}
	failures := 0
	for _, row := range rows {
		failures += len(row.Failures)
	}
	fmt.Fprintf(diagnostics, "%d rooms checked; %d unreachable targets/start failures. Grid: 10 units; walking only, intact cover.\n", len(rows), failures)
	if failures > 0 {
		return 1
	}
	return 0
}
func main() { os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr)) }
