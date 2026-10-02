package rift

// Tracking ends before the fixed warning, leaving time to leave the strike box.
// Coordinates live in the saved arena, so resuming never retargets a locked bolt.
func (r *Run) trackLightningMarkers() {
	if r.Paused {
		return
	}
	hazards := r.Arena().Hazards
	for i := range hazards {
		h := &hazards[i]
		if h.Kind != "tracking_lightning" || h.Disabled || h.Phase(r.Clock) >= .5 {
			continue
		}
		h.X = clamp(r.Player.X-h.W/2, 35, 1565-h.W)
		h.Y = clamp(r.Player.Y-h.H/2, 315, 490-h.H)
	}
}
