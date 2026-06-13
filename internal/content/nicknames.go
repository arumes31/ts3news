package content

import (
	"math/rand/v2"
	"strings"
	"ts3news/internal/i18n"
	"unicode"
)

// franchiseNicks maps a keyword found in a game's title to a themed nickname the
// bot adopts while announcing that game (feature: "VaultBoy" for Fallout, etc.).
// Uses lazy initialization to ensure i18n is loaded before T() calls.
var franchiseNicks map[string]string

func initFranchiseNicks() {
	if franchiseNicks != nil {
		return
	}
	franchiseNicks = map[string]string{
		"fallout":        i18n.T("content.nickname.vaultboy"),
		"elder":          i18n.T("content.nickname.dovahkiin"),
		"skyrim":         i18n.T("content.nickname.dovahkiin"),
		"witcher":        i18n.T("content.nickname.geralt_of_rivia"),
		"doom":           i18n.T("content.nickname.doomguy"),
		"halo":           i18n.T("content.nickname.master_chief"),
		"zelda":          i18n.T("content.nickname.hero_of_time"),
		"mario":          i18n.T("content.nickname.itsa_me"),
		"sonic":          i18n.T("content.nickname.gotta_go_fast"),
		"portal":         i18n.T("content.nickname.still_alive"),
		"minecraft":      i18n.T("content.nickname.steve"),
		"tomb":           i18n.T("content.nickname.lara_croft"),
		"assassin":       i18n.T("content.nickname.ezio_auditore"),
		"god of war":     i18n.T("content.nickname.kratos"),
		"resident":       i18n.T("content.nickname.stars_member"),
		"silent":         i18n.T("content.nickname.foggy_hill"),
		"dark souls":     i18n.T("content.nickname.chosen_undead"),
		"elden":          i18n.T("content.nickname.tarnished"),
		"cyberpunk":      i18n.T("content.nickname.choom_v"),
		"grand theft":    i18n.T("content.nickname.wanted_level_5"),
		"counter":        i18n.T("content.nickname.rush_b"),
		"half-life":      i18n.T("content.nickname.freeman_phd"),
		"metroid":        i18n.T("content.nickname.samus_aran"),
		"metal gear":     i18n.T("content.nickname.solid_snake"),
		"final fantasy":  i18n.T("content.nickname.chocobo_rider"),
		"star wars":      i18n.T("content.nickname.use_the_force"),
		"borderlands":    i18n.T("content.nickname.vault_hunter"),
		"diablo":         i18n.T("content.nickname.nephalem_hero"),
		"warcraft":       i18n.T("content.nickname.for_the_horde"),
		"overwatch":      i18n.T("content.nickname.payload_escort"),
		"bioshock":       i18n.T("content.nickname.would_you_kindly"),
		"far cry":        i18n.T("content.nickname.rookie_fc"),
		"hitman":         i18n.T("content.nickname.agent_47"),
		"mortal":         i18n.T("content.nickname.flawless_victory"),
		"street fighter": i18n.T("content.nickname.hadouken"),
		"pac-man":        i18n.T("content.nickname.waka_waka"),
		"tetris":         i18n.T("content.nickname.tspin_triple"),
		"among us":       i18n.T("content.nickname.not_the_impostor"),
		"terraria":       i18n.T("content.nickname.moon_lord"),
		"stardew":        i18n.T("content.nickname.junimo_farmer"),
		"hollow":         i18n.T("content.nickname.the_knight"),
	}
}

// gamerSuffixes are appended to a derived nick when no franchise keyword matches.
var gamerSuffixes = i18n.Pool("pool.gamer_suffix")

// NicknameForGame returns a themed TeamSpeak nickname for the bot to use while
// announcing the given game. It first looks for a known franchise keyword; if
// none match it derives a clean nick from the title. The result is always a
// valid, reasonably short (<= 28 char) nickname.
func NicknameForGame(title string) string {
	initFranchiseNicks()
	lower := strings.ToLower(title)
	for keyword, nick := range franchiseNicks {
		if strings.Contains(lower, keyword) {
			return clampNick(nick)
		}
	}

	// Fallback: PascalCase the first couple of significant words + a gamer suffix.
	var word strings.Builder
	wordCount := 0
	for _, w := range strings.Fields(title) {
		clean := keepAlnum(w)
		if clean == "" {
			continue
		}
		word.WriteString(capitalize(strings.ToLower(clean)))
		wordCount++
		if wordCount >= 2 {
			break
		}
	}
	base := word.String()
	if base == "" {
		base = "MrFree"
	}
	// Safety check for empty pool (can happen during init before i18n is fully loaded)
	suffixes := gamerSuffixes
	if len(suffixes) == 0 {
		suffixes = []string{"Gamer", "Hero", "Pro"}
	}
	// #nosec G404
	suffix := suffixes[rand.IntN(len(suffixes))] // #nosec G404
	return clampNick(base + suffix)
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func keepAlnum(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func clampNick(s string) string {
	const max = 28
	if len(s) > max {
		return s[:max]
	}
	return s
}
