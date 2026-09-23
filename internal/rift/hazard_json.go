package rift

import "encoding/json"

// UnmarshalJSON preserves jump evasion in older saved floor hazards. Newly
// authored hazards use the explicit flag; unknown legacy kinds default to false.
func (h *Hazard) UnmarshalJSON(data []byte) error {
	type plain Hazard
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if _, exists := fields["jumpable"]; !exists {
		switch value.Kind {
		case "fire", "ice", "poison", "thorns", "rune", "radiant", "void":
			value.Jumpable = true
		}
	}
	*h = Hazard(value)
	return nil
}
