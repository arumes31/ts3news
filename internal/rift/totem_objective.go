package rift

import (
	"fmt"
	"math"
)

func (r *Run) beginTotemObjective() {
	r.RoomObjective = &RoomObjective{Kind: "destroy_totems", Name: "Shatter the totems", Description: "Destroy all three ritual totems and defeat the patrol. Totems do not grant monster loot or kill credit.", Target: 3}
	mission := 1
	if r.Level != nil {
		mission = r.Level.ID
	}
	for i, point := range [][2]float64{{430, 330}, {820, 480}, {1240, 330}} {
		hp := math.Max(30, r.Build.Damage*3)
		actor := Actor{ID: fmt.Sprintf("l%d-r%d-totem%d", mission, r.Room, i+1), Name: fmt.Sprintf("Ritual totem %d", i+1), Kind: "totem", X: point[0], Y: point[1], HP: hp, MaxHP: hp, Facing: 1}
		settle(&actor, r.Arena().solidObstacles())
		r.Enemies = append(r.Enemies, actor)
	}
}

func (r *Run) updateTotemObjective() {
	o := r.RoomObjective
	if o == nil || o.Kind != "destroy_totems" {
		return
	}
	destroyed := 0
	for _, enemy := range r.Enemies {
		if enemy.Kind == "totem" && enemy.HP <= 0 {
			destroyed++
		}
	}
	o.Collected = destroyed
	o.Complete = destroyed == o.Target
}
