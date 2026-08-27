package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssWikiCatalogUsesAuthoritativeContent(t *testing.T) {
	t.Parallel()

	view := abyssWikiCatalog()
	if len(view.Categories) != 4 {
		t.Fatalf("wiki categories = %d, want 4", len(view.Categories))
	}
	wantKeys := []string{"gear", "mobs", "pacts", "affixes"}
	total := 0
	for index, category := range view.Categories {
		if category.Key != wantKeys[index] {
			t.Errorf("category %d key = %q, want %q", index, category.Key, wantKeys[index])
		}
		if len(category.Entries) == 0 {
			t.Errorf("category %q has no entries", category.Key)
		}
		for entryIndex, entry := range category.Entries {
			if entry.ID == "" || entry.Name == "" || entry.Kind == "" || entry.Description == "" || entry.Search == "" {
				t.Errorf("category %q entry %d is incomplete: %#v", category.Key, entryIndex, entry)
			}
			if entry.Search != strings.ToLower(entry.Search) {
				t.Errorf("category %q entry %q has non-canonical search text", category.Key, entry.ID)
			}
		}
		total += len(category.Entries)
	}
	if view.Total != total {
		t.Fatalf("wiki total = %d, want %d", view.Total, total)
	}
	if len(view.Categories[0].Entries) != len(content.AbyssGearCatalog()) {
		t.Error("wiki gear catalog does not match authoritative Abyss gear")
	}
	if len(view.Categories[1].Entries) != len(content.AbyssMobCatalog()) {
		t.Error("wiki monster catalog does not match authoritative encounter templates")
	}
	wantAffixes := len(abyssDailyMods) + len(content.ItemEffectCatalog()) + len(content.MobEffectCatalog())
	if len(view.Categories[3].Entries) != wantAffixes {
		t.Fatalf("wiki affixes = %d, want %d", len(view.Categories[3].Entries), wantAffixes)
	}
}

func TestAbyssWikiContentCatalogsAreDetached(t *testing.T) {
	t.Parallel()

	itemEffects := content.ItemEffectCatalog()
	mobs := content.AbyssMobCatalog()
	effects := content.MobEffectCatalog()
	if len(itemEffects) == 0 || len(mobs) == 0 || len(effects) == 0 {
		t.Fatal("wiki content catalog is empty")
	}
	firstItemEffect := itemEffects[0]
	itemEffects[0] = content.EffectNone
	if current := content.ItemEffectCatalog()[0]; current != firstItemEffect {
		t.Fatal("ItemEffectCatalog leaked caller mutation into canonical content")
	}
	firstMobName := mobs[0].Name
	mobs[0].Name = "mutated"
	mobs[0].Effects = append(mobs[0].Effects, content.EffectSilenced)
	if current := content.AbyssMobCatalog()[0]; current.Name != firstMobName || len(current.Effects) != 0 {
		t.Fatal("AbyssMobCatalog leaked caller mutation into canonical content")
	}
	firstEffectKey := effects[0].Key
	effects[0].Key = "mutated"
	if current := content.MobEffectCatalog()[0]; current.Key != firstEffectKey {
		t.Fatal("MobEffectCatalog leaked caller mutation into canonical content")
	}
}

func TestAbyssWikiAssetsAndPageWiring(t *testing.T) {
	t.Parallel()

	partial, err := webAssets.ReadFile("webassets/abyss_wiki.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_wiki.css")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{
		"Delver's Field Archive",
		"abyssWikiSearch",
		"data-wiki-tab",
		"data-wiki-panel",
		"textContent",
		"ArrowRight",
	} {
		if !strings.Contains(string(partial), token) {
			t.Errorf("wiki partial is missing %q", token)
		}
	}
	for _, token := range []string{".ab-wiki-ledger", "grid-column:1/-1", "@media(max-width:820px)"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("wiki styles are missing %q", token)
		}
	}
	for _, token := range []string{"/static/abyss_wiki.css", `{{template "abyssWiki" .}}`} {
		if !strings.Contains(string(page), token) {
			t.Errorf("Abyss page is missing wiki wiring %q", token)
		}
	}
	root := abyssAAARepositoryRoot(t)
	webSource, err := os.ReadFile(filepath.Join(root, "internal", "bot", "web.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(webSource), `/static/abyss_wiki.css`) {
		t.Fatal("wiki stylesheet route is missing")
	}
}
