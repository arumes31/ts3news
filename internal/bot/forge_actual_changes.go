package bot

import (
	"encoding/json"
	"ts3news/internal/content"
)

func forgeActualStatChanges(before, after json.RawMessage) []gearStatChange {
	var a, b struct {
		Item *content.Gear `json:"item"`
	}
	if json.Unmarshal(before, &a) != nil || json.Unmarshal(after, &b) != nil || a.Item == nil || b.Item == nil || a.Item.Unidentified || b.Item.Unidentified {
		return nil
	}
	old, next := a.Item.Stats.Details(), b.Item.Stats.Details()
	var changes []gearStatChange
	for i, s := range next {
		if s.Value != old[i].Value {
			changes = append(changes, gearStatChange{Code: s.Code, Label: s.Label, Combat: s.Combat, Before: old[i].Value, After: s.Value, Delta: s.Value - old[i].Value})
		}
	}
	return changes
}
