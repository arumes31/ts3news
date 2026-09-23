package rift

import (
	"fmt"
	"math"
	"slices"
	"time"

	"ts3news/internal/content"
)

const LevelCount = 100

type Obstacle struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

type Hazard struct {
	Obstacle
	Kind     string  `json:"kind"`
	Period   float64 `json:"period"`
	Offset   float64 `json:"offset"`
	Duration float64 `json:"duration"`
}

type Arena struct {
	MaxAttackers      int               `json:"max_attackers,omitempty"`
	HighCover         []Obstacle        `json:"high_cover,omitempty"`
	Name              string            `json:"name"`
	Obstacles         []Obstacle        `json:"obstacles"`
	Hazards           []Hazard          `json:"hazards"`
	Encounter         *EncounterPreview `json:"encounter,omitempty"`
	LootRarityCeiling string            `json:"loot_rarity_ceiling,omitempty"`
	Floor             string            `json:"floor,omitempty"`
}

// LootRarityCap is shared by campaign previews and server-owned gear rolls.
func LootRarityCap(room int) content.Rarity {
	if room == len(Rooms)-1 {
		return content.RarityLegendary
	}
	return content.RarityEpic
}

// EncounterPreview describes initial defenders and scaling relative to each
// monster's adapted Brawl template, before player defenses or combat effects.
type EncounterPreview struct {
	Enemies          int     `json:"enemies"`
	HealthMultiplier float64 `json:"health_multiplier"`
	DamageMultiplier float64 `json:"damage_multiplier"`
}

func missionExtraEnemies(id int) int         { return (id - 1) % 3 }
func missionHealthMultiplier(id int) float64 { return 1 + float64(id-1)*.004 }
func missionDamageMultiplier(id int) float64 { return 1 + float64(id-1)*.0025 }

// Level is copied into the saved expedition, so content changes never alter
// terrain underneath a player resuming a fight.
type Level struct {
	ID         int     `json:"id"`
	Region     int     `json:"region"`
	Name       string  `json:"name"`
	RegionName string  `json:"region_name"`
	Tactic     string  `json:"tactic"`
	Difficulty string  `json:"difficulty"`
	Color      string  `json:"color"`
	Rooms      []Arena `json:"rooms"`
}

