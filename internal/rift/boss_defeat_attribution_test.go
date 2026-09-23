package rift

import (
	"encoding/json"
	"testing"
)

func TestBossFatalHitIdentitySurvivesEncounterSave(t *testing.T) {
	for _, source := range []string{"slam", "projectile", "ordinary-enemy", "environment"} {
		t.Run(source, func(t *testing.T) {
			r := testRun()
			r.Player.X, r.Player.Y = 500, 410
			r.Player.HP = 1
			r.Enemies = []Actor{{ID: "boss-a", Name: "Moss King", Kind: "boss", X: 550, Y: 410, HP: 100, MaxHP: 100, Cooldown: 2}, {ID: "boss-b", Name: "Void Queen", Kind: "boss", X: 560, Y: 410, HP: 100, MaxHP: 100, Cooldown: 2}}
			switch source {
			case "slam":
				r.Enemies[1].Windup = .01
				r.Enemies[1].TargetX = 500
				r.Enemies[1].TargetY = 410
			case "projectile":
				r.Projectiles = []Projectile{{OwnerID: "boss-b", Enemy: true, X: 500, Y: 410, Life: 1, Power: 100}}
			case "ordinary-enemy":
				r.Enemies = append(r.Enemies, Actor{ID: "archer", Kind: "archer", HP: 100})
				r.Projectiles = []Projectile{{OwnerID: "archer", Enemy: true, X: 500, Y: 410, Life: 1, Power: 100}}
			case "environment":
				r.hurtPlayer(100, 500, 410)
			}
			r.tick(Input{}, .02)
			if r.Status != "defeated" {
				t.Fatal("fixture did not defeat player")
			}
			data, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var restored struct {
				LastEncounter struct {
					DefeatedByBoss string `json:"defeated_by_boss"`
				} `json:"last_encounter"`
			}
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Fatal(err)
			}
			want := ""
			if source == "slam" || source == "projectile" {
				want = "Void Queen"
			}
			if restored.LastEncounter.DefeatedByBoss != want {
				t.Fatalf("fatal source=%q, want %q", restored.LastEncounter.DefeatedByBoss, want)
			}
		})
	}
}
