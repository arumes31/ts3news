package content

import "math/rand/v2"

// RandomSource is the subset of a random generator used while resolving
// encounters. Callers can supply a seeded source for reproducible combat.
type RandomSource interface {
	Float64() float64
	IntN(int) int
}

type defaultRandomSource struct{}

func (defaultRandomSource) Float64() float64 {
	// #nosec G404 -- gameplay randomness is not security-sensitive.
	return rand.Float64()
}

func (defaultRandomSource) IntN(n int) int {
	// #nosec G404 -- gameplay randomness is not security-sensitive.
	return rand.IntN(n)
}

var gameplayRandom RandomSource = defaultRandomSource{}
