package main

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestIconCandidatesDescribeTheEquipmentOrAction(t *testing.T) {
	tests := []struct {
		name, family, kind, variant string
		want                        image.Point
	}{
		{"Great Health Potion", "items", "consumable", "", image.Pt(0, 6)},
		{"Intellect Elixir", "items", "consumable", "", image.Pt(1, 6)},
		{"Companion Revival Scroll", "items", "consumable", "", image.Pt(0, 7)},
		{"Trusty Longsword", "items", "gear", "MainHand", image.Pt(0, 0)},
		{"Earthshaker Hammer", "items", "gear", "MainHand", image.Pt(9, 0)},
		{"Sun-King Crown", "items", "gear", "Head", image.Pt(10, 1)},
		{"Shadow Assassin Hood", "items", "gear", "Head", image.Pt(8, 1)},
		{"Stormbringer Cloak", "items", "gear", "Back", image.Pt(4, 2)},
		{"Fiery Bolt", "skills", "skill", "Magic", image.Pt(0, 0)},
		{"Icy Bolt", "skills", "skill", "Magic", image.Pt(1, 0)},
		{"Holy Heal", "skills", "skill", "Buff", image.Pt(0, 1)},
		{"Arcane Shield", "skills", "skill", "Magic", image.Pt(1, 1)},
		{"Earthquake", "skills", "skill", "Physical", image.Pt(3, 0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := content.PixelArtEntry{Name: tt.name, Family: tt.family, Kind: tt.kind, Variant: tt.variant}
			cells := semanticCandidates(entry)
			if len(cells) != 1 || cells[0] != tt.want {
				t.Fatalf("%s candidates = %v, want reviewed cell %v", tt.name, cells, tt.want)
			}
		})
	}
}

func TestIdentityRemainsVisibleAfterThumbnailing(t *testing.T) {
	makeIcon := func(key string) []byte {
		icon := image.NewNRGBA(image.Rect(0, 0, 96, 96))
		for y := 20; y < 76; y++ {
			for x := 35; x < 61; x++ {
				icon.SetNRGBA(x, y, color.NRGBA{R: 180, G: 130, B: 70, A: 255})
			}
		}
		embedIdentity(icon, icon.Bounds(), content.PixelArtEntry{Key: key, Kind: "skill", Name: "Fiery Bolt"})
		// The former four-pixel watermark must not be the sole distinction.
		for y := 47; y <= 48; y++ {
			for x := 46; x <= 49; x++ {
				icon.SetNRGBA(x, y, color.NRGBA{})
			}
		}
		return icon.Pix
	}
	first, second := makeIcon("skill:example-a"), makeIcon("skill:example-b")
	if bytes.Equal(first, second) {
		t.Fatal("identity exists only in the invisible four-pixel watermark")
	}
	changed := 0
	for i := 0; i < len(first); i += 4 {
		if !bytes.Equal(first[i:i+4], second[i:i+4]) {
			changed++
		}
	}
	if changed < 200 {
		t.Fatalf("only %d pixels distinguish icons, want visible ornament and finish", changed)
	}
}

func TestMonsterPortraitMatchesItsCombatSpecies(t *testing.T) {
	for _, entry := range content.PixelArtCatalog() {
		if entry.Kind != "monster" {
			continue
		}
		if _, ok := specialSource(entry); !ok {
			t.Errorf("no reviewed anatomy for %s", entry.Name)
		}
	}
	for _, name := range []string{"Undead Rat", "Ghostly Rat", "Giant Rat"} {
		source, ok := monsterSource(strings.ToLower(name))
		if !ok || source.asset != "abyss_combat_bestiary_v2.png" || source.row != 0 {
			t.Errorf("%s is not a rat: %+v", name, source)
		}
	}
}
