package bot

import (
	"os"
	"strings"
	"testing"
)

type fixedAbyssDramaRandom struct {
	roll int
	line int
	call int
}

func (r *fixedAbyssDramaRandom) IntN(n int) int {
	r.call++
	if r.call == 1 {
		return r.roll % n
	}
	return r.line % n
}

func TestAbyssCriticalFumbleHasExactOnePercentRollBoundary(t *testing.T) {
	t.Parallel()

	triggered := 0
	for roll := range 100 {
		random := &fixedAbyssDramaRandom{roll: roll, line: 2}
		line := abyssCriticalFumbleLog("Hero", "Warden", true, random)
		if line == "" {
			if random.call != 1 {
				t.Fatalf("roll %d used %d draws without an event", roll, random.call)
			}
			continue
		}
		triggered++
		if random.call != 2 || !strings.Contains(line, "CRITICAL FUMBLE") ||
			!strings.Contains(line, "No combat effect") ||
			!strings.Contains(line, "Hero") || !strings.Contains(line, "Warden") {
			t.Fatalf("roll %d produced %q after %d draws", roll, line, random.call)
		}
	}
	if triggered != 1 {
		t.Fatalf("triggered events = %d, want exactly one across rolls 0-99", triggered)
	}
}

func TestAbyssCriticalFumbleIsSuppressedOutsideAbyss(t *testing.T) {
	t.Parallel()

	random := &fixedAbyssDramaRandom{}
	if line := abyssCriticalFumbleLog("Hero", "Warden", false, random); line != "" {
		t.Fatalf("non-Abyss fumble = %q", line)
	}
	if random.call != 0 {
		t.Fatalf("non-Abyss fumble consumed %d cosmetic draws", random.call)
	}
}

func TestAbyssCriticalFumbleSafelyRendersCombatantNames(t *testing.T) {
	t.Parallel()

	random := &fixedAbyssDramaRandom{}
	line := abyssCriticalFumbleLog(
		`<img src=x onerror=alert(1)>[color=#f00]`,
		`Warden[/color]`,
		true,
		random,
	)
	if strings.Contains(line, "[color") || strings.Contains(line, "[/color]") {
		t.Fatalf("fumble retained BBCode delimiters: %q", line)
	}
	html := abyssCombatLogHTML(line)
	if strings.Contains(html, "<img") || !strings.Contains(html, "&lt;img") {
		t.Fatalf("fumble combatant markup was not escaped: %q", html)
	}
}

func TestAbyssCriticalFumbleCombatContractHasNoMechanicalBranch(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("xp.go")
	if err != nil {
		t.Fatal(err)
	}
	combat := string(source)
	for _, removed := range []string{
		"dmg = dmg / 2",
		"au.fumbled",
		"embarrassed rage",
		"rand.Float64() < 0.03",
	} {
		if strings.Contains(combat, removed) {
			t.Errorf("legacy mechanical fumble remains: %q", removed)
		}
	}
	call := `abyssCriticalFumbleLog(u.Nickname, target.Name, true, defaultAbyssDramaRandom{})`
	if !strings.Contains(combat, call) {
		t.Fatalf("combat does not use the independent cosmetic source: want %q", call)
	}
}
