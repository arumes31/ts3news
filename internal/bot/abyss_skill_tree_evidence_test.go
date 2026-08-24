package bot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAbyssSkillTreeImprovementEvidenceGate(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate repository for evidence audit")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "..", ".."))
	payload, err := os.ReadFile(filepath.Join(root, "docs", "abyss-skill-tree-evidence.json"))
	if err != nil {
		t.Fatalf("read skill-tree evidence: %v", err)
	}
	var groups []struct {
		From     int      `json:"from"`
		To       int      `json:"to"`
		Evidence []string `json:"evidence"`
	}
	if err := json.Unmarshal(payload, &groups); err != nil {
		t.Fatalf("decode skill-tree evidence: %v", err)
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
			info, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative)))
			if err != nil || info.IsDir() {
				t.Errorf("evidence %q is missing or not a file", relative)
			}
		}
	}
	for number := 1; number <= 200; number++ {
		if !covered[number] {
			t.Errorf("improvement %d has no evidence group", number)
		}
	}
}
