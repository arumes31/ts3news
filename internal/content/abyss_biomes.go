package content

import "math/rand/v2"

// AbyssBiome is a cosmetic + light-mechanical reskin layered on top of an
// Abyss floor's zone: a themed name prefix and a small difficulty nudge,
// picked per depth bucket the same way abyssZoneName picks its zone name.
type AbyssBiome struct {
	Name     string
	Affinity string
	DiffMod  float64 // multiplier applied to the floor's difficulty
}

var abyssBiomesShallow = []AbyssBiome{
	{Name: "Mossbound", Affinity: "nature", DiffMod: 0.97},
	{Name: "Cinder-Choked", Affinity: "fire", DiffMod: 1.05},
	{Name: "Fogbound", Affinity: "water", DiffMod: 1.00},
	{Name: "Rootbound", Affinity: "nature", DiffMod: 0.98},
}

var abyssBiomesMid = []AbyssBiome{
	{Name: "Bloodrust", Affinity: "fire", DiffMod: 1.08},
	{Name: "Frostbitten", Affinity: "frost", DiffMod: 1.05},
	{Name: "Storm-Wracked", Affinity: "storm", DiffMod: 1.10},
	{Name: "Venom-Veiled", Affinity: "nature", DiffMod: 1.06},
}

var abyssBiomesDeep = []AbyssBiome{
	{Name: "Pyre-Eternal", Affinity: "fire", DiffMod: 1.17},
	{Name: "Voidscarred", Affinity: "void", DiffMod: 1.15},
	{Name: "Starless", Affinity: "void", DiffMod: 1.18},
	{Name: "Soul-Rent", Affinity: "spirit", DiffMod: 1.20},
	{Name: "Oblivion-Touched", Affinity: "void", DiffMod: 1.22},
}

// AbyssBiomeFor picks a depth-appropriate biome, mirroring the shallow/mid/deep
// bucketing already used for Abyss zone names.
func AbyssBiomeFor(depth int) AbyssBiome {
	pool := abyssBiomePool(depth)
	// #nosec G404 -- cosmetic/light-flavour selection
	return pool[rand.IntN(len(pool))]
}

// AbyssBiomeForAffinity chooses from the depth-appropriate pool and gives
// matching biomes triple weight. roll makes the selection deterministic for
// replayable encounters; callers should provide a random value from their
// encounter-scoped source.
func AbyssBiomeForAffinity(depth int, affinity string, roll int) AbyssBiome {
	pool := abyssBiomePool(depth)
	totalWeight := 0
	for _, biome := range pool {
		totalWeight++
		if affinity != "" && biome.Affinity == affinity {
			totalWeight += 2
		}
	}
	roll %= totalWeight
	if roll < 0 {
		roll += totalWeight
	}
	for _, biome := range pool {
		weight := 1
		if affinity != "" && biome.Affinity == affinity {
			weight = 3
		}
		if roll < weight {
			return biome
		}
		roll -= weight
	}
	return pool[len(pool)-1]
}

// AbyssBiomeWeight returns the size of the weighted roll range for a depth and
// affinity. It lets encounter callers draw exactly one bounded random value.
func AbyssBiomeWeight(depth int, affinity string) int {
	weight := 0
	for _, biome := range abyssBiomePool(depth) {
		weight++
		if affinity != "" && biome.Affinity == affinity {
			weight += 2
		}
	}
	return weight
}

func abyssBiomePool(depth int) []AbyssBiome {
	var pool []AbyssBiome
	switch {
	case depth < 10:
		pool = abyssBiomesShallow
	case depth < 30:
		pool = abyssBiomesMid
	default:
		pool = abyssBiomesDeep
	}
	return pool
}
