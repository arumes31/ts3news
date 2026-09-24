package rift

import (
	"encoding/json"
	"ts3news/internal/content"
)

// MarshalJSON exposes the original Abyss lore, falling back to its effect text.
func (d Drop) MarshalJSON() ([]byte, error) {
	type plain Drop
	description := ""
	if d.Gear != nil {
		description = d.Gear.Lore
		if description == "" {
			description = content.ItemEffectDescription(d.Gear.Special)
		}
	}
	return json.Marshal(struct {
		plain
		Description string `json:"gear_description,omitempty"`
	}{plain(d), description})
}
