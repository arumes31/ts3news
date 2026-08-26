package bot

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strconv"

	"ts3news/internal/content"
)

type abyssEnemyForecastView struct {
	Active    bool   `json:"active"`
	Concealed bool   `json:"concealed,omitempty"`
	Depth     int    `json:"depth,omitempty"`
	Key       string `json:"key,omitempty"`
	Icon      string `json:"icon,omitempty"`
	Name      string `json:"name,omitempty"`
	Signal    string `json:"signal,omitempty"`
	Counter   string `json:"counter,omitempty"`
}

var abyssEnemyForecasts = []abyssEnemyForecastView{
	{Key: "summoner", Icon: "🔮", Name: "Summoner", Signal: "On defeat, calls reinforcements.", Counter: "Leave it for last or prepare an add-clear."},
	{Key: "deathburst", Icon: "💥", Name: "Deathburst", Signal: "On defeat, damages the whole party.", Counter: "Stabilize before landing the finishing blow."},
	{Key: "regenerator", Icon: "♻️", Name: "Regenerator", Signal: "Restores health at the end of each round.", Counter: "Focus-fire; poison and burst deny recovery."},
	{Key: "berserker", Icon: "🩸", Name: "Berserker", Signal: "Enraged attacks deal 50% more damage.", Counter: "Defend early and break it before attrition sets in."},
}

func abyssEnemyForecast(uid string, run abyssRun, pacts []string) abyssEnemyForecastView {
	if !run.Active || run.Downed {
		return abyssEnemyForecastView{}
	}
	depth := run.Depth + 1
	if abyssHasPact(pacts, "blind") {
		return abyssEnemyForecastView{Active: true, Concealed: true, Depth: depth}
	}
	if depth%abyssBossEvery == 0 || abyssPactBossFloor(pacts, depth) {
		return abyssEnemyForecastView{
			Active: true, Depth: depth, Key: "ritualist", Icon: "🕯️", Name: "Summoning Ritual",
			Signal:  "Below 50% health, the boss channels reinforcements for one round.",
			Counter: "Use an ultimate during the channel to interrupt it.",
		}
	}

	sum := sha256.Sum256([]byte(uid + "\x00" + run.StartedAt.UTC().Format("20060102T150405.000000000Z07:00") + "\x00" + strconv.Itoa(depth)))
	forecast := abyssEnemyForecasts[binary.LittleEndian.Uint64(sum[:8])%uint64(len(abyssEnemyForecasts))]
	forecast.Active = true
	forecast.Depth = depth
	return forecast
}

func (b *Bot) abyssEnemyForecast(uid string, run abyssRun) abyssEnemyForecastView {
	return abyssEnemyForecast(uid, run, b.abyssRunPacts(uid))
}

func applyAbyssEnemyForecast(forecast abyssEnemyForecastView, mobs []content.Mob) string {
	if !forecast.Active || forecast.Concealed || len(mobs) == 0 {
		return ""
	}
	if forecast.Key == "ritualist" {
		return fmt.Sprintf("📡 SCOUT CONFIRMED · %s: %s", forecast.Name, forecast.Signal)
	}

	target := -1
	for i := range mobs {
		if !abyssEnemyHazard(&mobs[i]) {
			target = i
			break
		}
	}
	if target < 0 {
		return ""
	}
	mob := &mobs[target]
	switch forecast.Key {
	case "summoner":
		mob.DeathEffect = &content.MobDeathEffect{Name: "Call of the Deep", Type: content.DeathSummon}
	case "deathburst":
		mob.DeathEffect = &content.MobDeathEffect{Name: "Abyssal Rupture", Type: content.DeathExplosion}
	case "regenerator":
		appendAbyssMobEffect(mob, content.EffectRegen)
	case "berserker":
		appendAbyssMobEffect(mob, content.EffectEnraged)
	default:
		return ""
	}
	return fmt.Sprintf("📡 SCOUT CONFIRMED · %s manifests %s: %s", mob.Name, forecast.Name, forecast.Signal)
}

func appendAbyssMobEffect(mob *content.Mob, effect content.MobEffect) {
	for _, existing := range mob.Effects {
		if existing == effect {
			return
		}
	}
	mob.Effects = append(mob.Effects, effect)
}