// Campaign derives all missions from ten arena blueprints and ten regional
// rulesets. Every mission has different collision geometry and hazard timing.
func Campaign() []Level {
	regions := []string{"Mossbound Ruins", "Ember Forge", "Glacial Crossing", "Storm Spires", "Venom Mire", "Drowned Temple", "Bloodrust Barracks", "Moonlit Necropolis", "Starless Rift", "Obsidian Citadel"}
	names := []string{"Pilgrim's Gate", "Broken Well", "Pillar Watch", "Crossroads", "Twin Bastions", "Serpent Walk", "Hidden Alcoves", "Shattered Bridge", "Sentinel Rows", "Crown Arena"}
	tactics := []string{"An open approach with scattered cover", "Circle the well to flank the patrol", "Weave between staggered pillars", "Choose your crossing between four posts", "Use the gap between twin barricades", "Jump low walls along the winding route", "Draw defenders out of their alcoves", "Cross the broken spans between pulses", "Switch lanes between sentry lines", "Circle the ring and challenge its guardian"}
	colors := []string{"#a6ce7b", "#ff9b53", "#8bdfff", "#c6a1ff", "#bcdf64", "#65ded2", "#ed8d7c", "#bec4ff", "#be98ff", "#ffc77c"}
	kinds := []string{"thorns", "fire", "ice", "rune", "poison", "ice", "fire", "radiant", "void", "fire"}
	floors := []string{"grass", "metal", "ice", "stone", "mud", "water", "wood", "stone", "stone", "metal"}
	// Coordinates describe low cover, with a continuous bypass above and below.
	patterns := [][]Obstacle{
		{{780, 360, 65, 28}},
		{{760, 390, 110, 48}},
		{{450, 357, 42, 32}, {790, 424, 42, 32}, {1130, 357, 42, 32}},
		{{660, 359, 42, 27}, {930, 359, 42, 27}, {660, 434, 42, 27}, {930, 434, 42, 27}},
		{{620, 379, 95, 52}, {1010, 379, 95, 52}},
		{{460, 356, 90, 30}, {750, 429, 90, 30}, {1080, 356, 90, 30}},
		{{500, 361, 100, 28}, {540, 436, 60, 28}, {1070, 361, 100, 28}, {1070, 436, 60, 28}},
		{{480, 393, 105, 32}, {795, 393, 105, 32}, {1110, 393, 105, 32}},
		{{490, 364, 38, 28}, {750, 426, 38, 28}, {1000, 364, 38, 28}, {1250, 426, 38, 28}},
		{{650, 375, 40, 45}, {980, 375, 40, 45}, {800, 348, 65, 26}, {800, 444, 65, 26}},
	}
	levels := make([]Level, 0, LevelCount)
	for region, regionName := range regions {
		for layout, name := range names {
			id := region*10 + layout + 1
			level := Level{ID: id, Region: region, RegionName: regionName, Name: regionName + " · " + name, Tactic: tactics[layout], Color: colors[region], Difficulty: []string{"Wayfarer", "Veteran", "Champion", "Mythic"}[min(3, id/26)]}
			for room, suffix := range []string{"Approach", "Inner Court", "Guardian's Stand"} {
				arena := Arena{Name: name + " / " + suffix, Obstacles: []Obstacle{}, Hazards: []Hazard{}, Floor: floors[region]}
				arena.MaxAttackers = 2 + room
				arena.LootRarityCeiling = LootRarityCap(room).String()
				arena.Encounter = &EncounterPreview{Enemies: encounterCounts[room] + missionExtraEnemies(id), HealthMultiplier: roomHealthMultiplier(room) * missionHealthMultiplier(id), DamageMultiplier: missionDamageMultiplier(id)}
				for i, obstacle := range patterns[layout] {
					obstacle.X += float64(region*7 + room*19)
					obstacle.Y += float64((region+room+i)%3-1) * 4
					obstacle.W += float64(region%4) * 3
					if layout == 2 && i == 0 {
						arena.HighCover = append(arena.HighCover, obstacle)
					} else {
						arena.Obstacles = append(arena.Obstacles, obstacle)
					}
				}
				for h := 0; h < 1+(layout+room)%3; h++ {
					arena.Hazards = append(arena.Hazards, Hazard{Obstacle: Obstacle{X: 390 + float64((layout*91+region*47+room*73+h*310)%940), Y: 335 + float64((layout+region+room+h)%3)*49, W: 90 + float64(region)*5, H: 32}, Kind: kinds[region], Period: 7 - float64(region)*.23, Offset: float64((layout+room+h)%5) * .7, Duration: .8 + float64(layout%3)*.2})
				}
				level.Rooms = append(level.Rooms, arena)
			}
			levels = append(levels, level)
		}
	}
	return levels
}

func NewRunAtLevel(id string, build Build, now time.Time, catalog []content.Mob, levelID int) *Run {
	r := NewRunWithCatalog(id, build, now, catalog)
	r.setLevel(max(1, min(LevelCount, levelID)), catalog)
	return r
}

