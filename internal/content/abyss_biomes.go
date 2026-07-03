package content

import "math/rand/v2"

// AbyssBiome is a cosmetic + light-mechanical reskin layered on top of an
// Abyss floor's zone: a themed name prefix and a small difficulty nudge,
// picked per depth bucket the same way abyssZoneName picks its zone name.
type AbyssBiome struct {
	Name    string
	DiffMod float64 // multiplier applied to the floor's difficulty
}

var abyssBiomesShallow = []AbyssBiome{
	{"Mossbound", 0.97},
	{"Cinder-Choked", 1.05},
	{"Fogbound", 1.00},
	{"Rootbound", 0.98},
}

var abyssBiomesMid = []AbyssBiome{
	{"Bloodrust", 1.08},
	{"Frostbitten", 1.05},
	{"Storm-Wracked", 1.10},
	{"Venom-Veiled", 1.06},
}

var abyssBiomesDeep = []AbyssBiome{
	{"Voidscarred", 1.15},
	{"Starless", 1.18},
	{"Soul-Rent", 1.20},
	{"Oblivion-Touched", 1.22},
}

// AbyssBiomeFor picks a depth-appropriate biome, mirroring the shallow/mid/deep
// bucketing already used for Abyss zone names.
func AbyssBiomeFor(depth int) AbyssBiome {
	var pool []AbyssBiome
	switch {
	case depth < 10:
		pool = abyssBiomesShallow
	case depth < 30:
		pool = abyssBiomesMid
	default:
		pool = abyssBiomesDeep
	}
	// #nosec G404 -- cosmetic/light-flavour selection
	return pool[rand.IntN(len(pool))]
}
