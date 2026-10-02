package rift

import (
	"strings"
	"testing"
)

func TestCampaignIdentityValidation(t *testing.T) {
	tests := []struct {
		name   string
		change func([]Level) []Level
		want   string
	}{
		{"valid", func(levels []Level) []Level { return levels }, ""},
		{"duplicate ID", func(levels []Level) []Level { levels[1].ID = levels[0].ID; return levels }, "duplicate mission ID 1"},
		{"duplicate name", func(levels []Level) []Level { levels[1].Name = levels[0].Name; return levels }, "duplicate mission name"},
		{"whitespace alias", func(levels []Level) []Level { levels[1].Name = "  " + levels[0].Name + "  "; return levels }, "duplicate mission name"},
		{"case alias", func(levels []Level) []Level { levels[1].Name = strings.ToUpper(levels[0].Name); return levels }, "duplicate mission name"},
		{"empty name", func(levels []Level) []Level { levels[0].Name = " \t"; return levels }, "empty mission name"},
		{"zero ID", func(levels []Level) []Level { levels[0].ID = 0; return levels }, "outside 1..100"},
		{"out of range ID", func(levels []Level) []Level { levels[0].ID = LevelCount + 1; return levels }, "outside 1..100"},
		{"reordered IDs", func(levels []Level) []Level { levels[0], levels[1] = levels[1], levels[0]; return levels }, "position 1 has mission ID 2"},
		{"missing mission", func(levels []Level) []Level { return levels[:len(levels)-1] }, "expected 100 missions"},
		{"empty campaign", func(levels []Level) []Level { return nil }, "expected 100 missions"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			levels := test.change(Campaign())
			err := validateCampaignIdentity(levels)
			if test.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("got %v, want error containing %q", err, test.want)
			}
		})
	}
}
