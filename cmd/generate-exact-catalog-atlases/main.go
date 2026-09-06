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
	"ts3news/internal/i18n"

	xdraw "golang.org/x/image/draw"
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
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		log.Fatal(err)
	}
	content.InitLocalized()
	root, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	assetDir := filepath.Join(root, "internal", "bot", "webassets")
	entries := content.PixelArtCatalog()
	sources := loadSources(assetDir, entries)
	extraSources := make(map[string]image.Image)
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
		if special, ok := specialSource(entry); ok {
			art := extraSources[special.asset]
			if art == nil {
				file, err := os.Open(filepath.Join(assetDir, special.asset))
				if err != nil {
					log.Fatal(err)
				}
				art, err = png.Decode(file)
				_ = file.Close()
				if err != nil {
					log.Fatal(err)
				}
				extraSources[special.asset] = art
			}
			crop := special.bounds(art.Bounds())
			if !crop.In(art.Bounds()) {
				log.Fatalf("invalid icon source for %s", entry.Key)
			}
			xdraw.NearestNeighbor.Scale(page, destination.Inset(4), art, crop, draw.Over, nil)
		} else {
			draw.Draw(page, destination, source, image.Pt(baseCell.X*cellSize, baseCell.Y*cellSize), draw.Over)
		}
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
	if cell, ok := semanticCell(entry); ok {
		return []image.Point{cell}
	}
	if entry.Family != "items" {
		// Reviewed whole subjects in the first row. Later rows of several
		// original sheets contain fragments crossing the normalized cell grid.
		result := make([]image.Point, 0, content.PixelArtColumns)
		for column := 0; column < content.PixelArtColumns; column++ {
			result = append(result, image.Pt(column, 0))
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
	// Give the object itself an individual finish, preserving its light/shadow
	// and source color family. Identity must survive normal UI downsampling.
	for y := cell.Min.Y; y < cell.Max.Y; y++ {
		for x := cell.Min.X; x < cell.Max.X; x++ {
			pixel := target.NRGBAAt(x, y)
			if pixel.A == 0 {
				continue
			}
			channels := []*uint8{&pixel.R, &pixel.G, &pixel.B}
			for index, channel := range channels {
				factor := 0.80 + float64(digest[index])/637.5
				*channel = uint8(min(255, float64(*channel)*factor))
			}
			target.SetNRGBA(x, y, pixel)
		}
	}
	// Eight separated rune shards encode the identity in visible geometry.
	// They have no boxed badge/backdrop and leave the central subject legible.
	for index := 0; index < 8; index++ {
		angle := float64(index)*math.Pi/4 + float64(digest[3]%16)*math.Pi/128
		radius := 34 + float64(digest[4+index]%7)
		x := centerX + int(math.Cos(angle)*radius)
		y := centerY + int(math.Sin(angle)*radius)
		width, height := 2+int(digest[12+index]%3), 2+int(digest[20+index]%4)
		ink := color.NRGBA{R: 160 + digest[index]%96, G: 140 + digest[index+8]%100, B: 110 + digest[index+16]%130, A: 235}
		for dy := -height; dy <= height; dy++ {
			for dx := -width; dx <= width; dx++ {
				if math.Abs(float64(dx)/float64(width))+math.Abs(float64(dy)/float64(height)) > 1.25 {
					continue
				}
				point := image.Pt(x+dx, y+dy)
				if point.In(cell.Inset(3)) {
					target.SetNRGBA(point.X, point.Y, ink)
				}
			}
		}
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
	// The compiler and image previews may memory-map checked-in PNGs on
	// Windows. Encode a sibling first, then replace the completed asset.
	file, err := os.CreateTemp(filepath.Dir(path), ".catalog-*.png")
	if err != nil {
		log.Fatal(err)
	}
	temporary := file.Name()
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(file, source); err != nil {
		_ = file.Close()
		_ = os.Remove(temporary)
		log.Fatal(err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(temporary)
		log.Fatal(err)
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		log.Fatal(err)
	}
}
