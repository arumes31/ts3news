package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
	"ts3news/internal/rift"
)

func TestExportIsStableAndContainsCompleteCampaign(t *testing.T) {
	var first, second, diagnostics bytes.Buffer
	if execute(nil, &first, &diagnostics) != 0 || execute(nil, &second, &diagnostics) != 0 {
		t.Fatal(diagnostics.String())
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatal("export is nondeterministic")
	}
	var got campaignExport
	if err := json.Unmarshal(first.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Schema != 1 || got.MissionCount != 100 || got.RoomCount != 300 || got.RegionCount != 10 {
		t.Fatalf("incorrect summary: %+v", got)
	}
	if !reflect.DeepEqual(got.Missions, rift.Campaign()) {
		t.Fatal("export lost authored mission fields")
	}
	total := 0
	for _, count := range got.Objectives {
		total += count
	}
	if total != 300 || got.Objectives["combat"] == 0 || got.Objectives["survive_waves"] != 10 {
		t.Fatalf("incorrect objectives: %+v", got.Objectives)
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errors.New("output unavailable") }
func TestExportReportsUsageAndWriteFailures(t *testing.T) {
	for _, args := range [][]string{{"extra"}, {"-unknown"}} {
		var out, diagnostics bytes.Buffer
		if execute(args, &out, &diagnostics) != 2 || out.Len() != 0 || diagnostics.Len() == 0 {
			t.Fatal("invalid arguments were not rejected")
		}
	}
	var diagnostics bytes.Buffer
	if execute(nil, brokenWriter{}, &diagnostics) != 1 || diagnostics.Len() == 0 {
		t.Fatal("write failure hidden")
	}
	if execute([]string{"-h"}, io.Discard, &diagnostics) != 0 {
		t.Fatal("help failed")
	}
}
