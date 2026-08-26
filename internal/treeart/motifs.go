package treeart

import "image"

type motifKind uint8

const (
	motifBlade motifKind = iota
	motifCrossedBlades
	motifShield
	motifBurst
	motifCrown
	motifFlame
	motifTrident
	motifBolt
	motifHeart
	motifLeaf
	motifRing
	motifDiamond
	motifHourglass
	motifEye
	motifCrescent
	motifKey
	motifDice
)

func drawMotif(atlas *image.NRGBA, bounds image.Rectangle, seed uint64, palette artPalette, choices [8]motifKind) {
	kind := choices[seed%uint64(len(choices))]
	switch kind {
	case motifBlade:
		drawBlade(atlas, bounds, palette)
	case motifCrossedBlades:
		drawCrossedBlades(atlas, bounds, palette)
	case motifShield:
		drawShield(atlas, bounds, palette)
	case motifBurst:
		drawBurst(atlas, bounds, palette)
	case motifCrown:
		drawCrown(atlas, bounds, palette)
	case motifFlame:
		drawFlame(atlas, bounds, palette)
	case motifTrident:
		drawTrident(atlas, bounds, palette)
	case motifBolt:
		drawBolt(atlas, bounds, palette)
	case motifHeart:
		drawHeart(atlas, bounds, palette)
	case motifLeaf:
		drawLeaf(atlas, bounds, palette)
	case motifRing:
		drawRing(atlas, bounds, palette)
	case motifDiamond:
		drawDiamond(atlas, bounds, palette)
	case motifHourglass:
		drawHourglass(atlas, bounds, palette)
	case motifEye:
		drawEye(atlas, bounds, palette)
	case motifCrescent:
		drawCrescent(atlas, bounds, palette)
	case motifKey:
		drawKey(atlas, bounds, palette)
	case motifDice:
		drawDice(atlas, bounds, palette)
	}
}

func drawBlade(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 4, 12, 11, 3, palette.pale)
	line(atlas, bounds, 5, 13, 12, 4, palette.main)
	line(atlas, bounds, 3, 10, 8, 14, palette.bright)
}

func drawCrossedBlades(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 3, 3, 12, 12, palette.main)
	line(atlas, bounds, 12, 3, 3, 12, palette.pale)
	line(atlas, bounds, 4, 10, 6, 12, palette.bright)
	line(atlas, bounds, 9, 12, 11, 10, palette.bright)
}

func drawShield(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 4, 3, 11, 3, palette.bright)
	line(atlas, bounds, 4, 3, 4, 9, palette.main)
	line(atlas, bounds, 11, 3, 11, 9, palette.main)
	line(atlas, bounds, 4, 9, 8, 13, palette.dark)
	line(atlas, bounds, 11, 9, 8, 13, palette.pale)
	line(atlas, bounds, 8, 5, 8, 10, palette.bright)
}

func drawBurst(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	for _, ray := range [][4]int{{8, 2, 8, 13}, {2, 8, 13, 8}, {4, 4, 12, 12}, {12, 4, 4, 12}} {
		line(atlas, bounds, ray[0], ray[1], ray[2], ray[3], palette.main)
	}
	setPixel(atlas, bounds, 8, 8, palette.pale)
}

func drawCrown(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 3, 11, 12, 11, palette.dark)
	line(atlas, bounds, 3, 5, 5, 9, palette.main)
	line(atlas, bounds, 5, 9, 8, 4, palette.bright)
	line(atlas, bounds, 8, 4, 10, 9, palette.pale)
	line(atlas, bounds, 10, 9, 12, 5, palette.main)
}

func drawFlame(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 8, 2, 5, 8, palette.bright)
	line(atlas, bounds, 5, 8, 7, 13, palette.dark)
	line(atlas, bounds, 7, 13, 11, 10, palette.main)
	line(atlas, bounds, 11, 10, 10, 5, palette.pale)
	line(atlas, bounds, 10, 5, 8, 8, palette.bright)
}

