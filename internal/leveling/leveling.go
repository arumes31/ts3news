// Package leveling implements the bot's internal XP / leveling system: every time
// a user is poked they earn XP, which maps to one of 10000 fantasy-themed levels.
// Crossing configured level milestones can grant TeamSpeak server groups.
package leveling

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
)

// MaxLevel is the boundary for fixed tier names.
const MaxLevel = 10000

// absoluteMaxLevel bounds LevelForXP.
const absoluteMaxLevel = 1000000

const levelsPerTier = 30

// NumTiers is the number of level tiers (10000 / 30 ≈ 334).
const NumTiers = 334

// Procedural name pools used for late tiers and beyond.
var (
	epicAdjectives = []string{
		"Ascendant", "Eternal", "Astral", "Radiant", "Infernal", "Celestial", "Primordial", "Transcendent", "Mythic", "Abyssal",
		"Void-Touched", "Star-Forged", "Ancient", "Gilded", "Spectral", "Divine", "Forbidden", "Ethereal", "Storm-Born", "Shadow-Bound",
		"Nebulous", "Crystalline", "Forgotten", "Hallowed", "Vengeful", "Silent", "Shattered", "Burning", "Frozen", "Twisted",
	}
	epicTitles = []string{
		"Champion", "Warlord", "Sovereign", "Archon", "Overlord", "Demigod", "Titan", "Vanguard", "Conqueror", "Paragon",
		"Sentinel", "Exarch", "Oracle", "Deity", "Avatar", "Godslayer", "Reaper", "Prophet", "High-King", "Omni-Slayer",
		"Harbinger", "Executioner", "Watcher", "Keeper", "Lord", "Baron", "Shaman", "Sage", "Priest", "Templar",
	}
	epicRealms = []string{
		"Void", "Abyss", "Cosmos", "Eternity", "Storm", "Dawn", "Flame", "Frost", "Shadow", "Light",
		"Nebula", "Continuum", "Aether", "Hellfire", "Zero-Point", "Singularity", "Genesis", "Revelation", "Oblivion", "Nexus",
		"Underworld", "Sky-Reach", "Dream-World", "End-Times", "Sun-Forge", "Moon-Rise", "Deep-Sea", "Ever-Green", "Iron-Hold", "Dragon-Spire",
	}
)

// tierNames are the base fantasy rank names. We'll use a dense list and procedural expansion.
var baseTierNames = []string{
	"Drifter", "Pauper", "Peasant", "Thrall", "Bondman", "Scavenger", "Straggler", "Outcast", "Refugee", "Exile",
	"Vagabond", "Seeker", "Wanderer", "Nomad", "Initiate", "Neophyte", "Novice", "Apprentice", "Page", "Squire",
	"Footman", "Recruit", "Conscript", "Soldier", "Warrior", "Guard", "Sentry", "Watchman", "Warden", "Keeper",
	"Adventurer", "Mercenary", "Contractor", "Skirmisher", "Scout", "Ranger", "Pathfinder", "Tracker", "Hunter", "Stalker",
	"Knight", "Paladin", "Chevalier", "Cavalier", "Dragoon", "Templar", "Crusader", "Avenger", "Justicar", "Inquisitor",
	"Warlock", "Sorcerer", "Mage", "Wizard", "Arcanist", "Elementalist", "Summoner", "Necromancer", "Occultist", "Shaman",
	"Slayer", "Executioner", "Assassin", "Shinobi", "Ninja", "Raider", "Marauder", "Berserker", "Gladiator", "Combatant",
	"Battlemaster", "Commander", "Captain", "Major", "Colonel", "General", "Marshal", "Legate", "Centurion", "Tribune",
	"Champion", "Hero", "Legend", "Myth", "Vanguard", "Frontliner", "Guardian", "Defender", "Protector", "Sentinel",
	"Warlord", "Conqueror", "Overlord", "Sovereign", "Monarch", "Emperor", "Godslayer", "Titan", "Ascendant", "Transcendent",
}

// XP curve tuning for 10000 levels.
// Sharpened curve: 1-1000 is much easier, then it gets exponentially harder.
const (
	xpMin      = 20 // Increased from 10
	xpMax      = 50 // Increased from 25
	xpCurveK   = 1.0
)

// SubRank returns a level's rank within its tier (1..30).
func SubRank(level int) int {
	if level < 1 {
		level = 1
	}
	return (level-1)%levelsPerTier + 1
}

