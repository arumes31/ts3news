package main

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

func TestHazardCommandCoversAllRoomsAndValidatesWindow(t *testing.T) {
	var out, diagnostics bytes.Buffer
	if code := execute([]string{"-seconds=60"}, &out, &diagnostics); code != 0 {
		t.Fatalf("exit=%d: %s", code, diagnostics.String())
	}
	var rows []hazardRoomReport
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 300 {
		t.Fatalf("rows=%d", len(rows))
	}
	for i, row := range rows {
		if row.Mission != i/3+1 || row.Tier != i%3+1 || row.Horizon != 60 || row.Error != "" || len(row.Intervals) != 0 {
			t.Fatalf("unexpected row: %+v", row)
		}
	}
	for _, args := range [][]string{{"-seconds=0"}, {"-seconds=NaN"}, {"-seconds=3601"}, {"unexpected"}} {
		if execute(args, io.Discard, io.Discard) != 2 {
			t.Fatalf("invalid arguments accepted: %v", args)
		}
	}
}
