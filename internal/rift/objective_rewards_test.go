package rift

import (
	"encoding/json"
	"testing"
)

func rewardTestRun() *Run {
	r := testRun()
	level := Campaign()[0]
	r.Level = &level
	r.Room = len(Rooms) - 1
	r.Status = "cleared"
	r.Objectives = &MissionObjectives{Mission: r.Level.ID, Finished: true, RewardPerObjective: 5, Entries: []ObjectiveProgress{
		{ID: "timed", Status: "complete"}, {ID: "guard", Status: "complete"}, {ID: "no_damage", Status: "failed"}, {ID: "aerial_finish", Status: "active"},
	}}
	return r
}

func TestOptionalRewardSeparateFromFightLoot(t *testing.T) {
	r := rewardTestRun()
	r.Gold = 30
	r.Drops = []Drop{{ID: "fight", Gold: 30, Collected: true}}
	if got := r.PendingObjectiveGold(); got != 10 {
		t.Fatalf("bonus=%d", got)
	}
	if r.Gold != 30 || len(r.Drops) != 1 || r.Drops[0].Gold != 30 || r.BankedGold != 0 {
		t.Fatal("bonus changed fight loot")
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var restored Run
	if err = json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if restored.PendingObjectiveGold() != 10 {
		t.Fatal("saved reward offer changed")
	}
}

func TestOptionalRewardEligibility(t *testing.T) {
	for _, scenario := range []string{"early", "fighting", "defeated", "practice", "unfinished", "banked", "paid", "wrong mission", "legacy", "invalid offer"} {
		t.Run(scenario, func(t *testing.T) {
			r := rewardTestRun()
			switch scenario {
			case "early":
				r.Room = 0
			case "fighting":
				r.Status = "fighting"
			case "defeated":
				r.Status = "defeated"
			case "practice":
				r.Practice = &PracticeState{}
			case "unfinished":
				r.Objectives.Finished = false
			case "banked":
				r.Objectives.Banked = true
			case "paid":
				r.Objectives.RewardGold = 10
			case "wrong mission":
				r.Objectives.Mission++
			case "legacy":
				r.Objectives.RewardPerObjective = 0
			case "invalid offer":
				r.Objectives.RewardPerObjective = 10000
			}
			if r.PendingObjectiveGold() != 0 {
				t.Fatal("ineligible bonus")
			}
		})
	}
}

func TestOptionalRewardIgnoresDuplicateAndUnknownGoals(t *testing.T) {
	r := rewardTestRun()
	r.Objectives.Entries = append(r.Objectives.Entries, ObjectiveProgress{ID: "timed", Status: "complete"}, ObjectiveProgress{ID: "invented", Status: "complete"})
	if r.PendingObjectiveGold() != 10 {
		t.Fatal("duplicate or unknown reward")
	}
}
