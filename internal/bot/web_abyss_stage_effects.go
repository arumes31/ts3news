package bot

import "strings"

const abyssOverkillMarker = "[[ABYSS_OVERKILL]]"

func abyssOverkillHit(damage, remainingHP int) bool {
	if remainingHP <= 0 || damage <= remainingHP {
		return false
	}
	return damage-remainingHP > remainingHP
}

func markAbyssOverkillLog(line string, overkill bool) string {
	if !overkill {
		return line
	}
	return line + abyssOverkillMarker
}

func abyssCombatLogHTML(line string) string {
	overkill := strings.Contains(line, abyssOverkillMarker)
	line = strings.ReplaceAll(line, abyssOverkillMarker, "")
	html := bbToHTML(line)
	if overkill {
		html += `<span class="ab-overkill-signal" hidden></span>`
	}
	return html
}
