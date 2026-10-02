package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestComboDecaysAndRestarts(t *testing.T) {
	r := testRun()
	r.Status = "cleared"
	r.Enemies = nil
	r.tick(Input{Attack: true}, 0)
	r.tick(Input{}, .5)
	r.tick(Input{Attack: true}, 0)
	if r.Combo != 2 {
		t.Fatal("timely second attack did not chain")
	}
	r.tick(Input{}, 1.21)
	if r.Combo != 0 {
		t.Fatalf("idle combo=%d, want zero", r.Combo)
	}
	r.tick(Input{Attack: true}, 0)
	if r.Combo != 1 {
		t.Fatal("expired chain did not restart at one")
	}
	if r.Stats.HighestCombo != 2 {
		t.Fatal("decay erased highest combo record")
	}
}

func TestComboDecaySurvivesSaveAndPause(t *testing.T) {
	r := testRun()
	r.Status = "cleared"
	r.Enemies = nil
	r.tick(Input{Attack: true}, 0)
	r.tick(Input{}, .7)
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	restored.Paused = true
	restored.Step(Input{}, time.UnixMilli(restored.LastMS+200))
	if restored.Combo != 1 {
		t.Fatal("paused combo decayed")
	}
	restored.Paused = false
	restored.tick(Input{}, .4)
	if restored.Combo != 1 {
		t.Fatal("saved combo expired too early")
	}
	restored.tick(Input{}, .11)
	if restored.Combo != 0 {
		t.Fatal("save or pause extended combo lifetime")
	}
}

func TestComboCooldownDoesNotRefreshAndLegacyComboExpires(t *testing.T) {
	r := testRun()
	r.Status = "cleared"
	r.Enemies = nil
	r.tick(Input{Attack: true}, 0)
	r.tick(Input{Attack: true}, .1)
	if r.Combo != 1 || r.ComboTime >= comboWindow {
		t.Fatal("rejected attack refreshed combo")
	}
	legacy := testRun()
	legacy.Status = "cleared"
	legacy.Enemies = nil
	legacy.Combo = 2
	legacy.tick(Input{}, .1)
	if legacy.Combo != 2 {
		t.Fatal("legacy combo lost its initial grace window")
	}
	legacy.tick(Input{}, 1.11)
	if legacy.Combo != 0 {
		t.Fatal("legacy combo failed to expire")
	}
}
