package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestPreviewIsDeterministicAcrossEveryMission(t *testing.T) {
	for mission := 1; mission <= 100; mission++ {
		var first, second, diagnostics bytes.Buffer
		args := []string{fmt.Sprintf("-mission=%d", mission), "-seed=preview-check"}
		if execute(args, &first, &diagnostics) != 0 || execute(args, &second, &diagnostics) != 0 {
			t.Fatal(diagnostics.String())
		}
		if !bytes.Equal(first.Bytes(), second.Bytes()) {
			t.Fatalf("mission %d is nondeterministic", mission)
		}
		var got missionPreview
		if err := json.Unmarshal(first.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Schema != 1 || got.Level.ID != mission || len(got.Tiers) != 3 || got.Seed != "preview-check" || got.Catalog != "canonical_abyss" {
			t.Fatalf("invalid preview for %d", mission)
		}
		for index, tier := range got.Tiers {
			if tier.Tier != index+1 || len(tier.PlannedEnemies) == 0 || len(tier.EntryActors) == 0 || tier.Arena.Name == "" {
				t.Fatalf("missing tier %d in mission %d", index+1, mission)
			}
			if tier.Arena.Objective == "survive_waves" && len(tier.EntryActors) >= len(tier.PlannedEnemies) {
				t.Fatal("wave reserves presented as entry actors")
			}
		}
	}
}
func TestPreviewSeedChangesEncounters(t *testing.T) {
	var a, b, diagnostics bytes.Buffer
	if execute([]string{"-seed=one"}, &a, &diagnostics) != 0 || execute([]string{"-seed=two"}, &b, &diagnostics) != 0 {
		t.Fatal(diagnostics.String())
	}
	var first, second missionPreview
	if err := json.Unmarshal(a.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b.Bytes(), &second); err != nil {
		t.Fatal(err)
	}
	left, _ := json.Marshal(first.Tiers)
	right, _ := json.Marshal(second.Tiers)
	if bytes.Equal(left, right) {
		t.Fatal("seed did not affect encounters")
	}
}
func TestPreviewRejectsInvalidOptions(t *testing.T) {
	for _, args := range [][]string{{"-mission=0"}, {"-mission=101"}, {"-seed="}, {"-seed=  "}, {"-seed=" + strings.Repeat("x", 129)}, {"extra"}, {"-unknown"}} {
		var out, diagnostics bytes.Buffer
		if execute(args, &out, &diagnostics) != 2 || out.Len() != 0 || diagnostics.Len() == 0 {
			t.Fatalf("accepted %v", args)
		}
	}
	if execute([]string{"-h"}, io.Discard, io.Discard) != 0 {
		t.Fatal("help failed")
	}
	if execute(nil, failedWriter{}, io.Discard) != 1 {
		t.Fatal("output failure hidden")
	}
}

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
