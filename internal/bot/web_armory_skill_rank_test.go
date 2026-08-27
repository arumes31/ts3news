package bot

import (
	"strings"
	"testing"
)

func TestArmourySkillRankProgressContract(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/armory.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/style.css")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(page) + string(styles)
	for _, required := range []string{
		`class="sk-rank" role="progressbar"`,
		`aria-valuemax="9"`,
		`aria-valuenow="{{.UpgradeRank}}"`,
		`mulpct .UpgradeRank 9`,
		`@media (forced-colors: active)`,
	} {
		if !strings.Contains(combined, required) {
			t.Errorf("Armoury skill-rank contract missing %q", required)
		}
	}
}
