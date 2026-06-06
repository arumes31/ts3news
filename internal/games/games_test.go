package games

import "testing"

func TestKeyCrossSourceDedup(t *testing.T) {
	// GamerPower-style title vs. Epic-API bare title must normalise identically.
	gamerPower := Game{Title: "Songs of Conquest (Epic Games) Giveaway"}
	epic := Game{Title: "Songs of Conquest"}
	if gamerPower.Key() != epic.Key() {
		t.Errorf("keys differ: %q vs %q", gamerPower.Key(), epic.Key())
	}
	if gamerPower.Key() != "songsofconquest" {
		t.Errorf("unexpected key %q", gamerPower.Key())
	}

	steam := Game{Title: "Some Game [Steam] (DRM-Free) Giveaway"}
	if steam.Key() != "somegame" {
		t.Errorf("steam key = %q, want somegame", steam.Key())
	}
}

func TestDisplayTitleAndTrailer(t *testing.T) {
	g := Game{Title: "Songs of Conquest (Epic Games) Giveaway"}
	if got := g.DisplayTitle(); got != "Songs of Conquest" {
		t.Errorf("DisplayTitle = %q, want %q", got, "Songs of Conquest")
	}
	url := TrailerSearchURL(g.DisplayTitle())
	if url != "https://www.youtube.com/results?search_query=Songs+of+Conquest" {
		t.Errorf("trailer URL = %q", url)
	}
}

func TestPriceEUR(t *testing.T) {
	if p, ok := (Game{Worth: "€19.99"}).PriceEUR(); !ok || p != 19.99 {
		t.Errorf("PriceEUR(€19.99) = %v,%v", p, ok)
	}
	if _, ok := (Game{Worth: "N/A"}).PriceEUR(); ok {
		t.Error("N/A should have no price")
	}
}

func TestMatchesDRM(t *testing.T) {
	g := Game{Platforms: "PC, Epic Games Store"}
	if !matchesDRM(g, []string{"epic"}) {
		t.Error("expected epic match")
	}
	if matchesDRM(g, []string{"steam"}) {
		t.Error("did not expect steam match")
	}
}

func TestNormallyPaidAndActive(t *testing.T) {
	if (Game{Worth: "N/A"}).isNormallyPaid() {
		t.Error("N/A should not be normally paid")
	}
	if !(Game{Worth: "€19.99"}).isNormallyPaid() {
		t.Error("€19.99 should be normally paid")
	}
	if (Game{Worth: "$0.00"}).isNormallyPaid() {
		t.Error("$0.00 should not be normally paid")
	}
	if !(Game{EndDate: "N/A"}).isActive() {
		t.Error("open-ended should be active")
	}
	if (Game{EndDate: "2000-01-01 00:00:00"}).isActive() {
		t.Error("past end date should be inactive")
	}
}
