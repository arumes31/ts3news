package rift

import "testing"

func TestMythicHazardsKeepFullWarningBeforeDamage(t *testing.T) {
	checked := 0
	for _, level := range Campaign() {
		if level.Difficulty != "Mythic" {
			continue
		}
		for room, arena := range level.Rooms {
			for index, h := range arena.Hazards {
				checked++
				if h.Period <= 1.2+h.Duration {
					t.Fatalf("mission %d tier %d hazard %d has no recovery interval", level.ID, room+1, index)
				}
				for _, phase := range []float64{.00001, .3, .6, .9, 1.19999, 1.20001} {
					r := circleTestRun()
					r.RoomObjective = nil
					r.Level.Region = level.Region
					r.Level.Rooms[0] = Arena{Hazards: []Hazard{h}}
					r.Build.Armor = 0
					r.Player.X = h.X + h.W/2
					r.Player.Y = h.Y + h.H/2
					r.Clock = h.Period*2 - h.Offset + phase
					bounds := h.ContactBounds(r.Clock)
					r.Player.X = bounds.X + bounds.W/2
					before := r.Player.HP
					r.hazardTick()
					if (r.Player.HP < before) != (phase > 1.2) {
						t.Fatalf("mission %d tier %d hazard %d dealt wrong damage at phase %v", level.ID, room+1, index, phase)
					}
					if phase < 1.2 {
						found := false
						warningKind := "hazard_warning"
						if h.Kind == "falling_rock" {
							warningKind = "rock_warning"
						}
						for _, e := range r.Events {
							if e.Kind == warningKind {
								found = true
							}
						}
						if !found {
							t.Fatal("warning phase emitted no warning cue")
						}
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no maximum-difficulty hazards checked")
	}
	t.Logf("verified warning and damage boundaries for %d Mythic hazards", checked)
}

func TestMythicHazardWarningAllowsWalkingEscapeAfterReactionDelay(t *testing.T) {
	checked := 0
	for _, level := range Campaign() {
		if level.Difficulty != "Mythic" {
			continue
		}
		for room, arena := range level.Rooms {
			for index, h := range arena.Hazards {
				for _, fx := range []float64{.25, .5, .75} {
					for _, fy := range []float64{.25, .5, .75} {
						x, y := h.X+h.W*fx, h.Y+h.H*fy
						blocked := false
						for _, o := range arena.solidObstacles() {
							blocked = blocked || contains(o, x, y, 10)
						}
						if blocked {
							continue
						}
						checked++
						escaped := false
						for _, direction := range []Input{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
							r := circleTestRun()
							r.RoomObjective = nil
							r.Level.Region = level.Region
							// Isolate this warning's timing while preserving actual walls and ledges.
							copyArena := arena
							copyArena.Hazards = []Hazard{h}
							r.Level.Rooms[0] = copyArena
							r.Player.X = x
							r.Player.Y = y
							r.Build.Armor = 0
							r.Clock = h.Period*2 - h.Offset + .000001
							before := r.Player.HP
							for frame := 0; frame < 61; frame++ {
								in := Input{}
								if frame >= 15 {
									in = direction
								}
								r.tick(in, .02)
							}
							if r.Player.HP == before && r.Stats.HazardContacts == 0 && !contains(h.Obstacle, r.Player.X, r.Player.Y, 0) && r.Stats.Jumps == 0 {
								escaped = true
								break
							}
						}
						if !escaped {
							t.Fatalf("mission %d tier %d hazard %d cannot walk out from %.1f/%.1f after 300ms reaction", level.ID, room+1, index, x, y)
						}
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no walkable hazard positions checked")
	}
	t.Logf("verified walking escape from %d legal Mythic hazard positions", checked)
}
