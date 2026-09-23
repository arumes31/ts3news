package rift

import (
	"encoding/json"
	"testing"
)

func TestBossPhaseTransitionProvidesSavedRecoveryWindow(t *testing.T) {
	for _, damage := range []float64{550, 800} {
		r := testRun()
		r.Player.X, r.Player.Y = 500, 410
		r.Enemies = []Actor{{ID: "boss", Kind: "boss", ArtKey: "monster:test", X: 550, Y: 410, HP: 1000, MaxHP: 1000, Phase: 1, Windup: .01, AttackName: "Ground Slam"}}
		r.hurtEnemy(0, damage, "hit")
		boss := r.Enemies[0]
		if boss.Windup != 0 || boss.AttackName != "" || boss.Cooldown < 1 || boss.Pose != "stagger" {
			t.Fatal("phase transition did not stop the pending attack and grant recovery")
		}
		data, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var saved Run
		if err := json.Unmarshal(data, &saved); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 9; i++ {
			saved.enemyTick(0, .1)
		}
		if saved.Enemies[0].Attacks != 0 || saved.Enemies[0].Windup != 0 {
			t.Fatal("boss attacked during transition recovery")
		}
		remaining := saved.Enemies[0].Cooldown
		saved.hurtEnemy(0, 1, "hit")
		if saved.Enemies[0].Cooldown != remaining {
			t.Fatal("ordinary damage restarted phase respite")
		}
		saved.enemyTick(0, .11)
		if saved.Enemies[0].Windup <= 0 || saved.Enemies[0].Attacks != 0 {
			t.Fatal("boss did not restart with a full telegraph after respite")
		}
	}
}

func TestBossPhaseChangesDoNotDuplicateLoot(t *testing.T) {
	r := testRun()
	r.Enemies = []Actor{{ID: "boss", Kind: "boss", ArtKey: "monster:test", X: 500, Y: 410, HP: 1000, MaxHP: 1000, Phase: 1}}
	for _, damage := range []float64{550, 250} {
		r.hurtEnemy(0, damage, "hit")
		if len(r.Drops) != 0 || r.Stats.Kills != 0 || r.Stats.Bosses != 0 {
			t.Fatal("phase transition granted premature defeat rewards")
		}
	}
	r.hurtEnemy(0, 200, "hit")
	if len(r.Drops) != 1 || !r.Drops[0].NeedsGear || r.Stats.Kills != 1 || r.Stats.Bosses != 1 {
		t.Fatal("boss defeat did not grant exactly one reward")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	saved.hurtEnemy(0, 1000, "hit")
	saved.skillHit(0, 1000, Skill{Kind: "fire"}, 0, "")
	if len(saved.Drops) != 1 || saved.Stats.Kills != 1 || saved.Stats.Bosses != 1 {
		t.Fatal("post-defeat damage duplicated boss rewards")
	}
}
