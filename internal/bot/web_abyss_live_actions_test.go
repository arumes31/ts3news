package bot

import (
	"errors"
	"testing"
	"time"

	"ts3news/internal/content"
)

func TestAbyssLiveCombatBestAction(t *testing.T) {
	tests := []struct {
		name         string
		tactic       string
		allyHP       int
		options      []abyssLiveOption
		expectedKind string
		expectedID   string
	}{
		{
			name:   "balanced heals critical ally with skill",
			tactic: "balanced", allyHP: 30,
			options: []abyssLiveOption{
				{Kind: "skill", ID: "heal", Target: "ally", Power: 1},
				{Kind: "skill", ID: "blast", Target: "enemy", Power: 4},
			},
			expectedKind: "skill", expectedID: "heal",
		},
		{
			name:   "defensive spends potion earlier",
			tactic: "defensive", allyHP: 40,
			options: []abyssLiveOption{
				{Kind: "item", ID: "potion", Target: "ally", Count: 1, Power: 50},
				{Kind: "skill", ID: "blast", Target: "enemy", Power: 4},
			},
			expectedKind: "item", expectedID: "potion",
		},
		{
			name:   "conserve items attacks outside emergency",
			tactic: "conserve_items", allyHP: 20,
			options: []abyssLiveOption{
				{Kind: "item", ID: "potion", Target: "ally", Count: 1, Power: 50},
				{Kind: "ultimate", ID: "meteor", Target: "enemy", Power: 8},
			},
			expectedKind: "ultimate", expectedID: "meteor",
		},
		{
			name:   "chooses strongest affordable offense",
			tactic: "balanced", allyHP: 100,
			options: []abyssLiveOption{
				{Kind: "skill", ID: "expensive", Target: "enemy", Mana: 120, Power: 20},
				{Kind: "skill", ID: "bolt", Target: "enemy", Mana: 20, Power: 3},
			},
			expectedKind: "skill", expectedID: "bolt",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			combat := &abyssLiveCombat{
				round:   4,
				tactics: map[string]string{"user": test.tactic},
				allies: []abyssLiveCombatantView{
					{ID: "ally:user", HP: test.allyHP, MaxHP: 100, Mana: 100, MaxMana: 100},
				},
				enemies: []abyssLiveCombatantView{
					{ID: "enemy:0", HP: 90, MaxHP: 100},
					{ID: "enemy:1", HP: 25, MaxHP: 100},
				},
				options: map[string][]abyssLiveOption{"user": test.options},
			}
			action := combat.bestActionLocked("user")
			if action.Kind != test.expectedKind || action.AbilityID != test.expectedID {
				t.Fatalf("best action = %s/%s, want %s/%s", action.Kind, action.AbilityID, test.expectedKind, test.expectedID)
			}
			if action.TargetID != "enemy:1" && action.Kind != "item" && action.AbilityID != "heal" {
				t.Fatalf("offensive action target = %q, want lowest-health enemy", action.TargetID)
			}
			if !action.Automatic {
				t.Fatal("best action must be marked automatic")
			}
		})
	}
}

func TestAbyssLiveCombatSubmit(t *testing.T) {
	combat := &abyssLiveCombat{
		id:           "session",
		participants: map[string]bool{"user": true},
		phase:        "planning",
		round:        3,
		deadline:     time.Now().Add(time.Minute),
		options: map[string][]abyssLiveOption{
			"user": {
				{Kind: "attack", Name: "Basic Attack", Target: "enemy"},
				{Kind: "skill", ID: "heal", Name: "Heal", Target: "ally", Mana: 20},
			},
		},
		allies: []abyssLiveCombatantView{
			{ID: "ally:user", HP: 50, MaxHP: 100, Mana: 100, MaxMana: 100},
		},
		enemies: []abyssLiveCombatantView{
			{ID: "enemy:0", HP: 50, MaxHP: 100},
		},
		queued:      map[string]abyssLiveAction{},
		idempotency: map[string]bool{},
	}

	action := abyssLiveAction{
		SessionID: "session", Kind: "attack", TargetID: "enemy:0", Round: 3, IdempotencyKey: "same",
	}
	if err := combat.submit("user", action); err != nil {
		t.Fatalf("valid action rejected: %v", err)
	}
	if err := combat.submit("user", action); err != nil {
		t.Fatalf("idempotent retry rejected: %v", err)
	}
	if got := combat.queued["user"]; got.Automatic {
		t.Fatal("manual action marked automatic")
	}

	wrongSession := action
	wrongSession.SessionID = "another-session"
	wrongSession.IdempotencyKey = "wrong-session"
	if err := combat.submit("user", wrongSession); !errors.Is(err, errAbyssLiveStale) {
		t.Fatalf("wrong-session action error = %v, want errAbyssLiveStale", err)
	}

	stale := action
	stale.Round = 2
	stale.IdempotencyKey = "stale"
	if err := combat.submit("user", stale); !errors.Is(err, errAbyssLiveStale) {
		t.Fatalf("stale action error = %v, want errAbyssLiveStale", err)
	}

	invalidTarget := abyssLiveAction{
		SessionID: "session", Kind: "skill", AbilityID: "heal", TargetID: "enemy:0", Round: 3,
	}
	if err := combat.submit("user", invalidTarget); err == nil {
		t.Fatal("ally-targeted skill accepted an enemy target")
	}
}

func TestLiveTargetLookups(t *testing.T) {
	mobs := []*content.Mob{
		{Name: "alive", Stats: content.Stats{HP: 10}},
		{Name: "dead", Stats: content.Stats{HP: 0}},
	}
	if got := liveMobFromTarget("enemy:0", mobs); got != mobs[0] {
		t.Fatalf("liveMobFromTarget returned %p, want %p", got, mobs[0])
	}
	for _, target := range []string{"enemy:1", "enemy:9", "ally:user", "enemy:nope"} {
		if got := liveMobFromTarget(target, mobs); got != nil {
			t.Fatalf("liveMobFromTarget(%q) = %v, want nil", target, got)
		}
	}
}
