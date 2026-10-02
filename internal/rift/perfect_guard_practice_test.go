package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPerfectGuardPracticeUsesRealTimingAndIsolatedProgress(t *testing.T) {
	r, err := NewPracticeRun("perfect", testRun().Build, "perfect_guard", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 200; i++ {
		r.Step(Input{Guard: true}, time.UnixMilli(100000+int64(i)*50))
	}
	if r.Practice.Completed || r.Practice.PerfectGuards >= 3 || r.Stats.Guards < 3 {
		t.Fatal("holding guard completed timing drill or trainer did not attack")
	}
	if err := r.ResetPractice(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 600 && r.Status == "fighting"; i++ {
		guard := r.Enemies[0].Windup > 0 && r.Enemies[0].Windup <= .15
		r.Step(Input{Guard: guard}, time.UnixMilli(200000+int64(i)*50))
	}
	if !r.Practice.Completed || r.Practice.PerfectGuards != 3 {
		t.Fatalf("timed guards did not complete: %+v", r.Practice)
	}
	if r.Stats.PerfectGuards != 0 || len(r.History) != 0 || len(r.Drops) != 0 || r.Gold != 0 {
		t.Fatal("practice affected campaign records or rewards")
	}
	raw, _ := json.Marshal(r)
	var saved Run
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Practice.PerfectGuards != 3 {
		t.Fatal("progress lost on save")
	}
	if err := saved.ResetPractice(time.Unix(300, 0)); err != nil {
		t.Fatal(err)
	}
	if saved.Practice.PerfectGuards != 0 || saved.Practice.Completed {
		t.Fatal("reset retained timing progress")
	}
}
