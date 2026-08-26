package bot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestRecordAbyssSkillVarietyCountsSuccessfulDistinctPartyCasts(t *testing.T) {
	live := &abyssLiveCombat{
		varietySkills: make(map[string]struct{}),
		skillVariety:  newAbyssSkillVarietyView(0),
	}
	first := UserInCombat{EscrowLoot: true, live: live}
	second := UserInCombat{EscrowLoot: true, live: live}
	party := []activeUser{{u: &first}, {u: &second}}
	var logs []string

	recordAbyssSkillVariety(&first, content.Skill{ID: "skill-a", Name: "Alpha"}, party, &logs)
	recordAbyssSkillVariety(&first, content.Skill{ID: "skill-a", Name: "Alpha"}, party, &logs)
	recordAbyssSkillVariety(&second, content.Skill{ID: "skill-b", Name: "Beta"}, party, &logs)
	recordAbyssSkillVariety(&second, content.Skill{ID: "skill-a", Name: "Alpha"}, party, &logs)
	recordAbyssSkillVariety(&first, content.Skill{ID: "skill-c", Name: "Gamma"}, party, &logs)

	view := abyssSkillVarietyForActiveUsers(party)
	if view.Distinct != 3 || !view.Unlocked || live.skillVariety != view {
		t.Fatalf("variety = %+v, live = %+v", view, live.skillVariety)
	}
	if len(logs) != 3 {
		t.Fatalf("logs = %d, want one per party-distinct skill: %v", len(logs), logs)
	}
	if !strings.Contains(logs[2], "VARIETY COMPLETE") || !strings.Contains(logs[2], "+5%") {
		t.Fatalf("unlock log = %q", logs[2])
	}
}

func TestRecordAbyssSkillVarietyIgnoresNonAbyssAndMissingIDs(t *testing.T) {
	regular := UserInCombat{}
	abyss := UserInCombat{EscrowLoot: true}
	var logs []string
	recordAbyssSkillVariety(&regular, content.Skill{ID: "skill-a", Name: "Alpha"}, []activeUser{{u: &regular}}, &logs)
	recordAbyssSkillVariety(&abyss, content.Skill{Name: "No ID"}, []activeUser{{u: &abyss}}, &logs)
	if len(regular.abyssSkillsUsed) != 0 || len(abyss.abyssSkillsUsed) != 0 || len(logs) != 0 {
		t.Fatalf("ignored casts mutated state: regular=%v abyss=%v logs=%v", regular.abyssSkillsUsed, abyss.abyssSkillsUsed, logs)
	}
}

func TestApplyAbyssSkillVarietyXPRoundsUp(t *testing.T) {
	tests := []struct {
		name       string
		reward     int
		distinct   int
		wantReward int
		wantBonus  int
	}{
		{name: "locked", reward: 100, distinct: 2, wantReward: 100},
		{name: "minimum", reward: 1, distinct: 3, wantReward: 2, wantBonus: 1},
		{name: "exact", reward: 100, distinct: 3, wantReward: 105, wantBonus: 5},
		{name: "ceiling", reward: 21, distinct: 4, wantReward: 23, wantBonus: 2},
		{name: "zero", reward: 0, distinct: 3, wantReward: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotReward, gotBonus := applyAbyssSkillVarietyXP(test.reward, newAbyssSkillVarietyView(test.distinct))
			if gotReward != test.wantReward || gotBonus != test.wantBonus {
				t.Fatalf("apply(%d, %d) = (%d, %d), want (%d, %d)", test.reward, test.distinct, gotReward, gotBonus, test.wantReward, test.wantBonus)
			}
		})
	}
}

func TestAbyssLiveSnapshotIncludesSkillVariety(t *testing.T) {
	combat := &abyssLiveCombat{
		id:           "session",
		ownerUID:     "owner",
		participants: map[string]bool{"owner": true},
		tactics:      map[string]string{"owner": "balanced"},
		options:      map[string][]abyssLiveOption{},
		queued:       map[string]abyssLiveAction{},
		skillVariety: newAbyssSkillVarietyView(3),
	}
	snapshot := combat.snapshotFor("owner")
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SkillVariety.Distinct != 3 || !snapshot.SkillVariety.Unlocked || !strings.Contains(string(encoded), `"skill_variety":{"distinct":3`) {
		t.Fatalf("snapshot variety = %+v; JSON = %s", snapshot.SkillVariety, encoded)
	}
}

func TestAbyssSkillVarietyUIContract(t *testing.T) {
	partial, err := webAssets.ReadFile("webassets/abyss_skill_variety.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_skill_variety.css")
	if err != nil {
		t.Fatal(err)
	}
	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	live, err := webAssets.ReadFile("webassets/abyss_live.html")
	if err != nil {
		t.Fatal(err)
	}
	root := abyssAAARepositoryRoot(t)
	combatSource, err := os.ReadFile(filepath.Join(root, "internal", "bot", "xp.go"))
	if err != nil {
		t.Fatal(err)
	}
	settlementSource, err := os.ReadFile(filepath.Join(root, "internal", "bot", "web_abyss.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"VARIETY", "renderLiveSkillVariety", "textContent", "--variety-progress"} {
		if !strings.Contains(string(partial), token) {
			t.Errorf("variety partial is missing %q", token)
		}
	}
	for _, token := range []string{".ab-skill-variety", ".unlocked", "@media (max-width: 900px)"} {
		if !strings.Contains(string(styles), token) {
			t.Errorf("variety styles are missing %q", token)
		}
	}
	for _, token := range []string{"/static/abyss_skill_variety.css", `template "abyssSkillVarietyJS"`, "variety_bonus_xp"} {
		if !strings.Contains(string(page), token) {
			t.Errorf("Abyss page is missing %q", token)
		}
	}
	for _, token := range []string{`template "abyssSkillVarietyLive"`, "state.skill_variety"} {
		if !strings.Contains(string(live), token) {
			t.Errorf("live combat UI is missing %q", token)
		}
	}
	if !strings.Contains(string(combatSource), "recordAbyssSkillVariety(u, s, activeUsers, logs)") {
		t.Error("successful skill-cast path does not record variety")
	}
	for _, token := range []string{"applyAbyssSkillVarietyXP", `"skill_variety"`, `"variety_bonus_xp"`} {
		if !strings.Contains(string(settlementSource), token) {
			t.Errorf("Abyss settlement is missing %q", token)
		}
	}
}
