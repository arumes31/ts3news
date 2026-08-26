package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"ts3news/internal/content"
	"ts3news/internal/treeart"
)

func main() {
	output := flag.String("out", "internal/bot/webassets", "directory for generated atlas PNGs")
	flag.Parse()

	atlases, err := treeart.Generate(content.AbyssTree())
	if err != nil {
		fatal(err)
	}
	if err := os.MkdirAll(*output, 0o755); err != nil {
		fatal(fmt.Errorf("create output directory: %w", err))
	}
	for _, sheet := range content.AbyssTreeArtSheets {
		path := filepath.Join(*output, "abyss_atlas_"+sheet+".png")
		file, err := os.Create(path) // #nosec G304 - path is a developer-supplied generation target
		if err != nil {
			fatal(fmt.Errorf("create %s: %w", path, err))
		}
		encodeErr := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(file, atlases[sheet])
		closeErr := file.Close()
		if encodeErr != nil {
			fatal(fmt.Errorf("encode %s: %w", path, encodeErr))
		}
		if closeErr != nil {
			fatal(fmt.Errorf("close %s: %w", path, closeErr))
		}
		fmt.Printf("generated %s\n", path)
	}
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
