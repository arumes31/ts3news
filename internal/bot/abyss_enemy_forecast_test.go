package bot

import (
	"strings"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestAbyssEnemyForecastIsStableAndVariesAcrossDepths(t *testing.T) {
	run := abyssRun{Active: true, Depth: 11, StartedAt: time.Date(2026, time.August, 26, 8, 30, 0, 0, time.UTC)}
	want := abyssEnemyForecast("scout", run, nil)
	if got := abyssEnemyForecast("scout", run, nil); got != want {
		t.Fatalf("same run forecast changed: got %+v, want %+v", got, want)
	}
	if !want.Active || want.Depth != 12 || want.Key == "" || want.Signal == "" || want.Counter == "" {
		t.Fatalf("incomplete forecast: %+v", want)
	}

	seen := make(map[string]bool)
	for depth := 11; depth < 23; depth++ {
		run.Depth = depth
		seen[abyssEnemyForecast("scout", run, nil).Key] = true
	}
	if len(seen) < 2 {
		t.Fatalf("twelve depths produced only %d forecast archetype", len(seen))
	}
}

func TestAbyssEnemyForecastRespectsBlindAndBossFloors(t *testing.T) {
	run := abyssRun{Active: true, Depth: 3}
	blind := abyssEnemyForecast("scout", run, []string{"blind"})
	if !blind.Active || !blind.Concealed || blind.Key != "" || blind.Name != "" {
		t.Fatalf("blind forecast leaked intel: %+v", blind)
	}

	run.Depth = abyssBossEvery - 1
	boss := abyssEnemyForecast("scout", run, nil)
	if boss.Key != "ritualist" || !strings.Contains(boss.Counter, "ultimate") {
		t.Fatalf("boss forecast = %+v", boss)
	}

	run.Depth = 2
	pactBoss := abyssEnemyForecast("scout", run, []string{"deep_drums"})
	if pactBoss.Key != "ritualist" {
		t.Fatalf("pact boss forecast = %+v", pactBoss)
	}
}

func TestApplyAbyssEnemyForecastBindsMechanic(t *testing.T) {
	tests := []struct {
		key   string
		check func(*testing.T, content.Mob)
	}{
		{key: "summoner", check: func(t *testing.T, mob content.Mob) {
			if mob.DeathEffect == nil || mob.DeathEffect.Type != content.DeathSummon {
				t.Fatalf("death effect = %+v", mob.DeathEffect)
			}
		}},
		{key: "deathburst", check: func(t *testing.T, mob content.Mob) {
			if mob.DeathEffect == nil || mob.DeathEffect.Type != content.DeathExplosion {
				t.Fatalf("death effect = %+v", mob.DeathEffect)
			}
		}},
		{key: "regenerator", check: func(t *testing.T, mob content.Mob) {
			assertAbyssMobEffectCount(t, mob, content.EffectRegen, 1)
		}},
		{key: "berserker", check: func(t *testing.T, mob content.Mob) {
			assertAbyssMobEffectCount(t, mob, content.EffectEnraged, 1)
		}},
	}

	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			forecast := forecastByKey(t, test.key)
			mobs := []content.Mob{{Name: "Hazard: Rift"}, {Name: "Scout target", Effects: []content.MobEffect{effectForForecast(test.key)}}}
			logLine := applyAbyssEnemyForecast(forecast, mobs)
			if !strings.Contains(logLine, "Scout target") || !strings.Contains(logLine, forecast.Name) {
				t.Fatalf("confirmation log = %q", logLine)
			}
			test.check(t, mobs[1])
		})
	}
}

func TestAbyssEnemyPatternDescribesForecastMechanics(t *testing.T) {
	tests := []struct {
		mob  content.Mob
		want string
	}{
		{content.Mob{DeathEffect: &content.MobDeathEffect{Type: content.DeathSummon}}, "calls reinforcements"},
		{content.Mob{DeathEffect: &content.MobDeathEffect{Type: content.DeathExplosion}}, "whole party"},
		{content.Mob{Effects: []content.MobEffect{content.EffectRegen}}, "Restores health"},
		{content.Mob{Effects: []content.MobEffect{content.EffectEnraged}}, "50% more damage"},
	}
	for _, test := range tests {
		if got := abyssEnemyPattern(&test.mob); !strings.Contains(got, test.want) {
			t.Errorf("pattern %q does not contain %q", got, test.want)
		}
	}
}

func TestAbyssEnemyForecastUIContract(t *testing.T) {
	partial, err := webAssets.ReadFile("webassets/abyss_enemy_forecast.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_enemy_forecast.css")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"NEXT COMBAT SIGNAL", "SIGNAL JAMMED", "COUNTERMEASURE", "textContent", "renderAbyssEnemyForecast"} {
		if !strings.Contains(string(partial), token) {
			t.Errorf("forecast partial is missing %q", token)
		}
	}
	for _, token := range []string{".ab-enemy-forecast", ".is-concealed", "@media (max-width: 680px)"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("forecast styles are missing %q", token)
		}
	}
	for _, token := range []string{"/static/abyss_enemy_forecast.css", `template "abyssEnemyForecast"`, "d.enemy_forecast"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("Abyss page is missing %q", token)
		}
	}
}

func forecastByKey(t *testing.T, key string) abyssEnemyForecastView {
	t.Helper()
	for _, forecast := range abyssEnemyForecasts {
		if forecast.Key == key {
			forecast.Active = true
			return forecast
		}
	}
	t.Fatalf("unknown forecast key %q", key)
	return abyssEnemyForecastView{}
}

func effectForForecast(key string) content.MobEffect {
	switch key {
	case "regenerator":
		return content.EffectRegen
	case "berserker":
		return content.EffectEnraged
	default:
		return content.EffectWeakened
	}
}

func assertAbyssMobEffectCount(t *testing.T, mob content.Mob, effect content.MobEffect, want int) {
	t.Helper()
	count := 0
	for _, existing := range mob.Effects {
		if existing == effect {
			count++
		}
	}
	if count != want {
		t.Fatalf("%s count = %d, want %d (%v)", effect, count, want, mob.Effects)
	}
}
