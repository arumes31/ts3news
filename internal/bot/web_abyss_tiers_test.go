package bot

import (
	"strings"
	"testing"
)

func TestDecodeAbyssTierCatalog(t *testing.T) {
	t.Parallel()

	tiers, err := decodeAbyssTierCatalog(embeddedAbyssTierCatalog)
	if err != nil {
		t.Fatal(err)
	}
	if got := tiers["insanity"].DiffMult; got != 20 {
		t.Fatalf("insanity difficulty multiplier = %v, want 20", got)
	}
	if got := tiers["nightmare"].MinBest; got != 15 {
		t.Fatalf("nightmare minimum depth = %d, want 15", got)
	}
}

func TestDecodeAbyssTierCatalogRejectsUnsafeCatalogs(t *testing.T) {
	t.Parallel()

	valid := string(embeddedAbyssTierCatalog)
	tests := []struct {
		name    string
		catalog string
		wantErr string
	}{
		{"unknown field", strings.Replace(valid, `"name": "Normal"`, `"name": "Normal", "surprise": true`, 1), "unknown field"},
		{"wrong key order", strings.Replace(valid, `"key": "normal"`, `"key": "hell"`, 1), `key must be "normal"`},
		{"missing tier", valid[:strings.LastIndex(valid, ",\n  {")] + "\n]", "expected 4 tiers"},
		{"zero multiplier", strings.Replace(valid, `"difficulty_multiplier": 1.0`, `"difficulty_multiplier": 0`, 1), "finite and positive"},
		{"negative entry", strings.Replace(valid, `"entry_gold": 500`, `"entry_gold": -1`, 1), "must not be negative"},
		{"unordered depth", strings.Replace(valid, `"minimum_best_depth": 30`, `"minimum_best_depth": 5`, 1), "must be non-negative and ordered"},
		{"trailing value", valid + "{}", "multiple values"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeAbyssTierCatalog([]byte(test.catalog))
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("decode error = %v, want substring %q", err, test.wantErr)
			}
		})
	}
}
