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
	b := &Bot{Cfg: &config.Config{}}
	g := games.Game{Title: "Test Game", Worth: "20€"}
	lr := &levelResult{OldLevel: 1, NewLevel: 2, Awarded: 100, TotalXP: 100}
	pm := b.composePM(g, "http://short", nil, lr, []string{"note1", "note2"}, 50)
	if !strings.Contains(pm, "Test Game") || !strings.Contains(pm, "note1") || !strings.Contains(pm, "LvL: 2") {
		t.Errorf("pm = %q", pm)
	}
}

func TestXPForGame(t *testing.T) {
	b := &Bot{Cfg: &config.Config{}}
	g := games.Game{Worth: "10.00€"}
	xp := b.xpForGame(g)
	if xp <= 0 {
		t.Error("xpForGame returned zero or negative")
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
