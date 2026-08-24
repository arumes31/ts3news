package bot

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"ts3news/internal/content"
)

func FuzzAbyssTreePlanPreview(f *testing.F) {
	f.Add([]byte{1, 0, 2, 0, 3, 0}, uint16(100), uint8(30))
	f.Add([]byte{255, 255, 0, 0}, uint16(0), uint8(0))
	f.Fuzz(func(t *testing.T, raw []byte, points uint16, depth uint8) {
		if len(raw) > 400 {
			raw = raw[:400]
		}
		ids := make([]int, 0, len(raw)/2)
		for index := 0; index+1 < len(raw); index += 2 {
			ids = append(ids, int(raw[index])<<8|int(raw[index+1]))
		}
		analysis := analyzeAbyssTreePlan(content.AbyssTree(), nil, ids, int(points), int(depth), 0, -1)
		if len(analysis.IDs) > len(ids) || analysis.PlannedCost < 0 || analysis.CurrentCost < 0 {
			t.Fatalf("invalid preview invariants: %+v", analysis)
		}
		if analysis.Valid && abyssTreePlanCommitError(analysis) != "" {
			t.Fatalf("valid preview is not commit-compatible: %+v", analysis)
		}
	})
}

func FuzzAbyssTreeBuildCodeImport(f *testing.F) {
	valid, _ := json.Marshal(abyssTreeBuildCode{
		Version: 1, Schema: content.TreeCatalogSchemaVersion,
		Layout: content.AbyssTree().TopologyHash(), IDs: []int{1, 2, 3},
	})
	f.Add(base64.StdEncoding.EncodeToString(valid))
	f.Add("not-base64")
	f.Add("")
	f.Fuzz(func(t *testing.T, encoded string) {
		if len(encoded) > 200000 {
			encoded = encoded[:200000]
		}
		code, err := decodeAbyssTreeBuildCode(encoded)
		if err == nil {
			if code.Version != 1 || code.Schema != content.TreeCatalogSchemaVersion || code.Layout == "" || len(code.IDs) == 0 || len(code.IDs) > abyssTreePlanMaxNodes {
				t.Fatalf("accepted invalid build code: %+v", code)
			}
		}
	})
}
