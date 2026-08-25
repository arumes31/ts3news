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
