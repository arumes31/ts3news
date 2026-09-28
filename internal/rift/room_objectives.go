package rift

import "math"

type ObjectivePickup struct {
	ID        int     `json:"id"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Collected bool    `json:"collected"`
}

type ObjectiveZone struct {
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	RadiusX float64 `json:"radius_x"`
	RadiusY float64 `json:"radius_y"`
}

func (z ObjectiveZone) contains(actor Actor) bool {
	dx, dy := (actor.X-z.X)/z.RadiusX, (actor.Y-z.Y)/z.RadiusY
	return dx*dx+dy*dy <= 1
}

type RoomObjective struct {
	Gate *WaveGate `json:"gate,omitempty"`
	Lanes               []DefenseLane     `json:"lanes,omitempty"`
	Sequence            []int             `json:"sequence,omitempty"`
	PuzzleTouch         int               `json:"puzzle_touch"`
	Lantern             *Actor            `json:"lantern,omitempty"`
	Captives            []RescueCaptive   `json:"captives,omitempty"`
	BondActive          bool              `json:"bond_active"`
	CollapseX           float64           `json:"collapse_x"`
	CollapseHitCooldown float64           `json:"collapse_hit_cooldown"`
	Channels            []RitualChannel   `json:"channels,omitempty"`
	Escort              *Actor            `json:"escort,omitempty"`
	EscortMoving        bool              `json:"escort_moving"`
	BeaconTime          float64           `json:"beacon_time"`
	Targets             []string          `json:"targets,omitempty"`
	Relic               *ObjectivePickup  `json:"relic,omitempty"`
	Carrying            bool              `json:"carrying"`
	Waves               [][]Actor         `json:"waves,omitempty"`
	Wave                int               `json:"wave"`
	NextWaveSeconds     float64           `json:"next_wave_seconds"`
	Zone                *ObjectiveZone    `json:"zone,omitempty"`
	Seconds             float64           `json:"seconds"`
	Contested           bool              `json:"contested"`
	Charging            bool              `json:"charging"`
	Kind                string            `json:"kind"`
	Name                string            `json:"name"`
	Description         string            `json:"description"`
	Target              int               `json:"target"`
	Collected           int               `json:"collected"`
	Complete            bool              `json:"complete"`
	Pickups             []ObjectivePickup `json:"pickups"`
}

func (r *Run) beginRoomObjective() {
	r.RoomObjective = nil
	if r.Practice != nil {
		return
	}
	if r.Arena().Objective == "split_defense" {
		r.beginDefenseObjective()
		return
	}
	if r.Arena().Objective == "rune_gate" {
		r.beginRuneGateObjective()
		return
	}
	if r.Arena().Objective == "protect_lantern" {
		r.beginLanternObjective()
		return
	}
	if r.Arena().Objective == "rescue_companions" {
		r.beginRescueObjective()
		return
	}
	if r.Arena().Objective == "linked_guardians" {
		r.beginGuardianObjective()
		return
	}
	if r.Arena().Objective == "escape_collapse" {
		r.beginCollapseObjective()
		return
	}
	if r.Arena().Objective == "interrupt_ritual" {
		r.beginRitualObjective()
		return
	}
	if r.Arena().Objective == "escort_spirit" {
		r.beginEscortObjective()
		return
	}
	if r.Arena().Objective == "moving_beacons" {
		r.beginBeaconObjective()
		return
	}
	if r.Arena().Objective == "marked_hunt" {
		r.beginHuntObjective()
		return
	}
	if r.Arena().Objective == "disable_generators" {
		r.beginGeneratorObjective()
		return
	}
	if r.Arena().Objective == "carry_relic" {
		r.beginRelicObjective()
		return
	}
	if r.Arena().Objective == "destroy_totems" {
		r.beginTotemObjective()
		return
	}
	if r.Arena().Objective == "survive_waves" {
		r.beginWaveObjective()
		return
	}
	if r.Arena().Objective == "hold_circle" {
		r.RoomObjective = &RoomObjective{Kind: "hold_circle", Name: "Hold the circle", Description: "Charge the circle for 15 uncontested seconds, then defeat the patrol. Progress is kept when you leave.", Target: 15, Zone: &ObjectiveZone{X: 480, Y: 410, RadiusX: 80, RadiusY: 44}}
		return
	}
	if r.Arena().Objective != "sigils" {
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

func (r *Run) tickRoomObjective(dt float64) {
	r.collectRoomSigils()
	r.tickRuneGateObjective()
	r.updateRitualObjective()
	r.updateGuardianObjective()
	r.updateRescueObjective()
	r.tickEscortObjective(dt)
	r.tickBeaconObjective(dt)
	r.tickHuntObjective()
	r.tickRelicObjective()
	r.updateTotemObjective()
	r.tickWaveObjective(dt)
	o := r.RoomObjective
	if o == nil || o.Kind != "hold_circle" || o.Complete || o.Zone == nil || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	wasContested, wasCharging := o.Contested, o.Charging
	o.Contested = false
	for _, enemy := range r.Enemies {
		o.Contested = o.Contested || (enemy.HP > 0 && o.Zone.contains(enemy))
	}
	o.Charging = !o.Contested && r.Player.Jump <= .1 && o.Zone.contains(r.Player)
	if o.Contested && !wasContested {
		r.event("circle_contested", o.Zone.X, o.Zone.Y, 0)
	}
	if o.Charging && !wasCharging {
		r.event("circle_charge", o.Zone.X, o.Zone.Y, 0)
	}
	if o.Charging {
		o.Seconds = math.Min(float64(o.Target), o.Seconds+dt)
	}
	if o.Seconds >= float64(o.Target)-1e-9 {
		o.Seconds = float64(o.Target)
		o.Complete = true
		o.Charging = false
		r.event("circle_complete", o.Zone.X, o.Zone.Y, 0)
	}
}
