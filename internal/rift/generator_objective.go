package rift

import (
	"fmt"
	"math"
)

func (a Actor) isObjectiveProp() bool { return a.Kind == "totem" || a.Kind == "generator" }

func (r *Run) beginGeneratorObjective() {
	if r.Level == nil || len(r.Arena().Hazards) == 0 {
		return
	}
	hazards := r.Level.Rooms[r.Room].Hazards
	r.RoomObjective = &RoomObjective{Kind: "disable_generators", Name: "Shut down the generators", Description: "Destroy each generator to permanently disable its linked floor hazard, then defeat the patrol. Generators do not grant monster loot or kill credit.", Target: len(hazards)}
	for i := range hazards {
		id := fmt.Sprintf("l%d-r%d-generator%d", r.Level.ID, r.Room, i+1)
		hazards[i].GeneratorID = id
		hazards[i].Disabled = false
		hp := math.Max(30, r.Build.Damage*3)
		actor := Actor{ID: id, Name: fmt.Sprintf("Hazard generator %d", i+1), Kind: "generator", X: hazards[i].X + hazards[i].W/2, Y: hazards[i].Y - 45, HP: hp, MaxHP: hp, Facing: 1}
		settle(&actor, r.Arena().solidObstacles())
		r.Enemies = append(r.Enemies, actor)
	}
}

func (r *Run) disableGenerator(id string) {
	o := r.RoomObjective
	if o == nil || o.Kind != "disable_generators" || r.Level == nil {
		return
	}
	hazards := r.Level.Rooms[r.Room].Hazards
	disabled := 0
	for i := range hazards {
		h := &hazards[i]
		if h.GeneratorID == id && !h.Disabled {
			h.Disabled = true
			r.event("generator_shutdown", h.X+h.W/2, h.Y+h.H/2, 0)
		}
		if h.Disabled {
			disabled++
		}
	}
	o.Collected = disabled
	o.Complete = disabled == o.Target
}
