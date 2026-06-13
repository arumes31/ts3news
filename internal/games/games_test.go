package games

import (
	"testing"
)

func TestGameDisplayTitle(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Game Title [Steam]", "Game Title"},
		{"Epic Game (Epic Games) Giveaway", "Epic Game"},
		{"Mystery Game Giveaways", "Mystery Game"},
		{"[Steam] Borderlands 3 Giveaway", "Borderlands 3"},
		{"Always Free", "Always Free"},
		{"   ", ""},
	}
	for _, tt := range tests {
		g := Game{Title: tt.in}
		if got := g.DisplayTitle(); got != tt.want {
			t.Errorf("DisplayTitle(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGameKey(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"The Witcher 3: Wild Hunt", "thewitcher3wildhunt"},
		{"Witcher 3 [Steam]", "witcher3"},
		{"Witcher 3 giveaway", "witcher3"},
		{"Witcher-3 free", "witcher3"},
		{"!!!", ""}, // Fallback logic
	}
	for _, tt := range tests {
		g := Game{Title: tt.in}
		got := g.Key()
		if tt.want != "" && got != tt.want {
			t.Errorf("Key(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMatchesDRM(t *testing.T) {
	g := Game{Platforms: "PC, Steam, Epic Games"}
	tests := []struct {
		want []string
		got  bool
	}{
		{[]string{"steam"}, true},
		{[]string{"epic"}, true},
		{[]string{"gog"}, false},
		{[]string{"pc"}, true},
	}
	for _, tt := range tests {
		if got := matchesDRM(g, tt.want); got != tt.got {
			t.Errorf("matchesDRM(%v) = %v, want %v", tt.want, got, tt.got)
		}
	}
}

func TestPriceEUR(t *testing.T) {
	tests := []struct {
		worth string
		want  float64
		ok    bool
	}{
		{"$29.99", 29.99, true},
		{"€19.99", 19.99, true},
		{"£15.50", 15.5, true},
		{"10,00", 10.0, true},
		{"N/A", 0, false},
		{"FREE", 0, false},
		{"0.00", 0, false},
	}
	for _, tt := range tests {
		g := Game{Worth: tt.worth}
		val, ok := g.PriceEUR()
		if ok != tt.ok || (ok && val != tt.want) {
			t.Errorf("PriceEUR(%q) = %f, %v; want %f, %v", tt.worth, val, ok, tt.want, tt.ok)
		}
	}
}

func TestIsActive(t *testing.T) {
	g := Game{EndDate: "2099-01-01 00:00:00"}
	if !g.isActive() {
		t.Error("2099 should be active")
	}
	g2 := Game{EndDate: "2000-01-01 00:00:00"}
	if g2.isActive() {
		t.Error("2000 should not be active")
	}
	g3 := Game{EndDate: "N/A"}
	if !g3.isActive() {
		t.Error("N/A should be active")
	}
}

func TestFmtCents(t *testing.T) {
	tests := []struct {
		cents int
		curr  string
		want  string
	}{
		{1999, "EUR", "€19.99"},
		{2999, "USD", "$29.99"},
		{1550, "GBP", "£15.50"},
		{1000, "JPY", "JPY 10.00"},
	}
	for _, tt := range tests {
		if got := fmtCents(tt.cents, tt.curr); got != tt.want {
			t.Errorf("fmtCents(%d, %q) = %q, want %q", tt.cents, tt.curr, got, tt.want)
		}
	}
}

func TestCleanRedditTitle(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"[Steam] (Game) Witcher 3 free", "Witcher 3"},
		{"Epic Game Giveaway (100% off)", "Epic Game"},
		{"Mystery Game is free", "Mystery Game"},
	}
	for _, tt := range tests {
		if got := cleanRedditTitle(tt.in); got != tt.want {
			t.Errorf("cleanRedditTitle(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRedditDRM(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"[steam] game", "games.platform.steam"},
		{"[epic] game", "games.platform.epic"},
		{"[gog] game", "games.platform.gog"},
		{"unknown", ""},
	}
	for _, tt := range tests {
		if got := redditDRM(tt.title, ""); got != tt.want {
			t.Errorf("redditDRM(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}
