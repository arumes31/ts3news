package bot

import "ts3news/internal/content"

const abyssLiveEffectVampiric = "vampiric"

func liveMobAffix(effect content.MobEffect) abyssLiveEffect {
	view := abyssLiveEffect{
		Name:     string(effect),
		Duration: abyssLiveEncounterDuration,
	}
	info, ok := content.MobEffectDetails(effect)
	if !ok {
		return view
	}
	view.Key = info.Key
	view.Icon = info.Icon
	view.Description = info.Description
	view.Tone = info.Tone
	view.Affix = true
	return view
}

func liveVampiricMobAffix() abyssLiveEffect {
	return abyssLiveEffect{
		Name:        "Vampiric",
		Key:         abyssLiveEffectVampiric,
		Icon:        "🩸",
		Description: "Heals for 15% of direct damage dealt.",
		Tone:        "sustain",
		Duration:    abyssLiveEncounterDuration,
		Affix:       true,
	}
}
