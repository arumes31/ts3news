package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestValidationRunsEveryGateAndPreservesFailures(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	runner := func(args []string, out, diagnostics io.Writer) error {
		calls++
		if _, err := io.WriteString(out, "fixture output"); err != nil {
			return err
		}
		if calls == 2 {
			return errors.New("population budget failed")
		}
		return nil
	}
	passed, err := runChecks(context.Background(), dir, 8, 1, 30, "fixture", runner, io.Discard)
	if err != nil || passed || calls != 4 {
		t.Fatalf("passed=%v calls=%d err=%v", passed, calls, err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var results []checkResult
	if err = json.Unmarshal(raw, &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 4 || results[1].Passed || results[1].Error == "" || !results[3].Passed {
		t.Fatalf("results=%+v", results)
	}
	for _, result := range results {
		if _, err := os.Stat(filepath.Join(dir, result.Output)); err != nil {
			t.Fatal(err)
		}
	}
}
func TestValidationCancellationStopsFurtherGates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "summary.json"), []byte(`[{"passed":true}]`), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := runChecks(ctx, dir, 8, 1, 30, "fixture", func([]string, io.Writer, io.Writer) error { calls++; return nil }, io.Discard)
	if !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var results []checkResult
	if err = json.Unmarshal(raw, &results); err != nil {
		t.Fatal(err)
	}
	if len(results) != 4 {
		t.Fatal("stale summary retained")
	}
	for _, result := range results {
		if result.Passed || result.Error != "not run" {
			t.Fatalf("stale success: %+v", result)
		}
	}
}
