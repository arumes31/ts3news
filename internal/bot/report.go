package bot

import (
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

// report.go holds pure, dependency-light BBCode "visual" formatters for combat
// reports, progression summaries and atmosphere. Everything here is a pure
// function so it can be unit-tested without a live bot, DB or TS3 connection.
// Structural pieces (bars, colour wrapping, emoji maps) stay in code; anything
// the player reads as prose goes through i18n so it localises.

// ─── Colour palette (BBCode hex) ───
const (
	colDmg   = "#ff5555" // incoming/enemy damage
	colHeal  = "#55ff55" // healing / party damage
	colCrit  = "#ffcc00" // critical hits, fanfare
	colXP    = "#7faaff" // experience
	colMuted = "#9e9e9e" // lore, secondary text
	colWarn  = "#ff9900" // durability / warnings
	colGood  = "#55ff55"
	colBad   = "#ff5555"
)

// barFill builds a fixed-width progress bar like [|||||-----].
// cur is clamped to [0,max]; width is the number of cells.
func barFill(cur, maxV, width int) string {
	if width < 1 {
		width = 1
	}
	if maxV <= 0 {
		maxV = 1
	}
	if cur < 0 {
		cur = 0
	}
	if cur > maxV {
		cur = maxV
	}
	filled := int(float64(cur) / float64(maxV) * float64(width))
	if filled > width {
		filled = width
	}
	// A tiny non-zero value should still show one pip so "barely alive" reads.
	if filled == 0 && cur > 0 {
		filled = 1
	}
	return "[" + strings.Repeat("|", filled) + strings.Repeat("-", width-filled) + "]"
}

// hpRatioColor grades a 0..1 health ratio green→orange→red.
func hpRatioColor(ratio float64) string {
	switch {
	case ratio >= 0.6:
		return colGood
	case ratio >= 0.3:
		return colWarn
	default:
		return colBad
	}
}

// hpBar renders a colour-graded HP bar with a cur/max readout, e.g.
// [color=#55ff55][||||||----][/color] 120/200.
func hpBar(cur, maxV int) string {
	if maxV <= 0 {
		maxV = 1
	}
	ratio := float64(cur) / float64(maxV)
	bar := colorWrap(hpRatioColor(ratio), barFill(cur, maxV, 10))
	return fmt.Sprintf("%s %d/%d", bar, clampNonNeg(cur), maxV)
}

// xpBar renders a 0..1 progress fraction as a coloured bar plus percentage.
func xpBar(pct float64) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	return fmt.Sprintf("%s %d%%", colorWrap(colXP, barFill(int(pct*100), 100, 10)), int(pct*100))
}

// damageBar shows one side's contribution to total damage as a share bar.
func damageBar(part, total int) string {
	return barFill(part, total, 12)
}

// colorWrap wraps text in a BBCode colour tag.
func colorWrap(hex, s string) string { return "[color=" + hex + "]" + s + "[/color]" }

// colorDmg / colorHeal colourise a number for damage / healing readability.
func colorDmg(n int) string  { return colorWrap(colDmg, fmt.Sprintf("%d", n)) }
func colorHeal(n int) string { return colorWrap(colHeal, "+"+fmt.Sprintf("%d", n)) }

// mobTypeColor returns a rarity colour for a mob type (WoW-ish quality tiers).
func mobTypeColor(t content.MobType) string {
	switch t {
	case content.MobCommon:
		return "#cfcfcf" // grey
	case content.MobEliteMinion:
		return "#ffffff" // white
	case content.MobElite:
		return "#1eff00" // green
	case content.MobMiniboss:
		return "#0070dd" // blue
	case content.MobBoss:
		return "#a335ee" // purple
	case content.MobLegendary:
		return "#ff8000" // orange/legendary
	default:
		return "#cfcfcf"
	}
}

// mobTypeIcon returns a small rank glyph for a mob type.
func mobTypeIcon(t content.MobType) string {
	switch t {
	case content.MobElite:
		return "⭐"
	case content.MobMiniboss:
		return "✦"
	case content.MobBoss:
		return "👑"
	case content.MobLegendary:
		return "🌟"
	default:
		return ""
	}
}

