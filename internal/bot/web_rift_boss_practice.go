package bot

import (
	"errors"
	"time"
	"ts3news/internal/rift"
)

func newRiftPractice(req riftRequest, id string, build rift.Build, mode string, now time.Time) (*rift.Run, error) {
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
	return run, err
}

func resetRiftPractice(run *rift.Run, req riftRequest, now time.Time) error {
	if req.BossName == "" && req.BossPhase == 0 {
		if err := run.ResetPractice(now); err != nil {
			return err
		}
		if run.Practice.Mode == "boss" && req.SlowTelegraphs != nil {
			run.Practice.SlowTelegraphs = *req.SlowTelegraphs
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
	fresh.Practice.SlowTelegraphs = run.Practice.SlowTelegraphs
	if req.SlowTelegraphs != nil {
		fresh.Practice.SlowTelegraphs = *req.SlowTelegraphs
	}
	fresh.Revision, fresh.StartKey, fresh.Epoch = run.Revision, run.StartKey, run.Epoch
	*run = *fresh
	return nil
}
