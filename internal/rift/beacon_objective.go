package rift

import "math"

var beaconCenters = [3][2]float64{{350, 330}, {800, 480}, {1250, 330}}

func (r *Run) beginBeaconObjective() {
	r.RoomObjective = &RoomObjective{Kind: "moving_beacons", Name: "Capture the moving beacons", Description: "Stay grounded inside each moving ring for three seconds. Capture all three beacons and defeat the patrol. Leaving keeps capture progress.", Target: 3}
	r.positionBeacon()
}
func (r *Run) positionBeacon() {
	o := r.RoomObjective
	center := beaconCenters[min(o.Collected, len(beaconCenters)-1)]
	o.Zone = &ObjectiveZone{X: center[0] + 100*math.Sin(o.BeaconTime*.4), Y: center[1], RadiusX: 65, RadiusY: 35}
}
func (r *Run) tickBeaconObjective(dt float64) {
	o := r.RoomObjective
	if o == nil || o.Kind != "moving_beacons" || o.Complete || r.Practice != nil || r.Paused || r.Status != "fighting" || r.Player.HP <= 0 {
		return
	}
	o.BeaconTime += dt
	r.positionBeacon()
	wasCharging := o.Charging
	o.Charging = r.Player.Jump <= .1 && o.Zone.contains(r.Player)
	if o.Charging && !wasCharging {
		r.event("beacon_charge", o.Zone.X, o.Zone.Y, 0)
	}
	if !o.Charging {
		return
	}
	o.Seconds = math.Min(3, o.Seconds+dt)
	if o.Seconds < 3-1e-9 {
		return
	}
	o.Seconds = 3
	o.Collected++
	o.Charging = false
	r.event("beacon_captured", o.Zone.X, o.Zone.Y, float64(o.Collected))
	if o.Collected == o.Target {
		o.Complete = true
		r.event("beacons_complete", o.Zone.X, o.Zone.Y, 0)
		return
	}
	o.Seconds = 0
	o.BeaconTime = 0
	r.positionBeacon()
}
