// Command brawl-population-report reports campaign enemy budgets without a database.
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

type populationRow struct {
	Mission         int    `json:"mission"`
	Tier            int    `json:"tier"`
	Name            string `json:"name"`
	Objective       string `json:"objective"`
	PlannedEnemies  int    `json:"planned_enemies"`
	EntryEnemies    int    `json:"entry_enemies"`
	ObjectiveActors int    `json:"objective_actors"`
	Attackers       int    `json:"max_attackers"`
	Budget          int    `json:"enemy_budget"`
	OverBudget      bool   `json:"over_budget"`
}

func populationReport(limit int) []populationRow {
	catalog := content.AbyssMobCatalog()
	rows := make([]populationRow, 0, rift.LevelCount*len(rift.Rooms))
	for _, level := range rift.Campaign() {
		run := rift.NewRunAtLevel("author-population", rift.Build{HP: 300, Damage: 20}, time.Unix(100, 0), catalog, level.ID)
		for room := range level.Rooms {
			planned := len(run.EncounterPlan[room])
			ids := map[string]bool{}
			for _, enemy := range run.EncounterPlan[room] {
				ids[enemy.ID] = true
			}
			entry, props := 0, 0
			for _, actor := range run.Enemies {
				if ids[actor.ID] {
					entry++
				} else {
					props++
				}
			}
			attackers := run.Arena().MaxAttackers
			if attackers <= 0 {
				attackers = 3
			}
			attackers = min(attackers, 8)
			rows = append(rows, populationRow{level.ID, room + 1, level.Name, run.Arena().Objective, planned, entry, props, attackers, limit, planned > limit})
			run.Status = "cleared"
			run.NextRoom()
		}
	}
	return rows
}

func execute(args []string, out, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("brawl-population-report", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	limit := flags.Int("max-enemies", 8, "maximum planned enemies per room, including all waves")
	format := flags.String("format", "csv", "report format: csv or json")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 || *limit < 1 || (*format != "csv" && *format != "json") {
		fmt.Fprintln(diagnostics, "Use a positive -max-enemies and -format csv or json; positional arguments are unsupported.")
		return 2
	}
	rows := populationReport(*limit)
	violations := 0
	for _, row := range rows {
		if row.OverBudget {
			violations++
		}
	}
	var err error
	if *format == "json" {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(rows)
	} else {
		writer := csv.NewWriter(out)
		err = writer.Write([]string{"mission", "tier", "name", "objective", "planned_enemies", "entry_enemies", "objective_actors", "max_attackers", "enemy_budget", "over_budget"})
		for _, row := range rows {
			if err != nil {
				break
			}
			err = writer.Write([]string{strconv.Itoa(row.Mission), strconv.Itoa(row.Tier), row.Name, row.Objective, strconv.Itoa(row.PlannedEnemies), strconv.Itoa(row.EntryEnemies), strconv.Itoa(row.ObjectiveActors), strconv.Itoa(row.Attackers), strconv.Itoa(row.Budget), strconv.FormatBool(row.OverBudget)})
		}
		writer.Flush()
		if err == nil {
			err = writer.Error()
		}
	}
	if err != nil {
		fmt.Fprintln(diagnostics, err)
		return 2
	}
	fmt.Fprintf(diagnostics, "%d rooms; %d exceed the %d-enemy budget. Planned counts include later waves; objective actors are separate.\n", len(rows), violations, *limit)
	if violations > 0 {
		return 1
	}
	return 0
}
func main() { os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr)) }
