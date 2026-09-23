package rift

func (r *Run) beginHuntObjective() {
	candidates := []string{}
	for _, enemy := range r.Enemies {
		// Treasure goblins can leave the arena, so they cannot be mandatory hunt targets.
		if enemy.HP > 0 && enemy.Kind != "treasure" && !enemy.isObjectiveProp() {
			candidates = append(candidates, enemy.ID)
		}
	}
	if len(candidates) == 0 {
		return
	}
	count := min(3, len(candidates))
	o := &RoomObjective{Kind: "marked_hunt", Name: "Hunt the marked targets", Description: "Defeat the marked enemies to secure this tier. Surviving unmarked enemies retreat without granting kills or loot.", Target: count}
	for i := 0; i < count; i++ {
		o.Targets = append(o.Targets, candidates[i*len(candidates)/count])
	}
	r.RoomObjective = o
}

func (r *Run) tickHuntObjective() {
	o := r.RoomObjective
	if o == nil || o.Kind != "marked_hunt" || o.Complete || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	defeated := 0
	for _, id := range o.Targets {
		for _, enemy := range r.Enemies {
			if enemy.ID == id && enemy.HP <= 0 && enemy.Pose != "escape" {
				defeated++
				break
			}
		}
	}
	o.Collected = defeated
	if defeated != o.Target {
		return
	}
	o.Complete = true
	for i := range r.Enemies {
		if r.Enemies[i].HP > 0 {
			r.escapeEnemy(i)
		}
	}
	r.event("hunt_complete", r.Player.X, r.Player.Y, 0)
}
