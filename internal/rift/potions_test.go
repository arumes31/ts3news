package rift

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

func TestHealingPotionClampsTracksAndCoolsDown(t *testing.T) {
	r := testRun()
	r.Player.HP = r.Player.MaxHP - 20
	if err := r.UseHealingPotion(50); err != nil {
		t.Fatal(err)
	}
	if r.Player.HP != r.Player.MaxHP || r.Stats.Healing != 20 || r.Stats.PotionsUsed != 1 || r.SkillTimers["healing_potion"] != 8 {
		t.Fatal("incorrect confirmed potion effect")
	}
	r.Player.HP -= 10
	if err := r.UseHealingPotion(50); err == nil {
		t.Fatal("potion ignored cooldown")
	}
	if r.Stats.PotionsUsed != 1 || r.Player.HP != r.Player.MaxHP-10 {
		t.Fatal("rejected potion changed state")
	}
}

func TestHealingPotionRejectsUnavailableAndInvalidUse(t *testing.T) {
	for _, mode := range []string{"full", "dead", "paused", "cleared", "practice", "zero", "negative", "nan", "infinity"} {
		r := testRun()
		r.Player.HP = r.Player.MaxHP - 50
		amount := 20.0
		switch mode {
		case "full":
			r.Player.HP = r.Player.MaxHP
		case "dead":
			r.Player.HP = 0
		case "paused":
			r.Paused = true
		case "cleared":
			r.Status = "cleared"
		case "practice":
			r.Practice = &PracticeState{}
		case "zero":
			amount = 0
		case "negative":
			amount = -1
		case "nan":
			amount = math.NaN()
		case "infinity":
			amount = math.Inf(1)
		}
		before := r.Player
		if err := r.UseHealingPotion(amount); err == nil {
			t.Fatalf("accepted %s", mode)
		}
		if r.Player != before || r.Stats.PotionsUsed != 0 || r.Stats.Healing != 0 || r.SkillTimers["healing_potion"] != 0 {
			t.Fatalf("mutated %s", mode)
		}
	}
}

func TestHealingPotionCooldownSurvivesSaveAndFreezesOnPause(t *testing.T) {
	r := testRun()
	r.Player.HP -= 20
	if err := r.UseHealingPotion(10); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	now := time.UnixMilli(saved.LastMS)
	saved.SetPaused(true, now)
	saved.Step(Input{}, now.Add(10*time.Second))
	if saved.SkillTimers["healing_potion"] != 8 || saved.Stats.PotionsUsed != 1 {
		t.Fatal("saved cooldown advanced while paused")
	}
	saved.SetPaused(false, now.Add(10*time.Second))
	saved.Step(Input{}, now.Add(10100*time.Millisecond))
	if saved.SkillTimers["healing_potion"] >= 8 || saved.SkillTimers["healing_potion"] < 7.8 {
		t.Fatal("cooldown did not follow resumed combat time")
	}
}
