package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventHistoryBoundAcrossLongRun(t *testing.T) {
	run := NewRun("long-events", Build{HP: 100}, time.Now())
	before := run.Counter
	for i := 0; i < 10000; i++ {
		run.Land(.5)
		if len(run.Events) > 40 {
			t.Fatalf("event %d exceeded history bound", i)
		}
	}
	if len(run.Events) != 40 || run.Counter != before+10000 {
		t.Fatal("history bound changed the sequence counter")
	}
	for i, event := range run.Events {
		if event.ID != before+9961+i {
			t.Fatalf("event %d is not in the newest suffix", i)
		}
	}
	payload, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(payload, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Events) != 40 || saved.Counter != run.Counter {
		t.Fatal("saved history exceeded the live bound")
	}
}
