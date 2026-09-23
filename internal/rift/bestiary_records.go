package rift

// MonsterRecord contains observations made by this version, not inferred history.
type MonsterRecord struct {
	FastestClearSeconds *float64 `json:"fastest_clear_seconds,omitempty"`
	FirstSeenMS         int64    `json:"first_seen_ms"`
	Defeats             int      `json:"defeats"`
}

func (r *Run) observeMonster(actor Actor) {
	if r.Practice != nil || r.Level == nil || actor.Name == "" || actor.ArtKey != "monster:"+actor.Name {
		return
	}
	if r.MonsterRecords == nil {
		r.MonsterRecords = map[string]MonsterRecord{}
	}
	if _, exists := r.MonsterRecords[actor.ArtKey]; !exists {
		r.MonsterRecords[actor.ArtKey] = MonsterRecord{FirstSeenMS: max(int64(0), r.LastMS)}
	}
}

func (r *Run) observeRoomMonsters() {
	for _, actor := range r.Enemies {
		if actor.HP > 0 {
			r.observeMonster(actor)
		}
	}
}

func (r *Run) recordMonsterDefeat(actor Actor) {
	r.observeMonster(actor)
	if r.Practice != nil || r.Level == nil {
		return
	}
	if record, exists := r.MonsterRecords[actor.ArtKey]; exists {
		record.Defeats++
		r.MonsterRecords[actor.ArtKey] = record
	}
}

func (r *Run) inheritMonsterRecords(previous *Run) {
	combined := make(map[string]MonsterRecord, len(previous.MonsterRecords)+len(r.MonsterRecords))
	for key, record := range previous.MonsterRecords {
		combined[key] = record
	}
	for key, record := range r.MonsterRecords {
		if old, exists := combined[key]; exists {
			record.FirstSeenMS = min(record.FirstSeenMS, old.FirstSeenMS)
			record.Defeats += old.Defeats
			if old.FastestClearSeconds != nil && (record.FastestClearSeconds == nil || *old.FastestClearSeconds < *record.FastestClearSeconds) {
				seconds := *old.FastestClearSeconds
				record.FastestClearSeconds = &seconds
			}
		}
		combined[key] = record
	}
	r.MonsterRecords = combined
}

// recordBossClear measures the entire room, so surviving defenders still count.
func (r *Run) recordBossClear() {
	if r.Practice != nil || r.Level == nil || r.Status != "cleared" || r.RoomStartSeconds == nil {
		return
	}
	seconds := r.Stats.Seconds - *r.RoomStartSeconds
	if seconds < 0 {
		return
	}
	for _, actor := range r.Enemies {
		if actor.HP > 0 {
			return
		}
	}
	for _, actor := range r.Enemies {
		if actor.Kind != "boss" {
			continue
		}
		r.observeMonster(actor)
		record, exists := r.MonsterRecords[actor.ArtKey]
		if !exists {
			continue
		}
		if record.FastestClearSeconds == nil || seconds < *record.FastestClearSeconds {
			best := seconds
			record.FastestClearSeconds = &best
			r.MonsterRecords[actor.ArtKey] = record
		}
	}
}
