package bot

import (
	"encoding/json"
	"fmt"
	"strings"

	"ts3news/internal/content"
)

const abyssNemesisPrefix = "abyss_nemesis_"

type abyssNemesisState struct {
	Name      string `json:"name"`
	Level     int    `json:"level"`
	Victories int    `json:"victories"`
}

func abyssEnemyRole(mob *content.Mob) string {
	if mob == nil {
		return ""
	}
	if abyssEnemyHazard(mob) {
		return "hazard"
	}
	switch mob.Type {
	case content.MobBoss, content.MobLegendary:
		return "boss"
	case content.MobTreasureGoblin:
		return "runner"
	}
	if len(mob.Spells) > 0 && mob.Stats.STR >= mob.Stats.DEF {
		return "caster"
	}
	if mob.Stats.DEF > mob.Stats.STR {
		return "guardian"
	}
	if mob.Stats.SPD > mob.Stats.STR {
		return "skirmisher"
	}
	return "bruiser"
}

func abyssEnemyFaction(mob *content.Mob) string {
	if mob == nil {
		return ""
	}
	name := strings.ToLower(mob.Name)
	switch {
	case abyssEnemyHazard(mob):
		return "Abyssal Phenomena"
	case strings.Contains(name, "undead") || strings.Contains(name, "skeleton") ||
		strings.Contains(name, "lich") || strings.Contains(name, "ghost"):
		return "Graveborn"
	case strings.Contains(name, "void") || strings.Contains(name, "abyss") ||
		strings.Contains(name, "watcher"):
		return "Void Court"
	case strings.Contains(name, "goblin") || strings.Contains(name, "orc") ||
		strings.Contains(name, "troll"):
		return "Deepclaw Clan"
	case mob.Element == content.ElementFire:
		return "Ember Legion"
	case mob.Element == content.ElementWater:
		return "Drowned Host"
	default:
		return "Lost Delvers"
	}
}

func abyssEnemyPattern(mob *content.Mob) string {
	if mob == nil {
		return ""
	}
	if abyssEnemyHazard(mob) {
		return "Destroy before it detonates"
	}
	if mob.DeathEffect != nil {
		switch mob.DeathEffect.Type {
		case content.DeathSummon:
			return "On defeat, calls reinforcements"
		case content.DeathExplosion:
			return "On defeat, damages the whole party"
		}
	}
	for _, effect := range mob.Effects {
		switch effect {
		case content.EffectRegen:
			return "Restores health at the end of each round"
		case content.EffectEnraged:
			return "Enraged attacks deal 50% more damage"
		}
	}
	switch mob.Type {
	case content.MobTreasureGoblin:
		return "Attempts to flee from round 3"
	case content.MobBoss, content.MobLegendary:
		return "Phases at 50% and 25%; enrages if stalled"
	}
	switch abyssEnemyRole(mob) {
	case "caster":
		return "Alternates attacks with spell casts"
	case "guardian":
		return "Protects its pack with high defense"
	case "skirmisher":
		return "Pressures vulnerable targets quickly"
	default:
		return "Direct pressure"
	}
}

func abyssEnemyHazard(mob *content.Mob) bool {
	return mob != nil && strings.HasPrefix(mob.Name, "Hazard: ")
}

