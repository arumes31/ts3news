package rift

import (
	"errors"
	"time"
	"ts3news/internal/content"
)

// NewBossPracticeRun starts an isolated fight against a shared Abyss boss.
// Phase selection changes starting health and timing; reset preserves this snapshot.
func NewBossPracticeRun(id string, build Build, name string, phase int, now time.Time) (*Run, error) {
	return NewBossPracticeRunWithCatalog(id, build, name, phase, now, content.AbyssMobCatalog())
}

// NewBossPracticeRunWithCatalog also supports the server's live Abyss boss roster.
func NewBossPracticeRunWithCatalog(id string, build Build, name string, phase int, now time.Time, catalog []content.Mob) (*Run, error) {
	if phase < 1 || phase > len(bossPhaseTraining) {
		return nil, errors.New("invalid boss phase")
	}
	for _, mob := range catalog {
		if mob.Name != name {
			continue
		}
		boss := AdaptMonster(mob)
		if boss.Kind != "boss" {
			break
		}
		boss.ID = "practice-boss"
		boss.X, boss.Y = 560, 410
		boss.Phase = phase
		boss.HP = boss.MaxHP * float64(bossPhaseTraining[phase-1].AtHealthPercent) / 100
		return newBossPracticeSnapshot(id, build, boss, now)
	}
	return nil, errors.New("unknown practice boss")
}

func newBossPracticeSnapshot(id string, build Build, boss Actor, now time.Time) (*Run, error) {
	r, err := newPracticeRun(id, build, "boss", now)
	if err != nil {
		return nil, err
	}
	r.Practice.Arena.Name = "Boss practice arena"
	r.Practice.BossStart = &boss
	r.Enemies = []Actor{boss}
	return r, nil
}
