package bot

import (
	"bytes"
	"image/png"
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssTreePixelAtlasAssetsAndTemplateContract(t *testing.T) {
	t.Parallel()

	server, err := NewWebServer(nil)
	if err != nil {
		t.Fatalf("NewWebServer: %v", err)
	}
	treeTemplate := server.tmpl.Lookup("abysstree")
	if treeTemplate == nil {
		t.Fatal("Abyss Skill Web template is missing")
	}
	source := treeTemplate.Tree.Root.String()
	for _, marker := range []string{
		"TREE_ATLAS_URLS", "loadPassiveNodeIcon", "data-art-signature",
		"treeAtlasImages", "ctx.drawImage", "imageRendering = 'pixelated'",
	} {
		if !strings.Contains(source, marker) {
			t.Errorf("Abyss Skill Web atlas renderer is missing %q", marker)
		}
	}

	for _, sheet := range content.AbyssTreeArtSheets {
		name := "webassets/abyss_atlas_" + sheet + ".png"
		payload, err := webAssets.ReadFile(name)
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			continue
		}
		image, err := png.Decode(bytes.NewReader(payload))
		if err != nil {
			t.Errorf("decode %s: %v", name, err)
			continue
		}
		want := content.AbyssTreeArtCellSize * content.AbyssTreeArtColumns
		if image.Bounds().Dx() != want || image.Bounds().Dy() != want {
			t.Errorf("%s dimensions = %dx%d, want %dx%d", name, image.Bounds().Dx(), image.Bounds().Dy(), want, want)
		}
	}
}
