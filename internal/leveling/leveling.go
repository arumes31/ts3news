// Package leveling implements the bot's internal XP / leveling system: every time
// a user is poked they earn XP, which maps to one of 1000 fantasy-themed levels.
// Crossing configured level milestones can grant TeamSpeak server groups.
package leveling

import (
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

// MaxLevel is the highest attainable level.
const MaxLevel = 1000

// XP curve tuning.
//
// A user earns XP once per poke (one poke per *new* free game they receive).
// Assuming roughly one notification per day on average (there is usually at least
// one fresh Steam/Epic giveaway daily across all sources), reaching the level cap
// should take about ten years:
//
//	~1 poke/day * 365 * 10  ≈ 3650 pokes
//	avg XP per poke          = (xpMin+xpMax)/2 = 17.5
//	XP needed for level 1000 ≈ 3650 * 17.5     ≈ 63,875
//
// We model cumulative XP as XPForLevel(l) = xpCurveK * (l-1)^xpCurveExp. With
// exp 1.5 the climb is quick early and slow late; K is chosen so the cap lands
// near the ~64k target above (XPForLevel(1000) ≈ 63,150).
const (
	xpMin      = 10
	xpMax      = 25
	xpCurveK   = 2.0
	xpCurveExp = 1.5
)

// tierNames are 25 fantasy rank tiers; each spans 40 levels (25 * 40 = 1000).
var tierNames = []string{
	"Peasant", "Squire", "Footman", "Adventurer", "Mercenary",
	"Knight", "Ranger", "Warden", "Templar", "Crusader",
	"Vanguard", "Champion", "Slayer", "Warlord", "Battlemaster",
	"Conqueror", "Paladin", "Archon", "Warlock", "Sorcerer",
	"Mystic", "Demigod", "Ascendant", "Eternal", "Godslayer",
}

const levelsPerTier = MaxLevel / 25 // 40

// XPForLevel returns the cumulative XP required to reach the given level.
// Level 1 starts at 0 XP. See the curve-tuning notes above for the ~10-year design.
func XPForLevel(level int) int {
	if level <= 1 {
		return 0
	}
	if level > MaxLevel {
		level = MaxLevel
	}
	return int(math.Round(xpCurveK * math.Pow(float64(level-1), xpCurveExp)))
}

// LevelForXP returns the level (1..MaxLevel) corresponding to a total XP amount.
func LevelForXP(xp int) int {
	if xp <= 0 {
		return 1
	}
	level := 1
	for level < MaxLevel && XPForLevel(level+1) <= xp {
		level++
	}
	return level
}

// LevelName returns the fantasy name for a level, e.g. "Knight XII".
func LevelName(level int) string {
	if level < 1 {
		level = 1
	}
	if level > MaxLevel {
		level = MaxLevel
	}
	tier := (level - 1) / levelsPerTier
	if tier >= len(tierNames) {
		tier = len(tierNames) - 1
	}
	sub := (level-1)%levelsPerTier + 1 // 1..40
	return tierNames[tier] + " " + roman(sub)
}

// XPPerPoke returns a randomised XP award (used when the game price is unknown).
func XPPerPoke() int {
	return xpMin + rand.Intn(xpMax-xpMin+1) // xpMin..xpMax
}

// priceCapEUR is the price at which the price-based XP scale saturates.
const priceCapEUR = 60.0

// XPForPrice maps a game's original price to an XP award within [xpMin, xpMax].
// cheaperMoreXP=true gives more XP for cheaper games; false rewards pricier games.
// The midpoint price yields the average award, preserving the ~10-year curve.
func XPForPrice(priceEUR float64, cheaperMoreXP bool) int {
	if priceEUR < 0 {
		priceEUR = 0
	}
	if priceEUR > priceCapEUR {
		priceEUR = priceCapEUR
	}
	frac := priceEUR / priceCapEUR // 0 (free-ish) .. 1 (>= cap)
	if cheaperMoreXP {
		frac = 1 - frac
	}
	return int(math.Round(float64(xpMin) + frac*float64(xpMax-xpMin)))
}

// ParseLevelGroups parses a "level:groupID,level:groupID" string (e.g.
// "10:7,50:8,100:9") into a map of level -> server group id. Invalid entries
// are skipped.
func ParseLevelGroups(s string) map[int]int {
	out := map[int]int{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		l, g, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		lvl, err1 := strconv.Atoi(strings.TrimSpace(l))
		grp, err2 := strconv.Atoi(strings.TrimSpace(g))
		if err1 != nil || err2 != nil {
			continue
		}
		out[lvl] = grp
	}
	return out
}

// MilestonesCrossed returns the server group ids whose milestone level falls in
// the half-open interval (oldLevel, newLevel] — i.e. groups newly earned by
// leveling up from oldLevel to newLevel. The result is sorted by milestone level.
func MilestonesCrossed(oldLevel, newLevel int, groups map[int]int) []int {
	type ms struct{ level, group int }
	var crossed []ms
	for lvl, grp := range groups {
		if lvl > oldLevel && lvl <= newLevel {
			crossed = append(crossed, ms{lvl, grp})
		}
	}
	sort.Slice(crossed, func(i, j int) bool { return crossed[i].level < crossed[j].level })
	out := make([]int, len(crossed))
	for i, m := range crossed {
		out[i] = m.group
	}
	return out
}

// roman converts 1..3999 to a Roman numeral (used for sub-tier rank display).
func roman(n int) string {
	if n <= 0 {
		return "I"
	}
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
	var b strings.Builder
	for i, v := range vals {
		for n >= v {
			b.WriteString(syms[i])
			n -= v
		}
	}
	return b.String()
}
