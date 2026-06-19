package bot

import (
	"testing"

	"ts3news/internal/i18n"
)

func TestClientSafeChannelName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"Screaming Guerilla", true},     // en_US, plain ASCII
		{"Schreiender Freischärler", true}, // de_DE, Latin + ä
		{"Guérilla Hurlante", true},      // fr_FR, Latin + é
		{"Cold George's Lair", true},     // apostrophe + hyphen-free
		{"Iron-Forged Hall", true},       // hyphen
		{"Bright & Bold", true},          // ampersand
		{"Вопящий Мятежник", false},      // ru_RU, Cyrillic
		{"スクリーミング・ゲリラ", false},        // ja_JP, Katakana
		{"咆哮するゲリラ", false},             // CJK
		{"", false},                      // empty
		{"   ", false},                   // whitespace only
	}
	for _, c := range cases {
		if got := clientSafeChannelName(c.name); got != c.want {
			t.Errorf("clientSafeChannelName(%q) = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestPickChannelNameUniqueAndSafe(t *testing.T) {
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		t.Fatal(err)
	}

	// Repeatedly pick names while reserving each one; every pick must be unique,
	// non-empty, and client-safe (foreign borrows are filtered to safe chars).
	taken := make(map[string]bool)
	for i := 0; i < 200; i++ {
		name, _ := pickChannelName(taken)
		if name == "" {
			t.Fatalf("pick %d returned empty name", i)
		}
		if taken[name] {
			t.Fatalf("pick %d returned already-taken name %q", i, name)
		}
		if !clientSafeChannelName(name) {
			t.Fatalf("pick %d returned non-client-safe name %q", i, name)
		}
		taken[name] = true
	}
}

func TestPickChannelNameExhausted(t *testing.T) {
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		t.Fatal(err)
	}
	// Reserve every local name; the picker should return "" (caller keeps the
	// current name) rather than a duplicate. Foreign borrows may still succeed,
	// so accept either "" or a name absent from taken.
	taken := make(map[string]bool)
	for _, n := range i18n.Pool("channel.name") {
		taken[n] = true
	}
	if name, _ := pickChannelName(taken); name != "" && taken[name] {
		t.Errorf("pickChannelName returned taken name %q when local pool exhausted", name)
	}
}
