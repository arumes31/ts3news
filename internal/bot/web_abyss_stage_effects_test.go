package bot

import (
	"strings"
	"testing"
)

func TestAbyssOverkillRequiresMoreThanTwiceRemainingHealth(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name      string
		damage    int
		remaining int
		want      bool
	}{
		{name: "ordinary kill", damage: 120, remaining: 100},
		{name: "exactly double", damage: 200, remaining: 100},
		{name: "more than double", damage: 201, remaining: 100, want: true},
		{name: "already defeated", damage: 999, remaining: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := abyssOverkillHit(test.damage, test.remaining); got != test.want {
				t.Errorf("abyssOverkillHit(%d, %d) = %v, want %v", test.damage, test.remaining, got, test.want)
			}
		})
	}
}

func TestAbyssOverkillMarkerBecomesSafePresentationSignal(t *testing.T) {
	t.Parallel()

	marked := markAbyssOverkillLog("☠️ target defeated", true)
	html := abyssCombatLogHTML(marked)
	if !strings.Contains(html, `class="ab-overkill-signal" hidden`) {
		t.Fatalf("marked combat log lacks presentation signal: %q", html)
	}
	if strings.Contains(html, abyssOverkillMarker) {
		t.Fatalf("internal overkill marker leaked into rendered combat log: %q", html)
	}
	if plain := abyssCombatLogHTML("ordinary hit"); strings.Contains(plain, "ab-overkill-signal") {
		t.Fatalf("ordinary hit received overkill signal: %q", plain)
	}
}

func TestAbyssStageEffectsContracts(t *testing.T) {
	t.Parallel()

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatalf("read Abyss page: %v", err)
	}
	source := string(page)
	for _, required := range []string{
		"killsThisFloor=0; updateKillChip()",
		"function updateKillChip",
		"d.querySelector('.ab-overkill-signal')",
		"function stampOverkill",
		"function appendAbyssOverkillReceipt",
		"addLogHead(log,'ab-log-overkill-reward'",
		"d.classList.add('ab-log-kill')",
		"function markCrackedGear",
		"dur/maxDur<0.20",
		`data-max-dur="{{.MaxDurability}}"`,
		"animateNumber(el, curDepth, nd",
		"nd%10===0",
		"curMomentum >= 10 ? ' f3'",
		"if(/ENRAGED/.test(txt)) setStageEnraged(true)",
	} {
		if !strings.Contains(source, required) {
			t.Errorf("Abyss stage effects are missing %q", required)
		}
	}
	if strings.Contains(source, "if(fightMeta.actions<=3) stampOverkill") {
		t.Error("overkill presentation must not use the old action-count approximation")
	}

	uiCSS, err := webAssets.ReadFile("webassets/abyss_ui200.css")
	if err != nil {
		t.Fatalf("read UI-200 CSS: %v", err)
	}
	baseCSS, err := webAssets.ReadFile("webassets/style.css")
	if err != nil {
		t.Fatalf("read base CSS: %v", err)
	}
	styles := string(uiCSS) + string(baseCSS)
	for _, required := range []string{
		".ab-overkill",
		".ab-log-line.ab-log-overkill-reward",
		"@keyframes ab-kill-zoom",
		".ab-depth-ring.milestone",
		".abyss-side-gear.ab-cracked::after",
		".ab-threat-fill::after",
		"transition: width .5s ease",
		".ab-mom-flame.f2",
		".ab-mom-flame.f3",
		"@keyframes ab-enrage-breath",
		"@media (prefers-reduced-motion: reduce)",
	} {
		if !strings.Contains(styles, required) {
			t.Errorf("stage effects CSS is missing %q", required)
		}
	}
}