func (b *Bot) prepareAbyssEnemies(
	uid string,
	depth int,
	mobs []content.Mob,
	random combatRandomSource,
) ([]content.Mob, []string) {
	logs := make([]string, 0, 4)
	if depth >= 8 && random.Float64() < 0.12 {
		hp := max(30, depth*12)
		mobs = append(mobs, content.Mob{
			Name:      "Hazard: Volatile Rift",
			Type:      content.MobCommon,
			Level:     depth,
			Stats:     content.Stats{HP: hp, STR: max(1, depth/2), DEF: 0, SPD: 1},
			CurrentHP: hp,
			MaxHP:     hp,
			Element:   content.ElementAir,
			DeathEffect: &content.MobDeathEffect{
				Name: "Rift Collapse",
				Type: content.DeathExplosion,
			},
		})
		logs = append(logs, "⚠️ A Volatile Rift can be targeted and destroyed before it detonates.")
	}
	if depth >= 20 && random.Float64() < 0.08 {
		hp := max(100, depth*20)
		mobs = append(mobs, content.Mob{
			Name:      "Abyssal Invader",
			Type:      content.MobElite,
			Level:     depth + 2,
			Stats:     content.Stats{HP: hp, STR: max(10, depth*2), DEF: max(5, depth/2), SPD: 25},
			CurrentHP: hp,
			MaxHP:     hp,
			Element:   content.ElementPhysical,
			RewardXP:  depth * 5,
		})
		logs = append(logs, "🌀 A rare Void Court invasion breaches this biome!")
	}
	if rare, signature, ok := abyssNamedRareSpawn(depth, random); ok {
		mobs = append(mobs, rare)
		logs = append(logs, fmt.Sprintf("💠 NAMED RARE — %s carries the fixed signature drop %s.", rare.Name, signature))
	}
	if nemesis, ok := b.loadAbyssNemesis(uid); ok && depth >= 5 && random.Float64() < 0.25 {
		level := max(depth, nemesis.Level+nemesis.Victories)
		hp := max(150, level*24)
		mobs = append(mobs, content.Mob{
			Name:      "Nemesis: " + nemesis.Name,
			Type:      content.MobMiniboss,
			Level:     level,
			Stats:     content.Stats{HP: hp, STR: max(15, level*2), DEF: max(8, level), SPD: 30},
			CurrentHP: hp,
			MaxHP:     hp,
			Element:   content.ElementPhysical,
			RewardXP:  level * 10,
			Effects:   []content.MobEffect{content.EffectEnraged},
		})
		logs = append(logs, fmt.Sprintf("☠️ %s has returned as your nemesis (victories: %d).", nemesis.Name, nemesis.Victories))
	}
	if empowered, auraLog := applyScheduledAbyssEliteAura(depth, mobs); auraLog != "" {
		mobs = empowered
		logs = append(logs, auraLog)
	}

	factionCounts := make(map[string]int)
	eliteCount := 0
	for i := range mobs {
		mob := &mobs[i]
		if mob.MaxHP <= 0 {
			mob.MaxHP = mob.Stats.HP
			mob.CurrentHP = mob.MaxHP
		}
		switch mob.Type {
		case content.MobElite, content.MobMiniboss, content.MobBoss, content.MobLegendary:
			eliteCount++
			mob.MaxBreak = max(10, mob.MaxHP/5)
			mob.Break = mob.MaxBreak
			if len(mob.Effects) == 0 {
				affixes := []content.MobEffect{
					content.EffectEnraged,
					content.EffectArmored,
					content.EffectFleet,
					content.EffectRegen,
				}
				mob.Effects = append(mob.Effects, affixes[random.IntN(len(affixes))])
			}
		}
		factionCounts[abyssEnemyFaction(mob)]++
	}
	for faction, count := range factionCounts {
		if count < 2 || faction == "Abyssal Phenomena" {
			continue
		}
		for i := range mobs {
			if abyssEnemyFaction(&mobs[i]) == faction {
				mobs[i].Stats.STR = max(1, mobs[i].Stats.STR*105/100)
				mobs[i].Stats.DEF = max(0, mobs[i].Stats.DEF*105/100)
			}
		}
		logs = append(logs, fmt.Sprintf("⚔️ %s pack synergy: %d allies coordinate for +5%% STR/DEF.", faction, count))
	}
	if eliteCount >= 2 {
		logs = append(logs, "🔗 Coordinated elite pack: breaking one enemy creates the safest focus window.")
	}
	return mobs, logs
}

