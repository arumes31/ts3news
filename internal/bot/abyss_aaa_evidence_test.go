package bot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type abyssAAAEvidence struct {
	IDs          []int    `json:"ids"`
	Production   []string `json:"production"`
	Verification []string `json:"verification"`
}

func TestAbyssAAAEvidenceReferencesUniqueItemsAndExistingFiles(t *testing.T) {
	t.Parallel()

	root := abyssAAARepositoryRoot(t)
	entries := abyssAAAEvidenceEntries(t, root)
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

func abyssAAAEvidenceEntries(t *testing.T, root string) []abyssAAAEvidence {
	t.Helper()
	path := filepath.Join(root, "internal", "bot", "testdata", "abyss_aaa_evidence.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Abyss AAA evidence: %v", err)
	}
	var entries []abyssAAAEvidence
	if err := json.Unmarshal(content, &entries); err != nil {
		t.Fatalf("decode Abyss AAA evidence: %v", err)
	}
	return entries
}