func (r *Run) setLevel(id int, catalog []content.Mob) {
	level := Campaign()[id-1]
	r.Level = &level
	r.beginMissionHistory()
	r.RoomSplits = [3]*float64{}
	r.Room = 0
	// The whole shared bestiary remains eligible, including future additions.
	r.EncounterPlan = planEncounters(fmt.Sprintf("%s-level-%d", r.ID, id), catalog)
	for room := range r.EncounterPlan {
		actors := r.EncounterPlan[room]
		if len(actors) == 0 {
			continue
		}
		for extra := 0; extra < missionExtraEnemies(id); extra++ {
			actors = append(actors, actors[1%len(actors)])
		}
		for i := range actors {
			a := &actors[i]
			a.ID = fmt.Sprintf("l%d-r%d-e%d", id, room, i)
			a.X = 540 + float64(i)*170 + float64((id+room)%4)*25
			a.Y = 330 + float64((i+id+room)%4)*48
			a.HP *= missionHealthMultiplier(id)
			a.MaxHP = a.HP
			a.Damage *= missionDamageMultiplier(id)
			settle(a, level.Rooms[room].solidObstacles())
		}
		r.EncounterPlan[room] = actors
	}
	r.spawnRoom()
}

func (r *Run) Arena() Arena {
	if r.Practice != nil {
		return r.Practice.Arena
	}
	if r.Level != nil && r.Room >= 0 && r.Room < len(r.Level.Rooms) {
		return r.Level.Rooms[r.Room]
	}
	return Arena{}
}

func (r *Run) FloorMaterial() string {
	if a := r.Arena(); a.Floor != "" {
		return a.Floor
	}
	if r.Level != nil && r.Level.Region >= 0 && r.Level.Region < 10 {
		floors := []string{"grass", "metal", "ice", "stone", "mud", "water", "wood", "stone", "stone", "metal"}
		return floors[r.Level.Region]
	}
	return "stone"
}

// FinishCheckpoint is called only after rewards have been banked atomically.
// Advancing keeps the same run, build and receipt, without a page reload.
func (r *Run) FinishCheckpoint(kind string, catalog []content.Mob) {
	if r.Practice != nil {
		return
	}
	if r.Status != "cleared" {
		return
	}
	r.SetPaused(false, time.UnixMilli(r.LastMS))
	if r.Room == len(Rooms)-1 {
		r.finishMissionHistory("completed")
	} else if kind == "exit" {
		r.finishMissionHistory("exited")
	}
	if r.Room == len(Rooms)-1 && r.Level != nil && !slices.Contains(r.CompletedLevels, r.Level.ID) {
		r.CompletedLevels = append(r.CompletedLevels, r.Level.ID)
		slices.Sort(r.CompletedLevels)
	}
	if kind == "exit" {
		r.Status = "banked"
		return
	}
	if kind == "bank" {
		if r.Room == len(Rooms)-1 {
			r.Status = "complete"
			if r.Player.Pose != "victory" {
				r.event("victory", r.Player.X, r.Player.Y-30, 0)
			}
			r.Player.Pose = "victory"
			r.Player.PoseTime = 4.0
		}
		return
	}
	if r.NextRoom() {
		return
	}
	if kind == "advance" && r.Level != nil && r.Level.ID < LevelCount {
		r.Status = "fighting"
		r.Player.HP = math.Min(r.Player.MaxHP, r.Player.HP+r.Player.MaxHP*.25)
		r.Player.Mana = 100
		// These drops are already in the real inventory; keep the receipt but
		// bound the active snapshot to one mission's drops.
		r.Drops = []Drop{}
		r.setLevel(r.Level.ID+1, catalog)
		return
	}
	r.Status = "complete"
	if r.Player.Pose != "victory" {
		r.event("victory", r.Player.X, r.Player.Y-30, 0)
	}
	r.Player.Pose = "victory"
	r.Player.PoseTime = 4.0
}

func contains(o Obstacle, x, y, radius float64) bool {
	return x > o.X-radius && x < o.X+o.W+radius && y > o.Y-radius && y < o.Y+o.H+radius
}

// actorClearance is the ground footprint used by collision and navigation.
func actorClearance(a *Actor) float64 {
	if a.ID == "player" {
		return 10
	}
	switch a.Kind {
	case "boss":
		return 18
	case "treasure", "wolf":
		return 6
	default:
		return 10
	}
}

