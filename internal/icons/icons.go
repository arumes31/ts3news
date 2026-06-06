// Package icons generates small PNG icons for the XP level tiers. Each icon shows
// the tier number on a colour that escalates with rank (bronze -> silver -> gold
// -> prestige), plus small "prestige" pips for the highest tiers. No external
// font dependency is used — digits are drawn from a built-in 5x7 bitmap font.
package icons

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// 5x7 bitmap glyphs for the digits used in tier numbers.
var glyphs = map[rune][]string{
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11111", "00010", "00100", "00010", "00001", "10001", "01110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "11110", "00001", "00001", "10001", "01110"},
	'6': {"00110", "01000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00010", "01100"},
}

const (
	glyphW = 5
	glyphH = 7
)

// Icon renders a square PNG icon (sizePx x sizePx) showing the given number
// (e.g. the level). The background colour and prestige pips are driven by tier
// (1..maxTier), so higher ranks look warmer/prestige. Returns the PNG bytes.
func Icon(number, tier, maxTier, sizePx int) ([]byte, error) {
	if tier < 1 {
		tier = 1
	}
	if maxTier < 1 {
		maxTier = 1
	}
	img := image.NewRGBA(image.Rect(0, 0, sizePx, sizePx))

	bg := tierColor(tier, maxTier)
	fg := contrast(bg)
	border := darken(bg)

	// Fill background with a rounded-ish border.
	for y := 0; y < sizePx; y++ {
		for x := 0; x < sizePx; x++ {
			c := bg
			if x == 0 || y == 0 || x == sizePx-1 || y == sizePx-1 {
				c = border
			}
			img.Set(x, y, c)
		}
	}

	drawNumber(img, number, fg, sizePx)
	drawPrestigePips(img, tier, maxTier, fg, sizePx)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// drawNumber renders the tier number centred, scaled to fit the icon.
func drawNumber(img *image.RGBA, n int, fg color.RGBA, size int) {
	digits := itoa(n)
	scale := (size - 8) / glyphH // scale by height, leave margin
	if scale < 1 {
		scale = 1
	}
	totalW := len(digits)*(glyphW+1)*scale - scale
	startX := (size - totalW) / 2
	startY := (size - glyphH*scale) / 2

	x := startX
	for _, d := range digits {
		g := glyphs[d]
		for gy := 0; gy < glyphH; gy++ {
			row := g[gy]
			for gx := 0; gx < glyphW; gx++ {
				if row[gx] == '1' {
					fillRect(img, x+gx*scale, startY+gy*scale, scale, scale, fg)
				}
			}
		}
		x += (glyphW + 1) * scale
	}
}

// drawPrestigePips adds up to 5 small pips along the bottom for top-tier ranks.
func drawPrestigePips(img *image.RGBA, tier, maxTier int, fg color.RGBA, size int) {
	// Last fifth of tiers earn prestige pips (1..5).
	threshold := maxTier * 4 / 5
	if tier <= threshold {
		return
	}
	pips := tier - threshold
	if pips > 5 {
		pips = 5
	}
	pip := size / 12
	if pip < 1 {
		pip = 1
	}
	gap := pip
	totalW := pips*pip + (pips-1)*gap
	x := (size - totalW) / 2
	y := size - pip - 2
	for i := 0; i < pips; i++ {
		fillRect(img, x, y, pip, pip, fg)
		x += pip + gap
	}
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	b := img.Bounds()
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			if xx >= b.Min.X && xx < b.Max.X && yy >= b.Min.Y && yy < b.Max.Y {
				img.SetRGBA(xx, yy, c)
			}
		}
	}
}

// tierColor returns an escalating colour: low tiers brown/bronze, mid grey/silver,
// high gold, top prestige purple.
func tierColor(tier, maxTier int) color.RGBA {
	f := float64(tier-1) / float64(maxOf(maxTier-1, 1))
	switch {
	case f < 0.25: // bronze
		return lerp(color.RGBA{120, 72, 40, 255}, color.RGBA{170, 110, 60, 255}, f/0.25)
	case f < 0.5: // silver
		return lerp(color.RGBA{120, 120, 130, 255}, color.RGBA{180, 185, 195, 255}, (f-0.25)/0.25)
	case f < 0.75: // gold
		return lerp(color.RGBA{200, 160, 40, 255}, color.RGBA{235, 205, 70, 255}, (f-0.5)/0.25)
	default: // prestige
		return lerp(color.RGBA{120, 50, 170, 255}, color.RGBA{190, 60, 90, 255}, (f-0.75)/0.25)
	}
}

func lerp(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	mix := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), 255}
}

func darken(c color.RGBA) color.RGBA {
	// #nosec G115 -- Calculation results in max 153, fits safely in uint8
	d := func(x uint8) uint8 { return uint8(int(x) * 6 / 10) }
	return color.RGBA{d(c.R), d(c.G), d(c.B), 255}
}

// contrast returns black or white depending on background luminance.
func contrast(c color.RGBA) color.RGBA {
	lum := (int(c.R)*299 + int(c.G)*587 + int(c.B)*114) / 1000
	if lum > 140 {
		return color.RGBA{20, 20, 20, 255}
	}
	return color.RGBA{245, 245, 245, 255}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func maxOf(a, b int) int {
	if a > b {
		return a
	}
	return b
}
