package bot

import (
	"testing"

	"ts3news/internal/content"
)

func TestNormalizeAbyssCombatPosition(t *testing.T) {
	t.Parallel()

	for input, want := range map[string]string{
		"frontline":  "frontline",
		" BACKLINE ": "backline",
		"":           "frontline",
		"forged":     "frontline",
	} {
		if got := normalizeAbyssCombatPosition(input); got != want {
			t.Errorf("normalize position %q = %q, want %q", input, got, want)
		}
	}
}

func TestApplyAbyssCombatPositionUsesRunChoice(t *testing.T) {
	t.Parallel()

	front := UserInCombat{}
	applyAbyssCombatPosition(&front, nil)
	if front.Position != content.PositionFrontline {
		t.Fatalf("default position = %q, want Frontline", front.Position)
	}

	back := UserInCombat{}
	applyAbyssCombatPosition(&back, map[string]int64{
		abyssRunFlagPosition: abyssCombatPositions["backline"],
	})
	if back.Position != content.PositionBackline {
		t.Fatalf("selected position = %q, want Backline", back.Position)
	}

	applyAbyssCombatPosition(nil, nil)
}
