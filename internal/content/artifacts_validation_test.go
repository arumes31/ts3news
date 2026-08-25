package content

import (
	"strings"
	"testing"
)

func TestValidateGearCatalog(t *testing.T) {
	t.Parallel()
	if err := ValidateGearCatalog(); err != nil {
		t.Fatalf("production gear catalog: %v", err)
	}
}

func TestValidateGearCatalogRejectsBrokenContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		gear    []Gear
		effects []ItemEffect
		want    string
	}{
		{
			name: "empty ID",
			gear: []Gear{{Name: "Nameless"}},
			want: "empty ID",
		},
		{
			name: "duplicate ID",
			gear: []Gear{{ID: "same"}, {ID: "same"}},
			want: "duplicate gear ID",
		},
		{
			name: "undersized set",
			gear: []Gear{{ID: "single", SetID: "lonely"}},
			want: "want at least 2",
		},
		{
			name: "undocumented gear effect",
			gear: []Gear{{ID: "mystery", Special: ItemEffect("Mystery")}},
			want: "undocumented effect",
		},
		{
			name:    "undocumented registered effect",
			gear:    []Gear{{ID: "plain"}},
			effects: []ItemEffect{ItemEffect("Mystery")},
			want:    "no player-facing description",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := validateGearCatalog(test.gear, test.effects)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}