func settle(a *Actor, obstacles []Obstacle) {
	radius := actorClearance(a)
	for _, o := range obstacles {
		if !contains(o, a.X, a.Y, radius) {
			continue
		}
		distances := []float64{a.X - o.X + radius, o.X + o.W + radius - a.X, a.Y - o.Y + radius, o.Y + o.H + radius - a.Y}
		switch slices.Index(distances, slices.Min(distances)) {
		case 0:
			a.X = o.X - radius
		case 1:
			a.X = o.X + o.W + radius
		case 2:
			a.Y = o.Y - radius
		case 3:
			a.Y = o.Y + o.H + radius
		}
	}
	a.X = clamp(a.X, 35, Width-35)
	a.Y = clamp(a.Y, 315, 490)
}

func (a Arena) solidObstacles() []Obstacle {
	if len(a.HighCover) == 0 {
		return a.Obstacles
	}
	all := make([]Obstacle, 0, len(a.Obstacles)+len(a.HighCover))
	all = append(all, a.Obstacles...)
	return append(all, a.HighCover...)
}

// obstacleImpact returns the first intersection along a movement segment.
func obstacleImpact(x1, y1, x2, y2 float64, o Obstacle) (float64, bool) {
	enter, leave := 0.0, 1.0
	for _, axis := range [][4]float64{{x1, x2 - x1, o.X, o.X + o.W}, {y1, y2 - y1, o.Y, o.Y + o.H}} {
		start, delta, low, high := axis[0], axis[1], axis[2], axis[3]
		if delta == 0 {
			if start < low || start > high {
				return 0, false
			}
			continue
		}
		a, b := (low-start)/delta, (high-start)/delta
		if a > b {
			a, b = b, a
		}
		enter, leave = math.Max(enter, a), math.Min(leave, b)
		if enter > leave {
			return 0, false
		}
	}
	return enter, true
}

func (r *Run) clearProjectilePath(from, to *Actor) bool {
	for _, wall := range r.Arena().HighCover {
		if _, hit := obstacleImpact(from.X, from.Y, to.X, to.Y, wall); hit {
			return false
		}
	}
	return true
}

// clearMeleePath rejects segments that touch or cross a solid arena obstacle.
func (r *Run) clearMeleePath(from, to *Actor) bool {
	for _, o := range r.Arena().solidObstacles() {
		if _, hit := obstacleImpact(from.X, from.Y, to.X, to.Y, o); hit {
			return false
		}
	}
	return true
}

func (r *Run) clearPursuitPath(from, to *Actor) bool {
	arena := r.Arena()
	obstacles := arena.solidObstacles()
	if from.Jump > .1 {
		obstacles = arena.HighCover
	}
	radius := actorClearance(from) + 2
	for _, o := range obstacles {
		clearance := Obstacle{X: o.X - radius, Y: o.Y - radius, W: o.W + radius*2, H: o.H + radius*2}
		if _, hit := obstacleImpact(from.X, from.Y, to.X, to.Y, clearance); hit {
			return false
		}
	}
	return true
}

// knockbackActor uses short sweeps so an impulse cannot skip a thin obstacle.
func (r *Run) knockbackActor(a *Actor, dx, dy float64) {
	a.RouteX, a.RouteY = 0, 0
	steps := max(1, int(math.Ceil(math.Hypot(dx, dy)/8)))
	for range steps {
		r.moveActor(a, dx/float64(steps), dy/float64(steps), false)
	}
}

// detourLane tries the other side when the preferred passage is outside the
// arena or another obstacle has closed its horizontal crossing.
func detourLane(a *Actor, wall Obstacle, obstacles []Obstacle) float64 {
	radius := actorClearance(a)
	above, below := wall.Y-radius-4, wall.Y+wall.H+radius+4
	choices := []float64{above, below}
	if a.Y >= wall.Y+wall.H/2 {
		choices[0], choices[1] = below, above
	}
	for _, y := range choices {
		if y < 315 || y > 490 {
			continue
		}
		clear := true
		for _, o := range obstacles {
			expanded := Obstacle{o.X - radius, o.Y - radius, o.W + 2*radius, o.H + 2*radius}
			if _, hit := obstacleImpact(wall.X-radius-5, y, wall.X+wall.W+radius+5, y, expanded); hit {
				clear = false
				break
			}
		}
		if clear {
			return y
		}
	}
	return choices[0]
}

