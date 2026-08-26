package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssRescueSupportLifecycle(t *testing.T) {
	t.Parallel()

	flags := map[string]int64{
		abyssRunFlagExplorerGuardFloors: abyssExplorerSupportFloors,
		abyssRunFlagExplorerSupportID:   1,
	}
	view := abyssRescueSupportViewFromFlags(flags)
	if !view.Active || view.Name != "Mara Flint" || view.Remaining != 3 {
		t.Fatalf("initial support view = %#v", view)
	}
	for remaining := 2; remaining >= 0; remaining-- {
		if !tickAbyssRescueSupport(flags) {
			t.Fatalf("tick to %d fights reported no change", remaining)
		}
		view = abyssRescueSupportViewFromFlags(flags)
		if remaining == 0 {
			if view.Active || flags[abyssRunFlagExplorerSupportID] != 0 {
				t.Fatalf("expired support remained active: view=%#v flags=%v", view, flags)
			}
			continue
		}
		if !view.Active || view.Remaining != remaining {
			t.Fatalf("support after tick = %#v, want %d fights", view, remaining)
		}
	}
	if tickAbyssRescueSupport(flags) {
		t.Fatal("expired support changed on an extra tick")
	}
}

func TestAbyssRescueSupportRejectsInvalidIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		id   int64
	}{
		{name: "missing", id: 0},
		{name: "negative after decoding", id: -1},
		{name: "past catalog", id: int64(len(abyssExplorerNames) + 1)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := map[string]int64{
				abyssRunFlagExplorerGuardFloors: 3,
				abyssRunFlagExplorerSupportID:   tt.id,
			}
			if support := abyssRescueSupportFromFlags(flags, 100, 100); support != nil {
				t.Fatalf("invalid support ID %d produced %#v", tt.id, support)
			}
		})
	}
}

func TestAbyssRescueSupportAttacksLowestHPEnemy(t *testing.T) {
	t.Parallel()

	owner := &UserInCombat{
		UID: "owner", Nickname: "Hero", CurrentHP: 100,
		Stats:        content.Stats{HP: 100, STR: 400, SPD: 80},
		abyssSupport: &abyssRescueSupport{Name: "Tessa Quill", Remaining: 3, Power: 100, Speed: 79},
	}
	users := []activeUser{{u: owner}}
	high := &content.Mob{Name: "High target", Stats: content.Stats{HP: 5000, DEF: 100}, MaxHP: 5000}
	low := &content.Mob{Name: "Low target", Stats: content.Stats{HP: 3000, DEF: 100}, MaxHP: 5000}
	mobs := []*content.Mob{high, low}
	logs := []string{}
	totalDamage := 0
	(&Bot{}).applyAbyssRescueSupportTurn(
		users, &mobs, content.Zone{}, 1, &logs, &totalDamage, 1, 1,
		[]UserInCombat{*owner}, &[]LootResult{}, defaultCombatRandomSource{},
	)
	if high.Stats.HP != 5000 || low.Stats.HP != 2925 || totalDamage != 75 {
		t.Fatalf("support attack result: high=%d low=%d total=%d", high.Stats.HP, low.Stats.HP, totalDamage)
	}
	if len(logs) != 1 || !strings.Contains(logs[0], "Tessa Quill fights beside Hero") {
		t.Fatalf("support attack logs = %v", logs)
	}
}

func TestAbyssRescueSupportAppearsInLiveInitiativeAndUI(t *testing.T) {
	t.Parallel()

	owner := &UserInCombat{
		UID: "owner", Nickname: "Hero", CurrentHP: 100,
		Stats:        content.Stats{HP: 100, SPD: 80},
		abyssSupport: &abyssRescueSupport{Name: "Orin Vale", Remaining: 2, Power: 25, Speed: 79},
	}
	initiative := liveInitiative([]activeUser{{u: owner}}, nil, true)
	if len(initiative) != 2 || initiative[1].ID != "support:explorer" || initiative[1].Name != "Orin Vale" {
		t.Fatalf("live initiative = %#v", initiative)
	}

	page, err := webAssets.ReadFile("webassets/abyss.html")
	if err != nil {
		t.Fatal(err)
	}
	live, err := webAssets.ReadFile("webassets/abyss_live.html")
	if err != nil {
		t.Fatal(err)
	}
	styles, err := webAssets.ReadFile("webassets/abyss_command.css")
	if err != nil {
		t.Fatal(err)
	}
	joined := string(page) + string(live) + string(styles)
	for _, token := range []string{"explorerSupportChip", "attack once per round", "renderExplorerSupport", "!unit.is_player", "#explorerSupportChip[hidden]"} {
		if !strings.Contains(joined, token) {
			t.Errorf("rescue support GUI contract is missing %q", token)
		}
	}
}
