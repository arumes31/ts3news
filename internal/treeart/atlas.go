// Package treeart renders the deterministic pixel atlases used by the Abyss
// Skill Web. Keeping generation outside the HTTP package makes the checked-in
// assets reproducible without adding image work to server startup.
//
//go:generate go run ../../cmd/generate-abyss-tree-atlases -out ../bot/webassets
package treeart

import (
	"fmt"
	"image"
	"image/color"

	"ts3news/internal/content"
)

const (
	CellSize = content.AbyssTreeArtCellSize
	Columns  = content.AbyssTreeArtColumns
	Rows     = content.AbyssTreeArtRows
	Width    = CellSize * Columns
	Height   = CellSize * Rows
)

// Generate builds one fixed-size transparent atlas per Skill Web discipline.
// Fixed dimensions keep browser sprite coordinates stable as the catalog grows.
func Generate(tree *content.AbyssTreeData) (map[string]*image.NRGBA, error) {
	if tree == nil {
		return nil, fmt.Errorf("generate tree atlases: tree is nil")
	}
	atlases := make(map[string]*image.NRGBA, len(content.AbyssTreeArtSheets))
	for _, sheet := range content.AbyssTreeArtSheets {
		atlases[sheet] = image.NewNRGBA(image.Rect(0, 0, Width, Height))
	}
	for index := range tree.Nodes {
		node := tree.Nodes[index]
		atlas, ok := atlases[node.ArtSheet]
		if !ok {
			return nil, fmt.Errorf("generate tree atlases: node %d uses unknown sheet %q", node.ID, node.ArtSheet)
		}
		if node.ArtCell < 0 || node.ArtCell >= Columns*Rows {
			return nil, fmt.Errorf("generate tree atlases: node %d cell %d exceeds sheet capacity", node.ID, node.ArtCell)
		}
		drawNode(atlas, node)
	}
	return atlases, nil
}

// CellBounds returns a node-sized rectangle within a sheet.
func CellBounds(cell int) image.Rectangle {
	x := cell % Columns * CellSize
	y := cell / Columns * CellSize
	return image.Rect(x, y, x+CellSize, y+CellSize)
}

func setPixel(canvas *image.NRGBA, bounds image.Rectangle, x, y int, value color.NRGBA) {
	if x < 0 || y < 0 || x >= CellSize || y >= CellSize {
		return
	}
	canvas.SetNRGBA(bounds.Min.X+x, bounds.Min.Y+y, value)
}

func line(canvas *image.NRGBA, bounds image.Rectangle, x0, y0, x1, y1 int, value color.NRGBA) {
	dx, sx := abs(x1-x0), 1
	if x0 > x1 {
		sx = -1
	}
	dy, sy := -abs(y1-y0), 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		setPixel(canvas, bounds, x0, y0, value)
		if x0 == x1 && y0 == y1 {
			return
		}
		twice := 2 * err
		if twice >= dy {
			err += dy
			x0 += sx
		}
		if twice <= dx {
			err += dx
			y0 += sy
		}
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
