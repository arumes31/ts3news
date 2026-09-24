package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"testing"
)

func TestPopulationReportCoversCampaignAndSeparatesWavesAndProps(t *testing.T) {
	rows := populationReport(8)
	if len(rows) != 300 {
		t.Fatalf("rows=%d", len(rows))
	}
	waves, props, largest := 0, 0, 0
	for i, row := range rows {
		if row.Mission != i/3+1 || row.Tier != i%3+1 {
			t.Fatalf("missing room at %d", i)
		}
		if row.PlannedEnemies < row.EntryEnemies || row.EntryEnemies < 1 || row.OverBudget {
			t.Fatalf("invalid counts: %+v", row)
		}
		largest = max(largest, row.PlannedEnemies)
		if row.Objective == "survive_waves" {
			waves++
			if row.EntryEnemies >= row.PlannedEnemies {
				t.Fatal("later waves counted as entry enemies")
			}
		}
		if row.ObjectiveActors > 0 {
			props++
		}
	}
	if waves == 0 || props == 0 || largest != 8 {
		t.Fatalf("missing budget cases: waves=%d props=%d largest=%d", waves, props, largest)
	}
}
func TestPopulationCommandReportsViolationsAndFormats(t *testing.T) {
	var out, diagnostics bytes.Buffer
	if code := execute([]string{"-format=json", "-max-enemies=1"}, &out, &diagnostics); code != 1 {
		t.Fatalf("exit=%d", code)
	}
	var rows []populationRow
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 300 {
		t.Fatal("report stopped at first violation")
	}
	for _, row := range rows {
		if !row.OverBudget {
			t.Fatal("missing violation")
		}
	}
	out.Reset()
	diagnostics.Reset()
	if code := execute(nil, &out, &diagnostics); code != 0 {
		t.Fatalf("exit=%d: %s", code, diagnostics.String())
	}
	records, err := csv.NewReader(&out).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 301 || len(records[0]) != 10 {
		t.Fatal("invalid CSV shape")
	}
	for _, args := range [][]string{{"-max-enemies=0"}, {"-format=xml"}, {"unexpected"}} {
		if code := execute(args, io.Discard, io.Discard); code != 2 {
			t.Fatalf("invalid options accepted: %v", args)
		}
	}
}
