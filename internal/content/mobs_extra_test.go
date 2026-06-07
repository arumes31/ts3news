package content

import (
	"testing"
)

func TestMobLogic(t *testing.T) {
	m := Mob{
		Name: "Orc",
		Type: MobCommon,
		Level: 5,
		Stats: Stats{HP: 50, STR: 10},
	}
	if m.DisplayName() != "Lvl 5 Orc [Common] (0/0 HP)" {
		t.Errorf("DisplayName = %q", m.DisplayName())
	}
	if m.Score() == 0 {
		t.Error("Mob.Score() should not be zero")
	}
	
	clone := m.Clone()
	if clone.Name != m.Name || clone.Level != m.Level {
		t.Errorf("Clone failed: %+v", clone)
	}
	
	mobs := SpawnMobGroup(10, Zone{Name: "Test", Difficulty: 1.0}, 1.0, 1)
	if len(mobs) == 0 {
		t.Error("SpawnMobGroup returned empty")
	}
}

func TestSpawnMob(t *testing.T) {
	// Bosses require level 10+ in SpawnMob logic
	m := SpawnMob(10, true, 1.0)
	if m.Type != MobBoss {
		t.Errorf("SpawnMob with boss=true, lvl=10 should be a boss, got %v", m.Type)
	}
	m2 := SpawnMob(10, false, 1.0)
	if m2.Type == MobBoss {
		t.Error("SpawnMob with boss=false should not be a boss")
	}
}
