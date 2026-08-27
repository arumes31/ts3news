package bot

import (
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var abyssStaticAssetPattern = regexp.MustCompile(`/static/([A-Za-z0-9][A-Za-z0-9_.-]*\.(?:css|gif|js|png|svg|webp|woff|woff2))`)

func TestAbyssStaticAssetReferencesResolve(t *testing.T) {
	t.Parallel()

	entries, err := fs.ReadDir(webAssets, "webassets")
	if err != nil {
		t.Fatalf("read embedded web assets: %v", err)
	}

	references := make(map[string][]string)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "abyss") || !isAbyssAssetSource(name) {
			continue
		}
		body, readErr := webAssets.ReadFile("webassets/" + name)
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		for _, match := range abyssStaticAssetPattern.FindAllSubmatch(body, -1) {
			asset := string(match[1])
			references[asset] = append(references[asset], name)
		}
	}

	if len(references) < 100 {
		t.Fatalf("asset audit found only %d fixed references; expected the complete Abyss surface", len(references))
	}
	assets := make([]string, 0, len(references))
	for asset := range references {
		assets = append(assets, asset)
	}
	sort.Strings(assets)
	for _, asset := range assets {
		if _, readErr := webAssets.ReadFile("webassets/" + asset); readErr != nil {
			t.Errorf("missing embedded asset %q referenced by %s", asset, strings.Join(references[asset], ", "))
		}
	}
}

func isAbyssAssetSource(name string) bool {
	return strings.HasSuffix(name, ".html") || strings.HasSuffix(name, ".css") || strings.HasSuffix(name, ".js")
}
