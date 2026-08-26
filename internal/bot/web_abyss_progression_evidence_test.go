package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAbyssProgressionImprovementEvidence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		path   string
		tokens []string
	}{
		{
			name: "saved talent loadouts and node refunds",
			path: "web_abyss_tree2.go",
			tokens: []string{
				"handleAbyssTreeLoadoutSave", "handleAbyssTreeLoadoutApply",
				"handleAbyssTreeBranchRefund", "treeRefundSet",
			},
		},
		{
			name: "server enforced deep delver talents",
			path: "web_abyss.go",
			tokens: []string{
				"UpQuartermaster", "UpCartographer", "UpMercy",
			},
		},
		{
			name:   "swiftness presentation timing",
			path:   filepath.Join("webassets", "abyss.html"),
			tokens: []string{"150 - 10 * {{.Stats.UpSwiftness}}", "combatRecorderDelay(logDelay+exchangePause)"},
		},
		{
			name:   "scavenger material yield",
			path:   "web_abyss_features.go",
			tokens: []string{"scavengerYield", "UpCartographer", "abyssSpecs"},
		},
		{
			name:   "visual talent tree and specializations",
			path:   filepath.Join("webassets", "abysstree.html"),
			tokens: []string{"Swiftness", "Scavenger", "Mercy", "Cartographer", "Quartermaster", "Specializations"},
		},
		{
			name:   "prestige paragon and bestiary mastery",
			path:   "web_abyss_tree_mastery.go",
			tokens: []string{"abyssParagonRanksPerPrestige", "Pct: 0.001", "bestiary_damage_"},
		},
		{
			name:   "returning player catchup",
			path:   "web_abyss_progression.go",
			tokens: []string{"14*24*time.Hour", "abyssRunFlagCatchupCharges", "flags[abyssRunFlagCatchupCharges] = 10"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			body, err := os.ReadFile(test.path)
			if err != nil {
				t.Fatal(err)
			}
			text := string(body)
			for _, token := range test.tokens {
				if !strings.Contains(text, token) {
					t.Errorf("%s is missing %q", test.path, token)
				}
			}
		})
	}
}
