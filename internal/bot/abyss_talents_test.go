package bot

import (
	"crypto/sha256"
	"regexp"
	"strings"
	"testing"
	"ts3news/internal/content"
)

func TestAbyssTalentAllocationTradeoffs(t *testing.T) {
	base := []string{"warrior_t1_1", "warrior_t2_2", "warrior_t3_1", "warrior_t4_2", "warrior_t5_3"}
	if err := validateAbyssTalents("warrior", base, 5); err != nil {
		t.Fatal(err)
	}
	for _, ids := range [][]string{append(append([]string{}, base...), "warrior_t1_2"), {"warrior_t2_1"}, {"oracle_t1_1"}, {"warrior_t1_1", "warrior_t1_1"}} {
		if validateAbyssTalents("warrior", ids, 5) == nil {
			t.Fatal(ids)
		}
	}
	if validateAbyssTalents("warrior", base, 4) == nil {
		t.Fatal("unearned points")
	}
	sub := []string{"vanguard_t1_1", "vanguard_t1_2", "vanguard_t2_1", "vanguard_t2_2", "vanguard_t3_1", "vanguard_t3_2", "vanguard_t4_1", "vanguard_t4_2", "vanguard_t5_1", "vanguard_t6_1"}
	if err := validateAbyssTalents("vanguard", sub, 10); err != nil {
		t.Fatal(err)
	}
	if validateAbyssTalents("vanguard", append(sub, "vanguard_t6_2"), 15) == nil {
		t.Fatal("two capstones")
	}
	if validateAbyssTalents("vanguard", []string{"vanguard_t6_1"}, 10) == nil {
		t.Fatal("capstone shortcut")
	}
}
func TestAbyssTalentLegacyMigrationPreservesSubclass(t *testing.T) {
	state, err := decodeAbyssClassState(`{"version":1,"selected":"oracle","profiles":{"oracle":{"skills":["S0_1"],"pins":["S0_1"]}}}`)
	if err != nil {
		t.Fatal(err)
	}
	p := state.Progress["warden"]
	if state.Version != 2 || state.Class != "warden" || content.AbyssClassPoints(p.XP) != 5 || len(p.Foundation) != 5 || state.Profiles["oracle"].Skills[0] != "S0_1" {
		t.Fatal(state)
	}
	raw := `{"version":2,"class":"warrior","profiles":{},"progress":{}}`
	fresh, err := decodeAbyssClassState(raw)
	if err != nil || fresh.Progress["warrior"].XP != 0 {
		t.Fatal(fresh, err)
	}
}

func TestAbyssTalentFoundationActionsUnlockFromEarnedPoints(t *testing.T) {
	for _, tc := range []struct {
		xp   int64
		want int
	}{{0, 0}, {1000, 1}, {14999, 1}, {15000, 2}} {
		state := newAbyssClassState()
		state.Class = "warrior"
		state.Progress["warrior"] = abyssClassProgress{XP: tc.xp}
		u := UserInCombat{Stats: content.Stats{STR: 100, HP: 100}, CurrentHP: 100}
		(&Bot{}).applyAbyssClassBuild(&u, state)
		if len(u.Skills) != tc.want || u.AbyssClass != "warrior" || u.AbyssSubclass != "" {
			t.Fatal(tc, u)
		}
	}
}

func TestAbyssTalentArtworkIsEmbeddedAndVisuallyDistinct(t *testing.T) {
	hashes := map[[32]byte]bool{}
	title := regexp.MustCompile(`<title>.*?</title>`)
	for _, class := range content.AbyssClasses() {
		ids := []string{class.ID}
		for _, sub := range class.Subclasses {
			ids = append(ids, sub.ID)
		}
		for _, id := range ids {
			for _, node := range content.AbyssTalents(id).Nodes {
				raw, err := webAssets.ReadFile("webassets/" + strings.TrimPrefix(node.Art, "/static/"))
				if err != nil {
					t.Fatal(node.ID, err)
				}
				hash := sha256.Sum256(title.ReplaceAll(raw, nil))
				if hashes[hash] {
					t.Fatal("reused visible artwork", node.ID)
				}
				hashes[hash] = true
			}
		}
	}
	if len(hashes) != 306 {
		t.Fatal(len(hashes))
	}
}
