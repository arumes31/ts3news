package rift

import "math"

func (r *Run) beginRuneGateObjective() {
	orders := [][3]int{{1, 2, 3}, {1, 3, 2}, {2, 1, 3}, {2, 3, 1}, {3, 1, 2}, {3, 2, 1}}
	mission := 1
	if r.Level != nil {
		mission = r.Level.ID
	}
	order := orders[(mission-1)%len(orders)]
	o := &RoomObjective{Kind: "rune_gate", Name: "Open the rune gate", Description: "After defeating the patrol, step on the three seals in the displayed order. A wrong seal resets the sequence without damage. Step off a seal before activating another.", Target: 3, Sequence: append([]int(nil), order[:]...)}
	for i, point := range [][2]float64{{430, 330}, {800, 480}, {1200, 330}} {
		seal := Actor{X: point[0], Y: point[1]}
		settle(&seal, r.Arena().solidObstacles())
		o.Pickups = append(o.Pickups, ObjectivePickup{ID: i + 1, X: seal.X, Y: seal.Y})
	}
	r.RoomObjective = o
}

func (r *Run) tickRuneGateObjective() {
	o := r.RoomObjective
	if o == nil || o.Kind != "rune_gate" || o.Complete || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 || r.Player.Jump > .1 {
		return
	}
	for _, e := range r.Enemies {
		if e.HP > 0 {
			o.PuzzleTouch = 0
			return
		}
	}
	if r.Level != nil {
		for i := range r.Level.Rooms[r.Room].Hazards {
			r.Level.Rooms[r.Room].Hazards[i].Disabled = true
		}
	}
	touched := 0
	for _, seal := range o.Pickups {
		target := Actor{X: seal.X, Y: seal.Y}
		if math.Hypot(r.Player.X-seal.X, r.Player.Y-seal.Y) <= 28 && r.clearMeleePath(&r.Player, &target) {
			touched = seal.ID
			break
		}
	}
	if touched == o.PuzzleTouch {
		return
	}
	o.PuzzleTouch = touched
	if touched == 0 {
		return
	}
	if touched != o.Sequence[o.Collected] {
		o.Collected = 0
		for i := range o.Pickups {
			o.Pickups[i].Collected = false
		}
		r.event("rune_wrong", r.Player.X, r.Player.Y, 0)
		return
	}
	for i := range o.Pickups {
		if o.Pickups[i].ID == touched {
			o.Pickups[i].Collected = true
		}
	}
	o.Collected++
	r.event("rune_correct", r.Player.X, r.Player.Y, float64(o.Collected))
	if o.Collected == o.Target {
		o.Complete = true
		r.event("rune_gate_open", 1450, 330, 0)
	}
}
