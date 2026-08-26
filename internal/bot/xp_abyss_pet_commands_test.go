package bot

import (
	"math"
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestParseAbyssPetCommand(t *testing.T) {
	tests := []struct {
		value string
		want  abyssPetCommand
		valid bool
	}{
		{" FOCUS ", abyssPetCommandFocus, true},
		{"guard", abyssPetCommandGuard, true},
		{"free", abyssPetCommandFree, true},
		{"forged", abyssPetCommandFree, false},
	}
	for _, test := range tests {
		got, valid := parseAbyssPetCommand(test.value)
		if got != test.want || valid != test.valid {
			t.Fatalf("parse %q = (%q, %t), want (%q, %t)", test.value, got, valid, test.want, test.valid)
		}
	}
}

func TestSelectAbyssPetTargetFollowsCommand(t *testing.T) {
	first := &content.Mob{Name: "first", Stats: content.Stats{HP: 10}}
	second := &content.Mob{Name: "second", Stats: content.Stats{HP: 10}}
	alive := []*content.Mob{first, second}

	focus := &activeUser{petCommand: abyssPetCommandFocus, petFocus: second.Name}
	if got := selectAbyssPetTarget(alive, focus, fixedCombatRandom{}); got != second {
		t.Fatalf("focus target = %v, want second", got)
	}

	guard := &activeUser{petCommand: abyssPetCommandGuard, lastAttackers: map[*content.Mob]bool{second: true}}
	if got := selectAbyssPetTarget(alive, guard, fixedCombatRandom{}); got != second {
		t.Fatalf("guard retaliation = %v, want second", got)
	}

	free := &activeUser{petCommand: abyssPetCommandFree}
	if got := selectAbyssPetTarget(alive, free, fixedCombatRandom{intn: 1}); got != second {
		t.Fatalf("free target = %v, want random second", got)
	}
}

func TestMitigateAbyssPetGuard(t *testing.T) {
	pet := &content.Mob{Name: "Bulwark", Stats: content.Stats{HP: 10}}
	au := &activeUser{
		u:          &UserInCombat{EscrowLoot: true, Pets: []*content.Mob{pet}},
		petCommand: abyssPetCommandGuard,
	}
	remaining, guarded := mitigateAbyssPetGuard(au, 100)
	if remaining != 85 || guarded != 15 {
		t.Fatalf("guard = (%d, %d), want (85, 15)", remaining, guarded)
	}
	remaining, guarded = mitigateAbyssPetGuard(au, math.MaxInt)
	if remaining <= 0 || guarded <= 0 || remaining+guarded != math.MaxInt {
		t.Fatalf("maximum damage guard overflowed: (%d, %d)", remaining, guarded)
	}
	au.petCommand = abyssPetCommandFree
	if remaining, guarded = mitigateAbyssPetGuard(au, 100); remaining != 100 || guarded != 0 {
		t.Fatalf("free command guarded damage: (%d, %d)", remaining, guarded)
	}
}

func TestApplyAbyssLivePetCommand(t *testing.T) {
	pet := &content.Mob{Name: "Bulwark", Stats: content.Stats{HP: 10}}
	enemy := &content.Mob{Name: "Watcher", Stats: content.Stats{HP: 10}}
	au := &activeUser{u: &UserInCombat{Nickname: "Delver", Pets: []*content.Mob{pet}}}
	var logs []string

	if !applyAbyssLivePetCommand(au, abyssLiveAction{AbilityID: "focus", TargetID: "enemy:0"}, []*content.Mob{enemy}, &logs) {
		t.Fatal("focus command was rejected")
	}
	if au.petCommand != abyssPetCommandFocus || au.petFocus != enemy.Name || !strings.Contains(logs[0], "FOCUS TARGET") {
		t.Fatalf("focus command not applied: %#v, %v", au, logs)
	}
	if applyAbyssLivePetCommand(au, abyssLiveAction{AbilityID: "focus", TargetID: "enemy:9"}, []*content.Mob{enemy}, &logs) {
		t.Fatal("invalid focus target was accepted")
	}
	pet.Stats.HP = 0
	if applyAbyssLivePetCommand(au, abyssLiveAction{AbilityID: "guard"}, []*content.Mob{enemy}, &logs) {
		t.Fatal("dead companion accepted a command")
	}
}
