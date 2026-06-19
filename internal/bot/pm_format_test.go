package bot

import (
	"strings"
	"testing"
)

func countLines(s string) int {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func TestWritePackedBullets(t *testing.T) {
	// The full gear loadout from the example PM (one item ~22 bytes).
	gear := []string{
		"Novice Neck (47 dur)", "Novice Shoulders (47 dur)", "Novice Back (47 dur)",
		"Novice Chest (47 dur)", "Novice Wrists (47 dur)", "Novice Hands (47 dur)",
		"Novice Waist (47 dur)", "Novice Legs (47 dur)", "Novice Feet (47 dur)",
		"Novice Finger1 (47 dur)", "Novice Finger2 (47 dur)", "Novice Trinket1 (47 dur)",
		"Novice Trinket2 (47 dur)", "Novice MainHand (47 dur)", "Novice OffHand (47 dur)",
	}
	var sb strings.Builder
	writePackedBullets(&sb, gear, pmLineWidth)
	out := sb.String()

	lines := countLines(out)
	if lines >= len(gear) {
		t.Errorf("packing did not reduce lines: %d items -> %d lines", len(gear), lines)
	}
	// Every item must still be present.
	for _, g := range gear {
		if !strings.Contains(out, g) {
			t.Errorf("packed output dropped %q", g)
		}
	}
	// No packed line should greatly exceed the soft width (allow one item's slop).
	for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if len(ln) > pmLineWidth+40 {
			t.Errorf("line exceeds soft width: %d bytes: %q", len(ln), ln)
		}
	}
}

func TestWritePackedBulletsLongItem(t *testing.T) {
	long := strings.Repeat("x", pmLineWidth+50)
	var sb strings.Builder
	writePackedBullets(&sb, []string{"short", long, "tail"}, pmLineWidth)
	if got := countLines(sb.String()); got != 3 {
		t.Errorf("expected an over-long item on its own line (3 lines), got %d:\n%s", got, sb.String())
	}
}

func TestIsStandaloneCombatLine(t *testing.T) {
	standalone := []string{
		"⚔️ WAVE 1 of 1 [CR:676]: 2x Lvl 5 Corrupted Guard",
		"📍 Deadwind Pass [Diff: 1.61]",
		"📊 [center][size=12]BATTLE SUMMARY[/size][/center]",
		"🏁 VICTORY! Party defeated all 5 mobs.",
	}
	packable := []string{
		"☠️ Giant Zombie defeated by Daniel!",
		"🔥 Undead Skeleton cast Arcane Drain!",
		"⚠️ AMBUSH! Enemies attack first!",
	}
	for _, n := range standalone {
		if !isStandaloneCombatLine(n) {
			t.Errorf("expected standalone: %q", n)
		}
	}
	for _, n := range packable {
		if isStandaloneCombatLine(n) {
			t.Errorf("expected packable: %q", n)
		}
	}
}

func TestWriteCombatLogKeepsOrderAndPacksKills(t *testing.T) {
	notes := []string{
		"📍 Deadwind Pass [Diff: 1.61]",
		"⚔️ WAVE 1 of 1 [CR:676]: big wave",
		"⚠️ AMBUSH! Enemies attack first!",
		"☠️ Giant Zombie defeated by Daniel!",
		"☠️ Corrupted Guard defeated by Daniel!",
		"🔥 Frost Lich cast Holy Shield!",
		"🏁 VICTORY!",
	}
	var sb strings.Builder
	writeCombatLog(&sb, notes, pmLineWidth)
	out := sb.String()

	// Fewer lines than notes (the 4 short kill/cast/ambush lines collapse).
	if got := countLines(out); got >= len(notes) {
		t.Errorf("combat log not packed: %d notes -> %d lines", len(notes), got)
	}
	// Structural headers stay on their own line.
	for _, head := range []string{"📍 Deadwind Pass", "⚔️ WAVE 1 of 1", "🏁 VICTORY!"} {
		var found bool
		for _, ln := range strings.Split(out, "\n") {
			if strings.Contains(ln, head) && !strings.Contains(ln, " | ") {
				found = true
			}
		}
		if !found {
			t.Errorf("structural line not standalone: %q\n%s", head, out)
		}
	}
	// The zone header must still precede the victory line (order preserved).
	if strings.Index(out, "Deadwind Pass") > strings.Index(out, "VICTORY") {
		t.Errorf("combat order not preserved:\n%s", out)
	}
}
