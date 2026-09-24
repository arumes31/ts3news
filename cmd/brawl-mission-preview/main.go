// Command brawl-mission-preview emits deterministic authoring previews without a database.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/rift"
)

type tierPreview struct {
	Tier           int          `json:"tier"`
	Arena          rift.Arena   `json:"arena"`
	PlannedEnemies []rift.Actor `json:"planned_enemies"`
	EntryActors    []rift.Actor `json:"entry_actors"`
	PlayerSpawn    rift.Actor   `json:"player_spawn"`
}
type missionPreview struct {
	Schema  int           `json:"schema"`
	Seed    string        `json:"seed"`
	Catalog string        `json:"catalog"`
	Level   rift.Level    `json:"level"`
	Tiers   []tierPreview `json:"tiers"`
}

func execute(args []string, out, diagnostics io.Writer) int {
	flags := flag.NewFlagSet("brawl-mission-preview", flag.ContinueOnError)
	flags.SetOutput(diagnostics)
	mission := flags.Int("mission", 1, "mission ID, 1 through 100")
	seed := flags.String("seed", "author-preview", "deterministic encounter seed (1–128 bytes)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *mission < 1 || *mission > rift.LevelCount || strings.TrimSpace(*seed) == "" || len(*seed) > 128 {
		fmt.Fprintln(diagnostics, "Use -mission 1..100 and a nonblank -seed of at most 128 bytes; no positional arguments.")
		return 2
	}
	run := rift.NewRunAtLevel(*seed, rift.Build{Name: "Preview", HP: 300, Damage: 20}, time.Unix(100, 0).UTC(), content.AbyssMobCatalog(), *mission)
	preview := missionPreview{Schema: 1, Seed: *seed, Catalog: "canonical_abyss", Level: *run.Level, Tiers: make([]tierPreview, 0, len(rift.Rooms))}
	for room := range rift.Rooms {
		// Copy the actor slices before advancing the synthetic run.
		preview.Tiers = append(preview.Tiers, tierPreview{Tier: room + 1, Arena: run.Arena(), PlannedEnemies: append([]rift.Actor(nil), run.EncounterPlan[room]...), EntryActors: append([]rift.Actor(nil), run.Enemies...), PlayerSpawn: run.Player})
		if room < len(rift.Rooms)-1 {
			run.Status = "cleared"
			if !run.NextRoom() {
				fmt.Fprintln(diagnostics, "could not prepare the next preview tier")
				return 1
			}
		}
	}
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(preview); err != nil {
		fmt.Fprintf(diagnostics, "write mission preview: %v\n", err)
		return 1
	}
	return 0
}
func main() { os.Exit(execute(os.Args[1:], os.Stdout, os.Stderr)) }
