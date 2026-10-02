package rift

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPickupPracticeUsesRealRadiusWithoutRewards(t *testing.T) {
	r, err := NewPracticeRun("pickup", testRun().Build, "pickup", time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Drops) != 3 || len(r.Enemies) != 0 {
		t.Fatal("unsafe or missing demonstration")
	}
	for i := range r.Drops {
		d := r.Drops[i]
		if d.Gold != 0 || d.Gear != nil || d.NeedsGear {
			t.Fatal("demo contains economic reward")
		}
		r.Player.X = d.X + r.Practice.PickupRadius
		r.Player.Y = d.Y
		r.tick(Input{}, .02)
		if r.Drops[i].Collected {
			t.Fatal("boundary collected too early")
		}
		r.Player.X = d.X + r.Practice.PickupRadius - .01
		r.tick(Input{}, .02)
		if !r.Drops[i].Collected {
			t.Fatalf("token %d not collected: player=(%v,%v) drop=(%v,%v) radius=%v status=%s",i,r.Player.X,r.Player.Y,d.X,d.Y,r.Practice.PickupRadius,r.Status)
		}
		if i < 2 && r.Practice.Completed {
			t.Fatal("completed before all pickups")
		}
	}
	if !r.Practice.Completed || r.Status != "complete" || r.Gold != 0 || r.BankedGold != 0 || len(r.History) != 0 {
		t.Fatal("invalid completion or reward")
	}
	raw, _ := json.Marshal(r)
	var saved Run
	if err = json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if !saved.Practice.Completed || !saved.Drops[2].Collected {
		t.Fatal("progress lost on save")
	}
	if err = saved.ResetPractice(time.Unix(200, 0)); err != nil {
		t.Fatal(err)
	}
	if saved.Practice.Completed || len(saved.Drops) != 3 || saved.Drops[0].Collected {
		t.Fatal("reset failed")
	}
	saved.Drops = nil
	saved.Player.X = 1000
	saved.practiceTick()
	if saved.Practice.Completed {
		t.Fatal("missing tokens completed drill")
	}
}
