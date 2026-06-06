package icon

import (
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
)

// tierColors maps to 10 major rank groups to provide more variety.
var tierColors = []color.RGBA{
	{205, 127, 50, 255},  // Bronze
	{192, 192, 192, 255}, // Silver
	{255, 215, 0, 255},   // Gold
	{50, 205, 50, 255},   // Emerald
	{30, 144, 255, 255},  // DodgerBlue
	{148, 0, 211, 255},   // Amethyst
	{255, 69, 0, 255},    // OrangeRed
	{0, 255, 255, 255},   // Cyan
	{255, 20, 147, 255},  // DeepPink
	{255, 255, 255, 255}, // White/Diamond
}

const MaxTiers = 1000

// GenerateTierIcons creates 16x16 PNG icons for the given number of tiers.
// TS3 server group icons are strictly 16x16 pixels.
func GenerateTierIcons(outputDir string, numTiers int) error {
	if numTiers < 1 || numTiers > MaxTiers {
		return fmt.Errorf("invalid numTiers %d (must be 1-%d)", numTiers, MaxTiers)
	}
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return err
	}

	const S = 16.0

	for i := 0; i < numTiers; i++ {
		dc := gg.NewContext(16, 16)

		// tier is 1..numTiers
		tier := i + 1
		colorIdx := ((tier - 1) / 5) % len(tierColors)
		shapeIdx := (tier - 1) % 5

		baseColor := tierColors[colorIdx]
		dc.SetColor(baseColor)

		// Draw progression of shapes: Circle -> Square -> Diamond -> Pentagon -> Hexagon
		switch shapeIdx {
		case 0: // Circle
			dc.DrawCircle(S/2, S/2, S/2-1)
		case 1: // Square
			dc.DrawRectangle(1, 1, S-2, S-2)
		case 2: // Diamond
			dc.MoveTo(S/2, 1)
			dc.LineTo(S-1, S/2)
			dc.LineTo(S/2, S-1)
			dc.LineTo(1, S/2)
			dc.ClosePath()
		case 3: // Pentagon
			dc.DrawRegularPolygon(5, S/2, S/2, S/2-1, math.Pi)
		case 4: // Hexagon
			dc.DrawRegularPolygon(6, S/2, S/2, S/2-1, math.Pi/2)
		}
		dc.FillPreserve()

		// Black border
		dc.SetColor(color.Black)
		dc.SetLineWidth(1)
		dc.Stroke()

		// Draw the tier number in the center
		dc.SetFontFace(basicfont.Face7x13)
		if isLight(baseColor) {
			dc.SetColor(color.Black)
		} else {
			dc.SetColor(color.White)
		}
		
		text := fmt.Sprintf("%d", tier)
		dc.DrawStringAnchored(text, S/2, S/2-1, 0.5, 0.5)

		outPath := filepath.Join(outputDir, fmt.Sprintf("tier_%d.png", tier))
		if err := dc.SavePNG(outPath); err != nil {
			return err
		}
	}
	return nil
}

func isLight(c color.RGBA) bool {
	lum := (int(c.R)*299 + int(c.G)*587 + int(c.B)*114) / 1000
	return lum > 140
}
