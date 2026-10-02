//go:build e2e

package bot

import (
	"time"
	"ts3news/internal/rift"
)

// riftTerminalFixture supplies explicit result screens without real rewards.
// Unknown scenarios are left to the existing fixture handlers.
func riftTerminalFixture(scenario string, build rift.Build) *rift.Run {
	switch scenario {
	case "terminal-complete", "terminal-banked", "terminal-defeated", "terminal-expired", "victory-final":
	default:
		return nil
	}
	now := time.Now()
	run := rift.NewRunAtLevel(scenario, build, now, riftMobCatalog(now), 1)
	run.Epoch = "fixture"
	if scenario == "victory-final" {
		// Keep the final blow and its victory event in the production simulation.
		run.Room = len(run.Level.Rooms) - 1
		run.Level.Rooms[run.Room].Hazards = nil
		run.Level.Rooms[run.Room].Obstacles = nil
		run.RoomObjective = nil
		run.Enemies = []rift.Actor{{ID: "final-target", Name: "Final guardian", Kind: "goblin", X: run.Player.X + 45, Y: run.Player.Y, HP: 1, MaxHP: 1, Facing: -1, Cooldown: 1000}}
		run.SetPaused(true, now)
		return run
	}
	run.BankedGold = 50
	run.BankedAtMS = now.UnixMilli()
	switch scenario {
	case "terminal-defeated":
		run.Player.HP = 0
		run.Step(rift.Input{}, now.Add(100*time.Millisecond))
	case "terminal-expired":
		// Match the expired GET projection: secured gold is historical, not spendable.
		run.Epoch = "fixture-old"
		run.Status = "expired"
		run.CompletedLevels = []int{1, 7}
		run.History = map[int]rift.MissionHistory{
			1: {Definition: run.MissionDefinition, Attempts: 2, Completions: 2, LastOutcome: "completed", BestSeconds: 42, CompletedByClass: map[string]int{build.Class: 2}},
			7: {Attempts: 1, Completions: 1, LastOutcome: "completed", CompletedByClass: map[string]int{build.Class: 1}},
		}
		run.PastExpeditions.Gold = run.BankedGold
		run.BankedGold = 0
	case "terminal-complete", "terminal-banked":
		if scenario == "terminal-complete" {
			run.Room = 2
		}
		run.Enemies = append([]rift.Actor{}, run.EncounterPlan[run.Room]...)
		for i := range run.Enemies {
			run.Enemies[i].HP = 0
		}
		run.Status = "cleared"
		run.RecordEncounterSummary("cleared")
		action := "exit"
		if scenario == "terminal-complete" {
			action = "bank"
		}
		run.FinishCheckpoint(action, riftMobCatalog(now))
	}
	run.UpdateObjectives()
	return run
}
