package rift

import "math"

func (r *Run) beginGuardianObjective() {
	o := &RoomObjective{Kind: "linked_guardians", Name: "Defeat the linked guardians", Description: "The linked guardians take 50% less damage while within 240 units of one another. Separate them or defeat one to break the bond, then clear the patrol.", Target: 2}
	for _, e := range r.Enemies {
		if e.HP <= 0 || e.Kind == "treasure" || e.Kind == "boss" || e.isObjectiveProp() {
			continue
		}
		o.Targets = append(o.Targets, e.ID)
		if len(o.Targets) == 2 {
			break
		}
	}
	if len(o.Targets) != 2 {
		return
	}
	r.RoomObjective = o
	o.BondActive = r.guardianBondActive(o.Targets[0])
}

// Evaluate live positions and health so consecutive hits cannot use a stale bond.
func (r *Run) guardianBondActive(id string) bool {
	o := r.RoomObjective
	if o == nil || o.Kind != "linked_guardians" || o.Complete || len(o.Targets) != 2 || (id != o.Targets[0] && id != o.Targets[1]) {
		return false
	}
	var pair [2]*Actor
	for i := range r.Enemies {
		for n, target := range o.Targets {
			if r.Enemies[i].ID == target {
				pair[n] = &r.Enemies[i]
			}
		}
	}
	return pair[0] != nil && pair[1] != nil && pair[0].HP > 0 && pair[1].HP > 0 && math.Hypot(pair[0].X-pair[1].X, pair[0].Y-pair[1].Y) <= 240
}

func (r *Run) updateGuardianObjective() {
	o := r.RoomObjective
	if o == nil || o.Kind != "linked_guardians" || o.Complete || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	active := r.guardianBondActive(o.Targets[0])
	if active != o.BondActive {
		cue := "guardian_unlinked"
		if active {
			cue = "guardian_linked"
		}
		r.event(cue, r.Player.X, r.Player.Y, 0)
	}
	o.BondActive = active
	defeated := 0
	for _, id := range o.Targets {
		for _, e := range r.Enemies {
			if e.ID == id && e.HP <= 0 && e.Pose != "escape" {
				defeated++
				break
			}
		}
	}
	o.Collected = defeated
	if defeated == o.Target {
		o.Complete = true
		r.event("guardians_defeated", r.Player.X, r.Player.Y, 0)
	}
}