func drawTrident(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 8, 3, 8, 13, palette.pale)
	line(atlas, bounds, 4, 4, 8, 8, palette.main)
	line(atlas, bounds, 12, 4, 8, 8, palette.main)
	line(atlas, bounds, 4, 4, 4, 7, palette.bright)
	line(atlas, bounds, 12, 4, 12, 7, palette.bright)
}

func drawBolt(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 10, 2, 5, 8, palette.pale)
	line(atlas, bounds, 5, 8, 9, 8, palette.main)
	line(atlas, bounds, 9, 8, 6, 14, palette.bright)
}

func drawHeart(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 3, 6, 8, 13, palette.dark)
	line(atlas, bounds, 12, 6, 8, 13, palette.main)
	line(atlas, bounds, 3, 6, 5, 3, palette.bright)
	line(atlas, bounds, 5, 3, 8, 6, palette.pale)
	line(atlas, bounds, 8, 6, 10, 3, palette.pale)
	line(atlas, bounds, 10, 3, 12, 6, palette.bright)
}

func drawLeaf(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 3, 12, 12, 3, palette.pale)
	line(atlas, bounds, 5, 11, 3, 7, palette.main)
	line(atlas, bounds, 3, 7, 8, 3, palette.bright)
	line(atlas, bounds, 8, 3, 12, 3, palette.dark)
	line(atlas, bounds, 6, 9, 10, 9, palette.main)
}

func drawRing(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 5, 3, 10, 3, palette.bright)
	line(atlas, bounds, 3, 5, 3, 10, palette.main)
	line(atlas, bounds, 12, 5, 12, 10, palette.pale)
	line(atlas, bounds, 5, 12, 10, 12, palette.dark)
	setPixel(atlas, bounds, 8, 8, palette.bright)
}

func drawDiamond(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 8, 2, 13, 8, palette.pale)
	line(atlas, bounds, 13, 8, 8, 13, palette.main)
	line(atlas, bounds, 8, 13, 3, 8, palette.dark)
	line(atlas, bounds, 3, 8, 8, 2, palette.bright)
	line(atlas, bounds, 8, 4, 8, 11, palette.pale)
}

func drawHourglass(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 4, 3, 11, 3, palette.pale)
	line(atlas, bounds, 4, 12, 11, 12, palette.dark)
	line(atlas, bounds, 5, 4, 10, 11, palette.main)
	line(atlas, bounds, 10, 4, 5, 11, palette.bright)
}

func drawEye(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 3, 8, 7, 4, palette.main)
	line(atlas, bounds, 7, 4, 12, 8, palette.bright)
	line(atlas, bounds, 12, 8, 8, 11, palette.pale)
	line(atlas, bounds, 8, 11, 3, 8, palette.dark)
	setPixel(atlas, bounds, 8, 7, palette.pale)
	setPixel(atlas, bounds, 8, 8, palette.shadow)
}

func drawCrescent(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 9, 2, 5, 5, palette.pale)
	line(atlas, bounds, 5, 5, 5, 10, palette.main)
	line(atlas, bounds, 5, 10, 10, 13, palette.dark)
	line(atlas, bounds, 10, 13, 8, 10, palette.bright)
	line(atlas, bounds, 8, 10, 8, 5, palette.shadow)
}

func drawKey(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 4, 11, 11, 4, palette.pale)
	line(atlas, bounds, 5, 12, 12, 5, palette.main)
	line(atlas, bounds, 3, 11, 5, 13, palette.bright)
	line(atlas, bounds, 9, 7, 12, 10, palette.dark)
}

func drawDice(atlas *image.NRGBA, bounds image.Rectangle, palette artPalette) {
	line(atlas, bounds, 4, 3, 11, 3, palette.bright)
	line(atlas, bounds, 3, 4, 3, 11, palette.main)
	line(atlas, bounds, 12, 4, 12, 11, palette.pale)
	line(atlas, bounds, 4, 12, 11, 12, palette.dark)
	for _, point := range [][2]int{{5, 5}, {10, 5}, {8, 8}, {5, 10}, {10, 10}} {
		setPixel(atlas, bounds, point[0], point[1], palette.pale)
	}
}
