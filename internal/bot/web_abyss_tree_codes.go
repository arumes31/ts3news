package bot

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"ts3news/internal/content"
)

const abyssTreeBuildCodeMaxBytes = 128 * 1024

type abyssTreeBuildCode struct {
	Version int    `json:"v"`
	Schema  int    `json:"schema"`
	Layout  string `json:"layout"`
	IDs     []int  `json:"ids"`
}

func decodeAbyssTreeBuildCode(encoded string) (abyssTreeBuildCode, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" || len(encoded) > base64.StdEncoding.EncodedLen(abyssTreeBuildCodeMaxBytes) {
		return abyssTreeBuildCode{}, errors.New("build code is empty or too large")
	}
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(payload) > abyssTreeBuildCodeMaxBytes {
		return abyssTreeBuildCode{}, errors.New("build code is not valid base64")
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var code abyssTreeBuildCode
	if err := decoder.Decode(&code); err != nil {
		return abyssTreeBuildCode{}, errors.New("build code is not valid JSON")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return abyssTreeBuildCode{}, errors.New("build code contains trailing data")
	}
	if code.Version != 1 || code.Schema != content.TreeCatalogSchemaVersion {
		return abyssTreeBuildCode{}, fmt.Errorf("unsupported build-code version or schema")
	}
	if code.Layout == "" || len(code.IDs) == 0 || len(code.IDs) > abyssTreePlanMaxNodes {
		return abyssTreeBuildCode{}, errors.New("build code has incomplete or oversized content")
	}
	return code, nil
}
