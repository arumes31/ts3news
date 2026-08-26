package bot

import (
	"strings"
	"testing"
)

func TestAbyssRingSocketUIAndRouteContracts(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	partial, err := webAssets.ReadFile("webassets/abyss_ring_sockets.html")
	if err != nil {
		t.Fatal(err)
	}
	combined := string(page) + string(partial)
	for _, marker := range []string{
		`{{template "abyssRingSocketControl" .}}`,
		`{{template "abyssRingSocketJS" .}}`,
		`id="btnForgeRingSocketReroll"`,
		`/api/abyss/reroll_ring_sockets`,
		`'reroll_ring_sockets'`,
		`slot === 'Finger1' || slot === 'Finger2'`,
		`gems > 3`,
	} {
		if !strings.Contains(combined, marker) {
			t.Errorf("ring socket UI is missing %q", marker)
		}
	}
}
