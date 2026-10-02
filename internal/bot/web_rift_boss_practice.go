package bot

import (
	"errors"
	"time"
	"ts3news/internal/rift"
)

func newRiftPractice(req riftRequest, id string, build rift.Build, mode string, now time.Time) (*rift.Run, error) {
	if !rift.ValidHazardIntensity(req.HazardIntensity) || req.HazardIntensity != "" && mode != "hazard" {
		return nil, errors.New("invalid hazard practice intensity")
	}
	if req.EnrageChallenge != nil && mode != "boss" {
		return nil, errors.New("enrage challenge requires boss practice")
	}
	var run *rift.Run
	var err error
	if mode == "boss" && (req.BossName != "" || req.BossPhase != 0) {
		run, err = rift.NewBossPracticeRunWithCatalog(id, build, req.BossName, req.BossPhase, now, riftMobCatalog(now))
	} else {
		run, err = rift.NewPracticeRun(id, build, mode, now)
	}
	if err == nil && mode == "boss" && req.SlowTelegraphs != nil {
		run.Practice.SlowTelegraphs = *req.SlowTelegraphs
	}
	if err == nil && mode == "boss" {
		configureRiftEnrage(run, req.EnrageChallenge)
	}
	if err == nil && mode == "hazard" {
		err = run.ConfigureHazardPractice(req.HazardIntensity)
	}
	return run, err
}

func resetRiftPractice(run *rift.Run, req riftRequest, now time.Time) error {
	if req.EnrageChallenge != nil && (run.Practice == nil || run.Practice.Mode != "boss") {
		return errors.New("enrage challenge requires boss practice")
	}
	if !rift.ValidHazardIntensity(req.HazardIntensity) || req.HazardIntensity != "" && (run.Practice == nil || run.Practice.Mode != "hazard") {
		return errors.New("invalid hazard practice intensity")
	}
	if req.BossName == "" && req.BossPhase == 0 {
		if err := run.ResetPractice(now); err != nil {
			return err
		}
		if run.Practice.Mode == "boss" && req.SlowTelegraphs != nil {
			run.Practice.SlowTelegraphs = *req.SlowTelegraphs
		}
		configureRiftEnrage(run, req.EnrageChallenge)
		if req.HazardIntensity != "" {
			return run.ConfigureHazardPractice(req.HazardIntensity)
		}
		return nil
	}
	if run.Practice == nil || run.Practice.Mode != "boss" {
		return errors.New("boss selection requires boss practice")
	}
	fresh, err := rift.NewBossPracticeRunWithCatalog(run.ID, run.Build, req.BossName, req.BossPhase, now, riftMobCatalog(now))
	if err != nil {
		return err
	}
	fresh.Practice.FreezeMovement = run.Practice.FreezeMovement
	fresh.Practice.FreezeUsed = fresh.Practice.FreezeMovement
	fresh.Practice.SlowTelegraphs = run.Practice.SlowTelegraphs
	fresh.Practice.EnrageSeconds = run.Practice.EnrageSeconds
	configureRiftEnrage(fresh, req.EnrageChallenge)
	if req.SlowTelegraphs != nil {
		fresh.Practice.SlowTelegraphs = *req.SlowTelegraphs
	}
	fresh.Revision, fresh.StartKey, fresh.Epoch = run.Revision, run.StartKey, run.Epoch
	*run = *fresh
	return nil
}

func configureRiftEnrage(run *rift.Run, enabled *bool) {
	if enabled == nil {
		return
	}
	run.Practice.EnrageSeconds = 0
	if *enabled {
		run.Practice.EnrageSeconds = rift.BossPracticeEnrageSeconds
	}
}
