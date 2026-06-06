package content

import (
	"math/rand"
	"strings"
	"unicode"
)

// franchiseNicks maps a keyword found in a game's title to a themed nickname the
// bot adopts while announcing that game (feature: "VaultBoy" for Fallout, etc.).
var franchiseNicks = map[string]string{
	"fallout":     "VaultBoy",
	"elder":       "Dovahkiin",
	"skyrim":      "Dovahkiin",
	"witcher":     "GeraltOfRivia",
	"doom":        "Doomguy",
	"halo":        "MasterChief",
	"zelda":       "HeroOfTime",
	"mario":       "ItsaMe",
	"sonic":       "GottaGoFast",
	"portal":      "StillAlive",
	"minecraft":   "Steve",
	"tomb":        "LaraCroft",
	"assassin":    "EzioAuditore",
	"god of war":  "Kratos",
	"resident":    "STARSMember",
	"silent":      "FoggyHill",
	"dark souls":  "ChosenUndead",
	"elden":       "Tarnished",
	"cyberpunk":   "ChoomV",
	"grand theft": "WantedLevel5",
	"counter":     "RushB",
	"half-life":   "FreemanPhD",
	"metroid":     "SamusAran",
	"metal gear":  "SolidSnake",
	"final fantasy": "ChocoboRider",
	"star wars":   "UseTheForce",
	"borderlands": "VaultHunter",
	"diablo":      "NephalemHero",
	"warcraft":    "ForTheHorde",
	"overwatch":   "PayloadEscort",
	"bioshock":    "WouldYouKindly",
	"far cry":     "RookieFC",
	"hitman":      "Agent47",
	"mortal":      "FlawlessVictory",
	"street fighter": "Hadouken",
	"pac-man":     "WakaWaka",
	"tetris":      "TSpinTriple",
	"among us":    "NotTheImpostor",
	"terraria":    "MoonLord",
	"stardew":     "JunimoFarmer",
	"hollow":      "TheKnight",
}

// gamerSuffixes are appended to a derived nick when no franchise keyword matches.
var gamerSuffixes = []string{"Gamer", "Loot", "Hero", "Legend", "Pro", "Master", "Raider", "Hunter"}

// NicknameForGame returns a themed TeamSpeak nickname for the bot to use while
// announcing the given game. It first looks for a known franchise keyword; if
// none match it derives a clean nick from the title. The result is always a
// valid, reasonably short (<= 28 char) nickname.
func NicknameForGame(title string) string {
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
	suffix := gamerSuffixes[rand.Intn(len(gamerSuffixes))]
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
