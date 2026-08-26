package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCalculateAbyssFirstStrikeScalesAndCapsRelativeSpeed(t *testing.T) {
	tests := []struct {
		name        string
		eligible    bool
		attacker    int
		attackerMod float64
		target      int
		targetMod   float64
		want        abyssFirstStrikeBonus
	}{
		{name: "not opening phase", attacker: 200, target: 100},
		{name: "tie", eligible: true, attacker: 100, target: 100, want: abyssFirstStrikeBonus{AttackerSPD: 100, TargetSPD: 100}},
		{name: "slight edge", eligible: true, attacker: 101, target: 100, want: abyssFirstStrikeBonus{AttackerSPD: 101, TargetSPD: 100, BonusPct: 1}},
		{name: "half again as fast", eligible: true, attacker: 150, target: 100, want: abyssFirstStrikeBonus{AttackerSPD: 150, TargetSPD: 100, BonusPct: 10}},
		{name: "double speed reaches cap", eligible: true, attacker: 200, target: 100, want: abyssFirstStrikeBonus{AttackerSPD: 200, TargetSPD: 100, BonusPct: 20}},
		{name: "extreme edge remains capped", eligible: true, attacker: 900, target: 100, want: abyssFirstStrikeBonus{AttackerSPD: 900, TargetSPD: 100, BonusPct: 20}},
		{name: "effective modifiers", eligible: true, attacker: 100, attackerMod: 1.5, target: 100, targetMod: 0.8, want: abyssFirstStrikeBonus{AttackerSPD: 150, TargetSPD: 80, BonusPct: 17}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := calculateAbyssFirstStrike(test.eligible, test.attacker, test.attackerMod, test.target, test.targetMod)
			if got != test.want {
				t.Fatalf("calculateAbyssFirstStrike() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestApplyAbyssFirstStrikeUsesBoundedIntegerDamage(t *testing.T) {
	bonus := abyssFirstStrikeBonus{BonusPct: 17}
	if got := applyAbyssFirstStrike(100, bonus); got != 117 {
		t.Fatalf("applyAbyssFirstStrike(100) = %d, want 117", got)
	}
	if got := applyAbyssFirstStrike(1, bonus); got != 1 {
		t.Fatalf("small integer damage = %d, want no artificial minimum bonus", got)
	}
	if got := applyAbyssFirstStrike(-5, bonus); got != 0 {
		t.Fatalf("negative damage = %d, want 0", got)
	}
}

func TestAbyssFirstStrikeLogIsExactAndSafelyRendered(t *testing.T) {
	bonus := abyssFirstStrikeBonus{AttackerSPD: 150, TargetSPD: 100, BonusPct: 10}
	line := abyssFirstStrikeLog(`<img src=x onerror=alert(1)>[color=#f00]`, "Target", bonus)
	if !strings.Contains(line, "SPD 150 vs 100") || !strings.Contains(line, "+10%") {
		t.Fatalf("first-strike log lacks exact values: %q", line)
	}
	if strings.Contains(line, "[color") {
		t.Fatalf("first-strike actor retained BBCode delimiters: %q", line)
	}
	html := abyssCombatLogHTML(line)
	if strings.Contains(html, "<img") || !strings.Contains(html, "&lt;img") {
		t.Fatalf("first-strike actor was not escaped: %q", html)
	}
}

func TestAbyssFirstStrikeIntegrationAndUIContract(t *testing.T) {
	root := abyssAAARepositoryRoot(t)
	xpSource, err := os.ReadFile(filepath.Join(root, "internal", "bot", "xp.go"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	live, err := webAssets.ReadFile("webassets/abyss_live.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_first_strike.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{
		"w == 1 && r == 1",
		"openingMobSPD[mob] = abyssEffectiveSpeed",
		"openingPlayerPhase && h == 0",
		"applyAbyssFirstStrike(dmg, firstStrike)",
		"abyssFirstStrikeLog(u.Nickname, target.Name, firstStrike)",
	} {
		if !strings.Contains(string(xpSource), token) {
			t.Errorf("combat integration is missing %q", token)
		}
	}
	for _, token := range []string{"/static/abyss_first_strike.css", "ab-log-first-strike"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("Abyss page is missing %q", token)
		}
	}
	for _, token := range []string{"function appendLiveLogLine", "textContent=line", "ab-live-first-strike"} {
		if !strings.Contains(string(live), token) {
			t.Errorf("live log is missing %q", token)
		}
	}
	for _, token := range []string{".ab-log-line.ab-log-first-strike", ".ab-live-feed > .ab-live-first-strike", "prefers-reduced-motion"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("first-strike styles are missing %q", token)
		}
	}
}
