package bot

import (
	"strings"
	"testing"
	"ts3news/internal/config"
	"ts3news/internal/content"
	"ts3news/internal/games"
)

func TestSplitMessage(t *testing.T) {
	msg := "line1\nline2\nline3"
	chunks := splitMessage(msg, 10)
	if len(chunks) != 3 {
		t.Errorf("len(chunks) = %d, want 3", len(chunks))
	}
	if chunks[0] != "line1" {
		t.Errorf("chunks[0] = %q", chunks[0])
	}
}

func TestComposePoke(t *testing.T) {
	g := games.Game{Title: "Test Game"}
	poke := composePoke(g, "http://short", nil, nil)
	if !strings.Contains(poke, "Test Game") || !strings.Contains(poke, "http://short") {
		t.Errorf("poke = %q", poke)
	}
}

func TestComposePM(t *testing.T) {
	b := &Bot{Cfg: &config.Config{}}
	g := games.Game{Title: "Test Game", Worth: "20.00€", URL: "http://example.com"}
	lr := &levelResult{OldLevel: 1, NewLevel: 2, Awarded: 100, TotalXP: 100}
	
	// Test without theme
	pm := b.composePM(g, "http://short", nil, lr, []string{"note1", "note2", "10/10 dura"}, 50)
	if !strings.Contains(pm, "Test Game") || !strings.Contains(pm, "note1") || !strings.Contains(pm, "LvL: 2") {
		t.Errorf("pm without theme = %q", pm)
	}

	// Test with theme
	theme := &content.Theme{Emoji: "🎄", Banner: "Holiday!", Signoff: "Merry X-Mas"}
	pmTheme := b.composePM(g, "http://short", theme, lr, nil, 50)
	if !strings.Contains(pmTheme, "🎄") || !strings.Contains(pmTheme, "Holiday!") || !strings.Contains(pmTheme, "Merry X-Mas") {
		t.Errorf("pm with theme = %q", pmTheme)
	}
}

func TestXPForGame(t *testing.T) {
	b := &Bot{Cfg: &config.Config{}}
	
	tests := []struct {
		worth   string
		cheaper bool
	}{
		{"10.00€", false},
		{"0.00€", false},
		{"invalid", false},
		{"5.00€", true},
	}
	for _, tt := range tests {
		g := games.Game{Worth: tt.worth}
		b.Cfg.CheaperMoreXP = tt.cheaper
		xp := b.xpForGame(g)
		if xp <= 0 {
			t.Errorf("xpForGame(%q, cheaper=%v) = %d", tt.worth, tt.cheaper, xp)
		}
	}
}

func TestFormatGold(t *testing.T) {
	tests := []struct {
		v    int64
		want string
	}{
		{100, "100"},
		{1500, "1.5k"},
		{2000000, "2.0M"},
		{3000000000, "3.0B"},
	}
	for _, tt := range tests {
		if got := FormatGold(tt.v); got != tt.want {
			t.Errorf("FormatGold(%d) = %q, want %q", tt.v, got, tt.want)
		}
	}
}
