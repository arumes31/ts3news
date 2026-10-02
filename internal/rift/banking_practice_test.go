package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBankingPracticeRequiresCheckpointAndExplicitConfirmation(t *testing.T) {
	r, err := NewPracticeRun("banking", testRun().Build, "banking", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	reject := func() {
		t.Helper()
		before, _ := json.Marshal(r)
		if r.PracticeTool("practice_bank") == nil {
			t.Fatal("early banking accepted")
		}
		after, _ := json.Marshal(r)
		if string(before) != string(after) {
			t.Fatal("rejected banking changed state")
		}
	}
	reject()
	for _, drop := range r.Drops {
		r.Player.X = drop.X
		r.Player.Y = drop.Y
		r.tick(Input{}, .02)
	}
	reject()
	if r.Practice.Completed {
		t.Fatal("pickup skipped checkpoint")
	}
	r.Player.X = r.Practice.GoalX
	r.tick(Input{}, .02)
	if !r.Practice.CheckpointReady || r.Practice.Completed || r.Status != "fighting" {
		t.Fatal("checkpoint missing or auto banked")
	}
	raw, _ := json.Marshal(r)
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	r = &saved
	if err = r.PracticeTool("practice_bank"); err != nil {
		t.Fatal(err)
	}
	if !r.Practice.Completed || r.Status != "complete" {
		t.Fatal("confirmation failed")
	}
	for _, drop := range r.Drops {
		if !drop.Banked || drop.Gold != 0 || drop.Gear != nil || drop.NeedsGear {
			t.Fatal("invalid demonstration receipt")
		}
	}
	if r.Gold != 0 || r.BankedGold != 0 || len(r.BankedItems) != 0 || len(r.History) != 0 {
		t.Fatal("demo changed economy")
	}
	if err = r.ResetPractice(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if r.Practice.CheckpointReady || r.Practice.Completed || r.Drops[0].Banked {
		t.Fatal("reset retained bank state")
	}
	other, _ := NewPracticeRun("other", testRun().Build, "pickup", time.Now())
	if other.PracticeTool("practice_bank") == nil {
		t.Fatal("other drill accepted demo banking")
	}
}
