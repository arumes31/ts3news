package bot

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

type abyssWeeklyBossDrop struct {
	Material string
	Amount   int
	Weight   int
}

func abyssWeeklyBossDropTable(name string) []abyssWeeklyBossDrop {
	switch name {
	case "Nhal, the Starved Horizon":
		return []abyssWeeklyBossDrop{{"dust", 3, 60}, {"shard", 1, 30}, {"prism", 1, 10}}
	case "Veyra of the Thousand Eyes":
		return []abyssWeeklyBossDrop{{"shard", 2, 55}, {"core", 1, 35}, {"prism", 1, 10}}
	case "The Iron Leviathan":
		return []abyssWeeklyBossDrop{{"dust", 5, 50}, {"core", 1, 40}, {"prism", 1, 10}}
	case "Mournroot Prime":
		return []abyssWeeklyBossDrop{{"dust", 2, 45}, {"shard", 2, 35}, {"core", 1, 20}}
	default:
		return []abyssWeeklyBossDrop{{"dust", 2, 100}}
	}
}

func abyssWeeklyBossDropFor(name, week, uid string, now time.Time) abyssWeeklyBossDrop {
	table := abyssWeeklyBossDropTable(name)
	seed := strings.Join([]string{name, week, uid, now.UTC().Format("2006-01-02")}, "\x00")
	digest := sha256.Sum256([]byte(seed))
	roll := int(binary.BigEndian.Uint64(digest[:8]) % 100)
	for _, drop := range table {
		if roll < drop.Weight {
			return drop
		}
		roll -= drop.Weight
	}
	return table[len(table)-1]
}

func abyssWeeklyBossDropLabel(drop abyssWeeklyBossDrop) string {
	return fmt.Sprintf("%d× %s", drop.Amount, abyssMaterialName(drop.Material))
}

func abyssWeeklyBossDropSummary(name string) string {
	table := abyssWeeklyBossDropTable(name)
	parts := make([]string, 0, len(table))
	for _, drop := range table {
		parts = append(parts, fmt.Sprintf("%s (%d%%)", abyssWeeklyBossDropLabel(drop), drop.Weight))
	}
	return strings.Join(parts, " · ")
}
