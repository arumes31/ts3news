package rift

import "math"

// WaveFloorSegment saves a warned panel and its missing-ground state. Occupied
// panels wait until every living footprint has left before becoming impassable.
type WaveFloorSegment struct {
	Obstacle
	CollapseIn float64 `json:"collapse_in"`
	Collapsed  bool    `json:"collapsed"`
}

func (r *Run) warnWaveFloor() {
	for i := range r.RoomObjective.FloorSegments {
		panel := &r.RoomObjective.FloorSegments[i]
		panel.Collapsed = false
		panel.CollapseIn = 3 + float64(i)*.6
		r.event("floor_warning", panel.X+panel.W/2, panel.Y+panel.H/2, panel.CollapseIn)
	}
}

func (r *Run) tickWaveFloor(dt float64) {
	for i := range r.RoomObjective.FloorSegments {
		panel := &r.RoomObjective.FloorSegments[i]
		if panel.Collapsed {
			continue
		}
		panel.CollapseIn = math.Max(0, panel.CollapseIn-dt)
		if panel.CollapseIn > 1e-9 {
			continue
		}
		panel.CollapseIn = 0
		occupied := func(actor *Actor) bool {
			return actor.HP > 0 && contains(panel.Obstacle, actor.X, actor.Y, actorClearance(actor)+2)
		}
		blocked := occupied(&r.Player)
		for j := range r.Enemies {
			blocked = blocked || occupied(&r.Enemies[j])
		}
		if blocked {
			continue
		}
		panel.Collapsed = true
		r.event("floor_collapse", panel.X+panel.W/2, panel.Y+panel.H/2, 0)
	}
}

func (r *Run) restoreWaveFloor() {
	for i := range r.RoomObjective.FloorSegments {
		panel := &r.RoomObjective.FloorSegments[i]
		if panel.Collapsed {
			r.event("floor_restore", panel.X+panel.W/2, panel.Y+panel.H/2, 0)
		}
		panel.Collapsed = false
		panel.CollapseIn = 0
	}
}

// ValidFragileFloor leaves clear ground above and below each bounded panel.
func (a Arena) ValidFragileFloor() bool {
	if len(a.FragileFloor) > 4 {
		return false
	}
	if len(a.FragileFloor) > 0 && (a.Objective != "survive_waves" || a.Round != nil || len(a.Bridges) > 0) {
		return false
	}
	for i, p := range a.FragileFloor {
		for _, v := range []float64{p.X, p.Y, p.W, p.H} {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return false
			}
		}
		if p.X < 100 || p.X+p.W > 1500 || p.W < 60 || p.W > 160 || p.Y < 365 || p.Y+p.H > 445 || p.H < 30 || p.H > 70 {
			return false
		}
		for _, other := range a.FragileFloor[:i] {
			if p.X < other.X+other.W+80 && other.X < p.X+p.W+80 {
				return false
			}
		}
	}
	return true
}

// ValidWaveFloor matches live floor state to the expedition's frozen geometry.
// Content updates never replace that saved definition with today's campaign.
func (r *Run) ValidWaveFloor() bool {
	var frozen []Obstacle
	if r.Practice == nil && r.Level != nil && r.Room >= 0 && r.Room < len(r.Level.Rooms) {
		frozen = r.Level.Rooms[r.Room].FragileFloor
	}
	o := r.RoomObjective
	if o == nil || o.Kind != "survive_waves" {
		return len(frozen) == 0 && (o == nil || len(o.FloorSegments) == 0)
	}
	if len(o.FloorSegments) != len(frozen) {
		return false
	}
	for i, p := range o.FloorSegments {
		if p.Obstacle != frozen[i] || math.IsNaN(p.CollapseIn) || math.IsInf(p.CollapseIn, 0) || p.CollapseIn < 0 || p.CollapseIn > 3+float64(i)*.6 {
			return false
		}
		if p.Collapsed && p.CollapseIn != 0 {
			return false
		}
		if (o.Complete || o.NextWaveSeconds > 0) && (p.Collapsed || p.CollapseIn != 0) {
			return false
		}
	}
	return true
}
