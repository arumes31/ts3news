package bot

import (
	"bytes"
	"image/png"
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"ts3news/internal/rift"
)

func TestRiftRegionBackgroundPanelsCoverCampaign(t *testing.T) {
	source, err := webAssets.ReadFile("webassets/rift_renderer.js")
	if err != nil {
		t.Fatal(err)
	}
	matches := regexp.MustCompile(`const regionRows\s*=\s*\[([^\]]+)\]`).FindAllSubmatch(source, -1)
	if len(matches) != 1 {
		t.Fatal("expected one background row-boundary contract")
	}
	var boundaries []float64
	for _, raw := range strings.Split(string(matches[0][1]), ",") {
		value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			t.Fatalf("invalid background row boundary %q", raw)
		}
		boundaries = append(boundaries, value)
	}
	if len(boundaries) < 2 || boundaries[0] != 0 || boundaries[len(boundaries)-1] != 1 {
		t.Fatal("background rows must span the complete atlas")
	}
	columnMatch := regexp.MustCompile(`row=Math\.floor\(region/(\d+)\)`).FindAllSubmatch(source, -1)
	if len(columnMatch) != 1 {
		t.Fatal("expected one background column-count contract")
	}
	columns, err := strconv.Atoi(string(columnMatch[0][1]))
	if err != nil || columns <= 0 {
		t.Fatal("invalid background columns")
	}
	horizontal := regexp.MustCompile(`region%(\d+)\*background\.width/(\d+)\+2`).FindAllSubmatch(source, -1)
	if len(horizontal) != 1 || string(horizontal[0][1]) != string(columnMatch[0][1]) || string(horizontal[0][2]) != string(columnMatch[0][1]) {
		t.Fatal("background horizontal crop must use the same column count as row selection")
	}
	encoded, err := webAssets.ReadFile("webassets/rift_regions.png")
	if err != nil {
		t.Fatal(err)
	}
	image, err := png.DecodeConfig(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	// The current renderer trims two pixels from each edge of a panel.
	if float64(image.Width)/float64(columns) <= 4 {
		t.Fatal("background panels have no width after trimming")
	}
	for row := 0; row < len(boundaries)-1; row++ {
		if boundaries[row] < 0 || boundaries[row+1] > 1 || (boundaries[row+1]-boundaries[row])*float64(image.Height) <= 4 {
			t.Fatalf("background row %d has invalid or empty bounds", row)
		}
	}
	panels := columns * (len(boundaries) - 1)
	for _, level := range rift.Campaign() {
		if level.Region < 0 || level.Region >= panels {
			t.Errorf("mission %d selects region %d outside %d background panels", level.ID, level.Region, panels)
		}
	}
}
