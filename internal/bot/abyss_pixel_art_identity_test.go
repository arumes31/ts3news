package bot

import (
	"fmt"
	"testing"
	"unicode/utf16"
)

func abyssPixelArtIdentityForTest(key string) string {
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

func TestEveryAbyssBossHasUniquePixelArtIdentity(t *testing.T) {
	names := append([]string{}, abyssBossRoster...)
	for _, lore := range abyssBossLoreCatalog {
		names = append(names, lore.Boss)
	}
	for _, boss := range abyssSecretBosses {
		names = append(names, boss.Name)
	}

	seenNames := make(map[string]struct{}, len(names))
	seenArt := make(map[string]string, len(names))
	for _, name := range names {
		if _, duplicate := seenNames[name]; duplicate {
			continue
		}
		seenNames[name] = struct{}{}
		identity := abyssPixelArtIdentityForTest("monster:" + name)
		if previous, collision := seenArt[identity]; collision {
			t.Fatalf("boss art collision: %q and %q both use %s", previous, name, identity)
		}
		seenArt[identity] = name
	}
	if len(seenNames) < 7 {
		t.Fatalf("boss art coverage unexpectedly small: %d names", len(seenNames))
	}
}
