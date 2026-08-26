package treeart

import (
	"crypto/sha256"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"ts3news/internal/content"
)

func TestGeneratedAtlasesCoverEveryNodeWithUniquePixels(t *testing.T) {
	t.Parallel()

	tree := content.AbyssTree()
	atlases, err := Generate(tree)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	seen := make(map[[sha256.Size]byte]int, len(tree.Nodes))
	for _, node := range tree.Nodes {
		atlas := atlases[node.ArtSheet]
		digest, opaque := cellDigest(atlas, node.ArtCell)
		if !opaque {
			t.Errorf("node %d has an empty atlas cell", node.ID)
		}
		if previous, duplicate := seen[digest]; duplicate {
			t.Errorf("nodes %d and %d render identical pixel art", previous, node.ID)
		}
		seen[digest] = node.ID
	}
	if len(seen) != len(tree.Nodes) {
		t.Fatalf("unique rendered cells = %d, want %d", len(seen), len(tree.Nodes))
	}
}

func TestCheckedInAtlasesMatchGenerator(t *testing.T) {
	t.Parallel()

	generated, err := Generate(content.AbyssTree())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, sheet := range content.AbyssTreeArtSheets {
		sheet := sheet
		t.Run(sheet, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join("..", "bot", "webassets", "abyss_atlas_"+sheet+".png")
			file, err := os.Open(path) // #nosec G304 - fixed repository test fixture
			if err != nil {
				t.Fatalf("open generated atlas: %v", err)
			}
			defer func() { _ = file.Close() }()
			checkedIn, err := png.Decode(file)
			if err != nil {
				t.Fatalf("decode generated atlas: %v", err)
			}
			if !samePixels(checkedIn, generated[sheet]) {
				t.Fatalf("%s is stale; run go generate ./internal/treeart", path)
			}
		})
	}
}

func cellDigest(atlas image.Image, cell int) ([sha256.Size]byte, bool) {
	bounds := CellBounds(cell)
	pixels := make([]byte, 0, CellSize*CellSize*4)
	opaque := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			red, green, blue, alpha := atlas.At(x, y).RGBA()
			pixels = append(pixels, byte(red>>8), byte(green>>8), byte(blue>>8), byte(alpha>>8))
			opaque = opaque || alpha != 0
		}
	}
	return sha256.Sum256(pixels), opaque
}

func samePixels(left, right image.Image) bool {
	if left.Bounds() != right.Bounds() {
		return false
	}
	for y := left.Bounds().Min.Y; y < left.Bounds().Max.Y; y++ {
		for x := left.Bounds().Min.X; x < left.Bounds().Max.X; x++ {
			leftPixel := color.NRGBAModel.Convert(left.At(x, y)).(color.NRGBA)
			rightPixel := color.NRGBAModel.Convert(right.At(x, y)).(color.NRGBA)
			if leftPixel != rightPixel {
				return false
			}
		}
	}
	return true
}
