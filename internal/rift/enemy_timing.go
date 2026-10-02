package rift

import "hash/fnv"

// rangedCooldownOffset keeps each archer's cadence stable across saves and
// independent of simulation iteration order. Missing legacy IDs keep base timing.
func rangedCooldownOffset(e *Actor) float64 {
	if e.Kind != "archer" || e.ID == "" {
		return 0
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(e.ID))
	return float64(1+h.Sum32()%8) * .08
}
