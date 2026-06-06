// Package icons generates small PNG icons for the XP level tiers.
package icons

import (
	"bytes"
	"fmt"
	"image/color"
	"math"

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

// Icon renders a 16x16 PNG icon with a shape and color that evolves with tier.
func Icon(number, tier, maxTier, sizePx int) ([]byte, error) {
	S := float64(sizePx)
	dc := gg.NewContext(sizePx, sizePx)

	// Determine color and shape based on tier
	// tier is 1..NumTiers
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

	// Darker border
	dc.SetColor(color.RGBA{0, 0, 0, 255})
	dc.SetLineWidth(1)
	dc.Stroke()

	// Draw the number (sub-rank)
	dc.SetFontFace(basicfont.Face7x13)
	// Contrast text color
	if isLight(baseColor) {
		dc.SetColor(color.Black)
	} else {
		dc.SetColor(color.White)
	}
	
	text := fmt.Sprintf("%d", number)
	dc.DrawStringAnchored(text, S/2, S/2-1, 0.5, 0.5)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func isLight(c color.RGBA) bool {
	lum := (int(c.R)*299 + int(c.G)*587 + int(c.B)*114) / 1000
	return lum > 140
}
