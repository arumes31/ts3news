package rift

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"time"
	"ts3news/internal/content"
	"ts3news/internal/i18n"
)

// AdaptMonster reads identity and relative strengths from the canonical Abyss
// template. Only the conversion to action-combat units belongs to Brawl.
func AdaptMonster(m content.Mob) Actor {
	a := Actor{Name: m.Name, ArtKey: "monster:" + m.Name, Tier: string(m.Type), Element: strings.ToLower(string(m.Element)), Kind: "goblin", Facing: -1, Phase: 1}
	a.HP = 55 + math.Log2(1+float64(max(0, m.Stats.HP)))*4
	a.Damage = 12 + math.Log2(1+float64(max(0, max(m.Stats.STR, m.Stats.INT))))*2
	a.Armor = math.Min(.45, math.Log2(1+float64(max(0, m.Stats.DEF)))*.035)
	a.Speed = 70 + math.Log2(1+float64(max(0, m.Stats.SPD)))*8
	switch m.Type {
	case content.MobEliteMinion:
		a.HP *= 1.2
		a.Kind = "knight"
		a.Elite = true
	case content.MobElite:
		a.HP *= 1.5
		a.Kind = "knight"
		a.Elite = true
	case content.MobMiniboss:
		a.HP *= 2.2
		a.Kind = "knight"
		a.Elite = true
	case content.MobBoss, content.MobLegendary:
		a.HP *= 6
		a.Kind = "boss"
	case content.MobTreasureGoblin:
		a.Kind = "treasure"
		a.Speed *= 1.2
	}
	name := strings.ToLower(m.Name)
	ranged := len(m.Spells) > 0 || m.Stats.INT > m.Stats.STR
	for _, word := range []string{"lich", "archer", "mage", "wizard", "scribe", "weaver"} {
		ranged = ranged || strings.Contains(name, word)
	}
	if ranged && a.Kind != "boss" && a.Kind != "treasure" {
		a.Kind = "archer"
	}
	if a.Kind == "knight" {
		a.Shield = true
	}
	if a.Kind == "goblin" || a.Kind == "wolf" {
		a.Pack = true
	}
	a.Shot = monsterElementEffect(a.Element)
	if a.Shot == "" {
		a.Shot = "arrow"
	}
	if a.Element == "" || a.Element == "physical" {
		for _, theme := range []struct{ word, effect string }{{"fiery", "fire"}, {"fire", "fire"}, {"frost", "ice"}, {"ice", "ice"}, {"toxic", "poison"}, {"void", "void"}, {"shadow", "void"}} {
			if strings.Contains(name, theme.word) {
				a.Shot = theme.effect
				break
			}
		}
	}
	a.Healer = a.Healer || m.Name == i18n.T("mob.frost_lich")
	a.Charging = (a.Kind == "goblin" || a.Kind == "knight") && (a.Charging || m.Name == i18n.T("mob.raging_behemoth"))
	a.MaxHP = a.HP
	return a
}

func monsterElementEffect(element string) string {
	switch element {
	case "fire":
		return "fire"
	case "water", "frost":
		return "ice"
	case "earth", "nature":
		return "poison"
	case "air", "storm":
		return "rune"
	case "void", "shadow":
		return "void"
	case "holy", "spirit":
		return "radiant"
	}
	return ""
}

func computeReplaySeed(id string) uint64 {
	seed := sha256.Sum256([]byte(id))
	return binary.LittleEndian.Uint64(seed[:8]) & 0x1fffffffffffff
}

func newRunState(id string, build Build, now time.Time) *Run {
	r := &Run{Schema: 1, ID: id, ReplaySeed: computeReplaySeed(id), Build: build, Status: "fighting", LastMS: now.UnixMilli(), SavedAtMS: now.UnixMilli(), SkillTimers: map[string]float64{}, Drops: []Drop{}, BankedItems: []string{}}
	r.Player = Actor{ID: "player", Name: build.Name, Kind: build.Class, X: 160, Y: 410, HP: build.HP, MaxHP: build.HP, Mana: 100, Facing: 1, GuardStamina: 100}
	return r
}

// NewRunWithCatalog freezes only this expedition's encounters. Each new run
// reads the current catalog; a content update cannot rewrite a fight in progress.
func NewRunWithCatalog(id string, build Build, now time.Time, catalog []content.Mob) *Run {
	r := newRunState(id, build, now)
	r.EncounterPlan = planEncounters(id, catalog)
	r.spawnRoom()
	r.event("arrival", r.Player.X, r.Player.Y, 0)
	return r
}

var encounterCounts = [...]int{3, 4, 3}

func roomHealthMultiplier(room int) float64 { return 1 + float64(room)*.15 }

func planEncounters(id string, catalog []content.Mob) [][]Actor {
	var regular, bosses []Actor
	for _, mob := range catalog {
		a := AdaptMonster(mob)
		if a.Kind == "boss" {
			bosses = append(bosses, a)
		} else {
			regular = append(regular, a)
		}
	}
	if len(regular) == 0 {
		regular = bosses
	}
	if len(bosses) == 0 {
		bosses = regular
	}
	if len(regular) == 0 {
		return make([][]Actor, len(Rooms))
	}
	seed := sha256.Sum256([]byte(id))
	rng := rand.New(rand.NewPCG(binary.LittleEndian.Uint64(seed[:8]), binary.LittleEndian.Uint64(seed[8:16])))
	rng.Shuffle(len(regular), func(i, j int) { regular[i], regular[j] = regular[j], regular[i] })
	plan := make([][]Actor, len(Rooms))
	cursor := 0
	for room, count := range encounterCounts {
		for index := 0; index < count; index++ {
			a := regular[cursor%len(regular)]
			cursor++
			if room == len(Rooms)-1 && index == 0 {
				a = bosses[rng.IntN(len(bosses))]
				a.Kind = "boss"
				a.Phase = 1
			}
			a.ID = fmt.Sprintf("r%d-e%d", room, index)
			a.X = 580 + float64(index)*240
			a.Y = 355 + float64(index%3)*50
			a.Cooldown = 1 + float64(index)*.3
			a.HP *= roomHealthMultiplier(room)
			a.MaxHP = a.HP
			plan[room] = append(plan[room], a)
		}
	}
	return plan
}
