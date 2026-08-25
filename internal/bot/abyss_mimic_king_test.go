package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAbyssMimicKingChainRequiresThreeSurvivedBites(t *testing.T) {
	flags := map[string]int64{}
	firstBiteAwakened := advanceAbyssMimicChain(flags)
	secondBiteAwakened := advanceAbyssMimicChain(flags)
	if firstBiteAwakened || secondBiteAwakened {
		t.Fatal("Mimic King awakened before the third bite")
	}
	if !advanceAbyssMimicChain(flags) || flags[abyssRunFlagMimicsSurvived] != 3 {
		t.Fatalf("third bite did not awaken chain: %#v", flags)
	}
	resetAbyssMimicChain(flags)
	if flags[abyssRunFlagMimicsSurvived] != 0 {
		t.Fatalf("chain did not reset: %#v", flags)
	}
}

func TestAbyssMimicKingHitCannotKill(t *testing.T) {
	for _, current := range []int{1, 10, 100} {
		if got := abyssMimicKingSurvivalHP(current, 300); got < 1 {
			t.Errorf("survival HP from %d = %d", current, got)
		}
	}
	label, grant := abyssMimicKingGrant()
	if !strings.Contains(label, "Crown of False Gold") || grant.Type != "unique" || grant.UniqName == "" {
		t.Fatalf("king grant = %q / %+v", label, grant)
	}
}

func TestAbyssMimicKingUIContract(t *testing.T) {
	path := filepath.Join(abyssAAARepositoryRoot(t), "internal", "bot", "webassets", "abyss.html")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"state.type === 'mimic_king'", "mimic_king_challenge", "mimic_king_retreat", "Crown of False Gold"} {
		if !strings.Contains(string(raw), token) {
			t.Errorf("Mimic King UI is missing %q", token)
		}
	}
}