func applyAbyssBreakDamage(mob *content.Mob, damage int, logs *[]string) {
	if mob == nil || mob.MaxBreak <= 0 || damage <= 0 || mob.Stats.HP <= 0 {
		return
	}
	mob.Break -= max(1, damage/4)
	if mob.Break > 0 {
		return
	}
	mob.Break = 0
	mob.Stats.SPD = 0
	*logs = append(*logs, fmt.Sprintf("💥 %s is BROKEN and loses its next action!", mob.Name))
}

func (c *abyssLiveCombat) applyLiveBossAdaptation(
	mobs []*content.Mob,
	logs []string,
) []string {
	c.mu.Lock()
	if c.bossAdaptation != "" {
		adaptation := c.bossAdaptation
		c.mu.Unlock()
		applyBossAdaptation(mobs, adaptation)
		return logs
	}
	dominantKind := ""
	dominantCount := 0
	for kind, count := range c.actionCounts {
		if count > dominantCount {
			dominantKind = kind
			dominantCount = count
		}
	}
	if dominantCount < 3 {
		c.mu.Unlock()
		return logs
	}
	switch dominantKind {
	case "attack":
		c.bossAdaptation = "armored"
	case "skill", "ultimate":
		c.bossAdaptation = "fleet"
	case "defend", "item":
		c.bossAdaptation = "enraged"
	}
	adaptation := c.bossAdaptation
	c.mu.Unlock()
	if adaptation == "" || !applyBossAdaptation(mobs, adaptation) {
		return logs
	}
	return append(logs, fmt.Sprintf(
		"🧠 The boss adapts to repeated %s actions and becomes %s!",
		dominantKind,
		adaptation,
	))
}

func applyBossAdaptation(mobs []*content.Mob, adaptation string) bool {
	effect := content.EffectArmored
	switch adaptation {
	case "fleet":
		effect = content.EffectFleet
	case "enraged":
		effect = content.EffectEnraged
	case "armored":
	default:
		return false
	}
	changed := false
	for _, mob := range mobs {
		if mob == nil || (mob.Type != content.MobBoss && mob.Type != content.MobLegendary) {
			continue
		}
		hasEffect := false
		for _, existing := range mob.Effects {
			if existing == effect {
				hasEffect = true
				break
			}
		}
		if !hasEffect {
			mob.Effects = append(mob.Effects, effect)
			changed = true
		}
	}
	return changed
}

func (b *Bot) loadAbyssNemesis(uid string) (abyssNemesisState, bool) {
	var encoded string
	if err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssNemesisPrefix+uid).Scan(&encoded); err != nil {
		return abyssNemesisState{}, false
	}
	var state abyssNemesisState
	if json.Unmarshal([]byte(encoded), &state) != nil || state.Name == "" {
		return abyssNemesisState{}, false
	}
	return state, true
}

func (b *Bot) updateAbyssNemesis(uid string, mobs []*content.Mob, victory bool) {
	previous, hadPrevious := b.loadAbyssNemesis(uid)
	if victory {
		for _, mob := range mobs {
			if mob != nil && strings.TrimPrefix(mob.Name, "Nemesis: ") == previous.Name && mob.Stats.HP <= 0 {
				_, _ = b.DB.Exec("DELETE FROM app_meta WHERE key=$1", abyssNemesisPrefix+uid)
				return
			}
		}
		return
	}
	var killer *content.Mob
	for _, mob := range mobs {
		if mob != nil && mob.Stats.HP > 0 && !abyssEnemyHazard(mob) &&
			(killer == nil || mob.Score() > killer.Score()) {
			killer = mob
		}
	}
	if killer == nil {
		return
	}
	state := abyssNemesisState{Name: strings.TrimPrefix(killer.Name, "Nemesis: "), Level: killer.Level, Victories: 1}
	if hadPrevious && previous.Name == state.Name {
		state.Victories = previous.Victories + 1
	}
	encoded, err := json.Marshal(state)
	if err != nil {
		return
	}
	_, _ = b.DB.Exec(
		`INSERT INTO app_meta (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`,
		abyssNemesisPrefix+uid,
		string(encoded),
	)
}
