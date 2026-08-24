package bot

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"testing"
)

func TestAbyssForgeImprovementEvidenceGate(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository for forge evidence audit")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	payload, err := os.ReadFile(filepath.Join(root, "docs", "abyss-forge-evidence.json"))
	if err != nil {
		t.Fatalf("read forge evidence: %v", err)
	}
	var groups []struct {
		From     int      `json:"from"`
		To       int      `json:"to"`
		Evidence []string `json:"evidence"`
	}
	if err := json.Unmarshal(payload, &groups); err != nil {
		t.Fatalf("decode forge evidence: %v", err)
	}
	covered := make(map[int]bool, 200)
	for _, group := range groups {
		if group.From < 1 || group.To > 200 || group.From > group.To || len(group.Evidence) == 0 {
			t.Errorf("invalid evidence group: %+v", group)
			continue
		}
		for number := group.From; number <= group.To; number++ {
			if covered[number] {
				t.Errorf("improvement %d has overlapping evidence groups", number)
			}
			covered[number] = true
		}
		for _, relative := range group.Evidence {
			info, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(relative)))
			if statErr != nil || info.IsDir() {
				t.Errorf("evidence %q is missing or not a file", relative)
			}
		}
	}
	for number := 1; number <= 200; number++ {
		if !covered[number] {
			t.Errorf("improvement %d has no evidence group", number)
		}
	}

	ledger, err := os.Open(filepath.Join(root, "docs", "plans", "2026-08-24-abyss-forge-200.md"))
	if err != nil {
		t.Fatalf("open forge improvement ledger: %v", err)
	}
	defer func() { _ = ledger.Close() }()
	checkedLine := regexp.MustCompile(`^- \[x\] ([0-9]+) `)
	checked := make(map[int]bool, 200)
	scanner := bufio.NewScanner(ledger)
	for scanner.Scan() {
		match := checkedLine.FindStringSubmatch(scanner.Text())
		if len(match) != 2 {
			continue
		}
		number, parseErr := strconv.Atoi(match[1])
		if parseErr != nil || number < 1 || number > 200 || checked[number] {
			t.Errorf("invalid or duplicate checked ledger number %q", match[1])
			continue
		}
		checked[number] = true
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan forge improvement ledger: %v", err)
	}
	for number := 1; number <= 200; number++ {
		if !checked[number] {
			t.Errorf("forge improvement %d is not checked in the ledger", number)
		}
	}
}
