package bot

import (
	"time"
	"ts3news/internal/rift"
)

func applyRiftPracticeEnemy(run *rift.Run, request riftRequest, now time.Time) error {
	if request.Kind == "practice_spawn" {
		return run.SpawnPracticeEnemy(request.EnemyName, riftMobCatalog(now))
	}
	return run.ClearPracticeEnemies()
}
