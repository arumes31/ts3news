// Package content provides the flavour text the bot mixes into its messages:
// random greetings, holiday theming and game-based nicknames.
// Everything here is deterministic-free (uses math/rand/v2) and side-effect free,
// so it is trivially unit-testable.
package content

import (
	"math/rand/v2"
	"ts3news/internal/i18n"
)

// greetings holds 100 short, punchy opening lines for the poke / private message.
// Now uses i18n.Pool("greeting") to get the greeting pool.

// RandomGreeting returns one of the 100 greetings at random.
func RandomGreeting() string {
	pool := i18n.Pool("greeting")
	if len(pool) == 0 {
		return "Hey gamer!"
	}
	// #nosec G404
	return pool[rand.IntN(len(pool))] // #nosec G404
}

// GreetingCount is exposed for tests.
func GreetingCount() int {
	pool := i18n.Pool("greeting")
	return len(pool)
}
