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
