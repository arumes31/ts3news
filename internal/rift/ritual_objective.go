package rift

import "math"

type RitualChannel struct {
	EnemyID string  `json:"enemy_id"`
	Seconds float64 `json:"seconds"`
}

func (r *Run) beginRitualObjective() {
	o := &RoomObjective{Kind: "interrupt_ritual", Name: "Interrupt the ritual", Description: "Hit channelers to reset their eight-second ritual. Their pulses damage nearby fighters. Defeat all channelers and the patrol."}
	for _, e := range r.Enemies {
		if e.HP <= 0 || e.Kind == "treasure" || e.Kind == "boss" || e.isObjectiveProp() {
			continue
		}
		o.Channels = append(o.Channels, RitualChannel{EnemyID: e.ID})
		if len(o.Channels) == 3 {
			break
		}
	}
	if len(o.Channels) == 0 {
		return
	}
	o.Target = len(o.Channels)
	r.RoomObjective = o
}

func (r *Run) interruptRitual(id string) {
	o := r.RoomObjective
	if o == nil || o.Kind != "interrupt_ritual" || o.Complete {
		return
	}
	for i := range o.Channels {
		c := &o.Channels[i]
		if c.EnemyID == id && c.Seconds > 0 {
			c.Seconds = 0
			for _, e := range r.Enemies {
				if e.ID == id {
					r.event("ritual_interrupt", e.X, e.Y, 0)
					break
				}
			}
			return
		}
	}
}

func (r *Run) tickRitualEnemy(i int, dt float64) bool {
	o := r.RoomObjective
	if o == nil || o.Kind != "interrupt_ritual" {
		return false
	}
	e := &r.Enemies[i]
	for n := range o.Channels {
		c := &o.Channels[n]
		if c.EnemyID != e.ID {
			continue
		}
		if e.HP <= 0 || o.Complete || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 {
			return true
		}
		e.PoseTime = math.Max(0, e.PoseTime-dt)
		e.Knockdown = math.Max(0, e.Knockdown-dt)
		e.RecoilX = 0
		if e.PoseTime > 0 || e.Knockdown > 0 {
			return true
		}
		e.Pose = "cast"
		old := c.Seconds
		c.Seconds += dt
		if old < 6 && c.Seconds >= 6 {
			r.event("ritual_warning", e.X, e.Y, 0)
		}
		if c.Seconds >= 8 {
			c.Seconds = 0
			r.event("ritual_pulse", e.X, e.Y, 0)
			if math.Hypot((r.Player.X-e.X)/150, (r.Player.Y-e.Y)/85) <= 1 {
				r.hurtPlayerFromEnemy(math.Max(12, e.Damage), e.X, e.Y, e.ID)
			}
		}
		return true
	}
	return false
}

func (r *Run) updateRitualObjective() {
	o := r.RoomObjective
	if o == nil || o.Kind != "interrupt_ritual" || o.Complete {
		return
	}
	defeated := 0
	for _, c := range o.Channels {
		for _, e := range r.Enemies {
			if e.ID == c.EnemyID && e.HP <= 0 {
				defeated++
				break
			}
		}
	}
	o.Collected = defeated
	if defeated == o.Target {
		o.Complete = true
		r.event("ritual_complete", r.Player.X, r.Player.Y, 0)
	}
}
