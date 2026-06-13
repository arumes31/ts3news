package content

import (
	"os"
	"testing"
	"time"
	"ts3news/internal/i18n"
)

func TestMain(m *testing.M) {
	if err := i18n.InitWithLocale(i18n.LocaleEnUS); err != nil {
		panic("i18n init failed: " + err.Error())
	}
	os.Exit(m.Run())
}

func TestGreetingsCount(t *testing.T) {
	if got := GreetingCount(); got != 100 {
		t.Errorf("expected 100 greetings, got %d", got)
	}
	if RandomGreeting() == "" {
		t.Error("RandomGreeting returned empty")
	}
}

func TestNicknameForGame(t *testing.T) {
	if got := NicknameForGame("Fallout 4"); got != "VaultBoy" {
		t.Errorf("Fallout -> %q, want VaultBoy", got)
	}
	if got := NicknameForGame("Some Unknown Indie Title"); got == "" {
		t.Error("fallback nickname empty")
	}
	if got := NicknameForGame("A Very Extremely Long Game Title That Exceeds Limits Indeed"); len(got) > 28 {
		t.Errorf("nickname too long: %q (%d)", got, len(got))
	}
}

func TestCurrentTheme(t *testing.T) {
	xmas := time.Date(2026, time.December, 24, 12, 0, 0, 0, time.UTC)
	if th := CurrentTheme(xmas); th == nil || th.Name != "Christmas" {
		t.Errorf("expected Christmas theme on Dec 24, got %v", th)
	}
	ordinary := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	if th := CurrentTheme(ordinary); th != nil {
		t.Errorf("expected no theme on Jul 15, got %v", th)
	}
}
