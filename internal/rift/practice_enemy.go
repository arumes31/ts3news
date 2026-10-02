package rift

import (
	"errors"
	"ts3news/internal/content"
)

// SpawnPracticeEnemy replaces the free-practice target using the current Abyss roster.
func (r *Run) SpawnPracticeEnemy(name string, catalog []content.Mob) error {
	if r.Practice == nil || r.Practice.Mode != "skills" || r.Status != "fighting" {
		return errors.New("enemy selection requires active free practice")
	}
	for _, mob := range catalog {
		if mob.Name != name {
			continue
		}
		enemy := AdaptMonster(mob)
		enemy.ID = "practice-enemy"
		enemy.X, enemy.Y = 560, 410
		r.clearPracticeArena()
		r.Enemies = []Actor{enemy}
		return nil
	}
	return errors.New("unknown practice enemy")
}

// ClearPracticeEnemies removes targets and projectiles without changing player resources.
func (r *Run) ClearPracticeEnemies() error {
	if r.Practice == nil || r.Practice.Mode != "skills" || r.Status != "fighting" {
		return errors.New("enemy removal requires active free practice")
	}
	r.clearPracticeArena()
	return nil
}

func (r *Run) clearPracticeArena() {
	r.Enemies = []Actor{}
	r.Projectiles = []Projectile{}
	r.Marked = ""
	r.LastMarkEnd = nil
}
