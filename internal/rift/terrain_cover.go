package rift

import "math"

// TerrainCover is saved with the arena; broken wood stays broken on reload.
type TerrainCover struct {
	Shortcut bool `json:"shortcut,omitempty"`
	Obstacle
	ID       string  `json:"id"`
	Material string  `json:"material"`
	HP       float64 `json:"hp"`
	MaxHP    float64 `json:"max_hp"`
}

func (c TerrainCover) solid() bool { return c.Material == "stone" || c.HP > 0 }

func (a Arena) tallObstacles() []Obstacle {
	if len(a.Cover) == 0 {
		return a.HighCover
	}
	walls := append([]Obstacle(nil), a.HighCover...)
	for _, c := range a.Cover {
		if c.solid() {
			walls = append(walls, c.Obstacle)
		}
	}
	return walls
}

func (r *Run) damageTerrainCover(index int, damage float64) float64 {
	if r.Level == nil || index < 0 || index >= len(r.Level.Rooms[r.Room].Cover) || damage <= 0 || math.IsNaN(damage) || math.IsInf(damage, 0) {
		return 0
	}
	c := &r.Level.Rooms[r.Room].Cover[index]
	if c.Material != "wood" || c.HP <= 0 {
		return 0
	}
	dealt := math.Min(c.HP, damage)
	c.HP -= dealt
	kind := "cover_hit"
	if c.HP == 0 {
		kind = "cover_break"
	}
	r.event(kind, c.X+c.W/2, c.Y+c.H/2, dealt)
	return dealt
}

// projectileCoverImpact selects the first wall, never a farther destructible prop.
func (r *Run) projectileCoverImpact(x1, y1, x2, y2 float64) (float64, int) {
	arena := r.Arena()
	impact, index := 2.0, -1
	for _, wall := range arena.HighCover {
		if t, hit := obstacleImpact(x1, y1, x2, y2, wall); hit && t < impact {
			impact = t
		}
	}
	for i, c := range arena.Cover {
		if c.solid() {
			if t, hit := obstacleImpact(x1, y1, x2, y2, c.Obstacle); hit && t < impact {
				impact, index = t, i
			}
		}
	}
	return impact, index
}

func (r *Run) attackTerrainCover(damage float64) bool {
	arena := r.Arena()
	p := &r.Player
	nearest, index := math.Inf(1), -1
	for i, c := range arena.Cover {
		if c.Material != "wood" || !c.solid() {
			continue
		}
		target := Actor{X: clamp(p.X, c.X, c.X+c.W), Y: clamp(p.Y, c.Y, c.Y+c.H)}
		if !inBasicMeleeRange(p, &target) {
			continue
		}
		blocked := false
		for _, wall := range append(append([]Obstacle(nil), arena.Obstacles...), arena.HighCover...) {
			if _, hit := obstacleImpact(p.X, p.Y, target.X, target.Y, wall); hit {
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}
		for j, other := range arena.Cover {
			if i != j && other.solid() {
				if _, hit := obstacleImpact(p.X, p.Y, target.X, target.Y, other.Obstacle); hit {
					blocked = true
					break
				}
			}
		}
		distance := math.Hypot(target.X-p.X, target.Y-p.Y)
		if !blocked && distance < nearest {
			nearest, index = distance, i
		}
	}
	if r.damageTerrainCover(index, damage) > 0 {
		c := arena.Cover[index]
		r.geomancerTerrainCue(c.X+c.W/2, c.Y+c.H/2)
		return true
	}
	return false
}

func (r *Run) geomancerTerrainCue(x, y float64) {
	if r.Build.Class == "geomancer" {
		r.event("geomancer_terrain", x, y, 0)
	}
}
