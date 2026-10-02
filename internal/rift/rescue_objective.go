package rift

import (
	"fmt"
	"math"
)

type RescueCaptive struct {
	CageID  string  `json:"cage_id"`
	Name    string  `json:"name"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Freed   bool    `json:"freed"`
	FreedAt float64 `json:"freed_at"`
}

func (r *Run) beginRescueObjective() {
	o := &RoomObjective{Kind: "rescue_companions", Name: "Rescue the companions", Description: "Break both cages with attacks or spells to free the captive spirits, then defeat the patrol. Cages grant no monster kills or loot.", Target: 2}
	mission := 1
	if r.Level != nil {
		mission = r.Level.ID
	}
	for i, point := range [][2]float64{{480, 330}, {1140, 480}} {
		id := fmt.Sprintf("l%d-r%d-cage%d", mission, r.Room, i+1)
		hp := math.Max(30, r.Build.Damage*3)
		cage := Actor{ID: id, Name: fmt.Sprintf("Captive cage %d", i+1), Kind: "cage", X: point[0], Y: point[1], HP: hp, MaxHP: hp, Facing: 1}
		settle(&cage, r.Arena().solidObstacles())
		r.Enemies = append(r.Enemies, cage)
		o.Captives = append(o.Captives, RescueCaptive{CageID: id, Name: []string{"Lost scout", "Lost pathfinder"}[i], X: cage.X, Y: cage.Y})
	}
	r.RoomObjective = o
}

func (r *Run) updateRescueObjective() {
	o := r.RoomObjective
	if o == nil || o.Kind != "rescue_companions" || o.Complete {
		return
	}
	freed := 0
	for i := range o.Captives {
		c := &o.Captives[i]
		for _, e := range r.Enemies {
			if e.ID == c.CageID && e.HP <= 0 && !c.Freed {
				c.Freed = true
				c.FreedAt = r.Clock
				r.event("companion_freed", c.X, c.Y, 0)
			}
		}
		if c.Freed {
			freed++
		}
	}
	o.Collected = freed
	if freed == o.Target {
		o.Complete = true
		r.event("rescue_complete", r.Player.X, r.Player.Y, 0)
	}
}
