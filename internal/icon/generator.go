package icon

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
	"golang.org/x/image/font/basicfont"
)

// tierColors maps to 5 major rank groups (Bronze, Silver, Gold, Emerald, Amethyst).
var tierColors = []color.RGBA{
	{205, 127, 50, 255},  // 0-4: Bronze
	{192, 192, 192, 255}, // 5-9: Silver
	{255, 215, 0, 255},   // 10-14: Gold
	{50, 205, 50, 255},   // 15-19: Emerald
	{148, 0, 211, 255},   // 20-24: Amethyst
}

// GenerateTierIcons creates 16x16 PNG icons for the given number of tiers.
// TS3 server group icons are strictly 16x16 pixels.
func GenerateTierIcons(outputDir string, numTiers int) error {
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return err
	}

	const S = 16

	for i := 0; i < numTiers; i++ {
		dc := gg.NewContext(S, S)

		colorIdx := (i / 5) % len(tierColors)
		shapeIdx := i % 5

		baseColor := tierColors[colorIdx]

		dc.SetColor(baseColor)

		// Draw a progression of shapes within each color group
		switch shapeIdx {
		case 0: // Circle
			dc.DrawCircle(S/2, S/2, S/2-1)
		case 1: // Square
			dc.DrawRectangle(1, 1, S-2, S-2)
		case 2: // Diamond
			dc.MoveTo(S/2, 0)
			dc.LineTo(S, S/2)
			dc.LineTo(S/2, S)
			dc.LineTo(0, S/2)
			dc.ClosePath()
		case 3: // Pentagon
			dc.DrawRegularPolygon(5, S/2, S/2, S/2-1, 0)
		case 4: // Hexagon
			dc.DrawRegularPolygon(6, S/2, S/2, S/2-1, 0)
		}
		dc.FillPreserve()

		// Black border for contrast
		dc.SetColor(color.Black)
		dc.SetLineWidth(1)
		dc.Stroke()

		// Draw the tier number in the center
		dc.SetFontFace(basicfont.Face7x13)
		dc.SetColor(color.White)
		
		text := fmt.Sprintf("%d", i+1)
		// For a 16x16 icon, 7x13 font needs careful anchoring. 
		// Y is adjusted slightly to optically center the basicfont glyphs.
		dc.DrawStringAnchored(text, S/2, S/2-1, 0.5, 0.5)

		outPath := filepath.Join(outputDir, fmt.Sprintf("tier_%d.png", i+1))
		if err := dc.SavePNG(outPath); err != nil {
			return err
		}
	}
	return nil
}
