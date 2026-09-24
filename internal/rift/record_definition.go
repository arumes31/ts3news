package rift

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// levelDefinition identifies the frozen mission content, not the current catalog.
// A missing or unencodable definition remains unknown and must not be compared.
func levelDefinition(level *Level) string {
	if level == nil {
		return ""
	}
	data, err := json.Marshal(level)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return "level-v1:" + hex.EncodeToString(sum[:])
}
