package bot

import (
	"fmt"
	"hash/fnv"

	"ts3news/internal/content"
)

const abyssBossAdaptationKills = 10

type abyssBossAdaptation struct {
	Effect content.MobEffect
	Name   string
	Desc   string
}

type abyssBossAdaptationBossView struct {
	Name      string
	Kills     int
	Remaining int
	Active    bool
	Trick     string
	Desc      string
}

type abyssBossAdaptationForecastView struct {
	TargetDepth int
	Bosses      []abyssBossAdaptationBossView
}

var abyssBossAdaptations = []abyssBossAdaptation{
	{Effect: content.EffectEnraged, Name: "Predator's Recall", Desc: "+50% STR"},
	{Effect: content.EffectArmored, Name: "Iron Memory", Desc: "+50% DEF"},
	{Effect: content.EffectFleet, Name: "Learned Pursuit", Desc: "+50% SPD"},
	{Effect: content.EffectRegen, Name: "Deathless Lesson", Desc: "restores 5% HP each round"},
}

func abyssBossAdaptationFor(name string, priorKills int) (abyssBossAdaptation, bool) {
	if priorKills < abyssBossAdaptationKills || name == "" {
		return abyssBossAdaptation{}, false
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(name))
	return abyssBossAdaptations[int(hash.Sum32())%len(abyssBossAdaptations)], true
}

func (b *Bot) abyssBossKillCounts(uid string) map[string]int {
	rows, err := b.DB.Query(`SELECT boss_name,COUNT(*) FROM abyss_boss_kills
		WHERE client_uid=$1 GROUP BY boss_name`, uid)
	if err != nil {
		return map[string]int{}
	}
	defer func() { _ = rows.Close() }()
	counts := map[string]int{}
	for rows.Next() {
		var name string
		var kills int
		if rows.Scan(&name, &kills) == nil {
			counts[name] = kills
		}
	}
	return counts
}

func abyssBossAdaptationForecast(
	run abyssRun,
	chain abyssSecretBossChainView,
	counts map[string]int,
) abyssBossAdaptationForecastView {
	target := abyssNextNaturalBossDepth(run.Depth)
	names := abyssBossNamesAtDepth(target)
	if chain.Unlocked && !chain.Completed && chain.Stage >= 0 && chain.Stage < len(abyssSecretBosses) {
		target = chain.NextDepth
		names = []string{abyssSecretBosses[chain.Stage].Name}
	}
	view := abyssBossAdaptationForecastView{
		TargetDepth: target,
		Bosses:      make([]abyssBossAdaptationBossView, 0, len(names)),
	}
	for _, name := range names {
		kills := max(0, counts[name])
		boss := abyssBossAdaptationBossView{
			Name:      name,
			Kills:     kills,
			Remaining: max(0, abyssBossAdaptationKills-kills),
		}
		if adaptation, active := abyssBossAdaptationFor(name, kills); active {
			boss.Active = true
			boss.Trick = adaptation.Name
			boss.Desc = adaptation.Desc
		}
		view.Bosses = append(view.Bosses, boss)
	}
	return view
}

func (b *Bot) abyssBossAdaptationForecast(
	uid string,
	run abyssRun,
	chain abyssSecretBossChainView,
) abyssBossAdaptationForecastView {
	return abyssBossAdaptationForecast(run, chain, b.abyssBossKillCounts(uid))
}

func (b *Bot) applyAbyssBossAdaptations(uid string, mobs []content.Mob) ([]content.Mob, []string) {
	counts := b.abyssBossKillCounts(uid)
	logs := make([]string, 0, len(mobs))
	for index := range mobs {
		if mobs[index].Type != content.MobBoss {
			continue
		}
		adaptation, active := abyssBossAdaptationFor(mobs[index].Name, counts[mobs[index].Name])
		if !active {
			continue
		}
		present := false
		for _, effect := range mobs[index].Effects {
			present = present || effect == adaptation.Effect
		}
		if !present {
			mobs[index].Effects = append(mobs[index].Effects, adaptation.Effect)
		}
		logs = append(logs, fmt.Sprintf("[color=#f0b35a]🧠 Adaptive boss — %s remembers %d defeats: %s (%s).[/color]", mobs[index].Name, counts[mobs[index].Name], adaptation.Name, adaptation.Desc))
	}
	return mobs, logs
}
