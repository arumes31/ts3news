package main

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
)

func TestReachabilityCommandReportsEveryRoom(t *testing.T) {
	var out, diagnostics bytes.Buffer
	if code := execute([]string{"-seed=author-test"}, &out, &diagnostics); code != 0 {
		t.Fatalf("exit=%d: %s", code, diagnostics.String())
	}
	var rows []roomReport
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 300 {
		t.Fatalf("rooms=%d", len(rows))
	}
	for i, row := range rows {
		if row.Mission != i/3+1 || row.Tier != i%3+1 || row.Targets < 1 || len(row.Failures) != 0 {
			t.Fatalf("unexpected row: %+v", row)
		}
	}
	if execute([]string{"unexpected"}, io.Discard, io.Discard) != 2 {
		t.Fatal("positional argument accepted")
	}
}
