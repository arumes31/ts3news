package treeart

import (
	"hash/fnv"
	"image"
	"image/color"

	"ts3news/internal/content"
)

type artPalette struct {
	shadow color.NRGBA
	dark   color.NRGBA
	main   color.NRGBA
	bright color.NRGBA
	pale   color.NRGBA
}

var disciplinePalettes = [...]artPalette{
	{rgba(38, 8, 15), rgba(105, 20, 24), rgba(205, 52, 35), rgba(255, 142, 40), rgba(255, 229, 151)},
	{rgba(7, 32, 23), rgba(13, 91, 48), rgba(39, 176, 77), rgba(138, 231, 89), rgba(235, 255, 193)},
	{rgba(18, 9, 38), rgba(55, 24, 92), rgba(111, 49, 160), rgba(191, 107, 231), rgba(239, 211, 255)},
	{rgba(5, 26, 51), rgba(17, 71, 126), rgba(30, 142, 211), rgba(81, 222, 242), rgba(220, 251, 255)},
	{rgba(39, 24, 3), rgba(122, 70, 5), rgba(219, 143, 14), rgba(255, 211, 61), rgba(255, 247, 188)},
	{rgba(20, 4, 38), rgba(48, 13, 91), rgba(111, 24, 159), rgba(235, 41, 188), rgba(134, 224, 255)},
}

func rgba(red, green, blue uint8) color.NRGBA {
	return color.NRGBA{R: red, G: green, B: blue, A: 255}
}

func drawNode(atlas *image.NRGBA, node content.TreeNode) {
	bounds := CellBounds(node.ArtCell)
	palette := disciplinePalettes[node.Sector]
	seed := artSeed(node)
	drawFrame(atlas, bounds, node.Type, palette)
	switch node.Sector {
	case 0:
		drawWar(atlas, bounds, seed, palette)
	case 1:
		drawVitality(atlas, bounds, seed, palette)
	case 2:
		drawShadow(atlas, bounds, seed, palette)
	case 3:
		drawArcane(atlas, bounds, seed, palette)
	case 4:
		drawFortune(atlas, bounds, seed, palette)
	default:
		drawVoid(atlas, bounds, seed, palette)
	}
	drawIdentitySigil(atlas, bounds, uint64(node.ID), palette)
}

func artSeed(node content.TreeNode) uint64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(node.ArtSignature))
	return hash.Sum64()
}

func drawFrame(atlas *image.NRGBA, bounds image.Rectangle, kind string, palette artPalette) {
	switch kind {
	case content.TreeNodeNotable, content.TreeNodeBridge:
		for _, point := range [][2]int{{2, 2}, {3, 2}, {12, 2}, {13, 2}, {2, 12}, {2, 13}, {13, 12}, {13, 13}} {
			setPixel(atlas, bounds, point[0], point[1], palette.dark)
		}
	case content.TreeNodeKeystone:
		line(atlas, bounds, 4, 1, 11, 1, palette.bright)
		line(atlas, bounds, 1, 4, 1, 11, palette.bright)
		line(atlas, bounds, 14, 4, 14, 11, palette.bright)
		line(atlas, bounds, 4, 14, 11, 14, palette.bright)
	case content.TreeNodeAura:
		for _, point := range [][2]int{{7, 0}, {8, 0}, {7, 15}, {8, 15}, {0, 7}, {0, 8}, {15, 7}, {15, 8}} {
			setPixel(atlas, bounds, point[0], point[1], palette.pale)
		}
	case content.TreeNodeSocket:
		line(atlas, bounds, 7, 1, 14, 8, palette.bright)
		line(atlas, bounds, 14, 8, 7, 14, palette.bright)
		line(atlas, bounds, 7, 14, 1, 8, palette.bright)
		line(atlas, bounds, 1, 8, 7, 1, palette.bright)
	}
}

func drawWar(atlas *image.NRGBA, bounds image.Rectangle, seed uint64, palette artPalette) {
	drawMotif(atlas, bounds, seed, palette, [...]motifKind{
		motifBlade, motifCrossedBlades, motifShield, motifBurst,
		motifCrown, motifFlame, motifTrident, motifBolt,
	})
}

func drawVitality(atlas *image.NRGBA, bounds image.Rectangle, seed uint64, palette artPalette) {
	drawMotif(atlas, bounds, seed, palette, [...]motifKind{
		motifHeart, motifShield, motifLeaf, motifBurst,
		motifRing, motifDiamond, motifCrown, motifHourglass,
	})
}

func drawShadow(atlas *image.NRGBA, bounds image.Rectangle, seed uint64, palette artPalette) {
	drawMotif(atlas, bounds, seed, palette, [...]motifKind{
		motifEye, motifCrescent, motifBlade, motifRing,
		motifBurst, motifTrident, motifFlame, motifHourglass,
	})
}

func drawArcane(atlas *image.NRGBA, bounds image.Rectangle, seed uint64, palette artPalette) {
	drawMotif(atlas, bounds, seed, palette, [...]motifKind{
		motifDiamond, motifBurst, motifRing, motifBolt,
		motifHourglass, motifEye, motifTrident, motifFlame,
	})
}

func drawFortune(atlas *image.NRGBA, bounds image.Rectangle, seed uint64, palette artPalette) {
	drawMotif(atlas, bounds, seed, palette, [...]motifKind{
		motifRing, motifKey, motifDice, motifCrown,
		motifBurst, motifShield, motifDiamond, motifHourglass,
	})
}

func drawVoid(atlas *image.NRGBA, bounds image.Rectangle, seed uint64, palette artPalette) {
	drawMotif(atlas, bounds, seed, palette, [...]motifKind{
		motifEye, motifRing, motifCrescent, motifBurst,
		motifFlame, motifDiamond, motifTrident, motifHourglass,
	})
}

func drawIdentitySigil(atlas *image.NRGBA, bounds image.Rectangle, id uint64, palette artPalette) {
	positions := [...]image.Point{
		{3, 1}, {5, 1}, {7, 1}, {9, 1}, {11, 1}, {14, 3}, {14, 6}, {14, 10},
		{12, 14}, {10, 14}, {8, 14}, {6, 14}, {4, 14}, {1, 12}, {1, 8}, {1, 4},
	}
	colors := [...]color.NRGBA{palette.shadow, palette.dark, palette.bright, palette.pale}
	for index, point := range positions {
		state := (id >> (index * 2)) & 3
		setPixel(atlas, bounds, point.X, point.Y, colors[state])
	}
}
