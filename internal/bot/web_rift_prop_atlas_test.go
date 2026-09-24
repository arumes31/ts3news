package bot

import (
	"bytes"
	"encoding/json"
	"image/png"
	"regexp"
	"strconv"
	"testing"
	"ts3news/internal/rift"
)

// Check the actual renderer mapping and embedded image together, so editing a
// regional prop index cannot silently introduce an out-of-atlas source rectangle.
func TestRiftCoverPropIndicesFitAtlas(t *testing.T) {
	source, err := webAssets.ReadFile("webassets/rift_renderer.js")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`index=\s*(\[[^\]]+\])\[run\.level\?\.region\|\|0\],\s*sw=img\.width/(\d+),\s*sh=img\.height/(\d+)`)
	matches := pattern.FindAllSubmatch(source, -1)
	if len(matches) != 1 {
		t.Fatal("expected one regional prop mapping; update this content check if the renderer's atlas contract changes")
	}
	var indices []int
	if err := json.Unmarshal(matches[0][1], &indices); err != nil {
		t.Fatalf("prop indices must be integers: %v", err)
	}
	columns, _ := strconv.Atoi(string(matches[0][2]))
	rows, _ := strconv.Atoi(string(matches[0][3]))
	encoded, err := webAssets.ReadFile("webassets/rift_props.png")
	if err != nil {
		t.Fatal(err)
	}
	image, err := png.DecodeConfig(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if columns <= 0 || rows <= 0 || image.Width < columns || image.Height < rows {
		t.Fatalf("invalid prop grid %dx%d for image %dx%d", columns, rows, image.Width, image.Height)
	}
	for region, index := range indices {
		if index < 0 || index >= columns*rows {
			t.Errorf("region %d references prop %d outside %dx%d atlas", region, index, columns, rows)
		}
	}
	for _, level := range rift.Campaign() {
		if level.Region < 0 || level.Region >= len(indices) {
			t.Errorf("mission %d region %d has no prop mapping", level.ID, level.Region)
		}
	}
}
