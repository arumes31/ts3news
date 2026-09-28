package bot

import (
	"encoding/json"
	"strings"
	"testing"
	"ts3news/internal/content"
)

func TestRiftBestiaryCopiesCanonicalAliasesWithoutChangingIdentity(t *testing.T) {
	mobs := []content.Mob{{Name: "Canonical sentinel", Aliases: []string{"Former sentinel", "Sentinelle"}, Type: content.MobBoss}, {Name: "No alias", Type: content.MobCommon}}
	entries := riftBestiaryFromCatalog(mobs)
	if len(entries) != 2 || len(entries[0].Aliases) != 2 || entries[0].Aliases[0] != "Former sentinel" || entries[0].Name != mobs[0].Name || entries[0].ArtKey != "monster:"+mobs[0].Name {
		t.Fatal("aliases changed identity or were lost")
	}
	entries[0].Aliases[0] = "changed"
	if mobs[0].Aliases[0] != "Former sentinel" {
		t.Fatal("bestiary aliases share canonical storage")
	}
	raw, err := json.Marshal(entries[1])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "aliases") {
		t.Fatal("unavailable aliases should be omitted")
	}
}
