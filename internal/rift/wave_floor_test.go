package rift

import (
	"encoding/json"
	"math"
	"testing"
)

func floorWaveRun() *Run {
	r := waveTestRun()
	r.Enemies = nil
	for _, group := range r.RoomObjective.Waves {
		r.Enemies = append(r.Enemies, group...)
	}
	r.RoomObjective = nil
	r.Level.Rooms[0].FragileFloor = []Obstacle{{500, 375, 110, 65}}
	r.beginWaveObjective()
	return r
}

func TestWaveFloorWarnsSavesAndRebuildsBetweenRounds(t *testing.T) {
	r := floorWaveRun()
	f := &r.RoomObjective.FloorSegments[0]
	if f.Collapsed || f.CollapseIn != 3 {
		t.Fatal("missing floor warning")
	}
	r.tickWaveObjective(1)
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	saved.Paused = true
	saved.tickWaveObjective(5)
	if saved.RoomObjective.FloorSegments[0].CollapseIn != 2 {
		t.Fatal("paused floor warning advanced")
	}
	saved.Paused = false
	saved.tickWaveObjective(2)
	if !saved.RoomObjective.FloorSegments[0].Collapsed {
		t.Fatal("vacant floor did not collapse")
	}
	if saved.Arena().groundPath(450, 410, 660, 410, 10) {
		t.Fatal("missing ground remained walkable")
	}
	for i := range saved.Enemies {
		saved.Enemies[i].HP = 0
	}
	saved.tickWaveObjective(.02)
	if saved.RoomObjective.FloorSegments[0].Collapsed || !saved.Arena().groundPath(450, 410, 660, 410, 10) {
		t.Fatal("intermission did not rebuild floor")
	}
	saved.tickWaveObjective(2.5)
	if saved.RoomObjective.FloorSegments[0].CollapseIn != 3 || saved.RoomObjective.FloorSegments[0].Collapsed {
		t.Fatal("new round did not restart warning")
	}
}

func TestWaveFloorWaitsForPlayerAndEnemyFootprints(t *testing.T) {
	for _, enemy := range []bool{false, true} {
		r := floorWaveRun()
		actor := &r.Player
		if enemy {
			actor = &r.Enemies[0]
		}
		actor.X, actor.Y = 505, 410
		r.tickWaveObjective(4)
		if r.RoomObjective.FloorSegments[0].Collapsed {
			t.Fatal("floor collapsed under occupant")
		}
		actor.X = 450
		r.tickWaveObjective(.02)
		if !r.RoomObjective.FloorSegments[0].Collapsed {
			t.Fatal("vacated floor did not collapse")
		}
	}
}

func TestWaveFloorBlocksSweepsButNotProjectiles(t *testing.T) {
	r := floorWaveRun()
	r.tickWaveObjective(3)
	r.Player.X, r.Player.Y = 450, 410
	for _, jump := range []float64{0, .8} {
		r.Player.Jump = jump
		r.moveActor(&r.Player, 250, 0, false)
		if r.Player.X != 450 {
			t.Fatal("large move bypassed missing floor")
		}
	}
	r.Player.Jump = 0
	r.knockbackActor(&r.Player, 250, 0)
	if r.Player.X > 490 {
		t.Fatal("knockback crossed floor gap")
	}
	target := Actor{X: 700, Y: 410}
	if !r.clearProjectilePath(&r.Player, &target) {
		t.Fatal("floor gap became projectile cover")
	}
	r.Player.X, r.Player.Y = 450, 340
	r.moveActor(&r.Player, 250, 0, false)
	if r.Player.X != 700 {
		t.Fatal("upper bypass blocked")
	}
}

func TestWaveFloorPursuersUseBypasses(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		r := floorWaveRun()
		r.tickWaveObjective(3)
		actor := Actor{Kind: "goblin", HP: 100, X: 450, Y: 410}
		target := Actor{X: 700, Y: 410}
		if reverse {
			actor.X, target.X = target.X, actor.X
		}
		for i := 0; i < 160; i++ {
			r.moveActor(&actor, clamp(target.X-actor.X, -5, 5), clamp(target.Y-actor.Y, -5, 5), true)
		}
		if math.Hypot(actor.X-target.X, actor.Y-target.Y) > 10 {
			t.Fatalf("pursuer stuck at %.1f %.1f", actor.X, actor.Y)
		}
	}
}

func TestWaveFloorGeometryDoesNotAliasDefinition(t *testing.T) {
	r := floorWaveRun()
	r.RoomObjective.FloorSegments[0].X = 600
	if r.Level.Rooms[0].FragileFloor[0].X != 500 {
		t.Fatal("live floor aliases level")
	}
	level := cloneCampaignLevel(*r.Level)
	level.Rooms[0].FragileFloor[0].X = 800
	if r.Level.Rooms[0].FragileFloor[0].X != 500 {
		t.Fatal("campaign clone aliases floor")
	}
}

func TestWaveFloorFinalRepairAndEventsAreOncePerRound(t *testing.T) {
	r := floorWaveRun()
	counts := map[string]int{}
	seen := 0
	collect := func() {
		for _, event := range r.Events {
			if event.ID > seen {
				counts[event.Kind]++
				seen = event.ID
			}
		}
	}
	collect()
	for wave := 1; wave <= 3; wave++ {
		r.tickWaveObjective(3)
		collect()
		r.tickWaveObjective(1)
		collect()
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err = json.Unmarshal(raw, &saved); err != nil {
			t.Fatal(err)
		}
		if saved.Arena().groundPath(450, 410, 660, 410, 10) {
			t.Fatal("recovered missing floor lost collision")
		}
		for i := range r.Enemies {
			r.Enemies[i].HP = 0
		}
		r.tickWaveObjective(.02)
		collect()
		if r.RoomObjective.FloorSegments[0].Collapsed {
			t.Fatal("floor remained missing after wave")
		}
		if wave < 3 {
			r.tickWaveObjective(2.5)
			collect()
		}
	}
	if !r.RoomObjective.Complete || !r.Arena().groundPath(450, 410, 660, 410, 10) {
		t.Fatal("final clear did not restore floor")
	}
	for _, kind := range []string{"floor_warning", "floor_collapse", "floor_restore"} {
		if counts[kind] != 3 {
			t.Fatalf("%s fired %d times", kind, counts[kind])
		}
	}
}
