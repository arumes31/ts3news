package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"
	"testing"
)

func TestPopulationReportCoversCampaignAndSeparatesWavesAndProps(t *testing.T) {
	rows := populationReport(8, 1)
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
	if len(records) != 301 || len(records[0]) != 19 {
		t.Fatal("invalid CSV shape")
	}
	for _, args := range [][]string{{"-max-enemies=0"}, {"-format=xml"}, {"unexpected"}} {
		if code := execute(args, io.Discard, io.Discard); code != 2 {
			t.Fatalf("invalid options accepted: %v", args)
		}
	}
}

func TestBossBudgetCanFailIndependently(t *testing.T) {
	var out, diagnostics bytes.Buffer
	if code := execute([]string{"-format=json", "-max-bosses=0"}, &out, &diagnostics); code != 1 {
		t.Fatalf("exit=%d: %s", code, diagnostics.String())
	}
	var rows []struct {
		Tier       int  `json:"tier"`
		Planned    int  `json:"planned_bosses"`
		Entry      int  `json:"entry_bosses"`
		BossBudget int  `json:"boss_budget"`
		BossOver   bool `json:"boss_over_budget"`
		EnemyOver  bool `json:"over_budget"`
	}
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 300 {
		t.Fatal("incomplete report")
	}
	total := 0
	for _, row := range rows {
		want := 0
		if row.Tier == 3 {
			want = 1
		}
		if row.Planned != want || row.Entry != want || row.BossOver != (want > 0) || row.BossBudget != 0 || row.EnemyOver {
			t.Fatalf("wrong boss accounting: %+v", row)
		}
		total += row.Planned
	}
	if total != 100 {
		t.Fatalf("boss count=%d", total)
	}
	if code := execute([]string{"-max-bosses=-1"}, io.Discard, io.Discard); code != 2 {
		t.Fatal("negative boss budget accepted")
	}
}

func TestRewardCeilingsSeparateDropsFromFinalMissionBonus(t *testing.T) {
	var out bytes.Buffer
	if execute([]string{"-format=json"}, &out, io.Discard) != 0 {
		t.Fatal("report failed")
	}
	var rows []struct {
		Tier    int   `json:"tier"`
		Enemies int   `json:"planned_enemies"`
		Gold    int64 `json:"drop_gold_ceiling"`
		Gear    int   `json:"gear_count_ceiling"`
		Rarity  int   `json:"gear_rarity_ceiling"`
		Bonus   int64 `json:"objective_gold_ceiling"`
		Total   int64 `json:"total_gold_ceiling"`
	}
	if err := json.Unmarshal(out.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 300 {
		t.Fatal("missing rooms")
	}
	for _, row := range rows {
		bonus, rarity := int64(0), 3
		if row.Tier == 3 {
			// Fourteen offered challenges, including no-potions, at five gold each.
			bonus = 70
			rarity = 4
		}
		if row.Gold != int64(row.Enemies*15*row.Tier) || row.Gear != row.Enemies || row.Bonus != bonus || row.Rarity != rarity || row.Total != row.Gold+bonus {
			t.Fatalf("incorrect reward ceiling: %+v", row)
		}
	}
}
