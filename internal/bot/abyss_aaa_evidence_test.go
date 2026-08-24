package bot

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

type abyssAAAEvidence struct {
	IDs          []int    `json:"ids"`
	Production   []string `json:"production"`
	Verification []string `json:"verification"`
}

func TestAbyssAAAProgramContainsExactlyOneThousandSuggestions(t *testing.T) {
	t.Parallel()

	root := abyssAAARepositoryRoot(t)
	ledgers := []struct {
		name    string
		path    string
		pattern *regexp.Regexp
		count   int
	}{
		{name: "core", path: "ABYSS_IDEAS.md", pattern: regexp.MustCompile(`^([0-9]+)\.`), count: 400},
		{name: "ux", path: "ABYSS_UX_IDEAS.md", pattern: regexp.MustCompile(`^- \*\*UX-([0-9]+)\*\*`), count: 100},
		{name: "ui", path: "ABYSS_UI_200.md", pattern: regexp.MustCompile(`^([0-9]+)\.`), count: 200},
		{name: "extended", path: "ABYSS_IMPROVEMENTS_300.md", pattern: regexp.MustCompile(`^([0-9]+)\.`), count: 300},
	}

	total := 0
	for _, ledger := range ledgers {
		ledger := ledger
		t.Run(ledger.name, func(t *testing.T) {
			t.Parallel()
			got := abyssAAALedgerNumbers(t, filepath.Join(root, "docs", ledger.path), ledger.pattern)
			if len(got) != ledger.count {
				t.Fatalf("suggestion count = %d, want %d", len(got), ledger.count)
			}
			for index, number := range got {
				want := index + 1
				if number != want {
					t.Fatalf("suggestion %d has source ID %d, want %d", index, number, want)
				}
			}
		})
		total += ledger.count
	}
	if total != 1000 {
		t.Fatalf("global suggestion count = %d, want 1000", total)
	}
}

func TestAbyssAAAEvidenceReferencesUniqueItemsAndExistingFiles(t *testing.T) {
	t.Parallel()

	root := abyssAAARepositoryRoot(t)
	path := filepath.Join(root, "internal", "bot", "testdata", "abyss_aaa_evidence.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Abyss AAA evidence: %v", err)
	}
	var entries []abyssAAAEvidence
	if err := json.Unmarshal(content, &entries); err != nil {
		t.Fatalf("decode Abyss AAA evidence: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("Abyss AAA evidence ledger is empty")
	}

	seen := make(map[int]struct{})
	for entryIndex, entry := range entries {
		if len(entry.IDs) == 0 || len(entry.Production) == 0 || len(entry.Verification) == 0 {
			t.Fatalf("evidence entry %d is incomplete", entryIndex)
		}
		for _, id := range entry.IDs {
			if id < 1 || id > 1000 {
				t.Fatalf("evidence entry %d has invalid AAA ID %d", entryIndex, id)
			}
			if _, duplicate := seen[id]; duplicate {
				t.Fatalf("AAA-%04d has duplicate evidence", id)
			}
			seen[id] = struct{}{}
		}
		for _, evidencePath := range append(entry.Production, entry.Verification...) {
			clean := filepath.Clean(evidencePath)
			if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
				t.Fatalf("evidence entry %d escapes the repository: %q", entryIndex, evidencePath)
			}
			info, err := os.Stat(filepath.Join(root, clean))
			if err != nil {
				t.Fatalf("evidence entry %d references %q: %v", entryIndex, evidencePath, err)
			}
			if info.IsDir() {
				t.Fatalf("evidence entry %d references directory %q", entryIndex, evidencePath)
			}
		}
	}
}

func abyssAAARepositoryRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository for Abyss AAA evidence audit")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
}

func abyssAAALedgerNumbers(t *testing.T, path string, pattern *regexp.Regexp) []int {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open suggestion ledger: %v", err)
	}
	defer func() { _ = file.Close() }()

	numbers := make([]int, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		match := pattern.FindStringSubmatch(scanner.Text())
		if len(match) != 2 {
			continue
		}
		number, err := strconv.Atoi(match[1])
		if err != nil {
			t.Fatalf("parse source ID %q: %v", match[1], err)
		}
		numbers = append(numbers, number)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan suggestion ledger: %v", err)
	}
	return numbers
}
