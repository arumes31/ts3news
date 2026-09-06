// Command generate-abyss-talent-art writes the authored semantic vector icons
// used by the class talent catalog. Each node owns a separate immutable asset.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"ts3news/internal/content"
)

var glyphs = map[string]string{
	"primary":        `<path d="M29 10 35 16 24 37 19 39 18 33Z"/><path d="m15 30 12 7M17 38l-5 9"/>`,
	"health":         `<path d="M24 43C-5 25 11 3 24 19 38 3 54 25 24 43Z"/><path d="M8 27h9l4-8 6 15 4-7h10"/>`,
	"mana":           `<path d="M24 6C20 18 9 23 9 33a15 15 0 0 0 30 0C39 23 28 18 24 6Z"/><path d="m24 22-6 13h12Z"/>`,
	"builder_power":  `<path d="m8 42 13-14-4-4 23-16-14 23-4-4Z"/><path d="m5 17 10 2m17 21 3 6M8 7l8 7"/>`,
	"builder_shield": `<path d="m24 5 17 8v14c0 11-17 19-17 19S7 38 7 27V13Z"/><path d="M24 14v22m-10-11h20"/>`,
	"builder_heal":   `<path d="M12 42C8 18 18 8 41 9c1 25-10 31-29 33Z"/><path d="m12 42 20-24M14 30l14 1m-6-10 1 14"/>`,
	"finisher_power": `<path d="m24 3 5 14 15-5-8 13 10 11-17-1-5 14-5-14-17 1 10-11-8-13 15 5Z"/>`,
	"penetration":    `<path d="m24 7 16 7v15L24 44 8 29V14Z"/><path d="M3 43 42 4m-13 2 13-2-2 13"/>`,
	"mana_discount":  `<path d="M12 9h23l8 16-18 18L6 25Z"/><path d="M17 25h15"/><circle cx="25" cy="15" r="2"/>`,
	"resource_power": `<path d="m24 4 9 12-9 12-9-12ZM11 26l8 10-8 10-8-10Zm26 0 8 10-8 10-8-10Z"/>`,
	"finisher_heal":  `<path d="M9 18a16 16 0 1 1-1 20M8 9v12h12"/><path d="M24 18v18m-9-9h18"/>`,
	"regen":          `<path d="M7 22C9 5 35 5 41 19m0-10v12H29M41 31C38 48 12 47 7 34m0 11V32h12"/><path d="m25 16-7 12h8l-3 9 10-15h-9Z"/>`,
	"low_health":     `<path d="M23 44C-6 25 10 3 23 19 37 3 54 25 23 44Z"/><path d="m27 13-9 12 10 4-9 13"/>`,
	"marked_power":   `<circle cx="24" cy="25" r="15"/><circle cx="24" cy="25" r="6"/><path d="M24 3v10m0 24v10M2 25h10m24 0h10"/>`,
	"defense":        `<path d="M7 43V12h9v9h7V8h9v13h8v-9h7v31Z"/><path d="M20 43V30h12v13"/>`,
	"cap_burst":      `<path d="m24 2 5 15 15-8-8 15 14 5-16 4 7 15-15-10-7 12-2-17-16 4 11-13-10-9 16 2Z"/>`,
	"cap_aegis":      `<path d="m24 3 19 10v16L24 47 5 29V13Z"/><path d="M14 31V18h6v7h8v-7h6v13Z"/><path d="M24 11v8"/>`,
	"cap_siphon":     `<path d="M34 8C2-6 1 26 27 24c25-2 24 27-9 21M31 7l6 3-6 5M21 39l-8 5 8 4"/><path d="M24 15v19m-7-9h14"/>`,
}

// Class/subclass crests have distinct silhouettes, not just different colors.
var crests = []string{
	`M3 15 10 2l7 13-7-3Z`, `M3 3q20 6 0 14m0-14v14m0-7h15`, `M10 1l3 6 6 3-6 3-3 6-3-6-6-3 6-3Z`, `M4 17Q0 2 17 3Q18 17 4 17Zm0 0L14 6`, `M1 5l5 3 4-6 4 6 5-3-3 12H4Z`, `M4 2h12v4l-3 2v4l4 5H3l4-5V8L4 6Z`,
	`M10 1 18 5v7l-8 7-8-7V5Zm0 4v9`, `M2 3l7 4-5 6 4 5M18 3l-7 4 5 6-4 5`, `M10 1v18M1 10h18M4 4l12 12M4 16 16 4`, `M2 14l3-9 5 7 5-7 3 9-8 5Z`, `M10 1 18 16H2Zm0 5v8M5 14h10`, `M4 2h12L5 18h10Zm1 3 10 10`,
	`M10 1v18M1 10h18M4 4l12 12M4 16 16 4M6 1h8`, `M2 18 5 7l5-5 5 5 3 11ZM5 12h10`, `M3 2l7 5 7-5-4 10-3 7-3-7Z`, `M15 2C0 1 0 19 15 18C7 13 7 7 15 2Z`, `M3 4h14v12H3Zm4-3v18M13 1v18M1 8h18M1 12h18`, `M7 1h6v6l6 10-3 2H4l-3-2L7 7ZM5 13h10`,
}

func main() {
	check := flag.Bool("check", false, "verify assets without writing")
	flag.Parse()
	ids := []string{}
	for _, c := range content.AbyssClasses() {
		ids = append(ids, c.ID)
	}
	for _, c := range content.AbyssClasses() {
		for _, sub := range c.Subclasses {
			ids = append(ids, sub.ID)
		}
	}
	colors := []string{"#e8bf73", "#91c88e", "#a9bfff", "#a9d5ad", "#e895a8", "#82cdd5"}
	count := 0
	for treeIndex, id := range ids {
		tree := content.AbyssTalents(id)
		classIndex := 0
		for i, c := range content.AbyssClasses() {
			if c.ID == tree.Class {
				classIndex = i
			}
		}
		for _, node := range tree.Nodes {
			glyph, ok := glyphs[node.Effect]
			if !ok {
				panic(node.Effect)
			}
			svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><title>%s</title><path d="M12 2h56l10 10v56L68 78H12L2 68V12Z" fill="#101b2a" stroke="%s" stroke-width="2"/><path d="M10 24V12h12M58 12h10v12M10 56v12h12m36 0h10V56" fill="none" stroke="#687c93"/><g transform="translate(16 21)" fill="#263b50" stroke="%s" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">%s</g><g transform="translate(29 3)" fill="#152337" stroke="%s" stroke-width="1.5" stroke-linejoin="round"><path d="%s"/></g></svg>`+"\r\n", html.EscapeString(node.Name), colors[classIndex], colors[classIndex], glyph, colors[classIndex], crests[treeIndex])
			path := filepath.Join("internal", "bot", "webassets", "abyss_talents", node.ID+".svg")
			if *check {
				raw, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(raw, []byte(svg)) {
					panic("talent artwork drift: " + path)
				}
			} else {
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					panic(err)
				}
				if err := os.WriteFile(path, []byte(svg), 0644); err != nil {
					panic(err)
				}
			}
			count++
		}
	}
	fmt.Printf("Validated %d dedicated talent SVG assets.\n", count)
}
