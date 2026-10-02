package rift

import "math"

const terrainReactionLimit = 8
const terrainReactionWarning = 1.2

// armTerrainReaction consumes one connected cluster before warning it. Removing
// each selected prop from eligibility makes cyclic layouts terminate. All linked
// props share a fuse, so overlapping blast circles cannot multiply actor damage.
func (r *Run) armTerrainReaction(index int) {
	if r.Level == nil || r.Room < 0 || r.Room >= len(r.Level.Rooms) || r.Status != "fighting" {
		return
	}
	arena := &r.Level.Rooms[r.Room]
	if index < 0 || index >= len(arena.Cover) {
		return
	}
	root := &arena.Cover[index]
	if !root.Volatile || root.Material != "wood" || root.HP != 0 || root.BlastFuse != 0 {
		return
	}
	root.BlastFuse = terrainReactionWarning
	queue := []int{index}
	for head := 0; head < len(queue) && len(queue) < terrainReactionLimit; head++ {
		origin := arena.Cover[queue[head]]
		from := Actor{X: origin.X + origin.W/2, Y: origin.Y + origin.H/2}
		for next := range arena.Cover {
			c := &arena.Cover[next]
			if !c.Volatile || c.Material != "wood" || c.HP <= 0 || c.BlastFuse != 0 {
				continue
			}
			to := Actor{X: c.X + c.W/2, Y: c.Y + c.H/2}
			if !blastContains(&from, &to) || !terrainReactionLinkClear(*arena, from, to) {
				continue
			}
			c.HP, c.BlastFuse = 0, terrainReactionWarning
			queue = append(queue, next)
			if len(queue) == terrainReactionLimit {
				break
			}
		}
	}
	for _, index := range queue {
		c := arena.Cover[index]
		r.event("terrain_arming", c.X+c.W/2, c.Y+c.H/2, terrainReactionWarning)
	}
}

// Volatile props can relay ignition. Other solid cover blocks that link.
func terrainReactionLinkClear(arena Arena, from, to Actor) bool {
	for _, walls := range [][]Obstacle{arena.Obstacles, arena.HighCover} {
		for _, wall := range walls {
			if _, hit := obstacleImpact(from.X, from.Y, to.X, to.Y, wall); hit {
				return false
			}
		}
	}
	for _, c := range arena.Cover {
		if c.solid() && (!c.Volatile || c.Material != "wood") {
			if _, hit := obstacleImpact(from.X, from.Y, to.X, to.Y, c.Obstacle); hit {
				return false
			}
		}
	}
	return true
}

func (r *Run) terrainReactionTick(dt float64) {
	if r.Level == nil || r.Room < 0 || r.Room >= len(r.Level.Rooms) || r.Paused || r.Status != "fighting" || dt <= 0 || math.IsNaN(dt) || math.IsInf(dt, 0) {
		return
	}
	var blasts []Actor
	for i := range r.Level.Rooms[r.Room].Cover {
		c := &r.Level.Rooms[r.Room].Cover[i]
		if !c.Volatile || c.Material != "wood" || c.HP != 0 || c.BlastFuse <= 0 || math.IsNaN(c.BlastFuse) || math.IsInf(c.BlastFuse, 0) {
			continue
		}
		c.BlastFuse = math.Max(0, math.Min(terrainReactionWarning, c.BlastFuse)-dt)
		if c.BlastFuse < 1e-9 {
			c.BlastFuse = 0
		} else {
			continue
		}
		// A malformed saved room must not produce an unbounded event burst.
		if len(blasts) < terrainReactionLimit {
			blasts = append(blasts, Actor{X: c.X + c.W/2, Y: c.Y + c.H/2})
		}
	}
	if len(blasts) == 0 {
		return
	}
	for _, from := range blasts {
		r.event("terrain_blast", from.X, from.Y, 0)
	}
	contact := func(target *Actor) *Actor {
		if target.HP <= 0 || target.Jump >= .1 || target.Burrowed || target.isObjectiveProp() {
			return nil
		}
		for i := range blasts {
			if blastContains(&blasts[i], target) && r.clearMeleePath(&blasts[i], target) {
				return &blasts[i]
			}
		}
		return nil
	}
	damage := 12 + float64(max(0, min(9, r.Level.Region)))
	if from := contact(&r.Player); from != nil {
		r.hurtPlayerFromNamedHazard(damage, from.X, from.Y, HazardDefeat{Kind: "volatile cover", Jumpable: true})
	}
	for i := range r.Enemies {
		if contact(&r.Enemies[i]) != nil {
			r.applyEnemyDamage(i, damage, "hit", 0, true)
		}
	}
}