func (r *Run) moveActor(a *Actor, dx, dy float64, navigate bool) {
	arena := r.Arena()
	obstacles := arena.solidObstacles()
	if a.Jump > .1 {
		obstacles = arena.HighCover
	}
	radius := actorClearance(a)
	settle(a, obstacles)
	for _, o := range obstacles {
		if navigate && dx != 0 && contains(o, a.X+dx, a.Y, radius+2) {
			// Keep the bypass until the actor is beyond this obstacle in X.
			a.RouteY = detourLane(a, o, obstacles)
			a.RouteX = o.X + o.W + radius + 5
			if dx < 0 {
				a.RouteX = o.X - radius - 5
			}
		}
	}
	if navigate && a.RouteY != 0 {
		if dx > 0 && a.X >= a.RouteX || dx < 0 && a.X <= a.RouteX {
			a.RouteY = 0
		} else {
			dy = clamp(a.RouteY-a.Y, -math.Abs(dx), math.Abs(dx))
		}
	}
	nextX := clamp(a.X+dx, 35, Width-35)
	for _, o := range obstacles {
		if contains(o, nextX, a.Y, radius) {
			nextX = a.X
			break
		}
	}
	a.X = nextX
	nextY := clamp(a.Y+dy, 315, 490)
	for _, o := range obstacles {
		if contains(o, a.X, nextY, radius) {
			nextY = a.Y
			break
		}
	}
	a.Y = nextY
}

func (h Hazard) Phase(clock float64) float64 { return math.Mod(clock+h.Offset, h.Period) }

func (r *Run) hazardTick() {
	if r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	if r.SkillTimers == nil {
		r.SkillTimers = map[string]float64{}
	}
	for i, h := range r.Arena().Hazards {
		phase := h.Phase(r.Clock)
		warnKey := fmt.Sprintf("hazard-warn-%d", i)
		activeKey := fmt.Sprintf("hazard-active-%d", i)
		if phase < 1.2 {
			if r.SkillTimers[warnKey] <= 0 {
				r.SkillTimers[warnKey] = math.Max(1.5, h.Period-phase+.05)
				r.event("hazard_warning", h.X+h.W/2, h.Y+h.H/2, 0)
			}
		} else if phase >= 1.2 && phase < 1.2+h.Duration {
			r.SkillTimers[activeKey] = 1
		} else if phase >= 1.2+h.Duration {
			if r.SkillTimers[activeKey] > 0 {
				r.SkillTimers[activeKey] = 0
				r.event("hazard_deactivation", h.X+h.W/2, h.Y+h.H/2, 0)
			}
		}
		if phase < 1.2 || phase >= 1.2+h.Duration || !contains(h.Obstacle, r.Player.X, r.Player.Y, 0) || r.Player.Jump > .1 {
			continue
		}
		key := fmt.Sprintf("hazard-%d", i)
		if r.SkillTimers[key] > 0 || r.SkillTimers["hazard-hit"] > 0 {
			continue
		}
		r.SkillTimers[key] = 1
		r.SkillTimers["hazard-hit"] = .35
		region := 0
		if r.Level != nil {
			region = r.Level.Region
		}
		r.hurtPlayer(12+float64(region), r.Player.X, r.Player.Y)
		if r.Player.HP <= 0 {
			return
		}
		switch h.Kind {
		case "ice", "thorns", "poison":
			r.SkillTimers["slowed"] = 1.4
		case "void":
			r.moveActor(&r.Player, (h.X+h.W/2-r.Player.X)*.3, 0, false)
		}
		r.event(h.Kind, r.Player.X, r.Player.Y, 0)
	}
}
