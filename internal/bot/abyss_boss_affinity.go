package bot

import (
	"time"

	"ts3news/internal/content"
)

type abyssBossAffinity struct {
	Element       content.Element
	Name          string
	Icon          string
	WeakTo        content.Element
	StrongAgainst content.Element
	Color         string
}

type abyssBossAffinityForecastView struct {
	Name          string
	Icon          string
	Element       string
	WeakTo        string
	StrongAgainst string
	Color         string
	TargetDepth   int
	TwinBosses    bool
	Neutral       bool
	Secret        bool
}

func abyssBossAffinityForecastForSecret(run abyssRun, now time.Time, chain abyssSecretBossChainView) abyssBossAffinityForecastView {
	view := abyssBossAffinityForecast(run, now)
	if !chain.Unlocked || chain.Completed || chain.Stage < 0 || chain.Stage >= len(abyssSecretBosses) {
		return view
	}
	def := abyssSecretBosses[chain.Stage]
	view.Secret = true
	view.Name, view.Icon, view.Element = "Forbidden Signature", "⌬", string(def.Element)
	view.TargetDepth, view.TwinBosses = chain.NextDepth, false
	view.WeakTo = string(liveElementWeakness(def.Element))
	view.StrongAgainst = ""
	switch def.Element {
	case content.ElementFire:
		view.StrongAgainst, view.Color = string(content.ElementAir), "fire"
	case content.ElementWater:
		view.StrongAgainst, view.Color = string(content.ElementFire), "water"
	case content.ElementEarth:
		view.StrongAgainst, view.Color = string(content.ElementWater), "earth"
	case content.ElementAir:
		view.StrongAgainst, view.Color = string(content.ElementEarth), "air"
	default:
		view.Neutral, view.Color = true, "physical"
	}
	return view
}

var abyssBossAffinityRotation = []abyssBossAffinity{
	{Element: content.ElementFire, Name: "Cinder Crown", Icon: "🔥", WeakTo: content.ElementWater, StrongAgainst: content.ElementAir, Color: "fire"},
	{Element: content.ElementWater, Name: "Drowned Moon", Icon: "🌊", WeakTo: content.ElementEarth, StrongAgainst: content.ElementFire, Color: "water"},
	{Element: content.ElementEarth, Name: "Graven Root", Icon: "🜃", WeakTo: content.ElementAir, StrongAgainst: content.ElementWater, Color: "earth"},
	{Element: content.ElementAir, Name: "Riven Sky", Icon: "🌪", WeakTo: content.ElementFire, StrongAgainst: content.ElementEarth, Color: "air"},
}

func abyssDailyBossAffinity(now time.Time) abyssBossAffinity {
	day := now.UTC().Unix() / int64(24*time.Hour/time.Second)
	index := int(day % int64(len(abyssBossAffinityRotation)))
	if index < 0 {
		index += len(abyssBossAffinityRotation)
	}
	return abyssBossAffinityRotation[index]
}

func abyssBossAffinityForecast(run abyssRun, now time.Time) abyssBossAffinityForecastView {
	affinity := abyssDailyBossAffinity(now)
	target := abyssNextNaturalBossDepth(run.Depth)
	return abyssBossAffinityForecastView{
		Name: affinity.Name, Icon: affinity.Icon, Element: string(affinity.Element),
		WeakTo: string(affinity.WeakTo), StrongAgainst: string(affinity.StrongAgainst), Color: affinity.Color,
		TargetDepth: target, TwinBosses: target > abyssDoubleBossDepth,
	}
}
