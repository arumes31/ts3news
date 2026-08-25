package bot

// abyssBossSummonChoreography keeps boss summon flavor separate from combat
// resolution. Telegraphs deliberately retain "summoning ritual" because the
// live-combat planner uses that phrase to expose the interrupt window.
type abyssBossSummonChoreography struct {
	Telegraph string
	Arrival   string
	AddPrefix string
}

func abyssBossSummonFor(name string) abyssBossSummonChoreography {
	switch name {
	case "Gorgoroth the Firelord":
		return abyssBossSummonChoreography{
			Telegraph: "📣 Gorgoroth the Firelord raises his cinder horn in a summoning ritual! Fire an ULTIMATE this round to silence it!",
			Arrival:   "📣 Gorgoroth's horn answers — an Ashbound reinforcement charges through the flame!",
			AddPrefix: "Ashbound",
		}
	case "Malakor the Voidweaver":
		return abyssBossSummonChoreography{
			Telegraph: "📣 Malakor the Voidweaver tears open a void seam in a summoning ritual! Fire an ULTIMATE this round to seal it!",
			Arrival:   "📣 Malakor's seam splits wide — a Voidwoven reinforcement slips into the arena!",
			AddPrefix: "Voidwoven",
		}
	case "Azazoth the Slumbering Eye":
		return abyssBossSummonChoreography{
			Telegraph: "📣 Azazoth the Slumbering Eye opens a dreaming gate in a summoning ritual! Fire an ULTIMATE this round to wake it!",
			Arrival:   "📣 Azazoth's nightmare takes shape — a Dreamspawn reinforcement awakens!",
			AddPrefix: "Dreamspawn",
		}
	case "Abyssus, Heart of the Void":
		return abyssBossSummonChoreography{
			Telegraph: "📣 Abyssus, Heart of the Void beats the black heart in a summoning ritual! Fire an ULTIMATE this round to break its rhythm!",
			Arrival:   "📣 Abyssus completes the pulse — a Heartborn reinforcement claws free of the dark!",
			AddPrefix: "Heartborn",
		}
	default:
		return abyssBossSummonChoreography{
			Telegraph: "📣 " + name + " begins a summoning ritual! Fire an ULTIMATE this round to interrupt it!",
			Arrival:   "📣 " + name + " completes the ritual — reinforcements arrive!",
			AddPrefix: "Summoned",
		}
	}
}
