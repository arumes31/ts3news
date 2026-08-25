package bot

import (
	"strings"
	"testing"
)

func TestAbyssBuffBadgesUseAuthoritativeRestartSafeDurations(t *testing.T) {
	t.Parallel()
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	source := string(page)
	for _, required := range []string{
		"activeBuffState(pageConsumables)",
		"syncActiveBuffsFromConsumables(pageConsumables)",
		"item.Type==='Buff'",
		"Number(item.Duration)",
		"renderConsumables already synchronized the server-decremented durations",
		"without running a local countdown",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("authoritative buff UI is missing %q", required)
		}
	}
	if strings.Contains(source, "var activeBuffs=[]") || strings.Contains(source, "function tickBuffs()") {
		t.Error("buff badges still depend on a restart-unsafe client countdown")
	}
}
