package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"

	xdraw "golang.org/x/image/draw"
)

const (
	targetColumns = 14
	targetRows    = 12
	targetCell    = 96
	targetInset   = 4
)

type atlasGrid struct {
	columns int
	rows    int
}

type point struct {
	x int
	y int
}

var catalogAtlases = map[string]atlasGrid{
	"abyss_atlas_artifacts.png":  {columns: 14, rows: 11},
	"abyss_atlas_auras.png":      {columns: 12, rows: 12},
	"abyss_atlas_banners.png":    {columns: 13, rows: 10},
	"abyss_atlas_bosses.png":     {columns: 14, rows: 12},
	"abyss_atlas_charms.png":     {columns: 14, rows: 11},
	"abyss_atlas_companions.png": {columns: 14, rows: 12},
	"abyss_atlas_creatures.png":  {columns: 14, rows: 12},
	"abyss_atlas_emblems.png":    {columns: 12, rows: 12},
	"abyss_atlas_items.png":      {columns: 14, rows: 12},
	"abyss_atlas_mounts.png":     {columns: 12, rows: 12},
	"abyss_atlas_offhands.png":   {columns: 14, rows: 12},
	"abyss_atlas_pets.png":       {columns: 14, rows: 12},
	"abyss_atlas_ranged.png":     {columns: 14, rows: 12},
	"abyss_atlas_relics.png":     {columns: 13, rows: 12},
	"abyss_atlas_skills.png":     {columns: 14, rows: 12},
	"abyss_atlas_souls.png":      {columns: 14, rows: 12},
	"abyss_atlas_totems.png":     {columns: 14, rows: 10},
}

func main() {
	root := filepath.Join("internal", "bot", "webassets")
	for name, sourceGrid := range catalogAtlases {
		path := filepath.Join(root, name)
		if err := normalizeAtlas(path, sourceGrid); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", name, err)
			os.Exit(1)
		}
	}
}

func normalizeAtlas(path string, sourceGrid atlasGrid) error {
	source, err := loadAtlasSource(path)
	if err != nil {
		return err
	}

	targetWidth := targetColumns * targetCell
	targetHeight := targetRows * targetCell
	if source.Bounds().Dx() == targetWidth && source.Bounds().Dy() == targetHeight {
		return nil
	}

	target := image.NewNRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	for index := range sourceGrid.columns * sourceGrid.rows {
		sourceColumn := index % sourceGrid.columns
		sourceRow := index / sourceGrid.columns
		sourceRect := proportionalCell(source.Bounds(), sourceGrid, sourceColumn, sourceRow)
		sourceRect = trimSourceCell(sourceRect)
		cell := transparentCell(source, sourceRect)

		targetColumn := index % targetColumns
		targetRow := index / targetColumns
		targetRect := fittedCell(cell.Bounds(), targetColumn, targetRow)
		xdraw.CatmullRom.Scale(target, targetRect, cell, cell.Bounds(), draw.Over, nil)
	}

	cleanNormalizedAtlas(target)
	return writeAtlas(path, target)
}

// Generated source sheets do not always keep their artwork perfectly inside
// the requested grid. A narrow crop removes the neighbouring-cell slivers
// without touching the centered item silhouette.
func trimSourceCell(cell image.Rectangle) image.Rectangle {
	insetX := max(2, cell.Dx()/24)
	insetY := max(2, cell.Dy()/28)
	return image.Rect(cell.Min.X+insetX, cell.Min.Y+insetY, cell.Max.X-insetX, cell.Max.Y-insetY)
}

func loadAtlasSource(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	current, err := png.Decode(file)
	closeErr := file.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if current.Bounds().Dx() != targetColumns*targetCell || current.Bounds().Dy() != targetRows*targetCell {
		return current, nil
	}

	command := exec.Command("git", "show", "HEAD:"+filepath.ToSlash(path)) // #nosec G204 -- path comes from the fixed catalogAtlases map
	originalBytes, err := command.Output()
	if err != nil {
		return current, nil
	}
	original, err := png.Decode(bytes.NewReader(originalBytes))
	if err != nil {
		return nil, err
	}
	return original, nil
}

