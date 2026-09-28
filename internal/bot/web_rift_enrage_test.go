package bot

import (
	"encoding/json"
	"testing"
	"time"
	"ts3news/internal/rift"
)

func TestRiftBossEnrageSelectionResetAndValidation(t *testing.T) {
	enabled := true
	now := time.Unix(100, 0)
	run, err := newRiftPractice(riftRequest{EnrageChallenge: &enabled}, "enrage", rift.Build{HP: 100}, "boss", now)
	if err != nil {
		t.Fatal(err)
	}
	if run.Practice.EnrageSeconds != 30 {
		t.Fatal("challenge not enabled")
	}
	run.Clock = 31
	run.Practice.EnrageTriggered = true
	raw, _ := json.Marshal(run)
	if _, err = decodeRift(string(raw)); err != nil {
		t.Fatal(err)
	}
	if err = resetRiftPractice(run, riftRequest{}, now); err != nil || run.Practice.EnrageSeconds != 30 || run.Clock != 0 || run.Practice.EnrageTriggered {
		t.Fatal("reset lost challenge")
	}
	enabled = false
	if err = resetRiftPractice(run, riftRequest{BossName: run.Practice.BossStart.Name, BossPhase: 2, EnrageChallenge: &enabled}, now); err != nil || run.Practice.EnrageSeconds != 0 {
		t.Fatal("phase reset did not disable challenge")
	}
	for _, seconds := range []float64{-1, 1, 29, 31} {
		run.Practice.EnrageSeconds = seconds
		raw, _ = json.Marshal(run)
		if _, err = decodeRift(string(raw)); err == nil {
			t.Fatal("invalid enrage threshold accepted")
		}
	}
	run.Practice.EnrageSeconds = 30
	run.Practice.EnrageTriggered = true
	run.Clock = 29
	raw, _ = json.Marshal(run)
	if _, err = decodeRift(string(raw)); err == nil {
		t.Fatal("early trigger accepted")
	}
	if _, err = newRiftPractice(riftRequest{EnrageChallenge: &enabled}, "guard", run.Build, "guard", now); err == nil {
		t.Fatal("challenge entered other drill")
	}
}
