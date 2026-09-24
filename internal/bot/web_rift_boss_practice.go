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
	if err == nil && mode == "hazard" {
		err = run.ConfigureHazardPractice(req.HazardIntensity)
	}
	return run, err
}

func resetRiftPractice(run *rift.Run, req riftRequest, now time.Time) error {
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
	if req.SlowTelegraphs != nil {
		fresh.Practice.SlowTelegraphs = *req.SlowTelegraphs
	}
	fresh.Revision, fresh.StartKey, fresh.Epoch = run.Revision, run.StartKey, run.Epoch
	*run = *fresh
	return nil
}