func writeAtlas(path string, target image.Image) error {
	output, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(output, target); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

func cleanNormalizedAtlas(atlas *image.NRGBA) {
	for y := atlas.Bounds().Min.Y; y < atlas.Bounds().Max.Y; y++ {
		for x := atlas.Bounds().Min.X; x < atlas.Bounds().Max.X; x++ {
			pixel := atlas.NRGBAAt(x, y)
			cellX := x % targetCell
			cellY := y % targetCell
			atCellEdge := cellX < 2 || cellX >= targetCell-2 || cellY < 2 || cellY >= targetCell-2
			if atCellEdge {
				pixel.A = 0
				atlas.SetNRGBA(x, y, pixel)
			}
		}
	}
	for row := range targetRows {
		for column := range targetColumns {
			removeEdgeFragments(atlas, image.Rect(
				column*targetCell,
				row*targetCell,
				(column+1)*targetCell,
				(row+1)*targetCell,
			))
		}
	}
}

func removeEdgeFragments(atlas *image.NRGBA, cell image.Rectangle) {
	seen := make(map[point]bool, targetCell*targetCell)
	for y := cell.Min.Y; y < cell.Max.Y; y++ {
		for x := cell.Min.X; x < cell.Max.X; x++ {
			start := point{x: x, y: y}
			if seen[start] || atlas.NRGBAAt(x, y).A == 0 {
				continue
			}
			component := make([]point, 0, 64)
			queue := []point{start}
			seen[start] = true
			minX, maxX := x, x
			for len(queue) > 0 {
				current := queue[0]
				queue = queue[1:]
				component = append(component, current)
				minX = min(minX, current.x)
				maxX = max(maxX, current.x)
				for offsetY := -1; offsetY <= 1; offsetY++ {
					for offsetX := -1; offsetX <= 1; offsetX++ {
						neighbor := point{x: current.x + offsetX, y: current.y + offsetY}
						if (offsetX == 0 && offsetY == 0) || !neighbor.in(cell) || seen[neighbor] {
							continue
						}
						if atlas.NRGBAAt(neighbor.x, neighbor.y).A > 0 {
							seen[neighbor] = true
							queue = append(queue, neighbor)
						}
					}
				}
			}
			detachedSide := maxX < cell.Min.X+30 || minX >= cell.Max.X-30
			if !detachedSide {
				continue
			}
			for _, pixel := range component {
				value := atlas.NRGBAAt(pixel.x, pixel.y)
				value.A = 0
				atlas.SetNRGBA(pixel.x, pixel.y, value)
			}
		}
	}
}

func proportionalCell(bounds image.Rectangle, grid atlasGrid, column, row int) image.Rectangle {
	width := bounds.Dx()
	height := bounds.Dy()
	return image.Rect(
		bounds.Min.X+column*width/grid.columns,
		bounds.Min.Y+row*height/grid.rows,
		bounds.Min.X+(column+1)*width/grid.columns,
		bounds.Min.Y+(row+1)*height/grid.rows,
	)
}

func fittedCell(source image.Rectangle, column, row int) image.Rectangle {
	available := targetCell - 2*targetInset
	scale := min(
		float64(available)/float64(source.Dx()),
		float64(available)/float64(source.Dy()),
	)
	width := max(1, int(float64(source.Dx())*scale))
	height := max(1, int(float64(source.Dy())*scale))
	x := column*targetCell + (targetCell-width)/2
	y := row*targetCell + (targetCell-height)/2
	return image.Rect(x, y, x+width, y+height)
}

func transparentCell(source image.Image, sourceRect image.Rectangle) *image.NRGBA {
	cell := image.NewNRGBA(image.Rect(0, 0, sourceRect.Dx(), sourceRect.Dy()))
	draw.Draw(cell, cell.Bounds(), source, sourceRect.Min, draw.Src)

	queue := make([]point, 0, 2*(cell.Bounds().Dx()+cell.Bounds().Dy()))
	for x := cell.Bounds().Min.X; x < cell.Bounds().Max.X; x++ {
		queue = append(queue, point{x: x, y: cell.Bounds().Min.Y}, point{x: x, y: cell.Bounds().Max.Y - 1})
	}
	for y := cell.Bounds().Min.Y; y < cell.Bounds().Max.Y; y++ {
		queue = append(queue, point{x: cell.Bounds().Min.X, y: y}, point{x: cell.Bounds().Max.X - 1, y: y})
	}

	seen := make([]bool, cell.Bounds().Dx()*cell.Bounds().Dy())
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if !current.in(cell.Bounds()) {
			continue
		}
		index := current.y*cell.Bounds().Dx() + current.x
		if seen[index] || !isSheetBackground(cell.NRGBAAt(current.x, current.y)) {
			continue
		}
		seen[index] = true
		pixel := cell.NRGBAAt(current.x, current.y)
		pixel.A = 0
		cell.SetNRGBA(current.x, current.y, pixel)
		queue = append(queue,
			point{x: current.x - 1, y: current.y},
			point{x: current.x + 1, y: current.y},
			point{x: current.x, y: current.y - 1},
			point{x: current.x, y: current.y + 1},
		)
	}
	return cell
}

func (p point) in(bounds image.Rectangle) bool {
	return p.x >= bounds.Min.X && p.x < bounds.Max.X && p.y >= bounds.Min.Y && p.y < bounds.Max.Y
}

func isSheetBackground(pixel color.NRGBA) bool {
	return pixel.A == 0 || (pixel.R <= 42 && pixel.G <= 55 && pixel.B <= 70)
}
