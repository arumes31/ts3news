package rift

// EnemyDropGold is the floor-gold reward for one defeated monster in a zero-based room.
// Objective props and escaped monsters never create a reward drop.
func EnemyDropGold(room int) int64 {
	if room < 0 || room >= len(Rooms) {
		return 0
	}
	return int64(15 * (room + 1))
}
