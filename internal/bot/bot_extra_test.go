package bot

import (
	"strings"
	"testing"
	"ts3news/internal/config"
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
	// Skip for now due to DB dependencies in signature
	t.Skip("Skipping composePM test")
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
		{100, "[b]100[/b][color=#9e9e9e]g[/color]"},
		{1500, "[b]1.5[/b][color=#9e9e9e]k[/color]"},
		{2000000, "[b]2.0[/b][color=#9e9e9e]M[/color]"},
	}
	for _, tt := range tests {
		if got := FormatGold(tt.v); got != tt.want {
			t.Errorf("FormatGold(%d) = %q, want %q", tt.v, got, tt.want)
		}
	}
}
