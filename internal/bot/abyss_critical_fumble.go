package bot

import (
	cryptorand "crypto/rand"
	"fmt"
	"math/big"
)

const abyssCriticalFumbleChance = 1

type abyssDramaRandom interface {
	IntN(int) int
}

type defaultAbyssDramaRandom struct{}

var abyssCriticalFumbleLines = []string{
	"🎭 CRITICAL FUMBLE · %s salutes too early, catches the backswing, and still tags %s. (No combat effect)",
	"🎭 CRITICAL FUMBLE · %s trips over an imaginary cape, rolls through, and hits %s anyway. (No combat effect)",
	"🎭 CRITICAL FUMBLE · %s drops their weapon, catches it with the other hand, and bonks %s on schedule. (No combat effect)",
	"🎭 CRITICAL FUMBLE · %s attacks the dramatic lighting first; %s still takes the hit. (No combat effect)",
}

func (defaultAbyssDramaRandom) IntN(n int) int {
	if n <= 0 {
		panic("invalid Abyss drama random bound")
	}
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return n - 1
	}
	return int(value.Int64())
}

func abyssCriticalFumbleLog(
	attacker,
	target string,
	abyss bool,
	random abyssDramaRandom,
) string {
	if !abyss || random == nil || random.IntN(100) >= abyssCriticalFumbleChance {
		return ""
	}
	line := abyssCriticalFumbleLines[random.IntN(len(abyssCriticalFumbleLines))]
	return fmt.Sprintf(line, sanitizeBBCode(attacker), sanitizeBBCode(target))
}
