package bot

import (
	"strings"
	"testing"

	"ts3news/internal/content"
)

func TestAbyssBossWeakpointsUnlockAtHalfHealth(t *testing.T) {
	if got := abyssBossWeakpointsFor("boss", 51, 100); got != nil {
		t.Fatalf("weakpoints above half health = %#v, want nil", got)
	}
	points := abyssBossWeakpointsFor("boss", 50, 100)
	if len(points) != 2 || points[0].ID != abyssWeakpointHead || points[1].ID != abyssWeakpointArms {
		t.Fatalf("half-health weakpoints = %#v", points)
	}
	if got := abyssBossWeakpointsFor("elite", 25, 100); got != nil {
		t.Fatalf("elite weakpoints = %#v, want nil", got)
	}
}

func TestAbyssBossWeakpointResolution(t *testing.T) {
	boss := &content.Mob{Name: "Gorgoroth", Type: content.MobBoss, Stats: content.Stats{HP: 50}, MaxHP: 100}
	head := resolveAbyssBossWeakpoint(abyssWeakpointHead, boss)
	if head.DamageMultiplier != 1.2 || head.Silence || !strings.Contains(head.Log, "+20%") {
		t.Fatalf("head effect = %+v", head)
	}
	arms := resolveAbyssBossWeakpoint(abyssWeakpointArms, boss)
	if arms.DamageMultiplier != 1 || !arms.Silence || !strings.Contains(arms.Log, "silenced") {
		t.Fatalf("arms effect = %+v", arms)
	}
	silenceAbyssBoss(boss)
	silenceAbyssBoss(boss)
	if !consumeAbyssBossSilence(boss) || consumeAbyssBossSilence(boss) {
		t.Fatal("boss silence must apply once and be consumed once")
	}
}

func TestAbyssLiveWeakpointValidation(t *testing.T) {
	combat := &abyssLiveCombat{
		participants: map[string]bool{"user": true},
		options:      map[string][]abyssLiveOption{"user": {{Kind: "attack", Name: "Attack", Target: "enemy"}}},
		enemies:      []abyssLiveCombatantView{{ID: "enemy:0", Role: "boss", HP: 50, MaxHP: 100}},
	}
	action := abyssLiveAction{Kind: "attack", TargetID: "enemy:0", Weakpoint: abyssWeakpointHead}
	if !combat.validTargetLocked("user", action) {
		t.Fatal("valid head weakpoint was rejected")
	}
	combat.enemies[0].HP = 51
	if combat.validTargetLocked("user", action) {
		t.Fatal("weakpoint above half health was accepted")
	}
	combat.enemies[0].HP = 50
	action.Weakpoint = "invented"
	if combat.validTargetLocked("user", action) {
		t.Fatal("unknown weakpoint was accepted")
	}
}
