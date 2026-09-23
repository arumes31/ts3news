package bot

import (
	"reflect"
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftBossPracticeSelectionAndReset(t *testing.T) {
	now := time.Unix(100, 0)
	run, err := rift.NewPracticeRun("boss", rift.Build{HP: 100}, "boss", now)
	if err != nil {
		t.Fatal(err)
	}
	name := run.Enemies[0].Name
	req := riftRequest{Kind: "start", RequestID: "boss-practice-request", BossName: name, BossPhase: 3}
	if !validRiftRequest(req) {
		t.Fatal("valid selection rejected")
	}
	run, err = newRiftPractice(req, "boss", run.Build, "boss", now)
	if err != nil || run.Enemies[0].Phase != 3 {
		t.Fatalf("selected start failed: %v", err)
	}
	run.Revision = 8
	run.Epoch = "test"
	run.StartKey = "start-key"
	req.Kind = "practice_reset"
	req.BossPhase = 2
	if err = resetRiftPractice(run, req, now); err != nil {
		t.Fatal(err)
	}
	if run.Enemies[0].Phase != 2 || run.Revision != 8 || run.Epoch != "test" || run.StartKey != "start-key" {
		t.Fatal("reset lost phase or request identity")
	}
	before := *run
	req.BossName = "unknown"
	if err = resetRiftPractice(run, req, now); err == nil || !reflect.DeepEqual(before, *run) {
		t.Fatal("invalid selection mutated practice")
	}
	req.BossPhase = 4
	if validRiftRequest(req) {
		t.Fatal("invalid phase accepted")
	}
	ordinary, _ := rift.NewPracticeRun("guard", run.Build, "guard", now)
	if resetRiftPractice(ordinary, req, now) == nil {
		t.Fatal("boss reset entered ordinary drill")
	}
}

func TestRiftBossPracticeAcceptsWholeLiveRoster(t *testing.T) {
	now := time.Unix(100, 0)
	for _, mob := range riftMobCatalog(now) {
		if rift.AdaptMonster(mob).Kind != "boss" {
			continue
		}
		req := riftRequest{BossName: mob.Name, BossPhase: 3}
		run, err := newRiftPractice(req, "all-bosses", rift.Build{HP: 100}, "boss", now)
		if err != nil {
			t.Fatalf("%s: %v", mob.Name, err)
		}
		req.BossPhase = 2
		if err := resetRiftPractice(run, req, now); err != nil || run.Enemies[0].Phase != 2 {
			t.Fatalf("reset %s: %v", mob.Name, err)
		}
	}
}
