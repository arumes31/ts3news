package rift

import (
	"encoding/json"
	"testing"
)

func TestDefeatRecapPreservesEveryBossHealthAndPhase(t *testing.T) {
	r := testRun()
	r.Player.X, r.Player.Y = 500, 410
	r.Player.HP = 1
	r.Enemies = []Actor{{ID: "boss-a", Name: "Moss King", Kind: "boss", X: 550, Y: 410, HP: 400, MaxHP: 1000, Phase: 2, Cooldown: 2}, {ID: "boss-b", Name: "Void Queen", Kind: "boss", X: 560, Y: 410, HP: 100, MaxHP: 1000, Phase: 3, Windup: .01, TargetX: 500, TargetY: 410}}
	r.EncounterPlan[r.Room] = append([]Actor{}, r.Enemies...)
	r.tick(Input{}, .02)
	if r.Status != "defeated" {
		t.Fatal("fixture did not defeat player")
	}
	data, err := json.Marshal(r.LastEncounter)
	if err != nil {
		t.Fatal(err)
	}
	var recap struct {
		Bosses []struct {
			Name  string  `json:"name"`
			HP    float64 `json:"hp"`
			MaxHP float64 `json:"max_hp"`
			Phase int     `json:"phase"`
		} `json:"bosses"`
	}
	if err := json.Unmarshal(data, &recap); err != nil {
		t.Fatal(err)
	}
	if len(recap.Bosses) != 2 {
		t.Fatal("defeat recap omitted boss combat state")
	}
	if recap.Bosses[0].Name != "Moss King" || recap.Bosses[0].HP != 400 || recap.Bosses[0].MaxHP != 1000 || recap.Bosses[0].Phase != 2 || recap.Bosses[1].Name != "Void Queen" || recap.Bosses[1].HP != 100 || recap.Bosses[1].Phase != 3 {
		t.Fatal("defeat recap recorded incorrect boss state")
	}
	r.Enemies[0].HP = 0
	again, err := json.Marshal(r.LastEncounter)
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != string(data) {
		t.Fatal("recap changed with later actor mutation")
	}
}
