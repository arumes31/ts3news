package bot

import (
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestBarFill(t *testing.T) {
	cases := []struct {
		cur, max, width int
		want            string
	}{
		{0, 100, 10, "[----------]"},
		{100, 100, 10, "[||||||||||]"},
		{50, 100, 10, "[|||||-----]"},
		{1, 1000, 10, "[|---------]"}, // tiny but alive shows one pip
		{-5, 100, 10, "[----------]"},
		{200, 100, 10, "[||||||||||]"}, // clamps to max
		{5, 10, 0, "[|]"},              // width floored to 1
	}
	for _, c := range cases {
		if got := barFill(c.cur, c.max, c.width); got != c.want {
			t.Errorf("barFill(%d,%d,%d)=%q want %q", c.cur, c.max, c.width, got, c.want)
		}
	}
}

func TestBarFillZeroMaxNoPanic(t *testing.T) {
	if got := barFill(5, 0, 10); !strings.HasPrefix(got, "[") {
		t.Errorf("barFill with zero max should not panic, got %q", got)
	}
}

func TestHPBar(t *testing.T) {
	got := hpBar(120, 200)
	if !strings.Contains(got, "120/200") {
		t.Errorf("hpBar missing readout: %q", got)
	}
	if !strings.Contains(got, hpRatioColor(0.6)) {
		t.Errorf("hpBar should grade 0.6 ratio green-ish: %q", got)
	}
	// Negative current HP renders as 0, not a negative readout.
	if low := hpBar(-10, 100); !strings.Contains(low, "0/100") {
		t.Errorf("hpBar negative cur should clamp to 0: %q", low)
	}
}

func TestHPRatioColor(t *testing.T) {
	if hpRatioColor(0.9) != colGood || hpRatioColor(0.4) != colWarn || hpRatioColor(0.1) != colBad {
		t.Error("hpRatioColor grading wrong")
	}
}

func TestXPBar(t *testing.T) {
	if got := xpBar(0.5); !strings.Contains(got, "50%") {
		t.Errorf("xpBar(0.5) want 50%%, got %q", got)
	}
	if got := xpBar(2.0); !strings.Contains(got, "100%") {
		t.Errorf("xpBar over 1.0 should clamp to 100%%, got %q", got)
	}
	if got := xpBar(-1); !strings.Contains(got, "0%") {
		t.Errorf("xpBar negative should clamp to 0%%, got %q", got)
	}
}

func TestColorHelpers(t *testing.T) {
	if got := colorDmg(42); got != "[color="+colDmg+"]42[/color]" {
		t.Errorf("colorDmg=%q", got)
	}
	if got := colorHeal(7); got != "[color="+colHeal+"]+7[/color]" {
		t.Errorf("colorHeal=%q", got)
	}
}

func TestMobTypeColorDistinct(t *testing.T) {
	types := []content.MobType{
		content.MobCommon, content.MobEliteMinion, content.MobElite,
		content.MobMiniboss, content.MobBoss, content.MobLegendary,
	}
	seen := map[string]bool{}
	for _, ty := range types {
		c := mobTypeColor(ty)
		if !strings.HasPrefix(c, "#") {
			t.Errorf("mobTypeColor(%s)=%q not a hex", ty, c)
		}
		seen[c] = true
	}
	if len(seen) < 5 {
		t.Errorf("expected mostly-distinct rarity colours, got %d unique", len(seen))
	}
}

func TestColorMobName(t *testing.T) {
	boss := colorMobName("Ancient Dragon", content.MobBoss)
	if !strings.Contains(boss, mobTypeColor(content.MobBoss)) || !strings.Contains(boss, "👑") {
		t.Errorf("boss name should be coloured and crowned: %q", boss)
	}
	common := colorMobName("Rat", content.MobCommon)
	if strings.Contains(common, "👑") {
		t.Errorf("common mob should have no rank glyph: %q", common)
	}
}

func TestStatusEmoji(t *testing.T) {
	if statusEmoji(content.EffectPoisoned) != "🤢" {
		t.Error("poison emoji wrong")
	}
	if statusEmoji(content.MobEffect("nonexistent")) != "•" {
		t.Error("unknown effect should fall back")
	}
}

func TestStatEmoji(t *testing.T) {
	if statEmoji("str") != "💪" || statEmoji("DEF") != "🛡️" {
		t.Error("stat emoji mapping wrong / not case-insensitive")
	}
	if statEmoji("ZZZ") != "▫️" {
		t.Error("unknown stat should fall back")
	}
}

func TestGearScoreDelta(t *testing.T) {
	if got := gearScoreDelta(450, 450); got != "GS 450" {
		t.Errorf("no-change delta should omit parens: %q", got)
	}
	up := gearScoreDelta(465, 450)
	if !strings.Contains(up, "GS 465") || !strings.Contains(up, "+15") || !strings.Contains(up, colGood) {
		t.Errorf("upgrade delta wrong: %q", up)
	}
	down := gearScoreDelta(440, 450)
	if !strings.Contains(down, "-10") || !strings.Contains(down, colBad) {
		t.Errorf("downgrade delta wrong: %q", down)
	}
}

func TestStatDelta(t *testing.T) {
	up := statDelta("STR", 50, 60)
	if !strings.Contains(up, "50→60") || !strings.Contains(up, "💪") || !strings.Contains(up, colGood) {
		t.Errorf("statDelta up wrong: %q", up)
	}
	down := statDelta("DEF", 30, 20)
	if !strings.Contains(down, colBad) {
		t.Errorf("statDelta down should be red: %q", down)
	}
}

func TestCritFanfareAndHeaders(t *testing.T) {
	if cf := critFanfare(); !strings.Contains(cf, "[b]") || !strings.Contains(cf, colCrit) {
		t.Errorf("critFanfare missing bold/colour: %q", cf)
	}
	if h := centerHeader("X"); !strings.Contains(h, "[center]") || !strings.Contains(h, "[b]X[/b]") {
		t.Errorf("centerHeader wrong: %q", h)
	}
	if hr() != "[hr]" {
		t.Error("hr wrong")
	}
}

func TestWaveCountdown(t *testing.T) {
	if got := waveCountdown(1, 3); !strings.Contains(got, "1") || !strings.Contains(got, "3") {
		t.Errorf("waveCountdown should mention 1 and 3: %q", got)
	}
}

func TestZoneLoreDeterministic(t *testing.T) {
	a := zoneLore("Volcanic Eruption")
	b := zoneLore("Volcanic Eruption")
	if a != b {
		t.Errorf("zoneLore not deterministic: %q vs %q", a, b)
	}
	if a != "" && !strings.Contains(a, "[i]") {
		t.Errorf("zoneLore should be italicised: %q", a)
	}
}

func TestSeasonalPrefix(t *testing.T) {
	if seasonalPrefix(time.Date(2026, time.October, 31, 0, 0, 0, 0, time.UTC)) != "🎃" {
		t.Error("October should be pumpkin")
	}
	if seasonalPrefix(time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)) != "" {
		t.Error("July should be off-season")
	}
}

func TestDeathPenaltyLine(t *testing.T) {
	if got := deathPenaltyLine("Hero", 150); !strings.Contains(got, "Hero") || !strings.Contains(got, "150") {
		t.Errorf("deathPenaltyLine wrong: %q", got)
	}
}

func TestSignedColored(t *testing.T) {
	if !strings.Contains(signedColored(5), "+5") || !strings.Contains(signedColored(5), colGood) {
		t.Error("positive signedColored wrong")
	}
	if !strings.Contains(signedColored(-3), "-3") || !strings.Contains(signedColored(-3), colBad) {
		t.Error("negative signedColored wrong")
	}
}
