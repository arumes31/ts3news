package rift

import "math"

type ObjectivePickup struct {
	ID        int     `json:"id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Collected bool    `json:"collected"`
}

type RoomObjective struct {
	Kind        string            `json:"kind"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Target      int               `json:"target"`
	Collected   int               `json:"collected"`
	Complete    bool              `json:"complete"`
	Pickups     []ObjectivePickup `json:"pickups"`
}

func (r *Run) beginRoomObjective() {
	r.RoomObjective = nil
	if r.Practice != nil || r.Arena().Objective != "sigils" {
		return
	}
	objective := &RoomObjective{Kind: "sigils", Name: "Gather the sigils", Description: "Walk over all three sigils and defeat every enemy to clear this tier.", Target: 3}
	for i, point := range [][2]float64{{430, 330}, {820, 480}, {1240, 330}} {
		actor := Actor{X: point[0], Y: point[1]}
		settle(&actor, r.Arena().solidObstacles())
		objective.Pickups = append(objective.Pickups, ObjectivePickup{ID: i + 1, X: actor.X, Y: actor.Y})
	}
	r.RoomObjective = objective
}

func (r *Run) collectRoomSigils() {
	objective := r.RoomObjective
	if objective == nil || objective.Kind != "sigils" || objective.Complete || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 || r.Player.Jump > .1 {
		return
	}
	for i := range objective.Pickups {
		pickup := &objective.Pickups[i]
		target := Actor{X: pickup.X, Y: pickup.Y}
		if pickup.Collected || math.Hypot(r.Player.X-pickup.X, r.Player.Y-pickup.Y) > 28 || !r.clearMeleePath(&r.Player, &target) {
			continue
		}
		pickup.Collected = true
		objective.Collected++
		r.event("sigil_pickup", pickup.X, pickup.Y, float64(objective.Collected))
	}
	objective.Complete = objective.Collected == objective.Target
}
