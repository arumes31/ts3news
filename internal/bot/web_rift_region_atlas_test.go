package bot

import (
	"bytes"
	"encoding/json"
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
	sectionsRaw, err := webAssets.ReadFile("webassets/rift_region_sections.js")
	if err != nil {
		t.Fatal(err)
	}
	prefix := "window.RiftRegionSections="
	idx := bytes.Index(sectionsRaw, []byte(prefix))
	if idx < 0 {
		t.Fatal("expected window.RiftRegionSections definition")
	}
	jsonBytes := bytes.TrimSpace(sectionsRaw[idx+len(prefix):])
	jsonBytes = bytes.TrimSuffix(jsonBytes, []byte(";"))

	var sections struct {
		Version int `json:"version"`
		Width   int `json:"width"`
		Height  int `json:"height"`
		Regions []struct {
			X      int        `json:"x"`
			Y      int        `json:"y"`
			Width  int        `json:"width"`
			Height int        `json:"height"`
			Source [4]float64 `json:"source"`
			URL    string     `json:"url"`
		} `json:"regions"`
	}
	if err := json.Unmarshal(jsonBytes, &sections); err != nil {
		t.Fatalf("unmarshal rift_region_sections.js: %v", err)
	}
	columns := 2
	expectedPanels := columns * (len(boundaries) - 1)
	if len(sections.Regions) != expectedPanels {
		t.Fatalf("expected %d background panels, got %d", expectedPanels, len(sections.Regions))
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
	for i, reg := range sections.Regions {
		if reg.Width <= 4 || reg.Height <= 4 {
			t.Fatalf("region %d panel too small: %dx%d", i, reg.Width, reg.Height)
		}
		if reg.Source[2] <= 0 || reg.Source[3] <= 0 {
			t.Fatalf("region %d source rect invalid: %v", i, reg.Source)
		}
	}
	panels := len(sections.Regions)
	for _, level := range rift.Campaign() {
		if level.Region < 0 || level.Region >= panels {
			t.Errorf("mission %d selects region %d outside %d background panels", level.ID, level.Region, panels)
		}
	}
}