// colorMobName wraps a mob's display string in its rarity colour, prefixing a
// rank glyph for elite-and-above so the threat reads at a glance.
func colorMobName(name string, t content.MobType) string {
	ico := mobTypeIcon(t)
	if ico != "" {
		name = ico + " " + name
	}
	return colorWrap(mobTypeColor(t), name)
}

// statusEmoji maps a mob status effect to an at-a-glance emoji.
func statusEmoji(e content.MobEffect) string {
	switch e {
	case content.EffectPoisoned:
		return "🤢"
	case content.EffectBlinded:
		return "🌫️"
	case content.EffectEnraged:
		return "💢"
	case content.EffectArmored:
		return "🛡️"
	case content.EffectFleet:
		return "💨"
	case content.EffectWeakened:
		return "🥀"
	case content.EffectRegen:
		return "💖"
	default:
		return "•"
	}
}

// statEmoji maps a short stat name to a consistent emoji used everywhere.
func statEmoji(stat string) string {
	switch strings.ToUpper(stat) {
	case "STR":
		return "💪"
	case "DEF":
		return "🛡️"
	case "SPD":
		return "💨"
	case "INT":
		return "🧠"
	case "LCK":
		return "🍀"
	case "HP":
		return "❤️"
	case "STA":
		return "🫀"
	default:
		return "▫️"
	}
}

// gearScoreDelta formats "GS 450 (+15)" with a coloured, signed delta.
func gearScoreDelta(newScore, oldScore int) string {
	d := newScore - oldScore
	if d == 0 {
		return fmt.Sprintf("GS %d", newScore)
	}
	return fmt.Sprintf("GS %d (%s)", newScore, signedColored(d))
}

// statDelta formats "💪 STR 50→60" with arrow and stat emoji.
func statDelta(stat string, from, to int) string {
	arrow := "→"
	col := colMuted
	switch {
	case to > from:
		col = colGood
	case to < from:
		col = colBad
	}
	return fmt.Sprintf("%s %s %s", statEmoji(stat), strings.ToUpper(stat),
		colorWrap(col, fmt.Sprintf("%d%s%d", from, arrow, to)))
}

// signedColored renders a signed integer green when positive, red when negative.
func signedColored(d int) string {
	if d >= 0 {
		return colorWrap(colGood, "+"+fmt.Sprintf("%d", d))
	}
	return colorWrap(colBad, fmt.Sprintf("%d", d))
}

// critFanfare returns a bold, coloured critical-hit banner.
func critFanfare() string {
	return "[b]" + colorWrap(colCrit, i18n.T("bot.combat.crit_banner")) + "[/b]"
}

// waveCountdown returns a localised "Wave 1 of 3".
func waveCountdown(w, total int) string {
	return i18n.T("bot.combat.wave_countdown", w, total)
}

// hr returns a BBCode horizontal rule.
func hr() string { return "[hr]" }

// centerHeader wraps a title centred, bold and slightly enlarged.
func centerHeader(title string) string {
	return "[center][size=12][b]" + title + "[/b][/size][/center]"
}

// zoneLore returns a deterministic atmospheric one-liner for a zone, picked
// from a localised pool keyed off the zone name so each zone reads consistently.
func zoneLore(zoneName string) string {
	pool := i18n.Pool("pool.combat.zone_lore")
	if len(pool) == 0 {
		return ""
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(zoneName))
	line := pool[int(h.Sum32())%len(pool)]
	return colorWrap(colMuted, "[i]"+line+"[/i]")
}

// seasonalPrefix returns a seasonal emoji for the given time, or "" off-season.
func seasonalPrefix(t time.Time) string {
	switch t.Month() {
	case time.October:
		return "🎃"
	case time.December:
		return "❄️"
	case time.January:
		return "❄️"
	case time.February:
		return "💝"
	default:
		return ""
	}
}

// deathPenaltyLine spells out the XP cost of a defeat for clarity.
func deathPenaltyLine(nickname string, lostXP int) string {
	return i18n.T("bot.combat.death_penalty", nickname, lostXP)
}

func clampNonNeg(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
