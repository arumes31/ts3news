package bot

import (
	"errors"
	"time"
	"ts3news/internal/rift"
)

func newRiftPractice(req riftRequest, id string, build rift.Build, mode string, now time.Time) (*rift.Run, error) {
	if mode == "boss" && (req.BossName != "" || req.BossPhase != 0) {
		return rift.NewBossPracticeRunWithCatalog(id, build, req.BossName, req.BossPhase, now, riftMobCatalog(now))
	}
	return rift.NewPracticeRun(id, build, mode, now)
}

func resetRiftPractice(run *rift.Run, req riftRequest, now time.Time) error {
	if req.BossName == "" && req.BossPhase == 0 {
		return run.ResetPractice(now)
	}
	if run.Practice == nil || run.Practice.Mode != "boss" {
		return errors.New("boss selection requires boss practice")
	}
	fresh, err := rift.NewBossPracticeRunWithCatalog(run.ID, run.Build, req.BossName, req.BossPhase, now, riftMobCatalog(now))
	if err != nil {
		return err
	}
	fresh.Revision, fresh.StartKey, fresh.Epoch = run.Revision, run.StartKey, run.Epoch
	*run = *fresh
	return nil
}