// TierForLevel returns the 1..NumTiers tier a level belongs to.
func TierForLevel(level int) int {
	if level < 1 {
		level = 1
	}
	tier := (level-1)/levelsPerTier + 1
	if tier > NumTiers {
		tier = NumTiers
	}
	return tier
}

// TierName returns the fantasy name of a tier (1..NumTiers).
func TierName(tier int) string {
	idx := tier - 1
	if idx < len(baseTierNames) {
		return baseTierNames[idx]
	}
	// Procedural tiers for the rest
	adj := epicAdjectives[(idx/len(epicTitles))%len(epicAdjectives)]
	tit := epicTitles[idx%len(epicTitles)]
	return adj + " " + tit
}

// XPForLevel returns the cumulative XP required to reach the given level.
func XPForLevel(level int) int {
	if level <= 1 {
		return 0
	}
	// Dynamic exponent: grows as level increases.
	// Starts at 1.1 (very fast early levels), reaches ~1.6 at level 1000, and caps at 5.0.
	exponent := 1.1 + (float64(level) / 2000.0)
	if exponent > 5.0 {
		exponent = 5.0
	}
	
	val := math.Pow(float64(level-1), exponent)
	// Cap at a large integer to prevent overflow during search
	if val > 2e15 {
		return 2e15
	}
	return int(math.Round(val))
}

// LevelForXP returns the level for a total XP amount.
func LevelForXP(xp int) int {
	if xp <= 0 {
		return 1
	}
	// Binary search since the curve is no longer a simple static exponent.
	low, high := 1, absoluteMaxLevel
	ans := 1
	for low <= high {
		mid := low + (high-low)/2
		if XPForLevel(mid) <= xp {
			ans = mid
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return ans
}

// LevelName returns the fantasy name for a level.
func LevelName(level int) string {
	if level < 1 {
		level = 1
	}
	tier := TierForLevel(level)
	sub := SubRank(level)
	
	name := TierName(tier)
	
	// Add Realm flair for very high levels
	if level > 6000 {
		realm := epicRealms[(level/300)%len(epicRealms)]
		return fmt.Sprintf("%s of the %s %s", name, realm, roman(sub))
	}
	
	return name + " " + roman(sub)
}

// XPPerPoke returns a randomised XP award.
func XPPerPoke() int {
// #nosec G404
	return xpMin + rand.IntN(xpMax-xpMin+1) // #nosec G404
}

// XPForPrice maps a game's original price to an XP award.
func XPForPrice(priceEUR float64, cheaperMoreXP bool) int {
	const priceCapEUR = 60.0
	if priceEUR < 0 {
		priceEUR = 0
	}
	if priceEUR > priceCapEUR {
		priceEUR = priceCapEUR
	}
	frac := priceEUR / priceCapEUR
	if cheaperMoreXP {
		frac = 1 - frac
	}
	return int(math.Round(float64(xpMin) + frac*float64(xpMax-xpMin)))
}

// ParseLevelGroups parses a "level:groupID,level:groupID" string.
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

// MilestonesCrossed returns the server group ids newly earned.
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

// roman converts 1..9999 to a Roman numeral.
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

// LevelByName returns the level number for a level name.
func LevelByName(name string) (int, bool) {
	parts := strings.Fields(name)
	if len(parts) < 2 {
		return 0, false
	}
	
	rom := parts[len(parts)-1]
	sub := deroman(rom)
	if sub == 0 {
		return 0, false
	}

	// 1. Strip roman numeral and realm flair if present
	fullTName := strings.Join(parts[:len(parts)-1], " ")
	tName := fullTName
	if strings.Contains(fullTName, " of the ") {
		tName = strings.Split(fullTName, " of the ")[0]
	}
	
	// 2. Check all tiers (base + procedural)
	for t := 1; t <= NumTiers; t++ {
		if TierName(t) == tName {
			level := (t-1)*levelsPerTier + sub
			// Re-verify with full LevelName to handle realm flair and exact match
			if LevelName(level) == name {
				return level, true
			}
		}
	}

	return 0, false 
}

func deroman(s string) int {
	m := map[rune]int{'I': 1, 'V': 5, 'X': 10, 'L': 50, 'C': 100, 'D': 500, 'M': 1000}
	res := 0
	prev := 0
	for i := len(s) - 1; i >= 0; i-- {
		val, ok := m[rune(s[i])]
		if !ok { return 0 }
		if val < prev {
			res -= val
		} else {
			res += val
		}
		prev = val
	}
	return res
}
