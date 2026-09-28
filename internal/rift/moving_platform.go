package rift

import "math"

// The ferry is confined to an isolated drill. Its current deck geometry and
// ride progress are saved; combat time alone determines its next position.
func (r *Run) configureMovingPlatformPractice() {
	r.Practice.Arena.Name = "Moving platform challenge"
	r.Practice.GoalX = 1250
	r.Practice.Arena.Platforms = []RaisedPlatform{{Obstacle: Obstacle{300, 365, 180, 90}, ID: "practice-ferry", Rise: 16, Ramp: 24, Floor: "wood"}}
}

func (r *Run) tickMovingPlatformPractice() {
	if r.Practice == nil || r.Practice.Mode != "moving_platform" || r.Paused || r.Status != "fighting" || len(r.Practice.Arena.Platforms) != 1 {
		return
	}
	deck := &r.Practice.Arena.Platforms[0]
	previous := *deck
	// Eight seconds each way, with no teleport at either end of the route.
	phase := math.Mod(r.Clock, 16)
	if phase > 8 {
		phase = 16 - phase
	}
	deck.X = 300 + 75*phase
	dx := deck.X - previous.X
	carry := func(actor *Actor) float64 {
		before := actor.X
		radius := actorClearance(actor)
		aboard := actor.HP > 0 && actor.Jump <= .1 && actor.X >= previous.X+radius && actor.X <= previous.X+previous.W-radius && actor.Y >= previous.Y+radius && actor.Y <= previous.Y+previous.H-radius
		if aboard {
			steps := max(1, int(math.Ceil(math.Abs(dx)/8)))
			for range steps {
				r.moveActor(actor, dx/float64(steps), 0, false)
			}
		}
		actor.Elevation = r.Arena().Elevation(actor.X, actor.Y)
		return math.Abs(actor.X - before)
	}
	r.Practice.PlatformRide = math.Min(250, r.Practice.PlatformRide+carry(&r.Player))
	for i := range r.Enemies {
		carry(&r.Enemies[i])
	}
	r.Floor = r.FloorMaterial()
}
