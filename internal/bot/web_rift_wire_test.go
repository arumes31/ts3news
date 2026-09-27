package bot

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftLeanSnapshotReconstructsExactSavedRun(t *testing.T) {
	run := largeRiftReceipt(t)
	run.Status = "fighting"
	initial, err := riftWireSnapshot(run, "load", "")
	if err != nil {
		t.Fatal(err)
	}
	token := initial["snapshot_base"].(string)
	before, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	run.Player.X += 10
	run.Revision++
	run.Counter++
	lean, err := riftWireSnapshot(run, "step", token)
	if err != nil {
		t.Fatal(err)
	}
	if lean["snapshot_kind"] != "lean-v1" {
		t.Fatal("movement was not compact")
	}
	wire, err := json.Marshal(lean["run"])
	if err != nil {
		t.Fatal(err)
	}
	full, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	var old, decoded, want map[string]json.RawMessage
	json.Unmarshal(before, &old)
	json.Unmarshal(wire, &decoded)
	json.Unmarshal(full, &want)
	for key := range riftRetainedFields(run) {
		if _, exists := decoded[key]; exists {
			t.Fatal("retained field resent", key)
		}
		if value, exists := old[key]; exists {
			decoded[key] = value
		}
	}
	if !reflect.DeepEqual(decoded, want) {
		t.Fatal("reconstructed snapshot differs from saved run")
	}
	if len(wire)*2 >= len(full) {
		t.Fatalf("large receipt reduction below half: %d/%d", len(wire), len(full))
	}
	t.Logf("large campaign run: full%d bytes, lean%d bytes", len(full), len(wire))
	after, _ := json.Marshal(run)
	if string(after) != string(full) {
		t.Fatal("projection mutated saved run")
	}
}
func TestRiftLeanSnapshotFallsBackForDifferentBaselineOrTerminal(t *testing.T) {
	run := largeRiftReceipt(t)
	run.Status = "fighting"
	first, _ := riftWireSnapshot(run, "load", "")
	token := first["snapshot_base"].(string)
	for _, action := range []string{"load", "start", "pause", "next"} {
		got, err := riftWireSnapshot(run, action, token)
		if err != nil || got["snapshot_kind"] != "full" {
			t.Fatal(action)
		}
	}
	got, _ := riftWireSnapshot(run, "step", "stale")
	if got["snapshot_kind"] != "full" {
		t.Fatal("unknown base accepted")
	}
	run.Build.Name = "new frozen build"
	got, _ = riftWireSnapshot(run, "step", token)
	if got["snapshot_kind"] != "full" || got["snapshot_base"] == token {
		t.Fatal("changed build retained")
	}
	token = got["snapshot_base"].(string)
	run.Status = "cleared"
	got, _ = riftWireSnapshot(run, "step", token)
	if got["snapshot_kind"] != "full" {
		t.Fatal("terminal delta")
	}
	run.Status = "fighting"
	run.ID = "different-run"
	got, _ = riftWireSnapshot(run, "step", token)
	if got["snapshot_kind"] != "full" {
		t.Fatal("cross-run baseline")
	}
}

func BenchmarkRiftWireSnapshot(b *testing.B) {
	run, err := buildLargeRiftReceipt(rift.Build{HP: 300}, time.Unix(100, 0))
	if err != nil {
		b.Fatal(err)
	}
	run.Status = "fighting"
	first, err := riftWireSnapshot(run, "load", "")
	if err != nil {
		b.Fatal(err)
	}
	token := first["snapshot_base"].(string)
	b.Run("full", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			if _, err := json.Marshal(map[string]any{"ok": true, "run": run}); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("lean_including_token", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			response, err := riftWireSnapshot(run, "step", token)
			if err != nil {
				b.Fatal(err)
			}
			if _, err = json.Marshal(response); err != nil {
				b.Fatal(err)
			}
		}
	})
}
