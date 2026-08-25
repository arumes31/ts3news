package bot

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
)

const abyssTierCatalogEnv = "TS3NEWS_ABYSS_TIERS_FILE"

//go:embed abyss_tiers.json
var embeddedAbyssTierCatalog []byte

func mustLoadAbyssTierCatalog() map[string]abyssTier {
	source := "embedded abyss_tiers.json"
	data := embeddedAbyssTierCatalog
	if path := strings.TrimSpace(os.Getenv(abyssTierCatalogEnv)); path != "" {
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("load Abyss tier catalog %q: %v", path, err))
		}
		source = path
	}

	tiers, err := decodeAbyssTierCatalog(data)
	if err != nil {
		panic(fmt.Sprintf("invalid Abyss tier catalog %s: %v", source, err))
	}
	return tiers
}

func decodeAbyssTierCatalog(data []byte) (map[string]abyssTier, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var catalog []abyssTier
	if err := decoder.Decode(&catalog); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, err
	}
	if len(catalog) != len(abyssTierOrder) {
		return nil, fmt.Errorf("expected %d tiers, got %d", len(abyssTierOrder), len(catalog))
	}

	tiers := make(map[string]abyssTier, len(catalog))
	previousDepth := -1
	for index, tier := range catalog {
		expectedKey := abyssTierOrder[index]
		if tier.Key != expectedKey {
			return nil, fmt.Errorf("tier %d key must be %q, got %q", index, expectedKey, tier.Key)
		}
		if strings.TrimSpace(tier.Name) == "" {
			return nil, fmt.Errorf("tier %q name is required", tier.Key)
		}
		if !positiveFinite(tier.DiffMult) {
			return nil, fmt.Errorf("tier %q difficulty_multiplier must be finite and positive", tier.Key)
		}
		if !positiveFinite(tier.RewardMult) {
			return nil, fmt.Errorf("tier %q reward_multiplier must be finite and positive", tier.Key)
		}
		if tier.EntryGold < 0 {
			return nil, fmt.Errorf("tier %q entry_gold must not be negative", tier.Key)
		}
		if tier.MinBest < 0 || tier.MinBest < previousDepth {
			return nil, fmt.Errorf("tier %q minimum_best_depth must be non-negative and ordered", tier.Key)
		}
		previousDepth = tier.MinBest
		tiers[tier.Key] = tier
	}
	return tiers, nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("decode JSON: multiple values are not allowed")
		}
		return fmt.Errorf("decode JSON trailer: %w", err)
	}
	return nil
}

func positiveFinite(value float64) bool {
	return value > 0 && !math.IsInf(value, 0) && !math.IsNaN(value)
}
