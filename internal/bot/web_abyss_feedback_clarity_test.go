package bot

import (
	"os"
	"strings"
	"testing"
)

func TestAbyssUI141Through150Contracts(t *testing.T) {
	t.Parallel()

	pageBytes, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_ui200.css")
	if err != nil {
		t.Fatal(err)
	}
	goFiles := []string{"web_abyss.go", "web_abyss_loot.go", "xp.go"}
	server := ""
	for _, path := range goFiles {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		server += string(content)
	}
	page, css := string(pageBytes), string(styles)

	for _, contract := range []string{
		`if(d.new_record) recordBurst(d.depth)`,
		`if(d.global_record) recordBurst(d.depth,true)`,
		`if(d.pity_proc) pityProcFlash()`,
		`ab-starburst`, `ab-downed-beat`, `method: 'HEAD'`,
		`action') + ' need state review`, `action')+' manually`, `function coinArc()`, `function drainEscrow()`,
		`5*60*1000`, `ab_ui_ver`,
	} {
		if !strings.Contains(page, contract) {
			t.Errorf("Abyss feedback UI missing %q", contract)
		}
	}
	for _, contract := range []string{
		`.ab-depth-ring.ab-recordburst`, `.abyss-pity-hud.ab-pity-proc`,
		`.toast2.ab-starburst`, `.ab-btn-row.ab-downed-beat`,
		`button.primary:active:not(:disabled)`, `.ab-coin-arc`, `.ab-escrow.ab-crumble`, `.ab-newdot`,
	} {
		if !strings.Contains(css, contract) {
			t.Errorf("Abyss feedback stylesheet missing %q", contract)
		}
	}
	for _, contract := range []string{
		`NewRecord`, `PityProc`, `"new_record"`, `"global_record"`, `"pity_proc"`,
		`legendaryPity >= abyssLegendaryPityCap`, `pityProc = true`,
	} {
		if !strings.Contains(server, contract) {
			t.Errorf("Abyss feedback server signal missing %q", contract)
		}
	}
	for _, forbidden := range []string{
		`previousPity`, `bestNow`, `registerAbyssSafeRetry`, `opts.retry`, `ab-toast-retry`,
	} {
		if strings.Contains(page, forbidden) {
			t.Errorf("Abyss feedback still contains unsafe or inferred state %q", forbidden)
		}
	}
}
