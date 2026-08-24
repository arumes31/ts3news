package bot

import "math/rand/v2"

type combatRandomSource interface {
	Float64() float64
	IntN(int) int
}

type defaultCombatRandomSource struct{}

func combatRandomForUsers(users []UserInCombat) combatRandomSource {
	for i := range users {
		if users[i].live != nil {
			return users[i].live
		}
	}
	return defaultCombatRandomSource{}
}

func (defaultCombatRandomSource) Float64() float64 {
	// #nosec G404 -- gameplay randomness is not security-sensitive.
	return rand.Float64()
}

func (defaultCombatRandomSource) IntN(n int) int {
	// #nosec G404 -- gameplay randomness is not security-sensitive.
	return rand.IntN(n)
}

func (c *abyssLiveCombat) Float64() float64 {
	c.rngMu.Lock()
	defer c.rngMu.Unlock()
	c.ensureRandomLocked()
	c.randomDraws++
	return c.rng.Float64()
}

func (c *abyssLiveCombat) IntN(n int) int {
	c.rngMu.Lock()
	defer c.rngMu.Unlock()
	c.ensureRandomLocked()
	c.randomDraws++
	return c.rng.IntN(n)
}

func (c *abyssLiveCombat) randomDrawCount() uint64 {
	c.rngMu.Lock()
	defer c.rngMu.Unlock()
	return c.randomDraws
}

func (c *abyssLiveCombat) ensureRandomLocked() {
	if c.rng == nil {
		c.rng = rand.New(rand.NewPCG(c.randomSeed[0], c.randomSeed[1]))
	}
}
