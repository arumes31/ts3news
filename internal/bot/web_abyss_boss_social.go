package bot

import (
	"fmt"
	"hash/fnv"

	"ts3news/internal/content"
)

func abyssBossTip(name string) string {
	tips := []string{
		"Guard the opening volley, then spend burst after the first telegraph.",
		"Save one interrupt for the 50% summon and break armor before executing.",
		"Do not overlap defensive cooldowns; the final quarter is the real damage check.",
		"Clear adds before greedily tunneling the boss, then commit your Ultimate.",
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(name))
	return tips[int(hash.Sum32())%len(tips)]
}

func abyssBossFinale(timeline []combatTimelineFrame, rounds int) []combatTimelineFrame {
	if rounds <= 0 || len(timeline) == 0 {
		return nil
	}
	lastRound := timeline[len(timeline)-1].Round
	minimum := max(1, lastRound-rounds+1)
	lastByRound := map[int]combatTimelineFrame{}
	for _, frame := range timeline {
		if frame.Round >= minimum {
			lastByRound[frame.Round] = frame
		}
	}
	finale := make([]combatTimelineFrame, 0, rounds)
	for round := minimum; round <= lastRound; round++ {
		if frame, ok := lastByRound[round]; ok {
			finale = append(finale, frame)
		}
	}
	return finale
}

func abyssBossTaunts(name string, timeline []combatTimelineFrame) []string {
	thresholds := []struct {
		pct  int
		line string
	}{{50, "You have crossed the threshold. Now I remember your name."}, {25, "Come closer. Let the dark see which of us breaks first."}}
	lines := make([]string, 0, len(thresholds))
	for _, threshold := range thresholds {
		for _, frame := range timeline {
			if frame.EnemyMax > 0 && frame.EnemyHP*100 <= frame.EnemyMax*threshold.pct {
				lines = append(lines, fmt.Sprintf("[color=#e8899d]💬 R%d · %s: “%s”[/color]", frame.Round, name, threshold.line))
				break
			}
		}
	}
	return lines
}

func abyssInsanityWhisper(depth, variant int) string {
	whispers := []string{
		"The walls count your breaths, but stop one number before you do.",
		"A voice wearing your own name asks why you keep descending.",
		"For one heartbeat, every shadow points toward you.",
		"Something below applauds before the killing blow lands.",
	}
	return fmt.Sprintf("[color=#b388ff]◌ Floor %d whispers: %s[/color]", depth, whispers[max(0, variant)%len(whispers)])
}

func (b *Bot) recordAbyssDeath(uid string, depth int, mobs []*content.Mob) {
	killerName, killerFamily := abyssDeathKiller(mobs)
	_, _ = b.DB.Exec(`INSERT INTO abyss_deaths (client_uid,depth,killer_name,killer_family)
		VALUES ($1,$2,$3,$4)`, uid, max(depth, 0), killerName, killerFamily)
}

func abyssDeathKiller(mobs []*content.Mob) (string, string) {
	killerName, killerFamily, strongest := "The Abyss", "Unknown", -1
	for _, mob := range mobs {
		if mob == nil || mob.CurrentHP <= 0 {
			continue
		}
		if mob.CurrentHP > strongest {
			killerName, killerFamily, strongest = mob.Name, string(mob.Type), mob.CurrentHP
		}
	}
	return killerName, killerFamily
}
