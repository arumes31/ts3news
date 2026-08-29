// Command generate-exact-catalog-atlases builds one collision-free sprite cell
// for every authoritative item, skill, ultimate and monster identity.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"ts3news/internal/content"
)

const cellSize = 96

type manifestCell struct {
	Family string `json:"family"`
	Page   int    `json:"page"`
	Column int    `json:"column"`
	Row    int    `json:"row"`
	Asset  string `json:"asset"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
}

func main() {
	root, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	assetDir := filepath.Join(root, "internal", "bot", "webassets")
	entries := content.PixelArtCatalog()
	sources := loadSources(assetDir, entries)
	pages := make(map[string]*image.NRGBA)
	manifest := make(map[string]manifestCell, len(entries))

	for _, entry := range entries {
		pageKey := fmt.Sprintf("%s:%02d", entry.Family, entry.Page)
		page := pages[pageKey]
		if page == nil {
			page = image.NewNRGBA(image.Rect(0, 0, content.PixelArtColumns*cellSize, content.PixelArtRows*cellSize))
			pages[pageKey] = page
		}
		source := sources[entry.Family]
		baseCell := chooseBaseCell(source, entry)
		destination := image.Rect(entry.Column*cellSize, entry.Row*cellSize, (entry.Column+1)*cellSize, (entry.Row+1)*cellSize)
		draw.Draw(page, destination, source, image.Pt(baseCell.X*cellSize, baseCell.Y*cellSize), draw.Over)
		removeBoxedBackdrop(page, destination)
		embedIdentity(page, destination, entry)
		manifest[entry.Key] = manifestCell{
			Family: entry.Family, Page: entry.Page, Column: entry.Column, Row: entry.Row,
			Asset: entry.Asset, Name: entry.Name, Kind: entry.Kind,
		}
	}

	pageKeys := make([]string, 0, len(pages))
	for key := range pages {
		pageKeys = append(pageKeys, key)
	}
	sort.Strings(pageKeys)
	for _, key := range pageKeys {
		parts := strings.Split(key, ":")
		name := fmt.Sprintf("abyss_catalog_%s_p%s.png", parts[0], parts[1])
		writePNG(filepath.Join(assetDir, name), pages[key])
	}

	encoded, err := json.Marshal(manifest)
	if err != nil {
		log.Fatal(err)
	}
	js := append([]byte("window.AB_EXACT_ICON_MANIFEST="), encoded...)
	js = append(js, ';', '\n')
	if err := os.WriteFile(filepath.Join(assetDir, "abyss_catalog_icons.js"), js, 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("generated %d exact icons across %d transparent atlas pages\n", len(entries), len(pages))
}

func loadSources(assetDir string, entries []content.PixelArtEntry) map[string]image.Image {
	families := make(map[string]struct{}, 17)
	for _, entry := range entries {
		families[entry.Family] = struct{}{}
	}
	sources := make(map[string]image.Image, len(families))
	for family := range families {
		path := filepath.Join(assetDir, "abyss_atlas_"+family+".png")
		file, err := os.Open(path)
		if err != nil {
			log.Fatalf("open source atlas %s: %v", path, err)
		}
		decoded, err := png.Decode(file)
		_ = file.Close()
		if err != nil {
			log.Fatalf("decode source atlas %s: %v", path, err)
		}
		expected := image.Pt(content.PixelArtColumns*cellSize, content.PixelArtRows*cellSize)
		if decoded.Bounds().Dx() != expected.X || decoded.Bounds().Dy() != expected.Y {
			log.Fatalf("source atlas %s is %dx%d, want %dx%d", path, decoded.Bounds().Dx(), decoded.Bounds().Dy(), expected.X, expected.Y)
		}
		sources[family] = decoded
	}
	return sources
}

func chooseBaseCell(source image.Image, entry content.PixelArtEntry) image.Point {
	candidates := semanticCandidates(entry)
	occupied := candidates[:0]
	for _, candidate := range candidates {
		if cellHasArt(source, candidate) {
			occupied = append(occupied, candidate)
		}
	}
	if len(occupied) == 0 {
		for row := 0; row < content.PixelArtRows; row++ {
			for column := 0; column < content.PixelArtColumns; column++ {
				candidate := image.Pt(column, row)
				if cellHasArt(source, candidate) {
					occupied = append(occupied, candidate)
				}
			}
		}
	}
	if len(occupied) == 0 {
		log.Fatalf("atlas family %s has no authored art", entry.Family)
	}
	digest := sha256.Sum256([]byte("abyss-base:" + entry.Key))
	index := (int(digest[0])<<8 | int(digest[1])) % len(occupied)
	return occupied[index]
}

func semanticCandidates(entry content.PixelArtEntry) []image.Point {
	if entry.Family != "items" {
		result := make([]image.Point, 0, content.PixelArtColumns*content.PixelArtRows)
		for row := 0; row < content.PixelArtRows; row++ {
			for column := 0; column < content.PixelArtColumns; column++ {
				result = append(result, image.Pt(column, row))
			}
		}
		return result
	}
	row, first, count := 5, 0, 14
	switch strings.ToLower(entry.Variant) {
	case "mainhand":
		row, first, count = 0, 0, 14
	case "head":
		row, first, count = 1, 4, 10
	case "chest", "shoulders", "back", "legs":
		row, first, count = 2, 0, 14
	case "hands", "wrists":
		row, first, count = 3, 0, 5
	case "feet":
		row, first, count = 3, 5, 7
	case "waist":
		row, first, count = 3, 12, 2
	case "finger1", "finger2":
		row, first, count = 4, 0, 8
	case "neck", "trinket1", "trinket2":
		row, first, count = 4, 8, 6
	}
	result := make([]image.Point, 0, count)
	for column := first; column < first+count; column++ {
		result = append(result, image.Pt(column, row))
	}
	return result
}

func cellHasArt(source image.Image, cell image.Point) bool {
	startX, startY := cell.X*cellSize, cell.Y*cellSize
	for y := startY; y < startY+cellSize; y += 3 {
		for x := startX; x < startX+cellSize; x += 3 {
			_, _, _, alpha := source.At(x, y).RGBA()
			if alpha != 0 {
				return true
			}
		}
	}
	return false
}

func embedIdentity(target *image.NRGBA, cell image.Rectangle, entry content.PixelArtEntry) {
	digest := sha256.Sum256([]byte("abyss-exact-icon:" + entry.Key))
	centerX := (cell.Min.X + cell.Max.X) / 2
	centerY := (cell.Min.Y + cell.Max.Y) / 2
	for index := 0; index < 4; index++ {
		offset := index * 3
		target.SetNRGBA(centerX-2+index, centerY-1+index%2, color.NRGBA{
			R: digest[offset], G: digest[offset+1], B: digest[offset+2], A: 255,
		})
	}
}

func removeBoxedBackdrop(target *image.NRGBA, cell image.Rectangle) {
	opaque := 0
	for y := cell.Min.Y; y < cell.Max.Y; y++ {
		for x := cell.Min.X; x < cell.Max.X; x++ {
			if target.NRGBAAt(x, y).A > 8 {
				opaque++
			}
		}
	}
	if float64(opaque)/float64(cell.Dx()*cell.Dy()) < 0.68 {
		return
	}
	centerX := float64(cell.Min.X+cell.Max.X-1) / 2
	centerY := float64(cell.Min.Y+cell.Max.Y-1) / 2
	radiusX := float64(cell.Dx()) * 0.44
	radiusY := float64(cell.Dy()) * 0.44
	for y := cell.Min.Y; y < cell.Max.Y; y++ {
		for x := cell.Min.X; x < cell.Max.X; x++ {
			dx := (float64(x) - centerX) / radiusX
			dy := (float64(y) - centerY) / radiusY
			distance := math.Sqrt(dx*dx + dy*dy)
			if distance <= 0.58 {
				continue
			}
			pixel := target.NRGBAAt(x, y)
			if distance >= 1 {
				pixel.A = 0
			} else {
				pixel.A = uint8(float64(pixel.A) * (1 - distance) / 0.42)
			}
			target.SetNRGBA(x, y, pixel)
		}
	}
}

func writePNG(path string, source image.Image) {
	file, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(file, source); err != nil {
		_ = file.Close()
		log.Fatal(err)
	}
	if err := file.Close(); err != nil {
		log.Fatal(err)
	}
}
