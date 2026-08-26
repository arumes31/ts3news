package bot

import "strings"

const (
	abyssDoTMarker      = "[[ABYSS_DOT]]"
	abyssExecuteMarker  = "[[ABYSS_EXECUTE]]"
	abyssOverkillMarker = "[[ABYSS_OVERKILL]]"
)

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

func markAbyssDoTLog(line string, damageOverTime bool) string {
	if !damageOverTime {
		return line
	}
	return line + abyssDoTMarker
}

func markAbyssExecuteLog(line string, executeReady bool) string {
	if !executeReady {
		return line
	}
	return line + abyssExecuteMarker
}

func abyssCombatLogHTML(line string) string {
	damageOverTime := strings.Contains(line, abyssDoTMarker)
	executeReady := strings.Contains(line, abyssExecuteMarker)
	overkill := strings.Contains(line, abyssOverkillMarker)
	line = strings.ReplaceAll(line, abyssDoTMarker, "")
	line = strings.ReplaceAll(line, abyssExecuteMarker, "")
	line = strings.ReplaceAll(line, abyssOverkillMarker, "")
	html := bbToHTML(line)
	if damageOverTime {
		html += `<span class="ab-dot-signal" hidden></span>`
	}
	if executeReady {
		html += `<span class="ab-execute-signal" hidden></span>`
	}
	if overkill {
		html += `<span class="ab-overkill-signal" hidden></span>`
	}
	return html
}
