package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAbyssUI141Through160Contracts(t *testing.T) {
	t.Parallel()

	assets := map[string]string{}
	for _, name := range []string{
		"webassets/abyss.html",
		"webassets/abyss_accessibility.html",
		"webassets/abyss_accessibility.css",
		"webassets/abyss_live.html",
		"webassets/abyss_ui200.css",
		"webassets/abyss_longterm.html",
		"webassets/partials.html",
	} {
		content, err := webAssets.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		assets[name] = string(content)
	}

	page := assets["webassets/abyss.html"] + assets["webassets/abyss_longterm.html"]
	for _, contract := range []string{
		`if(d.new_record) recordBurst(d.depth)`,
		`if(d.pity_proc) pityProcFlash()`,
		`ab-starburst`, `ab-downed-beat`, `netFailed`, `btn-busy`,
		`method: 'HEAD'`, `need state review`, `manually`,
		`coinArc()`, `drainEscrow()`, `initIdleReminder()`, `initNewDots()`,
		`id="abyssDepthDial" role="group"`, `function updateDepthA11y()`,
		`aria-live="off"`, `id="vaultOverlay" aria-hidden="true" inert`,
		`id="bossCard" aria-hidden="true" inert`,
		`key:'ab_colorblind'`,
	} {
		if !strings.Contains(page, contract) {
			t.Errorf("Abyss page missing UI contract %q", contract)
		}
	}
	for _, inferred := range []string{`d.depth>best`, `previousPity`} {
		if strings.Contains(page, inferred) {
			t.Errorf("Abyss page still infers authoritative feedback with %q", inferred)
		}
	}
	for _, unsafe := range []string{
		`registerAbyssSafeRetry`, `clearAbyssSafeRetry`, `opts.retry`,
		`ab-toast-retry`, `function retryable`, `retry your last action`,
	} {
		if strings.Contains(page, unsafe) {
			t.Errorf("Abyss page contains unsafe mutation-retry contract %q", unsafe)
		}
	}

	a11y := assets["webassets/abyss_accessibility.html"]
	for _, contract := range []string{
		`700-(Date.now()-lastAnnouncement)`, `applyAbyssColorblind`,
		`var rarityGlyphs=['●','■','▲','◆'`,
		`function initStageSwipe()`, `confirmModal('Swipe gesture selected `,
		`function initLongPressDetails()`, `550`, `setAttribute('aria-label'`,
		`setAttribute('aria-hidden','true')`,
	} {
		if !strings.Contains(a11y, contract) {
			t.Errorf("Abyss accessibility layer missing contract %q", contract)
		}
	}

	css := assets["webassets/abyss_accessibility.css"] + assets["webassets/abyss_ui200.css"]
	for _, contract := range []string{
		`.ab-skip-link:focus`, `body.ab-colorblind .ab-rarity-glyph`,
		`.ab-longpress-active`, `body.ab-high-contrast`,
		`@media (prefers-reduced-motion: reduce)`, `.ab-recordburst`,
		`.ab-pity-proc`, `.ab-starburst`, `@keyframes ab-heartbeat`, `.ab-coin-arc`,
		`.ab-newdot`,
	} {
		if !strings.Contains(css, contract) {
			t.Errorf("Abyss styles missing contract %q", contract)
		}
	}

	if !strings.Contains(assets["webassets/partials.html"], `<a class="ab-skip-link" href="#abyssControls">`) {
		t.Error("shared shell missing first-tab-stop skip link")
	}
	if !strings.Contains(assets["webassets/abyss_live.html"], `id="liveFeed" aria-live="off"`) {
		t.Error("live combat feed must not announce every log mutation")
	}
}

func TestAbyssRecordAndPitySignalsAreServerAuthoritative(t *testing.T) {
	t.Parallel()

	root := abyssAAARepositoryRoot(t)
	contracts := map[string][]string{
		"internal/bot/web_abyss.go": {
			`NewRecord: depth > st.BestDepth`, `"new_record"`,
			`"pity_proc"`, `"global_record": isRecord`,
		},
		"internal/bot/web_abyss_loot.go": {
			`PityProc bool`, `pityProc = true`, `PityProc: pityProc`,
		},
		"internal/bot/xp.go": {`PityProc bool`},
	}
	for name, required := range contracts {
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		for _, contract := range required {
			if !strings.Contains(string(content), contract) {
				t.Errorf("%s missing authoritative feedback contract %q", name, contract)
			}
		}
	}
}
