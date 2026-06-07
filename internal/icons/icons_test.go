package icons

import (
	"bytes"
	"image/color"
	"image/png"
	"testing"
)

func TestTierIconValidPNG(t *testing.T) {
	for _, tier := range []int{1, 5, 13, 20, 25} {
		data, err := Icon(tier, tier, 25, 32)
		if err != nil {
			t.Fatalf("tier %d: %v", tier, err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("tier %d: decode: %v", tier, err)
		}
		if b := img.Bounds(); b.Dx() != 32 || b.Dy() != 32 {
			t.Errorf("tier %d: size %dx%d, want 32x32", tier, b.Dx(), b.Dy())
		}
	}
}

func TestTierIconDeterministic(t *testing.T) {
	a, _ := Icon(7, 1, 25, 32)
	b, _ := Icon(7, 1, 25, 32)
	if !bytes.Equal(a, b) {
		t.Error("icon generation is not deterministic")
	}
	// Different numbers should look different (so icon ids differ on the server).
	c, _ := Icon(8, 1, 25, 32)
	if bytes.Equal(a, c) {
		t.Error("different tiers produced identical icons")
	}
}

func TestIsLight(t *testing.T) {
	if !isLight(color.RGBA{255, 255, 255, 255}) {
		t.Error("white should be light")
	}
	if isLight(color.RGBA{0, 0, 0, 255}) {
		t.Error("black should not be light")
	}
}
