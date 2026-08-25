package content

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func pixelArtIdentityForTest(key string) string {
	hash := func(text string) uint32 {
		value := uint32(2166136261)
		for _, codeUnit := range utf16.Encode([]rune(text)) {
			value ^= uint32(codeUnit)
			value *= 16777619
		}
		return value
	}
	return fmt.Sprintf("%08x%08x", hash("abyss-art-a:"+key), hash("abyss-art-b:"+key))
}

func TestEveryCatalogEntryHasUniquePixelArtIdentity(t *testing.T) {
	initSkills()
	initUltimateSkills()
	initMobs()

	keys := make([]string, 0, len(allSkills)+len(allUltimateSkills)+len(allGear)+len(baseMobs))
	for _, skill := range allSkills {
		keys = append(keys, "skill:"+skill.ID)
	}
	for _, ultimate := range allUltimateSkills {
		keys = append(keys, "ultimate:"+ultimate.ID)
	}
	for _, gear := range allGear {
		keys = append(keys, "item:"+gear.ID)
	}
	for _, consumable := range append(append([]Consumable{}, allConsumables...), abyssExclusiveConsumables...) {
		keys = append(keys, "item:"+consumable.ID)
	}
	seenMobs := make(map[string]struct{}, len(baseMobs))
	for _, mob := range baseMobs {
		if _, exists := seenMobs[mob.Name]; exists {
			continue
		}
		seenMobs[mob.Name] = struct{}{}
		keys = append(keys, "monster:"+mob.Name)
	}
	for index, artifact := range corruptedArtifacts {
		keys = append(keys, fmt.Sprintf("item:artifact:%d:%s", index, artifact.Name))
	}

	if len(keys) < 2500 {
		t.Fatalf("catalog coverage unexpectedly small: %d entries", len(keys))
	}
	identities := make(map[string]string, len(keys))
	for _, key := range keys {
		identity := pixelArtIdentityForTest(key)
		if previous, exists := identities[identity]; exists {
			t.Fatalf("pixel-art identity collision: %q and %q both use %s", previous, key, identity)
		}
		identities[identity] = key
	}
	t.Logf(
		"verified unique pixel-art identities for %d catalog entries (%d skills, %d ultimates, %d gear, %d monsters)",
		len(keys), len(allSkills), len(allUltimateSkills), len(allGear), len(seenMobs),
	)
}

func TestMobCatalogInitializationIsIdempotent(t *testing.T) {
	initMobs()
	before := len(baseMobs)
	for range 10 {
		initMobs()
	}
	if len(baseMobs) != before {
		t.Fatalf("mob catalog grew from %d to %d after repeated initialization", before, len(baseMobs))
	}
}
