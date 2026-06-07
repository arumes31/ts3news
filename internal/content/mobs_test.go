package content

import (
	"testing"
)

func TestSpawnMobTypes(t *testing.T) {
	// Test spawning various types
	types := make(map[MobType]bool)
	for i := 0; i < 1000; i++ {
		m := SpawnMob(30, false, 1.0)
		types[m.Type] = true
	}

	expectedTypes := []MobType{MobCommon, MobEliteMinion, MobElite, MobMiniboss, MobBoss, MobLegendary}
	for _, et := range expectedTypes {
		if !types[et] {
			t.Errorf("Expected to spawn at least one %s mob in 1000 tries, but none were found", et)
		}
	}
}

func TestSpawnBoss(t *testing.T) {
	m := SpawnMob(15, true, 1.0)
	if m.Type != MobBoss {
		t.Errorf("Expected MobBoss when isBoss is true and level >= 10, got %s", m.Type)
	}
}
