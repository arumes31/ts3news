package rift

import (
	"encoding/json"
	"testing"
)

func missReactionRun() *Run {
	r := testRun()
	r.Player.X = 300
	r.Player.Y = 410
	r.Player.Facing = 1
	r.Enemies = []Actor{{ID: "near", Kind: "goblin", X: 180, Y: 410, HP: 100, MaxHP: 100, Speed: 80}, {ID: "far", Kind: "knight", X: 140, Y: 410, HP: 100, MaxHP: 100, Speed: 80}}
	return r
}

func TestMissReactionSelectsOneEnemyWithoutInstantDamage(t *testing.T) {
	r := missReactionRun()
	hp := r.Player.HP
	r.tick(Input{Attack: true}, .02)
	if r.Stats.BasicMisses != 1 || r.Enemies[0].ReactMissTimer <= 0 || r.Enemies[1].ReactMissTimer != 0 {
		t.Fatal("miss did not select nearest fighter exactly once")
	}
	if r.Player.HP != hp || r.Enemies[0].Windup != 0 || r.Enemies[0].X <= 180 {
		t.Fatal("reaction must approach without attacking")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 20; n++ {
		saved.enemyTick(0, .02)
	}
	if saved.Enemies[0].ReactMissTimer != 0 {
		t.Fatal("reaction did not expire after reload")
	}
	if saved.Player.HP != hp {
		t.Fatal("reaction bypassed normal attack warning")
	}
}

func TestMissReactionIgnoresHitsCoverAndRejectedAttacks(t *testing.T) {
	for _, mode := range []string{"hit", "cover", "cooldown", "guard"} {
		r := missReactionRun()
		in := Input{Attack: true}
		switch mode {
		case "hit":
			r.Enemies = append(r.Enemies, Actor{ID: "target", Kind: "goblin", X: 340, Y: 410, HP: 100, MaxHP: 100})
		case "cover":
			r.Level = &Level{Rooms: []Arena{{Cover: []TerrainCover{{Obstacle: Obstacle{330, 390, 20, 40}, ID: "wood", Material: "wood", HP: 100, MaxHP: 100}}}}}
		case "cooldown":
			r.Player.Cooldown = 1
		case "guard":
			in.Guard = true
		}
		r.tick(in, .02)
		for _, e := range r.Enemies {
			if e.ReactMissTimer > 0 {
				t.Fatalf("reaction for %s", mode)
			}
		}
	}
}

func TestMissReactionExcludesBusyHiddenAndSpecializedEnemies(t *testing.T) {
	for _, mode := range []string{"dead", "patrol", "wall", "windup", "recovery", "stagger", "archer", "boss", "treasure", "charge"} {
		r := missReactionRun()
		r.Enemies = r.Enemies[:1]
		e := &r.Enemies[0]
		switch mode {
		case "dead":
			e.HP = 0
		case "patrol":
			e.Patrol = true
		case "wall":
			r.Level = &Level{Rooms: []Arena{{HighCover: []Obstacle{{230, 350, 20, 100}}}}}
		case "windup":
			e.Windup = 1
		case "recovery":
			e.Cooldown = 1
		case "stagger":
			e.Pose = "stagger"
			e.PoseTime = 1
		case "charge":
			e.ChargeActive = true
		default:
			e.Kind = mode
		}
		r.tick(Input{Attack: true}, .02)
		if e.ReactMissTimer > 0 {
			t.Fatalf("ineligible enemy reacted: %s", mode)
		}
	}
}

func TestHitCancelsMissReaction(t *testing.T) {
	r := missReactionRun()
	r.tick(Input{Attack: true}, .02)
	r.hurtEnemy(0, 10, "hit")
	if r.Enemies[0].ReactMissTimer != 0 {
		t.Fatal("hit failed to cancel reaction")
	}
}

func TestControlEffectsAndBlockedRoutesCancelMissReaction(t *testing.T) {
	for _, mode := range []string{"knockdown", "stagger", "wall"} {
		r := missReactionRun()
		r.tick(Input{Attack: true}, .02)
		if r.Enemies[0].ReactMissTimer <= 0 {
			t.Fatal("missing initial reaction")
		}
		switch mode {
		case "knockdown":
			r.Enemies[0].Knockdown = .5
		case "stagger":
			r.Enemies[0].Pose = "stagger"
			r.Enemies[0].PoseTime = .5
		case "wall":
			r.Level = &Level{Rooms: []Arena{{HighCover: []Obstacle{{230, 350, 20, 100}}}}}
		}
		r.enemyTick(0, .02)
		if r.Enemies[0].ReactMissTimer != 0 {
			t.Fatalf("reaction survived %s", mode)
		}
	}
}
