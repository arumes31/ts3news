package rift

import "math"

// projectileBuckets owns only enemy indices, never actor or snapshot pointers.
// Rebuild after enemy movement, then reuse for every friendly shot in that tick.
// Hits change health and recoil animation, but do not move enemy coordinates.
type projectileBuckets struct {
	heads [16]int
	next  []int
}

func projectileBucket(x float64) int {
	if x <= 0 {
		return 0
	}
	if x >= Width {
		return 15
	}
	return int(x / 100)
}

func (b *projectileBuckets) reset(enemies []Actor) {
	for i := range b.heads {
		b.heads[i] = -1
	}
	if cap(b.next) < len(enemies) {
		b.next = make([]int, len(enemies))
	} else {
		b.next = b.next[:len(enemies)]
	}
	for i := len(enemies) - 1; i >= 0; i-- {
		if enemies[i].HP <= 0 || math.IsNaN(enemies[i].X) {
			continue
		}
		cell := projectileBucket(enemies[i].X)
		b.next[i] = b.heads[cell]
		b.heads[cell] = i
	}
}

func (b *projectileBuckets) target(enemies []Actor, shot Projectile) int {
	if math.IsNaN(shot.X) {
		return -1
	}
	ref := shot.Skill.Reference()
	first, last := projectileBucket(shot.X-ref.Horizontal), projectileBucket(shot.X+ref.Horizontal)
	target := -1
	for cell := first; cell <= last; cell++ {
		for i := b.heads[cell]; i >= 0; i = b.next[i] {
			if target >= 0 && i >= target {
				break
			}
			e := &enemies[i]
			if e.HP > 0 && math.Abs(shot.X-e.X) < ref.Horizontal && math.Abs(shot.Y-e.Y) < ref.Depth {
				target = i
				break
			}
		}
	}
	return target
}
