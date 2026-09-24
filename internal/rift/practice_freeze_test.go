package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPracticeFreezeStopsEnemiesButPreservesPlayerAndAttacks(t *testing.T) {
	r, err := NewPracticeRun("freeze", testRun().Build, "guard", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if err := r.PracticeTool("practice_freeze"); err != nil {
		t.Fatal(err)
	}
	x, y := r.Enemies[0].X, r.Enemies[0].Y
	r.moveActor(&r.Enemies[0], 80, 20, true)
	if r.Enemies[0].X != x || r.Enemies[0].Y != y {
		t.Fatal("frozen enemy moved")
	}
	px := r.Player.X
	r.moveActor(&r.Player, -20, 0, false)
	if r.Player.X == px {
		t.Fatal("player frozen")
	}
	copy := r.Player
	r.moveActor(&copy, -10, 0, false)
	if copy.X == r.Player.X { t.Fatal("player movement probe frozen") }
	r.Player.X = px // Keep the trainer in attack range after testing player movement.
	for i := 1; i <= 100; i++ {
		r.Step(Input{}, time.UnixMilli(100000+int64(i)*50))
	}
	if r.Enemies[0].X != x || r.Enemies[0].Y != y || r.Stats.DamageTaken == 0 {
		t.Fatal("freeze moved enemy or suppressed attacks")
	}
	raw, _ := json.Marshal(r)
	var saved Run
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if err := saved.ResetPractice(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if !saved.Practice.FreezeMovement {
		t.Fatal("reset lost freeze choice")
	}
	if err := saved.PracticeTool("practice_freeze"); err != nil {
		t.Fatal(err)
	}
	saved.moveActor(&saved.Enemies[0], 20, 0, true)
	if saved.Enemies[0].X == x {
		t.Fatal("unfreeze did not restore movement")
	}
	campaign := testRun()
	if err := campaign.PracticeTool("practice_freeze"); err == nil {
		t.Fatal("campaign accepted freeze")
	}
	if len(r.History) != 0 || len(r.Drops) != 0 || r.Gold != 0 {
		t.Fatal("freeze created campaign progress")
	}
}
