package bot

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
)

const (
	abyssRecentGearFloorWindow = int64(20)
	abyssRecentGearHistoryMax  = 512
)

type abyssRecentGearEntry struct {
	GearID       string `json:"gear_id"`
	FloorOrdinal int64  `json:"floor_ordinal"`
}

func abyssRecentGearKey(uid string) string {
	return "abyss_recent_gear_" + uid
}

func normalizeAbyssRecentGear(entries []abyssRecentGearEntry, currentFloor int64) []abyssRecentGearEntry {
	if currentFloor <= 0 {
		return nil
	}
	seen := make(map[string]bool, len(entries))
	normalized := make([]abyssRecentGearEntry, 0, min(len(entries), abyssRecentGearHistoryMax))
	for index := len(entries) - 1; index >= 0 && len(normalized) < abyssRecentGearHistoryMax; index-- {
		entry := entries[index]
		if entry.GearID == "" || entry.FloorOrdinal <= 0 || entry.FloorOrdinal > currentFloor ||
			currentFloor-entry.FloorOrdinal > abyssRecentGearFloorWindow || seen[entry.GearID] {
			continue
		}
		seen[entry.GearID] = true
		normalized = append(normalized, entry)
	}
	for left, right := 0, len(normalized)-1; left < right; left, right = left+1, right-1 {
		normalized[left], normalized[right] = normalized[right], normalized[left]
	}
	return normalized
}

func abyssRecentGearIDSet(entries []abyssRecentGearEntry) map[string]bool {
	ids := make(map[string]bool, len(entries))
	for _, entry := range entries {
		ids[entry.GearID] = true
	}
	return ids
}

func (b *Bot) abyssRecentGearProtection(uid string) (map[string]bool, int64) {
	var lifetimeFloors int64
	if err := b.DB.QueryRow("SELECT abyss_lifetime_floors FROM users WHERE client_uid=$1", uid).Scan(&lifetimeFloors); err != nil {
		return map[string]bool{}, 1
	}
	if lifetimeFloors == math.MaxInt64 {
		return map[string]bool{}, math.MaxInt64
	}
	currentFloor := max(lifetimeFloors+1, int64(1))
	var raw string
	if err := b.DB.QueryRow("SELECT value FROM app_meta WHERE key=$1", abyssRecentGearKey(uid)).Scan(&raw); err != nil {
		return map[string]bool{}, currentFloor
	}
	var entries []abyssRecentGearEntry
	if json.Unmarshal([]byte(raw), &entries) != nil {
		return map[string]bool{}, currentFloor
	}
	return abyssRecentGearIDSet(normalizeAbyssRecentGear(entries, currentFloor)), currentFloor
}

func recordAbyssRecentGearDrop(db dbOrTx, uid, gearID string) error {
	if gearID == "" {
		return nil
	}
	var lifetimeFloors int64
	if err := db.QueryRow("SELECT abyss_lifetime_floors FROM users WHERE client_uid=$1", uid).Scan(&lifetimeFloors); err != nil {
		return fmt.Errorf("read Abyss lifetime floors: %w", err)
	}
	if lifetimeFloors == math.MaxInt64 {
		return fmt.Errorf("abyss lifetime floor counter exhausted")
	}
	currentFloor := max(lifetimeFloors+1, int64(1))
	var raw string
	err := db.QueryRow("SELECT value FROM app_meta WHERE key=$1 FOR UPDATE", abyssRecentGearKey(uid)).Scan(&raw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("lock recent Abyss gear: %w", err)
	}
	var entries []abyssRecentGearEntry
	if err == nil && json.Unmarshal([]byte(raw), &entries) != nil {
		entries = nil
	}
	entries = normalizeAbyssRecentGear(entries, currentFloor)
	filtered := entries[:0]
	for _, entry := range entries {
		if entry.GearID != gearID {
			filtered = append(filtered, entry)
		}
	}
	entries = append(filtered, abyssRecentGearEntry{GearID: gearID, FloorOrdinal: currentFloor})
	data, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("encode recent Abyss gear: %w", err)
	}
	if _, err := db.Exec(`INSERT INTO app_meta (key,value) VALUES ($1,$2)
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`, abyssRecentGearKey(uid), string(data)); err != nil {
		return fmt.Errorf("save recent Abyss gear: %w", err)
	}
	return nil
}
